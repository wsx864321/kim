package gateway

import (
	"sync"
)

// connPool 连接池，管理所有活跃连接
type connPool struct {
	// connsByID key: connID, value: *connection
	connsByID sync.Map
	// connsByUserID key: userID, value: map[connID]*connection (一个用户可能有多个设备)
	connsByUserID sync.Map
	// mu rwmutex
	mu sync.RWMutex
}

func newConnPool() *connPool {
	return &connPool{}
}

// add 添加连接
func (p *connPool) add(conn *connection) {
	// 添加到 connID 索引
	p.connsByID.Store(conn.id, conn)

	// 添加到 userID 索引
	userConns, _ := p.connsByUserID.LoadOrStore(conn.userID, &sync.Map{})
	userConns.(*sync.Map).Store(conn.id, conn)
}

// remove 移除连接
func (p *connPool) remove(conn *connection) {
	// 从 connID 索引删除
	p.connsByID.Delete(conn.id)

	// 从 userID 索引删除
	if userConns, ok := p.connsByUserID.Load(conn.userID); ok {
		userConns.(*sync.Map).Delete(conn.id)
		// 如果该用户没有其他连接了，删除 userID 索引
		count := 0
		userConns.(*sync.Map).Range(func(key, value interface{}) bool {
			count++
			return false
		})
		if count == 0 {
			p.connsByUserID.Delete(conn.userID)
		}
	}
}

// getByID 根据连接ID获取连接
func (p *connPool) getByID(connID int) (*connection, bool) {
	conn, ok := p.connsByID.Load(connID)
	if !ok {
		return nil, false
	}
	return conn.(*connection), true
}

// getByUserID 根据用户ID获取所有连接
func (p *connPool) getByUserID(userID string) []*connection {
	userConns, ok := p.connsByUserID.Load(userID)
	if !ok {
		return nil
	}

	conns := make([]*connection, 0)
	userConns.(*sync.Map).Range(func(key, value interface{}) bool {
		conns = append(conns, value.(*connection))
		return true
	})
	return conns
}

// getAll 获取所有连接（用于广播等场景）
func (p *connPool) getAll() []*connection {
	conns := make([]*connection, 0)
	p.connsByID.Range(func(key, value interface{}) bool {
		conns = append(conns, value.(*connection))
		return true
	})
	return conns
}

// count 返回连接总数
func (p *connPool) count() int {
	count := 0
	p.connsByID.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
