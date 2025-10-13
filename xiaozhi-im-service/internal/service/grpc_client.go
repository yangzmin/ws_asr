package service

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "xiaozhi-grpc-proto/generated/go/ai_service"
	"xiaozhi-im-service/internal/config"
)

// GRPCClient gRPC客户端
type GRPCClient struct {
	conn         *grpc.ClientConn
	client       pb.AIServiceClient
	config       *config.GRPCConfig
	logger       *logrus.Logger
	mu           sync.RWMutex
	isConnected  bool
	reconnectCh  chan struct{}
	stopCh       chan struct{}
	streams      map[string]*StreamInfo
	streamsMu    sync.RWMutex
}

// StreamInfo 流信息
type StreamInfo struct {
	Stream      pb.AIService_ChatStreamClient
	Context     context.Context
	Cancel      context.CancelFunc
	ConnID      string
	CreatedAt   time.Time
	LastUsedAt  time.Time
	mu          sync.RWMutex
}

// NewGRPCClient 创建gRPC客户端
func NewGRPCClient(cfg *config.GRPCConfig, logger *logrus.Logger) *GRPCClient {
	return &GRPCClient{
		config:      cfg,
		logger:      logger,
		reconnectCh: make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
		streams:     make(map[string]*StreamInfo),
	}
}

// Start 启动gRPC客户端
func (g *GRPCClient) Start(ctx context.Context) error {
	if err := g.connect(); err != nil {
		return fmt.Errorf("初始连接失败: %v", err)
	}

	// 启动连接监控和重连机制
	go g.monitorConnection(ctx)
	go g.heartbeat(ctx)

	g.logger.Info("gRPC客户端已启动")
	return nil
}

// connect 建立gRPC连接
func (g *GRPCClient) connect() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 关闭现有连接
	if g.conn != nil {
		g.conn.Close()
	}

	// 创建新连接
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Duration(g.config.HeartbeatSeconds) * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
	}

	conn, err := grpc.Dial(g.config.AIServiceAddr, opts...)
	if err != nil {
		return fmt.Errorf("连接gRPC服务失败: %v", err)
	}

	g.conn = conn
	g.client = pb.NewAIServiceClient(conn)
	g.isConnected = true

	g.logger.WithField("addr", g.config.AIServiceAddr).Info("gRPC连接已建立")
	return nil
}

// IsReady 检查客户端是否就绪
func (g *GRPCClient) IsReady() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.isConnected && g.conn != nil && g.conn.GetState() == connectivity.Ready
}

// CreateStream 创建双向流
func (g *GRPCClient) CreateStream(connectionID string) (*StreamInfo, error) {
	if !g.IsReady() {
		return nil, ErrGRPCClientNotReady
	}

	g.streamsMu.Lock()
	defer g.streamsMu.Unlock()

	// 检查是否已存在流
	if stream, exists := g.streams[connectionID]; exists {
		return stream, nil
	}

	// 创建新的流上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建双向流
	stream, err := g.client.ChatStream(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建流失败: %v", err)
	}

	streamInfo := &StreamInfo{
		Stream:     stream,
		Context:    ctx,
		Cancel:     cancel,
		ConnID:     connectionID,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	g.streams[connectionID] = streamInfo
	g.logger.WithField("connection_id", connectionID).Info("gRPC流已创建")

	return streamInfo, nil
}

// GetStream 获取流
func (g *GRPCClient) GetStream(connectionID string) (*StreamInfo, bool) {
	g.streamsMu.RLock()
	defer g.streamsMu.RUnlock()

	stream, exists := g.streams[connectionID]
	if exists {
		stream.mu.Lock()
		stream.LastUsedAt = time.Now()
		stream.mu.Unlock()
	}
	return stream, exists
}

// RemoveStream 移除流
func (g *GRPCClient) RemoveStream(connectionID string) {
	g.streamsMu.Lock()
	defer g.streamsMu.Unlock()

	if stream, exists := g.streams[connectionID]; exists {
		stream.Cancel()
		delete(g.streams, connectionID)
		g.logger.WithField("connection_id", connectionID).Info("gRPC流已移除")
	}
}

// SendMessage 发送消息到流
func (g *GRPCClient) SendMessage(connectionID string, req *pb.ChatRequest) error {
	stream, exists := g.GetStream(connectionID)
	if !exists {
		return fmt.Errorf("流不存在: %s", connectionID)
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	if err := stream.Stream.Send(req); err != nil {
		g.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"error":         err,
		}).Error("发送消息失败")
		return err
	}

	stream.LastUsedAt = time.Now()
	return nil
}

