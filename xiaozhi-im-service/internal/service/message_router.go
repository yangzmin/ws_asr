package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	pb "xiaozhi-grpc-proto/generated/go/ai_service"
	"xiaozhi-im-service/internal/model"
)

// MessageRouter 消息路由器
type MessageRouter struct {
	grpcClient *GRPCClient
	connMgr    *ConnectionManager
	logger     *logrus.Logger
	mu         sync.RWMutex
	isRunning  bool
	stopCh     chan struct{}
}

// NewMessageRouter 创建消息路由器
func NewMessageRouter(grpcClient *GRPCClient, connMgr *ConnectionManager, logger *logrus.Logger) *MessageRouter {
	return &MessageRouter{
		grpcClient: grpcClient,
		connMgr:    connMgr,
		logger:     logger,
		stopCh:     make(chan struct{}),
	}
}

// Start 启动消息路由器
func (mr *MessageRouter) Start(ctx context.Context) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if mr.isRunning {
		return nil
	}

	mr.isRunning = true
	mr.logger.Info("消息路由器已启动")
	return nil
}

// Stop 停止消息路由器
func (mr *MessageRouter) Stop() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if !mr.isRunning {
		return
	}

	mr.isRunning = false
	close(mr.stopCh)
	mr.logger.Info("消息路由器已停止")
}

// RouteWebSocketMessage 路由WebSocket消息到gRPC
func (mr *MessageRouter) RouteWebSocketMessage(connectionID string, wsMsg *model.WebSocketMessage) error {
	if !mr.isRunning {
		return ErrServiceStopped
	}

	// 转换WebSocket消息为gRPC请求
	grpcReq, err := mr.convertWebSocketToGRPC(wsMsg)
	if err != nil {
		mr.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"message_type":  wsMsg.Type,
			"error":         err,
		}).Error("WebSocket消息转换失败")
		return fmt.Errorf("消息转换失败: %v", err)
	}

	// 发送到gRPC服务
	if err := mr.grpcClient.SendMessage(connectionID, grpcReq); err != nil {
		mr.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"message_type":  wsMsg.Type,
			"error":         err,
		}).Error("发送gRPC消息失败")
		return fmt.Errorf("发送gRPC消息失败: %v", err)
	}

	mr.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"message_type":  wsMsg.Type,
	}).Debug("WebSocket消息已路由到gRPC")

	return nil
}

// RouteGRPCMessage 路由gRPC消息到WebSocket
func (mr *MessageRouter) RouteGRPCMessage(connectionID string, grpcResp *pb.ChatResponse) error {
	if !mr.isRunning {
		return ErrServiceStopped
	}

	// 转换gRPC响应为WebSocket消息
	wsMsg, err := mr.convertGRPCToWebSocket(grpcResp)
	if err != nil {
		mr.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"error":         err,
		}).Error("gRPC消息转换失败")
		return fmt.Errorf("消息转换失败: %v", err)
	}

	// 获取WebSocket连接
	conn, exists := mr.connMgr.GetConnection(connectionID)
	if !exists {
		return ErrConnectionNotFound
	}

	// 序列化消息
	data, err := json.Marshal(wsMsg)
	if err != nil {
		mr.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"error":         err,
		}).Error("WebSocket消息序列化失败")
		return fmt.Errorf("消息序列化失败: %v", err)
	}

	// 发送到WebSocket连接
	if err := conn.Send(data); err != nil {
		mr.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"error":         err,
		}).Error("发送WebSocket消息失败")
		return fmt.Errorf("发送WebSocket消息失败: %v", err)
	}

	mr.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"message_type":  wsMsg.Type,
	}).Debug("gRPC消息已路由到WebSocket")

	return nil
}

// StartMessageLoop 启动消息循环处理
func (mr *MessageRouter) StartMessageLoop(connectionID string) {
	go mr.handleGRPCMessages(connectionID)
}

