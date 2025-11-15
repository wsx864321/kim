package krpc

import (
	"github.com/wsx864321/kim/pkg/krpc/registry"
	"google.golang.org/grpc"
)

type serverOptions struct {
	serviceName string
	port        int
	weight      int
	registry    registry.Registrar
}

type clientOptions struct {
	serviceName  string
	direct       bool
	url          string
	registry     registry.Registrar
	interceptors []grpc.UnaryClientInterceptor
}

type ServerOption func(opts *serverOptions)

type ClientOption func(opts *clientOptions)

// WithServiceName set serviceName
func WithServiceName(serviceName string) ServerOption {
	return func(opts *serverOptions) {
		opts.serviceName = serviceName
	}
}

// WithPort set port
func WithPort(port int) ServerOption {
	return func(opts *serverOptions) {
		opts.port = port
	}
}

// WithWeight set weight
func WithWeight(weight int) ServerOption {
	return func(opts *serverOptions) {
		opts.weight = weight
	}
}

// WithRegistry set registry
func WithRegistry(registry registry.Registrar) ServerOption {
	return func(opts *serverOptions) {
		opts.registry = registry
	}
}

// WithClientRegistry set registry
func WithClientRegistry(registry registry.Registrar) ClientOption {
	return func(opts *clientOptions) {
		opts.registry = registry
	}
}

// WithClientInterceptors set interceptors
func WithClientInterceptors(interceptors ...grpc.UnaryClientInterceptor) ClientOption {
	return func(opts *clientOptions) {
		opts.interceptors = interceptors
	}
}

// WithClientServiceName set serviceName
func WithClientServiceName(serviceName string) ClientOption {
	return func(opts *clientOptions) {
		opts.serviceName = serviceName
	}
}

// WithDirect 是否直连服务地址
func WithDirect(direct bool) ClientOption {
	return func(opts *clientOptions) {
		opts.direct = direct
	}
}

// WithURL 直接设置服务地址
func WithURL(url string) ClientOption {
	return func(opts *clientOptions) {
		opts.url = url
	}
}