// ReceiveMessage 从流接收消息
func (g *GRPCClient) ReceiveMessage(connectionID string) (*pb.ChatResponse, error) {
	stream, exists := g.GetStream(connectionID)
	if !exists {
		return nil, fmt.Errorf("流不存在: %s", connectionID)
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	resp, err := stream.Stream.Recv()
	if err != nil {
		if err == io.EOF {
			g.logger.WithField("connection_id", connectionID).Info("gRPC流已结束")
			return nil, ErrGRPCStreamClosed
		}
		g.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"error":         err,
		}).Error("接收消息失败")
		return nil, err
	}

	return resp, nil
}

// HealthCheck 健康检查
func (g *GRPCClient) HealthCheck(ctx context.Context) error {
	if !g.IsReady() {
		return ErrGRPCClientNotReady
	}

	req := &pb.HealthCheckRequest{}
	_, err := g.client.HealthCheck(ctx, req)
	return err
}

// monitorConnection 监控连接状态
func (g *GRPCClient) monitorConnection(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(g.config.ReconnectSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.stopCh:
			return
		case <-ticker.C:
			if !g.IsReady() {
				g.logger.Warn("gRPC连接断开，尝试重连")
				if err := g.reconnect(); err != nil {
					g.logger.WithError(err).Error("重连失败")
				}
			}
		case <-g.reconnectCh:
			if err := g.reconnect(); err != nil {
				g.logger.WithError(err).Error("重连失败")
			}
		}
	}
}

// reconnect 重连
func (g *GRPCClient) reconnect() error {
	// 清理所有现有流
	g.cleanupAllStreams()

	// 重新连接
	if err := g.connect(); err != nil {
		return err
	}

	g.logger.Info("gRPC重连成功")
	return nil
}

// heartbeat 心跳检测
func (g *GRPCClient) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(g.config.HeartbeatSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.stopCh:
			return
		case <-ticker.C:
			if g.IsReady() {
				healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				if err := g.HealthCheck(healthCtx); err != nil {
					g.logger.WithError(err).Warn("健康检查失败")
					select {
					case g.reconnectCh <- struct{}{}:
					default:
					}
				}
				cancel()
			}
		}
	}
}

// cleanupAllStreams 清理所有流
func (g *GRPCClient) cleanupAllStreams() {
	g.streamsMu.Lock()
	defer g.streamsMu.Unlock()

	for connID, stream := range g.streams {
		stream.Cancel()
		delete(g.streams, connID)
	}
	g.logger.Info("所有gRPC流已清理")
}

// cleanupInactiveStreams 清理不活跃的流
func (g *GRPCClient) cleanupInactiveStreams(timeout time.Duration) {
	g.streamsMu.Lock()
	defer g.streamsMu.Unlock()

	now := time.Now()
	var toRemove []string

	for connID, stream := range g.streams {
		stream.mu.RLock()
		lastUsed := stream.LastUsedAt
		stream.mu.RUnlock()

		if now.Sub(lastUsed) > timeout {
			toRemove = append(toRemove, connID)
		}
	}

	for _, connID := range toRemove {
		if stream, exists := g.streams[connID]; exists {
			stream.Cancel()
			delete(g.streams, connID)
			g.logger.WithField("connection_id", connID).Info("清理不活跃gRPC流")
		}
	}
}

// GetStreamCount 获取流数量
func (g *GRPCClient) GetStreamCount() int {
	g.streamsMu.RLock()
	defer g.streamsMu.RUnlock()
	return len(g.streams)
}

// Stop 停止gRPC客户端
func (g *GRPCClient) Stop() {
	close(g.stopCh)
	g.cleanupAllStreams()

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.conn != nil {
		g.conn.Close()
		g.conn = nil
	}
	g.isConnected = false

	g.logger.Info("gRPC客户端已停止")
}