// handleGRPCMessages 处理gRPC消息
func (mr *MessageRouter) handleGRPCMessages(connectionID string) {
	defer func() {
		if r := recover(); r != nil {
			mr.logger.WithFields(logrus.Fields{
				"connection_id": connectionID,
				"panic":         r,
			}).Error("gRPC消息处理发生panic")
		}
	}()

	for {
		select {
		case <-mr.stopCh:
			return
		default:
			// 接收gRPC消息
			grpcResp, err := mr.grpcClient.ReceiveMessage(connectionID)
			if err != nil {
				if err == ErrGRPCStreamClosed {
					mr.logger.WithField("connection_id", connectionID).Info("gRPC流已关闭")
					return
				}
				mr.logger.WithFields(logrus.Fields{
					"connection_id": connectionID,
					"error":         err,
				}).Error("接收gRPC消息失败")
				continue
			}

			// 路由到WebSocket
			if err := mr.RouteGRPCMessage(connectionID, grpcResp); err != nil {
				mr.logger.WithFields(logrus.Fields{
					"connection_id": connectionID,
					"error":         err,
				}).Error("路由gRPC消息失败")
			}
		}
	}
}

// convertWebSocketToGRPC 转换WebSocket消息为gRPC请求
func (mr *MessageRouter) convertWebSocketToGRPC(wsMsg *model.WebSocketMessage) (*pb.ChatRequest, error) {
	req := &pb.ChatRequest{
		Timestamp: wsMsg.Timestamp,
	}

	switch wsMsg.Type {
	case "hello":
		helloMsg := &model.HelloMessage{}
		if wsMsg.Data != nil {
			if err := mr.mapToStruct(wsMsg.Data, helloMsg); err != nil {
				return nil, fmt.Errorf("解析hello消息失败: %v", err)
			}
		}

		req.Message = &pb.ChatRequest_Hello{
			Hello: &pb.HelloMessage{
				AudioParams: mr.convertAudioParams(helloMsg.AudioParams),
			},
		}

	case "listen":
		listenMsg := &model.ListenMessage{}
		if err := mr.mapToStruct(wsMsg.Data, listenMsg); err != nil {
			return nil, fmt.Errorf("解析listen消息失败: %v", err)
		}

		req.Message = &pb.ChatRequest_Listen{
			Listen: &pb.ListenMessage{
				State: listenMsg.State,
				Mode:  listenMsg.Mode,
				Text:  listenMsg.Text,
			},
		}

	case "chat":
		chatMsg := &model.ChatMessage{}
		if err := mr.mapToStruct(wsMsg.Data, chatMsg); err != nil {
			return nil, fmt.Errorf("解析chat消息失败: %v", err)
		}

		req.Message = &pb.ChatRequest_Chat{
			Chat: &pb.ChatMessage{
				Text: chatMsg.Text,
			},
		}

	case "abort":
		abortMsg := &model.AbortMessage{}
		if wsMsg.Data != nil {
			if err := mr.mapToStruct(wsMsg.Data, abortMsg); err != nil {
				return nil, fmt.Errorf("解析abort消息失败: %v", err)
			}
		}

		req.Message = &pb.ChatRequest_Abort{
			Abort: &pb.AbortMessage{
				Reason: abortMsg.Reason,
			},
		}

	case "vision":
		visionMsg := &model.VisionMessage{}
		if err := mr.mapToStruct(wsMsg.Data, visionMsg); err != nil {
			return nil, fmt.Errorf("解析vision消息失败: %v", err)
		}

		req.Message = &pb.ChatRequest_Vision{
			Vision: &pb.VisionMessage{
				Cmd:    visionMsg.Cmd,
				Params: visionMsg.Params,
			},
		}

	case "image":
		imageMsg := &model.ImageMessage{}
		if err := mr.mapToStruct(wsMsg.Data, imageMsg); err != nil {
			return nil, fmt.Errorf("解析image消息失败: %v", err)
		}

		req.Message = &pb.ChatRequest_Image{
			Image: &pb.ImageMessage{
				Text:      imageMsg.Text,
				ImageData: mr.convertImageData(imageMsg.ImageData),
			},
		}

	case "mcp":
		mcpMsg := &model.MCPMessage{}
		if err := mr.mapToStruct(wsMsg.Data, mcpMsg); err != nil {
			return nil, fmt.Errorf("解析mcp消息失败: %v", err)
		}

		req.Message = &pb.ChatRequest_Mcp{
			Mcp: &pb.MCPMessage{
				Method: mcpMsg.Method,
				Params: mcpMsg.Params,
			},
		}

	case "audio":
		// 处理音频数据
		audioData, ok := wsMsg.Data["audio_data"].([]byte)
		if !ok {
			return nil, fmt.Errorf("无效的音频数据")
		}

		req.Message = &pb.ChatRequest_Audio{
			Audio: &pb.AudioData{
				Data: audioData,
			},
		}

	default:
		return nil, fmt.Errorf("不支持的消息类型: %s", wsMsg.Type)
	}

	return req, nil
}

