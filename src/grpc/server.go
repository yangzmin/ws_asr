package grpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	pb "xiaozhi-grpc-proto/generated/go/ai_service"
	"xiaozhi-server-go/src/configs"
	"xiaozhi-server-go/src/core"
	"xiaozhi-server-go/src/core/auth"
	"xiaozhi-server-go/src/core/pool"
	"xiaozhi-server-go/src/core/utils"
	"xiaozhi-server-go/src/services"
	"xiaozhi-server-go/src/task"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// GRPCServer gRPC服务器
type GRPCServer struct {
	pb.UnimplementedAIServiceServer
	config            *configs.Config
	logger            *utils.Logger
	server            *grpc.Server
	poolManager       *pool.PoolManager
	taskManager       *task.TaskManager
	userConfigService services.UserAIConfigService
	authManager       *auth.AuthManager
	streams           map[string]*StreamHandler
	streamsMu         sync.RWMutex
}

// StreamHandler 流处理器
type StreamHandler struct {
	stream       pb.AIService_ChatStreamServer
	connectionID string
	handler      *core.ConnectionHandler
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	lastActive   time.Time
}

// NewGRPCServer 创建gRPC服务器
func NewGRPCServer(
	config *configs.Config,
	logger *utils.Logger,
	poolManager *pool.PoolManager,
	taskManager *task.TaskManager,
	userConfigService services.UserAIConfigService,
	authManager *auth.AuthManager,
) *GRPCServer {
	return &GRPCServer{
		config:            config,
		logger:            logger,
		poolManager:       poolManager,
		taskManager:       taskManager,
		userConfigService: userConfigService,
		authManager:       authManager,
		streams:           make(map[string]*StreamHandler),
	}
}

// Start 启动gRPC服务器
func (s *GRPCServer) Start(ctx context.Context, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("监听端口失败: %v", err)
	}

	// 创建gRPC服务器选项
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Second,
			MaxConnectionAge:      30 * time.Second,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Second,
			Timeout:               1 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB
		grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4MB
	}

	s.server = grpc.NewServer(opts...)

	// 注册服务
	pb.RegisterAIServiceServer(s.server, s)

	// 启用反射（用于调试）
	reflection.Register(s.server)

	s.logger.Info("gRPC服务器启动在端口: %d", port)

	// 启动服务器
	go func() {
		if err := s.server.Serve(lis); err != nil {
			s.logger.Error("gRPC服务器运行失败: %v", err)
		}
	}()

	// 监听关闭信号
	go func() {
		<-ctx.Done()
		s.logger.Info("收到关闭信号，开始关闭gRPC服务器...")
		s.Stop()
	}()

	// 启动流清理任务
	go s.cleanupInactiveStreams(ctx)

	return nil
}

// Stop 停止gRPC服务器
func (s *GRPCServer) Stop() {
	if s.server != nil {
		// 清理所有流
		s.streamsMu.Lock()
		for id, handler := range s.streams {
			handler.cancel()
			delete(s.streams, id)
		}
		s.streamsMu.Unlock()

		// 优雅关闭服务器
		s.server.GracefulStop()
		s.logger.Info("gRPC服务器已关闭")
	}
}

