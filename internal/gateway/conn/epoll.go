package conn

import (
	"golang.org/x/sys/unix"
	"net"
	"sync"
	"sync/atomic"
)

type epoll struct {
	// fd epoll的fd
	fd int
	// tales key:fd value:*connection
	tables sync.Map
	// count 当前epoll中监听的fd数量
	count int32
}

func newEpoll() (*epoll, error) {
	fd, err := unix.EpollCreate(0)
	if err != nil {
		return nil, err
	}
	return &epoll{
		fd:     fd,
		tables: sync.Map{},
		count:  0,
	}, nil
}

// add 添加新的fd到epoll中，当前先采用水平触发模式
func (e *epoll) add(conn *connection) error {
	// 这里只有tcp链接才会走epoll，因此可以直接断言获取fd
	file, err := conn.conn.(*net.TCPConn).File()
	if err != nil {
		return err
	}
	fd := int32(file.Fd())
	event := &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     fd,
	}
	err = unix.EpollCtl(e.fd, unix.EPOLL_CTL_ADD, int(fd), event)
	if err != nil {
		return err
	}

	e.tables.Store(fd, conn)
	atomic.AddInt32(&e.count, 1)

	return nil
}

// remove 从epoll中删除一个fd
func (e *epoll) remove(conn *connection) error {
	// 这里只有tcp链接才会走epoll，因此可以直接断言获取fd
	file, err := conn.conn.(*net.TCPConn).File()
	if err != nil {
		return err
	}

	err = unix.EpollCtl(e.fd, unix.EPOLL_CTL_DEL, int(file.Fd()), nil)
	if err != nil {
		return err
	}

	e.tables.Delete(int32(file.Fd()))
	atomic.AddInt32(&e.count, -1)

	return nil
}

// wait 等待epoll事件的发生
func (e *epoll) wait(mesc int) ([]*connection, error) {
	// 这里的100是一次性处理的最大事件数，可以根据业务调整,后续改成配置项
	events := make([]unix.EpollEvent, 100)
	n, err := unix.EpollWait(e.fd, events, mesc)

	conns := make([]*connection, 0, n)
	for i := 0; i < n; i++ {
		if conn, ok := e.tables.Load(events[i].Fd); ok {
			conns = append(conns, conn.(*connection))
		}
	}
	return conns, err
}

// getCount 返回当前epoll中监听的fd数量
func (e *epoll) getCount() int32 {
	return atomic.LoadInt32(&e.count)
}
