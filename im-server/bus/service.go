package bus

import (
    "fmt"
    "io"
    "net"
    "net/http"
    "time"
    "strings"
    "sync"

    "angrymiao-ai-server/src/configs"
    "angrymiao-ai-server/src/core/auth"
    "angrymiao-ai-server/src/core/utils"

    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    "google.golang.org/grpc"
)

// ImMessage 双向流消息结构（JSON）
type ImMessage struct {
    Event       string            `json:"event"`
    SessionID   string            `json:"session_id"`
    Headers     map[string]string `json:"headers,omitempty"`
    MessageType int               `json:"message_type,omitempty"`
    Payload     []byte            `json:"payload,omitempty"`
}

// IMBusService 用于 gRPC 注册时的类型校验（空接口即可）
type IMBusService interface{}

// IMBusServer gRPC 总线 + WS 服务
type IMBusServer struct {
    config     *configs.Config
    logger     *utils.Logger
    server     *grpc.Server
    upgrader   *websocket.Upgrader
    authToken  *auth.AuthToken

    // 会话连接表：sessionID -> wsConn
    conns      sync.Map

    // AI 总线连接（单路）
    busMu      sync.Mutex
    busStream  grpc.ServerStream
}

func NewIMBusServer(cfg *configs.Config, logger *utils.Logger) *IMBusServer {
    return &IMBusServer{
        config:    cfg,
        logger:    logger,
        upgrader:  &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
        authToken: auth.NewAuthToken(cfg.Server.Token),
    }
}

// Start 启动 gRPC 总线服务
func (s *IMBusServer) Start(lis net.Listener) error {
    // 使用默认编解码，依赖客户端传递 content-subtype=\"json\"，并通过 encoding.RegisterCodec 注册 JSON
    s.server = grpc.NewServer()
    s.registerService()
    go func() {
        if err := s.server.Serve(lis); err != nil {
            s.logger.Error("gRPC 总线服务终止: %v", err)
        }
    }()
    return nil
}

func (s *IMBusServer) Stop() {
    if s.server != nil {
        s.server.GracefulStop()
    }
}

// registerService 动态注册服务描述
func (s *IMBusServer) registerService() {
    svc := &grpc.ServiceDesc{
        ServiceName: "im.IMBus",
        HandlerType: (*IMBusService)(nil),
        Streams: []grpc.StreamDesc{
            {
                StreamName:    "MessageBus",
                ServerStreams: true,
                ClientStreams: true,
                Handler:       s.messageBusHandler,
            },
        },
    }
    s.server.RegisterService(svc, s)
}

// messageBusHandler 处理双向流
func (s *IMBusServer) messageBusHandler(srv interface{}, stream grpc.ServerStream) error {
    s.busMu.Lock()
    s.busStream = stream
    s.busMu.Unlock()
    s.logger.Info("AI 总线连接已建立")

    // 读取 AI -> 客户端 的下行消息
    for {
        msg := &ImMessage{}
        if err := stream.RecvMsg(msg); err != nil {
            if err == io.EOF {
                s.logger.Warn("AI 总线连接关闭")
                return nil
            }
            s.logger.Error("读取 AI 总线消息失败: %v", err)
            return err
        }
        s.forwardToClient(msg)
    }
}

// forwardToClient 将 AI 下行数据写入对应 WS 连接
func (s *IMBusServer) forwardToClient(msg *ImMessage) {
    if msg == nil || msg.SessionID == "" {
        return
    }
    val, ok := s.conns.Load(msg.SessionID)
    if !ok {
        s.logger.Warn("收到下行消息但会话不存在: %s", msg.SessionID)
        return
    }
    conn := val.(*websocket.Conn)
    if msg.Event == "session_close" {
        _ = conn.Close()
        s.conns.Delete(msg.SessionID)
        return
    }
    // 默认写文本/二进制
    mtype := websocket.TextMessage
    if msg.MessageType != 0 {
        mtype = msg.MessageType
    }
    if err := conn.WriteMessage(mtype, msg.Payload); err != nil {
        s.logger.Warn("向会话 %s 写消息失败: %v", msg.SessionID, err)
    }
}

