package service

import (
	"context"
	"fmt"
	"io"
	"log"
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
	conn        *grpc.ClientConn
	client      pb.AIServiceClient
	config      *config.GRPCConfig
	logger      *logrus.Logger
	mu          sync.RWMutex
	isConnected bool
	reconnectCh chan struct{}
	stopCh      chan struct{}
	streams     map[string]*StreamInfo
	streamsMu   sync.RWMutex
}

// StreamInfo 流信息
type StreamInfo struct {
	Stream     pb.AIService_ChatStreamClient
	Context    context.Context
	Cancel     context.CancelFunc
	ConnID     string
	CreatedAt  time.Time
	LastUsedAt time.Time
	mu         sync.RWMutex
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

	g.logger.WithField("addr", g.config.AIServiceAddr).Info("开始建立gRPC连接")

	// 关闭现有连接
	if g.conn != nil {
		g.logger.Debug("关闭现有连接")
		g.conn.Close()
		g.isConnected = false
	}

	// 创建新连接
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // 30秒发送keepalive ping
			Timeout:             5 * time.Second,  // 5秒超时
			PermitWithoutStream: true,             // 允许在没有活动流时发送keepalive
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
	}

	conn, err := grpc.Dial(g.config.AIServiceAddr, opts...)
	if err != nil {
		g.logger.WithError(err).Error("连接gRPC服务失败")
		return fmt.Errorf("连接gRPC服务失败: %v", err)
	}

	// 等待连接就绪
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		state := conn.GetState()
		g.logger.WithField("state", state).Debug("连接状态")

		if state == connectivity.Ready {
			break
		}

		if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			conn.Close()
			return fmt.Errorf("连接失败，状态: %v", state)
		}

		if !conn.WaitForStateChange(ctx, state) {
			conn.Close()
			return fmt.Errorf("连接超时")
		}
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

	if g.conn == nil || !g.isConnected {
		return false
	}

	state := g.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// CreateStream 创建双向流
