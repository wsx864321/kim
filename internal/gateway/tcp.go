package gateway

import (
	"context"
	"errors"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/pkg/log"
	"google.golang.org/grpc"
)

// SessionServiceClient Session 服务客户端接口
type SessionServiceClient interface {
	RefreshSessionTTL(ctx context.Context, in *sessionpb.RefreshSessionTTLReq, opts ...grpc.CallOption) (*sessionpb.RefreshSessionTTLResp, error)
}

type TCPTransport struct {
	port               int
	ln                 *net.TCPListener
	ep                 *epoll
	connPool           *connPool
	handler            EventHandler
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	stopped            int32
	heartbeatTimeout   time.Duration
	numWorkers         int
	gatewayID          string               // Gateway 节点ID
	sessionClient      SessionServiceClient // Session 服务客户端
	timeWheel          *timeWheel           // 时间轮定时器
	refreshTTLInterval time.Duration        // 刷新TTL的间隔（默认60s）
}

// NewTCPTransport 创建 TCP Transport
func NewTCPTransport(port int, opts ...TCPOption) (*TCPTransport, error) {
	ep, err := newEpoll()
	if err != nil {
		return nil, err
	}

	ln, err := net.ListenTCP("tcp", &net.TCPAddr{Port: port})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	t := &TCPTransport{
		port:               port,
		ln:                 ln,
		ep:                 ep,
		connPool:           newConnPool(),
		ctx:                ctx,
		cancel:             cancel,
		heartbeatTimeout:   180 * time.Second,
		numWorkers:         2 * runtime.NumCPU(),
		gatewayID:          "default",        // 默认 Gateway ID
		refreshTTLInterval: 60 * time.Second, // 默认60秒刷新一次TTL
	}

	for _, opt := range opts {
		opt(t)
	}

	// 初始化时间轮（槽数等于间隔秒数，每1秒转动一次）
	slots := int(t.refreshTTLInterval.Seconds())
	if slots <= 0 {
		slots = 60 // 默认60个槽
	}
	t.timeWheel = newTimeWheel(t.refreshTTLInterval, slots)

	return t, nil
}

// Start 启动服务
func (t *TCPTransport) Start() error {
	if atomic.LoadInt32(&t.stopped) == 1 {
		return errors.New("transport already stopped")
	}

	// 启动 accept 处理协程
	t.acceptLoop()

	for i := 0; i < t.numWorkers; i++ {
		t.wg.Add(1)
		go t.eventLoop(i)
	}

	// 启动心跳检测协程
	t.wg.Add(1)
	go t.heartbeatLoop()

	// 启动时间轮定时器
	if t.timeWheel != nil && t.sessionClient != nil {
		t.timeWheel.start(t.refreshSessionTTL)
	}

	return nil
}

// Stop 停止服务
func (t *TCPTransport) Stop() error {
	if !atomic.CompareAndSwapInt32(&t.stopped, 0, 1) {
		return nil
	}

	t.cancel()
	t.ln.Close()

	// 停止时间轮
	if t.timeWheel != nil {
		t.timeWheel.stop()
	}

	// 关闭所有连接
	conns := t.connPool.getAll()
	for _, conn := range conns {
		conn.close()
	}

	t.wg.Wait()
	return nil
}

// acceptLoop accept 循环，多协程处理 accept
func (t *TCPTransport) acceptLoop() {
	for i := 0; i < runtime.NumCPU(); i++ {
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()

			for {
				select {
				case <-t.ctx.Done():
					return
				default:
					conn, err := t.ln.AcceptTCP()
					if err != nil {
						if atomic.LoadInt32(&t.stopped) == 1 {
							return
						}
						if ne, ok := err.(net.Error); ok && ne.Temporary() {
							log.Warn(context.Background(), "accept temp err", log.String("error", err.Error()))
							time.Sleep(10 * time.Millisecond)
							continue
						}
						log.Error(context.Background(), "accept err", log.String("error", err.Error()))
						return
					}

					// 设置 TCP 选项
					conn.SetNoDelay(true)
					conn.SetKeepAlive(true)
					conn.SetKeepAlivePeriod(30 * time.Second)

					// 异步处理新连接（避免阻塞 accept）
					go t.handleNewConnection(conn)
				}
			}
		}()
	}

}