// HandleWebSocket WS 入口（HTTP）
func (s *IMBusServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // 兼容浏览器通过 query 传递 Header
    if s.config.Transport.WebSocket.Browser {
        q := r.URL.Query()
        if v := q.Get("device-id"); v != "" { r.Header.Set("Device-Id", v) }
        if v := q.Get("client-id"); v != "" { r.Header.Set("Client-Id", v) }
        if v := q.Get("session-id"); v != "" { r.Header.Set("Session-Id", v) }
        if v := q.Get("transport-type"); v != "" { r.Header.Set("Transport-Type", v) }
        if v := q.Get("token"); v != "" { r.Header.Set("Authorization", "Bearer "+v); r.Header.Set("Token", v) }
    }

    userID, err := s.verifyJWTAuth(r)
    if err != nil {
        s.logger.Warn("WebSocket 认证失败: %v device-id: %s", err, r.Header.Get("Device-Id"))
        http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
        return
    }
    r.Header.Set("User-Id", fmt.Sprintf("%d", userID))

    // 在升级前，确保 AI 总线已连接，否则返回 503
    if !s.waitBusConnected(2 * time.Second) {
        s.logger.Warn("AI 总线未连接，拒绝 WS 升级: device=%s", r.Header.Get("Device-Id"))
        http.Error(w, "Service Unavailable: AI bus not connected", http.StatusServiceUnavailable)
        return
    }

    conn, err := s.upgrader.Upgrade(w, r, nil)
    if err != nil {
        s.logger.Error("WebSocket 升级失败: %v", err)
        return
    }

    sessionID := r.Header.Get("Session-Id")
    if sessionID == "" { sessionID = uuid.NewString() }
    s.conns.Store(sessionID, conn)

    s.logger.Info("WS 会话建立: session=%s device=%s", sessionID, r.Header.Get("Device-Id"))

    // 通知 AI 会话打开
    s.busSend(&ImMessage{
        Event:     "session_open",
        SessionID: sessionID,
        Headers: map[string]string{
            "Device-Id":     r.Header.Get("Device-Id"),
            "Client-Id":     r.Header.Get("Client-Id"),
            "Session-Id":    sessionID,
            "Transport-Type": r.Header.Get("Transport-Type"),
            "Authorization": r.Header.Get("Authorization"),
            "User-Id":       r.Header.Get("User-Id"),
        },
    })

    // 读循环: 客户端 -> AI
    go func() {
        defer func() {
            s.conns.Delete(sessionID)
            s.busSend(&ImMessage{Event: "session_close", SessionID: sessionID})
            _ = conn.Close()
        }()
        for {
            mtype, data, err := conn.ReadMessage()
            if err != nil {
                if websocket.IsUnexpectedCloseError(err) {
                    s.logger.Warn("WS 连接异常关闭: %v", err)
                }
                return
            }
            s.busSend(&ImMessage{
                Event:       "data",
                SessionID:   sessionID,
                MessageType: mtype,
                Payload:     data,
            })
        }
    }()
}

// verifyJWTAuth 验证JWT并返回 userID
func (s *IMBusServer) verifyJWTAuth(r *http.Request) (uint, error) {
    authHeader := r.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        return 0, fmt.Errorf("缺少或无效的Authorization头")
    }
    token := authHeader[7:]
    isValid, deviceID, userID, err := s.authToken.VerifyToken(token)
    if err != nil || !isValid {
        return 0, fmt.Errorf("JWT token验证失败: %v", err)
    }
    reqDevice := r.Header.Get("Device-Id")
    if reqDevice != deviceID {
        return 0, fmt.Errorf("设备ID与token不匹配: 请求=%s, token=%s", reqDevice, deviceID)
    }
    s.logger.Info("用户认证成功: userID=%d, deviceID=%s", userID, deviceID)
    return userID, nil
}

// busSend 安全发送到 AI 总线
func (s *IMBusServer) busSend(msg *ImMessage) {
    s.busMu.Lock()
    defer s.busMu.Unlock()
    if s.busStream == nil {
        s.logger.Warn("AI 总线未连接，消息丢弃: %s", msg.Event)
        return
    }
    if err := s.busStream.SendMsg(msg); err != nil {
        s.logger.Warn("向 AI 总线发送失败: %v", err)
    }
}

// waitBusConnected 在指定超时时间内等待 AI 总线连接建立
func (s *IMBusServer) waitBusConnected(timeout time.Duration) bool {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        s.busMu.Lock()
        connected := s.busStream != nil
        s.busMu.Unlock()
        if connected {
            return true
        }
        time.Sleep(100 * time.Millisecond)
    }
    return false
}