// convertGRPCToWebSocket 转换gRPC响应为WebSocket消息
func (mr *MessageRouter) convertGRPCToWebSocket(grpcResp *pb.ChatResponse) (*model.WebSocketMessage, error) {
	wsMsg := &model.WebSocketMessage{
		Timestamp: grpcResp.Timestamp,
		Data:      make(map[string]interface{}),
	}

	switch resp := grpcResp.Response.(type) {
	case *pb.ChatResponse_HelloResponse:
		wsMsg.Type = "hello_response"
		wsMsg.Data["server_audio_params"] = mr.convertAudioParamsFromPB(resp.HelloResponse.ServerAudioParams)
		wsMsg.Data["status"] = resp.HelloResponse.Status

	case *pb.ChatResponse_SttResponse:
		wsMsg.Type = "stt_response"
		wsMsg.Data["text"] = resp.SttResponse.Text
		wsMsg.Data["is_final"] = resp.SttResponse.IsFinal

	case *pb.ChatResponse_TtsResponse:
		wsMsg.Type = "tts_response"
		wsMsg.Data["state"] = resp.TtsResponse.State
		wsMsg.Data["text"] = resp.TtsResponse.Text
		wsMsg.Data["text_index"] = resp.TtsResponse.TextIndex

	case *pb.ChatResponse_EmotionResponse:
		wsMsg.Type = "emotion_response"
		wsMsg.Data["emotion"] = resp.EmotionResponse.Emotion

	case *pb.ChatResponse_AudioResponse:
		wsMsg.Type = "audio_response"
		wsMsg.Data["audio_data"] = resp.AudioResponse.AudioData
		wsMsg.Data["text"] = resp.AudioResponse.Text
		wsMsg.Data["round"] = resp.AudioResponse.Round
		wsMsg.Data["text_index"] = resp.AudioResponse.TextIndex
		wsMsg.Data["format"] = resp.AudioResponse.Format

	case *pb.ChatResponse_ErrorResponse:
		wsMsg.Type = "error_response"
		wsMsg.Data["error_code"] = resp.ErrorResponse.ErrorCode
		wsMsg.Data["error_message"] = resp.ErrorResponse.ErrorMessage

	case *pb.ChatResponse_StatusResponse:
		wsMsg.Type = "status_response"
		wsMsg.Data["status"] = resp.StatusResponse.Status
		wsMsg.Data["details"] = resp.StatusResponse.Details

	default:
		return nil, fmt.Errorf("不支持的gRPC响应类型")
	}

	return wsMsg, nil
}

// convertAudioParams 转换音频参数
func (mr *MessageRouter) convertAudioParams(params *model.AudioParams) *pb.AudioParams {
	if params == nil {
		return nil
	}
	return &pb.AudioParams{
		Format:        params.Format,
		SampleRate:    params.SampleRate,
		Channels:      params.Channels,
		FrameDuration: params.FrameDuration,
	}
}

// convertAudioParamsFromPB 从protobuf转换音频参数
func (mr *MessageRouter) convertAudioParamsFromPB(params *pb.AudioParams) *model.AudioParams {
	if params == nil {
		return nil
	}
	return &model.AudioParams{
		Format:        params.Format,
		SampleRate:    params.SampleRate,
		Channels:      params.Channels,
		FrameDuration: params.FrameDuration,
	}
}

// convertImageData 转换图片数据
func (mr *MessageRouter) convertImageData(data *model.ImageData) *pb.ImageData {
	if data == nil {
		return nil
	}
	return &pb.ImageData{
		Url:    data.URL,
		Data:   data.Data,
		Format: data.Format,
	}
}

// mapToStruct 将map转换为结构体
func (mr *MessageRouter) mapToStruct(data map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}