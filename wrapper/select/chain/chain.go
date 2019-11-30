package chain

import (
	"context"
	"strings"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
)

type chainWrapper struct {
	opts Options
	client.Client
}

func (w *chainWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	if val, ok := metadata.Get(ctx, w.opts.chainKey); ok && len(val) > 0 {
		chains := strings.Split(val, w.opts.chainSep)
		nOpts := append(opts, client.WithSelectOption(
			selector.WithFilter(w.filterChain(chains)),
		))
		return w.Client.Call(ctx, req, rsp, nOpts...)
	}

	return w.Client.Call(ctx, req, rsp, opts...)
}

func (w *chainWrapper) filterChain(chains []string) selector.Filter {
	return func(old []*registry.Service) []*registry.Service {
		var services []*registry.Service

		chain := ""
		for _, service := range old {
			serv := new(registry.Service)
			var nodes []*registry.Node

			for _, node := range service.Nodes {
				if node.Metadata == nil {
					continue
				}

				val := node.Metadata[w.opts.labelKey]
				if len(val) > 0 && (chain == val || inArray(val, chains)) {
					chain = val
					nodes = append(nodes, node)
				}
			}

			// only add service if there's some nodes
			if len(nodes) > 0 {
				// copy
				*serv = *service
				serv.Nodes = nodes
				services = append(services, serv)
			}
		}

		if len(services) == 0 {
			return old
		}

		return services
	}
}

func inArray(s string, d []string) bool {
	for _, v := range d {
		if s == v {
			return true
		}
	}
	return false
}

func NewClientWrapper(opts ...Option) client.Wrapper {
	options := newOptions(opts...)
	return func(c client.Client) client.Client {
		return &chainWrapper{
			opts:   options,
			Client: c,
		}
	}
}