// handleNewConnection 处理新连接（在独立协程中，避免阻塞 accept）
func (t *TCPTransport) handleNewConnection(conn net.Conn) {
	ctx := context.Background()
	// 设置初始读取超时（用于读取登录包）
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 读取并验证登录包
	packet, err := DecodePacket(conn)
	if err != nil {
		log.Warn(ctx, "decode logic packet failed", log.String("error", err.Error()), log.String("remote", conn.RemoteAddr().String()))
		conn.Close()
		return
	}

	if packet.MsgType != MsgTypeLogin {
		log.Warn(ctx, "first packet must be logic", log.Any("msgType", packet.MsgType))
		conn.Close()
		return
	}
	session, err := t.handler.OnLogin(ctx, conn, packet.Body, t.gatewayID)
	if err != nil {
		log.Warn(ctx, "logic failed", log.String("error", err.Error()), log.String("remote", conn.RemoteAddr().String()))
		conn.Close()
		return
	}

	// 解析平台类型
	var platformType PlatformType
	switch session.DeviceType {
	case sessionpb.DeviceType_DEVICE_TYPE_WEB:
		platformType = PlatformTypeWeb
	case sessionpb.DeviceType_DEVICE_TYPE_MOBILE:
		platformType = PlatformTypeMobile
	case sessionpb.DeviceType_DEVICE_TYPE_PC:
		platformType = PlatformTypePC
	case sessionpb.DeviceType_DEVICE_TYPE_BOT:
		platformType = PlatformTypeBot
	case sessionpb.DeviceType_DEVICE_TYPE_PAD:
		platformType = PlatformTypePAD
	default:
		platformType = PlatformTypeUnknown
	}

	// 创建连接对象
	expireTime := time.Unix(session.ExpireAt, 0)
	c := &connection{
		id:           session.GetConnId(),
		userID:       session.GetUserId(),
		platformType: platformType,
		deviceID:     session.GetDeviceId(),
		expireTime:   expireTime,
		conn:         conn,
		lastActiveAt: time.Now(),
	}

	//  添加到连接池
	t.connPool.add(c)

	//  添加到 epoll
	if err := t.ep.add(c); err != nil {
		log.Error(ctx, "add to epoll failed", log.String("error", err.Error()))
		t.connPool.remove(c)
		conn.Close()
		return
	}

	// 添加到时间轮，用于定期刷新Session TTL
	if t.timeWheel != nil {
		t.timeWheel.add(c)
	}

	// 通知上层连接建立
	if t.handler != nil {
		if err := t.handler.OnConnect(ctx, c); err != nil {
			log.Warn(ctx, "onConnect handler failed", log.String("error", err.Error()))
		}
	}

	log.Info(ctx, "new connection established", log.String("userID", session.UserId), log.Uint64("connID", c.id), log.String("deviceID", session.DeviceId))
}

// eventLoop epoll 事件循环（多协程处理）
func (t *TCPTransport) eventLoop(workerID int) {
	defer t.wg.Done()

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			// 等待 epoll 事件，超时时间 100ms
			conns, err := t.ep.wait(100)
			if err != nil {
				if atomic.LoadInt32(&t.stopped) == 1 {
					return
				}
				log.Warn(context.Background(), "epoll wait error", log.String("error", err.Error()), log.Int("worker", workerID))
				continue
			}

			// 处理就绪的连接
			for _, conn := range conns {
				// todo 生成一个带tracing的上下文
				ctx := context.Background()
				t.handleConnectionRead(ctx, conn)
			}
		}
	}
}

// handleConnectionRead 处理连接读取
func (t *TCPTransport) handleConnectionRead(ctx context.Context, conn *connection) {
	// 更新活跃时间
	conn.updateActiveTime()

	// 读取数据包（不设置超时，使用默认的）
	packet, err := DecodePacket(conn.conn)
	if err != nil { // 不管是什么原因的错误，都断开连接（读取超时、数据错误、连接关闭等等）
		log.Debug(context.Background(), "read packet failed", log.String("error", err.Error()), log.Uint64("connID", conn.id))
		t.handleDisconnect(ctx, conn, err.Error())
		return
	}

	// 处理不同类型的消息
	switch packet.MsgType {
	case MsgTypePing:
		// 心跳包，回复 Pong
		t.sendPong(ctx, conn)
	case MsgTypeLogout:
		// 登出
		t.handleDisconnect(ctx, conn, "logout")
	case MsgTypeUpstream:
		// 上行消息（客户端→服务端），通知上层处理
		if t.handler != nil {
			if err := t.handler.OnMessage(ctx, conn, packet.Body); err != nil {
				log.Warn(context.Background(), "onMessage handler failed", log.String("error", err.Error()), log.Uint64("connID", conn.id))
			}
		}
	default:
		log.Warn(context.Background(), "unknown msg type", log.Any("msgType", packet.MsgType), log.Uint64("connID", conn.id))
	}
}

// sendPong 发送心跳响应
func (t *TCPTransport) sendPong(ctx context.Context, conn *connection) {
	pongPacket := Packet{
		MsgType: MsgTypePong,
		Body:    nil,
	}
	data, err := EncodePacket(pongPacket)
	if err != nil {
		log.Warn(context.Background(), "encode pong failed", log.String("error", err.Error()))
		return
	}
	conn.conn.Write(data)

	// 更新活跃时间
	conn.updateActiveTime()

	// 通知上层收到心跳
	if t.handler != nil {
		t.handler.OnHeartbeat(ctx, conn)
	}
}

