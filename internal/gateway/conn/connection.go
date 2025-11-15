package conn

import (
	"net"
	"sync"
	"time"
)

type PlatformType int

const (
	PlatformTypeUnknown PlatformType = iota
	PlatformTypeMobile               // web端
	PlatformTypeWeb                  // iOS端
	PlatformTypePC                   // android端
	PlatformTypePAD                  // PC端
	PlatformTypeBot                  // 机器人、第三方接入
)

type connection struct {
	id           uint64
	fd           int
	userID       string
	platformType PlatformType
	deviceID     string
	expireTime   time.Time
	conn         net.Conn
	lastActiveAt time.Time // 最后活跃时间，用于心跳检测
	mu           sync.RWMutex
}

// updateActiveTime 更新最后活跃时间
func (c *connection) updateActiveTime() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastActiveAt = time.Now()
}

// getLastActiveTime 获取最后活跃时间
func (c *connection) getLastActiveTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastActiveAt
}

// close 关闭连接
func (c *connection) close() error {
	return c.conn.Close()
}

// 实现 Connection 接口

// ID 返回连接ID
func (c *connection) ID() uint64 {
	return c.id
}

// UserID 返回用户ID
func (c *connection) UserID() string {
	return c.userID
}

// PlatformType 返回平台类型
func (c *connection) PlatformType() PlatformType {
	return c.platformType
}

// DeviceID 返回设备ID
func (c *connection) DeviceID() string {
	return c.deviceID
}

// RemoteAddr 返回远程地址
func (c *connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Conn 返回底层连接
func (c *connection) Conn() net.Conn {
	return c.conn
}
