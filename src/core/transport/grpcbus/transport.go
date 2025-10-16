package grpcbus

import (
    "context"
    "fmt"
    "net/http"
    "sync"

    "angrymiao-ai-server/src/configs"
    "angrymiao-ai-server/src/core/transport"
    "angrymiao-ai-server/src/core/utils"
)

// GrpcBusTransport 将 im-server 的 gRPC Bus 作为传输层，对上复用现有 ConnectionHandlerFactory
type GrpcBusTransport struct {
    cfg        *configs.Config
    logger     *utils.Logger
    factory    transport.ConnectionHandlerFactory
    client     *IMBusClient
    cancel     context.CancelFunc
    wg         sync.WaitGroup
    handlers   sync.Map // sessionID -> transport.ConnectionHandler
    connections sync.Map // sessionID -> *GrpcConnection
}

func NewGrpcBusTransport(cfg *configs.Config, logger *utils.Logger, factory transport.ConnectionHandlerFactory) *GrpcBusTransport {
    return &GrpcBusTransport{cfg: cfg, logger: logger, factory: factory}
}

func (t *GrpcBusTransport) GetType() string { return "grpcbus" }

func (t *GrpcBusTransport) SetConnectionHandler(factory transport.ConnectionHandlerFactory) { t.factory = factory }

func (t *GrpcBusTransport) Start(ctx context.Context) error {
    if t.factory == nil { return fmt.Errorf("connection handler factory not set") }
    addr := fmt.Sprintf("%s:%d", t.cfg.Transport.WebSocket.IP, t.cfg.Transport.WebSocket.Port+1)
    cctx, cancel := context.WithCancel(ctx)
    t.cancel = cancel
    client, err := NewIMBusClient(cctx, addr)
    if err != nil { return err }
    t.client = client

    t.wg.Add(1)
    go func() {
        defer t.wg.Done()
        for {
            msg, err := t.client.Recv()
            if err != nil {
                t.logger.Error("grpcbus recv error: %v", err)
                return
            }
            t.handleIncoming(msg)
        }
    }()

    t.logger.Info("GrpcBusTransport connected to %s", addr)
    return nil
}

func (t *GrpcBusTransport) Stop() error {
    if t.cancel != nil { t.cancel() }
    if t.client != nil { _ = t.client.Close() }
    t.wg.Wait()
    // 关闭所有连接
    t.connections.Range(func(key, value any) bool {
        if c, ok := value.(*GrpcConnection); ok { _ = c.Close() }
        return true
    })
    return nil
}

func (t *GrpcBusTransport) GetActiveConnectionCount() int {
    count := 0
    t.connections.Range(func(_, _ any) bool { count++; return true })
    return count
}

func (t *GrpcBusTransport) handleIncoming(msg *ImMessage) {
    switch msg.Event {
    case EventSessionOpen:
        // 构造伪 HTTP 请求，传递头信息到处理器
        req := &http.Request{Header: http.Header{}}
        for k, v := range msg.Headers {
            req.Header.Set(k, v)
        }
        conn := NewGrpcConnection(msg.SessionID, req, t.client)
        handler := t.factory.CreateHandler(conn, req)
        t.connections.Store(msg.SessionID, conn)
        t.handlers.Store(msg.SessionID, handler)
        t.wg.Add(1)
        go func() {
            defer t.wg.Done()
            handler.Handle()
            t.connections.Delete(msg.SessionID)
            t.handlers.Delete(msg.SessionID)
        }()
    case EventData:
        if v, ok := t.connections.Load(msg.SessionID); ok {
            if c, ok := v.(*GrpcConnection); ok {
                c.PushIncoming(msg)
            }
        }
    case EventSessionClose:
        if v, ok := t.handlers.Load(msg.SessionID); ok {
            if h, ok := v.(transport.ConnectionHandler); ok { h.Close() }
        }
        if v, ok := t.connections.Load(msg.SessionID); ok {
            if c, ok := v.(*GrpcConnection); ok { _ = c.Close() }
        }
    default:
        // 其他事件直接作为数据透传
        if v, ok := t.connections.Load(msg.SessionID); ok {
            if c, ok := v.(*GrpcConnection); ok { c.PushIncoming(msg) }
        }
    }
}

// 编译期校验满足 Transport 接口
var _ transport.Transport = (*GrpcBusTransport)(nil)