// handleDisconnect 处理连接断开
func (t *TCPTransport) handleDisconnect(ctx context.Context, conn *connection, reason string) {
	// 从时间轮移除
	if t.timeWheel != nil {
		t.timeWheel.remove(conn.id)
	}

	// 从 epoll 移除
	t.ep.remove(conn)

	// 从连接池移除
	t.connPool.remove(conn)

	// 关闭连接
	conn.close()

	// 通知上层
	if t.handler != nil {
		t.handler.OnDisconnect(ctx, conn, reason)
	}

	log.Info(
		ctx,
		"connection closed",
		log.Uint64("connID", conn.id),
		log.String("userID", conn.userID),
		log.String("reason", reason),
	)
}

// heartbeatLoop 心跳检测循环
func (t *TCPTransport) heartbeatLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			conns := t.connPool.getAll()

			ctx := context.Background()
			log.Info(ctx, "heartbeat check", log.Int("connections", len(conns)))
			for _, conn := range conns {
				lastActive := conn.getLastActiveTime()
				if now.Sub(lastActive) > t.heartbeatTimeout {
					// 心跳超时
					if t.handler != nil {
						t.handler.OnHeartbeatTimeout(ctx, conn)
					}
					t.handleDisconnect(ctx, conn, "heartbeat timeout")
				}
			}
			log.Info(ctx, "heartbeat check completed")
		}
	}
}

// SetHandler 设置事件处理器
func (t *TCPTransport) SetHandler(h EventHandler) {
	t.handler = h
}

// Send 发送消息到指定连接
func (t *TCPTransport) Send(ctx context.Context, connID int, data []byte) error {
	conn, ok := t.connPool.getByID(connID)
	if !ok {
		return errors.New("connection not found")
	}

	packet := Packet{
		MsgType: MsgTypePush,
		Body:    data,
	}

	encoded, err := EncodePacket(packet)
	if err != nil {
		return err
	}

	if _, err := conn.conn.Write(encoded); err != nil {
		log.Warn(ctx, "send message failed", log.String("error", err.Error()), log.Uint64("connID", uint64(connID)))
		return err
	}

	return nil
}

// BatchSend 批量发送消息到多个连接（发送相同消息）
func (t *TCPTransport) BatchSend(ctx context.Context, connIDs []int, data []byte) ([]uint64, error) {
	if len(connIDs) == 0 {
		return nil, nil
	}

	// 编码数据包（所有连接使用相同消息）
	packet := Packet{
		MsgType: MsgTypePush,
		Body:    data,
	}

	encoded, err := EncodePacket(packet)
	if err != nil {
		return nil, err
	}

	// 批量发送
	failConns := make([]uint64, 0)
	for _, connID := range connIDs {
		conn, ok := t.connPool.getByID(connID)
		if !ok {
			log.Warn(ctx, "connection not found", log.Uint64("connID", uint64(connID)))
			failConns = append(failConns, uint64(connID))
			continue
		}

		if _, err := conn.conn.Write(encoded); err != nil {
			log.Warn(ctx, "send batch message failed", log.String("error", err.Error()), log.Uint64("connID", uint64(connID)))
			failConns = append(failConns, uint64(connID))
		}
	}

	return failConns, nil
}

// CloseConn 关闭指定连接
func (t *TCPTransport) CloseConn(ctx context.Context, connID int) error {
	conn, ok := t.connPool.getByID(connID)
	if !ok {
		return errors.New("connection not found")
	}

	// 使用 handleDisconnect 确保完整清理
	t.handleDisconnect(ctx, conn, "closed by server")
	return nil
}

// refreshSessionTTL 刷新Session TTL的回调函数
func (t *TCPTransport) refreshSessionTTL(conns []*connection) {
	if t.sessionClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	for _, conn := range conns {
		if ctx.Err() != nil { // 上下文已超时或取消，停止处理
			log.Warn(ctx, "refresh session timeout")
			return
		}
		// 检查连接是否仍然有效
		if _, ok := t.connPool.getByID(int(conn.id)); !ok {
			// 连接已断开，跳过
			continue
		}

		// 获取最后活跃时间
		lastActiveAt := conn.getLastActiveTime().Unix()

		err := t.handler.OnRefreshSession(ctx, conn, lastActiveAt)
		if err != nil {
			// 刷新失败，需要关闭连接
			t.handleDisconnect(ctx, conn, "refresh session timeout")
			continue
		}
		// 刷新成功，将连接重新添加到时间轮，以便下次继续刷新
		if t.timeWheel != nil {
			t.timeWheel.add(conn)
		}

		log.Debug(
			ctx,
			"session TTL refreshed",
			log.Uint64("connID", conn.id),
			log.String("userID", conn.userID),
		)
	}
}
