package gateway

import (
	"context"
	"errors"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/pkg/log"
	"github.com/wsx864321/kim/pkg/xerr"
	"google.golang.org/grpc"
)

// SessionServiceClient Session 服务客户端接口
type SessionServiceClient interface {
	Login(ctx context.Context, in *sessionpb.LoginReq, opts ...grpc.CallOption) (*sessionpb.LoginResp, error)
}

type TCPTransport struct {
	port             int
	ln               *net.TCPListener
	ep               *epoll
	connPool         *connPool
	handler          EventHandler
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	stopped          int32
	heartbeatTimeout time.Duration
	numWorkers       int
	gatewayID        string               // Gateway 节点ID
	sessionClient    SessionServiceClient // Session 服务客户端
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
		port:             port,
		ln:               ln,
		ep:               ep,
		connPool:         newConnPool(),
		ctx:              ctx,
		cancel:           cancel,
		heartbeatTimeout: 180 * time.Second,
		numWorkers:       2 * runtime.NumCPU(),
		gatewayID:        "default", // 默认 Gateway ID
	}

	for _, opt := range opts {
		opt(t)
	}

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

	return nil
}

// Stop 停止服务
func (t *TCPTransport) Stop() error {
	if !atomic.CompareAndSwapInt32(&t.stopped, 0, 1) {
		return nil
	}

	t.cancel()
	t.ln.Close()

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
	// 设置初始读取超时（用于读取登录包）
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 读取并验证登录包
	packet, err := DecodePacket(conn)
	if err != nil {
		log.Warn(context.Background(), "decode login packet failed", log.String("error", err.Error()), log.String("remote", conn.RemoteAddr().String()))
		conn.Close()
		return
	}

	if packet.MsgType != MsgTypeLogin {
		log.Warn(context.Background(), "first packet must be login", log.Any("msgType", packet.MsgType))
		conn.Close()
		return
	}

	// 生成临时连接ID（用于 Login 请求）
	tempConnID := connIDGenerator.NextID()

	// 调用 Session.Login
	// packet.Body 应该是客户端发送的序列化后的 AuthInfo
	loginReq := &sessionpb.LoginReq{
		Payload:    packet.Body, // 直接使用 packet.Body（已序列化的 AuthInfo）
		ConnId:     strconv.FormatUint(tempConnID, 10),
		RemoteAddr: conn.RemoteAddr().String(),
		GatewayId:  t.gatewayID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loginResp, err := t.sessionClient.Login(ctx, loginReq)
	if err != nil {
		log.Warn(ctx, "call session login failed", log.String("error", err.Error()), log.String("remote", conn.RemoteAddr().String()))
		conn.Close()
		return
	}

	// 检查登录结果
	if loginResp.Code != xerr.OK.Code() {
		log.Warn(ctx, "session login failed", log.Int("code", int(loginResp.Code)), log.String("message", loginResp.Message), log.String("remote", conn.RemoteAddr().String()))
		conn.Close()
		return
	}

	// 3. 登录成功，从响应中获取会话信息
	session := loginResp.GetData().GetSession()
	if session == nil {
		log.Error(ctx, "session data is nil in login response")
		conn.Close()
		return
	}

	// 解析连接ID（从 session 中获取，或使用临时ID）
	connID := tempConnID
	if session.ConnId != "" {
		parsedID, err := strconv.ParseUint(session.ConnId, 10, 64)
		if err == nil {
			connID = parsedID
		}
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

	// 4. 创建连接对象
	expireTime := time.Unix(session.ExpireAt, 0)
	c := &connection{
		id:           connID,
		userID:       session.GetUserId(),
		platformType: platformType,
		deviceID:     session.GetDeviceId(),
		expireTime:   expireTime,
		conn:         conn,
		lastActiveAt: time.Now(),
	}

	// 5. 添加到连接池
	t.connPool.add(c)

	// 6. 添加到 epoll
	if err := t.ep.add(c); err != nil {
		log.Error(ctx, "add to epoll failed", log.String("error", err.Error()))
		t.connPool.remove(c)
		conn.Close()
		return
	}

	// 7. 通知上层连接建立
	if t.handler != nil {
		if err := t.handler.OnConnect(c); err != nil {
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
	if err != nil {
		// 读取失败，可能是连接关闭或超时
		if errors.Is(err, net.ErrClosed) {
			return
		}
		log.Debug(context.Background(), "read packet failed", log.String("error", err.Error()), log.Uint64("connID", conn.id))
		t.handleDisconnect(conn, "read error")
		return
	}

	// 处理不同类型的消息
	switch packet.MsgType {
	case MsgTypePing:
		// 心跳包，回复 Pong
		t.sendPong(conn)
	case MsgTypeLogout:
		// 登出
		t.handleDisconnect(conn, "logout")
	case MsgTypeUpstream:
		// 上行消息（客户端→服务端），通知上层处理
		if t.handler != nil {
			if err := t.handler.OnMessage(conn, packet.Body); err != nil {
				log.Warn(context.Background(), "onMessage handler failed", log.String("error", err.Error()), log.Uint64("connID", conn))
			}
		}
	default:
		log.Warn(context.Background(), "unknown msg type", log.Any("msgType", packet.MsgType), log.Uint64("connID", conn.id))
	}
}

// sendPong 发送心跳响应
func (t *TCPTransport) sendPong(conn *connection) {
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
		t.handler.OnHeartbeat(conn)
	}
}

// handleDisconnect 处理连接断开
func (t *TCPTransport) handleDisconnect(conn *connection, reason string) {
	// 从 epoll 移除
	t.ep.remove(conn)

	// 从连接池移除
	t.connPool.remove(conn)

	// 关闭连接
	conn.close()

	// 通知上层
	if t.handler != nil {
		t.handler.OnDisconnect(conn, reason)
	}

	log.Info(
		context.Background(),
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

			for _, conn := range conns {
				lastActive := conn.getLastActiveTime()
				if now.Sub(lastActive) > t.heartbeatTimeout {
					// 心跳超时
					if t.handler != nil {
						t.handler.OnHeartbeatTimeout(conn)
					}
					t.handleDisconnect(conn, "heartbeat timeout")
				}
			}
		}
	}
}

// SetHandler 设置事件处理器
func (t *TCPTransport) SetHandler(h EventHandler) {
	t.handler = h
}

// Send 发送消息到指定连接
func (t *TCPTransport) Send(connID int, data []byte) error {
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
		log.Warn(context.Background(), "send message failed", log.String("error", err.Error()), log.Uint64("connID", uint64(connID)))
		return err
	}

	return nil
}

// BatchSend 批量发送消息到多个连接（发送相同消息）
func (t *TCPTransport) BatchSend(connIDs []int, data []byte) error {
	if len(connIDs) == 0 {
		return nil
	}

	// 编码数据包（所有连接使用相同消息）
	packet := Packet{
		MsgType: MsgTypePush,
		Body:    data,
	}

	encoded, err := EncodePacket(packet)
	if err != nil {
		return err
	}

	// 批量发送
	var firstErr error
	for _, connID := range connIDs {
		conn, ok := t.connPool.getByID(connID)
		if !ok {
			if firstErr == nil {
				firstErr = errors.New("connection not found")
			}
			continue
		}

		if _, err := conn.conn.Write(encoded); err != nil {
			log.Warn(context.Background(), "send batch message failed", log.String("error", err.Error()), log.Uint64("connID", uint64(connID)))
			if firstErr == nil {
				firstErr = err
			}
			// 继续发送其他连接，不中断
		}
	}

	return firstErr // 返回第一个错误（如果有）
}

// CloseConn 关闭指定连接
func (t *TCPTransport) CloseConn(connID int) error {
	conn, ok := t.connPool.getByID(connID)
	if !ok {
		return errors.New("connection not found")
	}

	// 使用 handleDisconnect 确保完整清理
	t.handleDisconnect(conn, "closed by server")
	return nil
}
