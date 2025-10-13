package grpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	pb "xiaozhi-grpc-proto/generated/go/ai_service"
	"xiaozhi-server-go/src/configs"
	"xiaozhi-server-go/src/core/auth"
	"xiaozhi-server-go/src/core/pool"
	"xiaozhi-server-go/src/core/transport"
	"xiaozhi-server-go/src/core/utils"
	"xiaozhi-server-go/src/services"
	"xiaozhi-server-go/src/task"
)

// GRPCServer gRPC服务器
type GRPCServer struct {
	pb.UnimplementedAIServiceServer
	config            *configs.Config
	logger            *utils.Logger
	server            *grpc.Server
	poolManager       *pool.PoolManager
	taskManager       *task.TaskManager
	userConfigService *services.UserAIConfigService
	authManager       *auth.AuthManager
	streams           map[string]*StreamHandler
	streamsMu         sync.RWMutex
}

// StreamHandler 流处理器
type StreamHandler struct {
	stream       pb.AIService_ChatStreamServer
	connectionID string
	handler      transport.ConnectionHandler
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
	userConfigService *services.UserAIConfigService,
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

	// 创建连接处理器
	handlerFactory := transport.NewDefaultConnectionHandlerFactory(
		s.config,
		s.poolManager,
		s.taskManager,
		s.logger,
		s.userConfigService,
	)

	handler := handlerFactory.CreateHandler(connectionID, "grpc")

	// 创建流处理器
	streamHandler := &StreamHandler{
		stream:       stream,
		connectionID: connectionID,
		handler:      handler,
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
func (s *GRPCServer) handleIncomingMessages(streamHandler *StreamHandler) error {
	for {
		select {
		case <-streamHandler.ctx.Done():
			return streamHandler.ctx.Err()
		default:
			// 接收消息
			req, err := streamHandler.stream.Recv()
			if err != nil {
				if err == io.EOF {
					s.logger.Info("gRPC流客户端关闭: %s", streamHandler.connectionID)
					return err
				}
				s.logger.Error("接收gRPC消息失败: %v", err)
				return err
			}

			// 更新活跃时间
			streamHandler.mu.Lock()
			streamHandler.lastActive = time.Now()
			streamHandler.mu.Unlock()

			// 转换并处理消息
			if err := s.processGRPCRequest(streamHandler, req); err != nil {
				s.logger.Error("处理gRPC请求失败: %v", err)
				// 发送错误响应
				s.sendErrorResponse(streamHandler, "PROCESS_ERROR", err.Error())
			}
		}
	}
}

// handleOutgoingMessages 处理发送消息
func (s *GRPCServer) handleOutgoingMessages(streamHandler *StreamHandler) error {
	// 这里需要实现从连接处理器获取响应消息的逻辑
	// 由于现有架构中连接处理器是基于WebSocket设计的，
	// 我们需要适配到gRPC流
	
	// 创建一个通道来接收响应消息
	responseCh := make(chan *pb.ChatResponse, 100)
	
	// 注册响应处理器到连接处理器
	// 这需要修改现有的连接处理器接口
	
	for {
		select {
		case <-streamHandler.ctx.Done():
			return streamHandler.ctx.Err()
		case resp := <-responseCh:
			if err := streamHandler.stream.Send(resp); err != nil {
				s.logger.Error("发送gRPC响应失败: %v", err)
				return err
			}
		}
	}
}

// processGRPCRequest 处理gRPC请求
func (s *GRPCServer) processGRPCRequest(streamHandler *StreamHandler, req *pb.ChatRequest) error {
	// 将gRPC请求转换为内部消息格式
	message, err := s.convertGRPCRequestToMessage(req)
	if err != nil {
		return fmt.Errorf("转换gRPC请求失败: %v", err)
	}

	// 调用连接处理器处理消息
	// 这里需要适配现有的连接处理器接口
	// 由于现有处理器是基于WebSocket设计的，需要进行适配
	
	s.logger.Debug("处理gRPC消息: %s, 类型: %T", streamHandler.connectionID, req.Message)
	
	// TODO: 实现具体的消息处理逻辑
	// 这需要根据现有的连接处理器实现进行适配
	
	return nil
}

// convertGRPCRequestToMessage 转换gRPC请求为内部消息格式
func (s *GRPCServer) convertGRPCRequestToMessage(req *pb.ChatRequest) (interface{}, error) {
	switch msg := req.Message.(type) {
	case *pb.ChatRequest_Hello:
		return map[string]interface{}{
			"type": "hello",
			"data": map[string]interface{}{
				"audio_params": s.convertAudioParams(msg.Hello.AudioParams),
			},
			"timestamp": req.Timestamp,
		}, nil

	case *pb.ChatRequest_Listen:
		return map[string]interface{}{
			"type": "listen",
			"data": map[string]interface{}{
				"state": msg.Listen.State,
				"mode":  msg.Listen.Mode,
				"text":  msg.Listen.Text,
			},
			"timestamp": req.Timestamp,
		}, nil

	case *pb.ChatRequest_Chat:
		return map[string]interface{}{
			"type": "chat",
			"data": map[string]interface{}{
				"text": msg.Chat.Text,
			},
			"timestamp": req.Timestamp,
		}, nil

	case *pb.ChatRequest_Abort:
		return map[string]interface{}{
			"type": "abort",
			"data": map[string]interface{}{
				"reason": msg.Abort.Reason,
			},
			"timestamp": req.Timestamp,
		}, nil

	case *pb.ChatRequest_Vision:
		return map[string]interface{}{
			"type": "vision",
			"data": map[string]interface{}{
				"cmd":    msg.Vision.Cmd,
				"params": msg.Vision.Params,
			},
			"timestamp": req.Timestamp,
		}, nil

	case *pb.ChatRequest_Image:
		return map[string]interface{}{
			"type": "image",
			"data": map[string]interface{}{
				"text":       msg.Image.Text,
				"image_data": s.convertImageData(msg.Image.ImageData),
			},
			"timestamp": req.Timestamp,
		}, nil

	case *pb.ChatRequest_Mcp:
		return map[string]interface{}{
			"type": "mcp",
			"data": map[string]interface{}{
				"method": msg.Mcp.Method,
				"params": msg.Mcp.Params,
			},
			"timestamp": req.Timestamp,
		}, nil

	case *pb.ChatRequest_Audio:
		return map[string]interface{}{
			"type": "audio",
			"data": map[string]interface{}{
				"audio_data": msg.Audio.Data,
			},
			"timestamp": req.Timestamp,
		}, nil

	default:
		return nil, fmt.Errorf("不支持的消息类型: %T", msg)
	}
}

// convertAudioParams 转换音频参数
func (s *GRPCServer) convertAudioParams(params *pb.AudioParams) map[string]interface{} {
	if params == nil {
		return nil
	}
	return map[string]interface{}{
		"format":         params.Format,
		"sample_rate":    params.SampleRate,
		"channels":       params.Channels,
		"frame_duration": params.FrameDuration,
	}
}

// convertImageData 转换图片数据
func (s *GRPCServer) convertImageData(data *pb.ImageData) map[string]interface{} {
	if data == nil {
		return nil
	}
	return map[string]interface{}{
		"url":    data.Url,
		"data":   data.Data,
		"format": data.Format,
	}
}

// sendErrorResponse 发送错误响应
func (s *GRPCServer) sendErrorResponse(streamHandler *StreamHandler, errorCode, errorMessage string) {
	resp := &pb.ChatResponse{
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Response: &pb.ChatResponse_ErrorResponse{
			ErrorResponse: &pb.ErrorResponse{
				ErrorCode:    errorCode,
				ErrorMessage: errorMessage,
			},
		},
	}

	if err := streamHandler.stream.Send(resp); err != nil {
		s.logger.Error("发送错误响应失败: %v", err)
	}
}

// HealthCheck 实现健康检查接口
func (s *GRPCServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status: "SERVING",
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