func (g *GRPCClient) CreateStream(connectionID string) (*StreamInfo, error) {
	fmt.Println("11111111111111111")
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
	fmt.Println("streamstream", stream)
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
		stream.LastUsedAt = time.Now()
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

// ReceiveMessage 从流接收消息
func (g *GRPCClient) ReceiveMessage(connectionID string) (*pb.ChatResponse, error) {
	stream, exists := g.GetStream(connectionID)
	if !exists {
		return nil, fmt.Errorf("流不存在: %s", connectionID)
	}

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

	g.logger.WithField("interval", g.config.ReconnectSeconds).Info("连接监控已启动")

	for {
		select {
		case <-ctx.Done():
			g.logger.Info("连接监控已停止 - 上下文取消")
			return
		case <-g.stopCh:
			g.logger.Info("连接监控已停止 - 收到停止信号")
			return
		case <-ticker.C:
			isReady := g.IsReady()
			g.logger.WithField("isReady", isReady).Debug("检查连接状态")

			if !isReady {
				g.logger.Warn("gRPC连接断开，尝试重连")
				if err := g.reconnect(); err != nil {
					g.logger.WithError(err).Error("重连失败")
				}
			} else {
				g.logger.Debug("连接状态正常")
			}
		case <-g.reconnectCh:
			g.logger.Info("收到重连请求")
			if err := g.reconnect(); err != nil {
				g.logger.WithError(err).Error("重连失败")
			}
		}
	}
}

// reconnect 重连
func (g *GRPCClient) reconnect() error {
	g.logger.Info("开始重连流程")

	// 清理所有现有流
	g.cleanupAllStreams()
	g.logger.Debug("已清理所有现有流")

	// 重新连接
	if err := g.connect(); err != nil {
		g.logger.WithError(err).Error("重连失败")
		return err
	}

	g.logger.Info("gRPC重连成功")
	return nil
}

// heartbeat 心跳检测
func (g *GRPCClient) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(g.config.HeartbeatSeconds) * time.Second)
	defer ticker.Stop()

	g.logger.WithField("interval", g.config.HeartbeatSeconds).Info("心跳检测已启动")

	for {
		select {
		case <-ctx.Done():
			g.logger.Info("心跳检测已停止 - 上下文取消")
			return
		case <-g.stopCh:
			g.logger.Info("心跳检测已停止 - 收到停止信号")
			return
		case <-ticker.C:
			isReady := g.IsReady()
			g.logger.WithField("isReady", isReady).Debug("执行心跳检测")

			if isReady {
				healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				if err := g.HealthCheck(healthCtx); err != nil {
					g.logger.WithError(err).Warn("健康检查失败，触发重连")
					select {
					case g.reconnectCh <- struct{}{}:
					default:
						g.logger.Warn("重连通道已满，跳过此次重连请求")
					}
				} else {
					g.logger.Debug("健康检查成功")
				}
				cancel()
			} else {
				g.logger.Warn("连接未就绪，跳过健康检查")
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

func (g *GRPCClient) testSend(sessionID, deviceID, clientID string, messageType int32, data []byte) {
	stream, err := g.client.ChatStream(context.Background())
	if err != nil {
		log.Fatalf("创建聊天流失败: %v", err)
	}

	// 发送测试消息
	testMessage := &pb.ChatRequest{
		SessionId:   sessionID,
		DeviceId:    deviceID,
		ClientId:    clientID,
		MessageType: messageType,
		MessageData: data,
		Timestamp:   time.Now().Unix(),
	}
	fmt.Printf("发送消息: %s\n", string(testMessage.MessageData))
	err = stream.Send(testMessage)
	if err != nil {
		g.logger.WithError(err).Error("发送消息失败")
		return
	}
}

// SendMessage 发送消息到AI服务端
func (g *GRPCClient) SendMessage(sessionID, deviceID, clientID string, messageType int32, data []byte) error {
	if !g.isConnected {
		return ErrNotConnected
	}
	fmt.Println("2222222222222222")

	// 使用sessionID作为connectionID来管理流
	connectionID := sessionID
	if connectionID == "" {
		return fmt.Errorf("sessionID不能为空")
	}

	// 先尝试获取现有流
	stream, exists := g.GetStream(connectionID)
	fmt.Println("stream exists:", exists)

	// 如果流不存在，创建新流
	if !exists {
		var err error
		stream, err = g.CreateStream(connectionID)
		fmt.Println("created stream:", stream)
		if err != nil {
			g.logger.WithFields(logrus.Fields{
				"connection_id": connectionID,
				"session_id":    sessionID,
				"error":         err,
			}).Error("获取或创建流失败")
			return fmt.Errorf("获取或创建流失败: %v", err)
		}
	}

	// 构造请求
	req := &pb.ChatRequest{
		SessionId:   sessionID,
		DeviceId:    deviceID,
		ClientId:    clientID,
		MessageType: messageType,
		MessageData: data,
		Timestamp:   time.Now().Unix(),
	}

	g.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"session_id":    sessionID,
		"device_id":     deviceID,
		"client_id":     clientID,
		"message_type":  messageType,
		"data_length":   len(data),
	}).Debug("发送消息到AI服务端")

	// 发送消息
	if err := stream.Stream.Send(req); err != nil {
		g.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"session_id":    sessionID,
			"error":         err,
		}).Error("发送消息失败，尝试重新创建流")

		// 移除失效的流
		g.RemoveStream(connectionID)

		// 重新创建流并重试
		stream, err = g.CreateStream(connectionID)
		if err != nil {
			return fmt.Errorf("重新创建流失败: %v", err)
		}

		if err := stream.Stream.Send(req); err != nil {
			g.RemoveStream(connectionID)
			return fmt.Errorf("重试发送消息失败: %v", err)
		}
	}

	g.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"session_id":    sessionID,
		"message_type":  messageType,
	}).Debug("消息发送成功")

	stream.LastUsedAt = time.Now()
	return nil
}

// SendTextMessage 发送文本消息（messageType=1）
func (g *GRPCClient) SendTextMessage(sessionID, deviceID, clientID string, jsonData []byte) error {
	fmt.Println("55555555")
	return g.SendMessage(sessionID, deviceID, clientID, 1, jsonData)
}

// SendBinaryMessage 发送二进制音频消息（messageType=2）
func (g *GRPCClient) SendBinaryMessage(sessionID, deviceID, clientID string, audioData []byte) error {
	return g.SendMessage(sessionID, deviceID, clientID, 2, audioData)
}

// 删除重复的SendJSONMessage方法定义