// ChatStream 实现双向流聊天接口
func (s *GRPCServer) ChatStream(stream pb.AIService_ChatStreamServer) error {
	// 生成连接ID
	connectionID := fmt.Sprintf("grpc_%d", time.Now().UnixNano())

	// 创建上下文
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// gRPC 适配暂未直接接入 ConnectionHandler，后续可在此创建适配的连接并传入处理器

	// 创建流处理器
	streamHandler := &StreamHandler{
		stream:       stream,
		connectionID: connectionID,
		handler:      nil,
		ctx:          ctx,
		cancel:       cancel,
		lastActive:   time.Now(),
	}

	// 注册流处理器
	s.streamsMu.Lock()
	s.streams[connectionID] = streamHandler
	s.streamsMu.Unlock()

	// 清理资源
	defer func() {
		s.streamsMu.Lock()
		delete(s.streams, connectionID)
		s.streamsMu.Unlock()
		s.logger.Info("gRPC流已关闭: %s", connectionID)
	}()

	s.logger.Info("gRPC流已建立: %s", connectionID)

	// 启动消息处理
	errCh := make(chan error, 2)

	// 接收消息协程
	go func() {
		errCh <- s.handleIncomingMessages(streamHandler)
	}()

	// 发送消息协程
	go func() {
		errCh <- s.handleOutgoingMessages(streamHandler)
	}()

	// 等待任一协程结束
	select {
	case err := <-errCh:
		if err != nil && err != io.EOF {
			s.logger.Error("gRPC流处理错误: %v", err)
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// handleIncomingMessages 处理接收到的消息
// 修复handleIncomingMessages方法中的processGRPCRequest调用
func (s *GRPCServer) handleIncomingMessages(streamHandler *StreamHandler) error {
	for {
		select {
		case <-streamHandler.ctx.Done():
			return streamHandler.ctx.Err()
		default:
			req, err := streamHandler.stream.Recv()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("接收消息失败: %v", err)
			}

			// 更新最后活跃时间
			streamHandler.lastActive = time.Now()
			// 处理消息 - 修复参数传递
			if err := s.processGRPCRequest(streamHandler, int(req.MessageType), req.MessageData); err != nil {
				s.logger.Error("处理gRPC请求失败: %v", err)
				s.sendErrorResponse(streamHandler, "PROCESS_ERROR", err.Error())
			}
		}
	}
}

// 修复processGRPCRequest方法中的NewConnectionHandler调用参数
func (s *GRPCServer) processGRPCRequest(streamHandler *StreamHandler, messageType int, message []byte) error {
	// 创建连接适配器
	adapter := NewGRPCConnectionAdapter(streamHandler.stream, streamHandler.connectionID)

	// 创建模拟的HTTP请求
	req, err := http.NewRequest("POST", "/grpc", nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 获取提供者集合
	providerSet, err := s.poolManager.GetProviderSet()
	if err != nil {
		return fmt.Errorf("获取提供者集合失败: %v", err)
	}

	// 创建连接处理器 - 修复参数顺序
	handler := core.NewConnectionHandler(
		s.config,
		providerSet,
		s.logger,
		req,
		context.Background(),
	)
	fmt.Println("processGRPCRequest 接收message", string(message))

	// 将消息放入适配器的消息队列
	if err := adapter.PutMessage(messageType, message); err != nil {
		return fmt.Errorf("消息放入队列失败: %v", err)
	}

	// 启动消息处理协程
	go handler.Handle(adapter)
	
	// 启动响应处理协程
	go s.handleConnectionResponses(streamHandler, adapter)

	return nil
}

// handleConnectionResponses 处理连接处理器的响应 - 新增方法
func (s *GRPCServer) handleConnectionResponses(streamHandler *StreamHandler, adapter *GRPCConnectionAdapter) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("处理连接响应时发生panic: %v", r)
		}
	}()

	for {
		select {
		case <-streamHandler.ctx.Done():
			s.logger.Info("流上下文已取消，停止响应处理")
			return
		case <-adapter.stopChan:
			s.logger.Info("适配器已关闭，停止响应处理")
			return
		default:
			// 由于现有的ConnectionHandler是通过WriteMessage直接发送响应的，
			// 我们不需要额外的响应处理逻辑，适配器会直接通过stream.Send发送
			// 这里只需要保持协程活跃，监听上下文取消
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// handleOutgoingMessages 处理发送消息 - 重构后的方法
func (s *GRPCServer) handleOutgoingMessages(streamHandler *StreamHandler) error {
	// 现在这个方法主要用于处理流的生命周期管理
	// 实际的消息发送已经通过GRPCConnectionAdapter的WriteMessage方法处理
	
	s.logger.Info("开始处理gRPC流的输出消息")
	
	// 等待流上下文结束
	<-streamHandler.ctx.Done()
	
	s.logger.Info("gRPC流上下文已结束，停止处理输出消息")
	return streamHandler.ctx.Err()
}

// convertGRPCRequestToMessage 转换gRPC请求为消息
func (s *GRPCServer) convertGRPCRequestToMessage(req *pb.ChatRequest) (interface{}, error) {
	// 根据消息类型转换请求
	switch req.MessageType {
	case 1: // 文本消息
		return map[string]interface{}{
			"type":    "text",
			"content": string(req.MessageData),
		}, nil
	case 2: // 二进制消息（如音频）
		return map[string]interface{}{
			"type": "binary",
			"data": req.MessageData,
		}, nil
	default:
		return nil, fmt.Errorf("不支持的消息类型: %d", req.MessageType)
	}
}

// sendErrorResponse 发送错误响应
func (s *GRPCServer) sendErrorResponse(streamHandler *StreamHandler, errorCode, errorMessage string) {
	errorResp := &pb.ChatResponse{
		ResponseType: 999, // 错误响应类型
		ResponseData: []byte(fmt.Sprintf(`{"error_code": "%s", "error_message": "%s"}`, errorCode, errorMessage)),
		Timestamp:    time.Now().UnixNano() / int64(time.Millisecond),
	}

	if err := streamHandler.stream.Send(errorResp); err != nil {
		s.logger.Error("发送错误响应失败: %v", err)
	}
}

// HealthCheck 实现健康检查接口
func (s *GRPCServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status: pb.HealthCheckResponse_SERVING,
	}, nil
}

// cleanupInactiveStreams 清理不活跃的流
func (s *GRPCServer) cleanupInactiveStreams(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.streamsMu.Lock()
			now := time.Now()
			var toRemove []string

			for id, handler := range s.streams {
				handler.mu.RLock()
				lastActive := handler.lastActive
				handler.mu.RUnlock()

				if now.Sub(lastActive) > 5*time.Minute {
					toRemove = append(toRemove, id)
				}
			}

			for _, id := range toRemove {
				if handler, exists := s.streams[id]; exists {
					handler.cancel()
					delete(s.streams, id)
					s.logger.Info("清理不活跃gRPC流: %s", id)
				}
			}
			s.streamsMu.Unlock()
		}
	}
}

// GetStreamCount 获取当前流数量
func (s *GRPCServer) GetStreamCount() int {
	s.streamsMu.RLock()
	defer s.streamsMu.RUnlock()
	return len(s.streams)
}
