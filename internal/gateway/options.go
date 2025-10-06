package gateway

import (
	"time"
)

type TCPOption func(transport *TCPTransport)

// WithTCPHeartbeatTimeout 设置心跳超时时间
func WithTCPHeartbeatTimeout(d time.Duration) TCPOption {
	return func(o *TCPTransport) {
		o.heartbeatTimeout = d
	}
}

// WithTCPNumWorkers 设置工作线协程数量
func WithTCPNumWorkers(n int) TCPOption {
	return func(o *TCPTransport) {
		o.numWorkers = n
	}
}

// WithGatewayID 设置 Gateway 节点ID
func WithGatewayID(gatewayID string) TCPOption {
	return func(o *TCPTransport) {
		o.gatewayID = gatewayID
	}
}

// WithSessionClient 设置 Session 服务客户端
func WithSessionClient(client SessionServiceClient) TCPOption {
	return func(o *TCPTransport) {
		o.sessionClient = client
	}
}
