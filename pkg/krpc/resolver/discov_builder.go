package resolver

import (
	"context"
	"fmt"
	"github.com/wsx864321/kim/pkg/krpc/registry"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

type RegistryBuilder struct {
	registry registry.Registrar
}

// NewRegistryBuilder ...
func NewRegistryBuilder(d registry.Registrar) resolver.Builder {
	return &RegistryBuilder{
		registry: d,
	}
}

func (r *RegistryBuilder) Scheme() string {
	return RegistryBuilderScheme
}

func (r *RegistryBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	r.registry.GetService(context.TODO(), r.getServiceName(target))
	serviceName := r.getServiceName(target)
	listener := func() {
		service := r.registry.GetService(context.TODO(), serviceName)
		var addrs []resolver.Address
		for _, item := range service.Endpoints {
			attr := attributes.New("weight", item.Weight)
			addr := resolver.Address{
				Addr:       fmt.Sprintf("%s:%d", item.IP, item.Port),
				Attributes: attr,
			}

			addrs = append(addrs, addr)
		}

		cc.UpdateState(resolver.State{
			Addresses: addrs,
		})
	}

	r.registry.AddListener(context.TODO(), listener)
	listener()

	return r, nil
}

func (r *RegistryBuilder) getServiceName(target resolver.Target) string {
	return target.Endpoint()
}

func (r *RegistryBuilder) Close() {
}

func (r *RegistryBuilder) ResolveNow(options resolver.ResolveNowOptions) {
}
