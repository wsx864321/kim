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

// WithRefreshTTLInterval 设置刷新Session TTL的间隔时间
func WithRefreshTTLInterval(d time.Duration) TCPOption {
	return func(o *TCPTransport) {
		o.refreshTTLInterval = d
		// 如果时间轮已创建，需要重新创建
		if o.timeWheel != nil {
			o.timeWheel.stop()
		}
		// 重新创建时间轮
		slots := int(d.Seconds())
		if slots <= 0 {
			slots = 60 // 默认60个槽
		}
		o.timeWheel = newTimeWheel(d, slots)
	}
}
