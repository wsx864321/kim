package gateway

import (
	"net"
)

// Transport 底层传输抽象接口
type Transport interface {
	// Start 启动服务
	Start() error
	// Stop 停止服务
	Stop() error
	// SetHandler 设置事件回调
	SetHandler(h EventHandler)
	// Send 发送消息到指定连接
	Send(connID int, data []byte) error
	// BatchSend 批量发送消息到多个连接（发送相同消息）
	BatchSend(connIDs []int, data []byte) error
	// CloseConn 关闭指定连接
	CloseConn(connID int) error
}

// EventHandler 定义 Transport 生命周期回调
type EventHandler interface {
	// OnConnect 连接建立且已鉴权
	OnConnect(conn Connection) error
	// OnMessage 收到业务消息
	OnMessage(conn Connection, data []byte) error
	// OnDisconnect 连接断开
	OnDisconnect(conn Connection, reason string)
	// OnHeartbeatTimeout 心跳超时
	OnHeartbeatTimeout(conn Connection)
	// OnHeartbeat 收到心跳消息
	OnHeartbeat(conn Connection)
}

// Connection 连接信息接口，提供给 EventHandler 使用（屏蔽一些参数）
type Connection interface {
	// ID 返回连接ID
	ID() uint64
	// UserID 返回用户ID
	UserID() string
	// PlatformType 返回平台类型
	PlatformType() PlatformType
	// DeviceID 返回设备ID
	DeviceID() string
	// RemoteAddr 返回远程地址
	RemoteAddr() net.Addr
	// Conn 返回底层连接
	Conn() net.Conn
}
