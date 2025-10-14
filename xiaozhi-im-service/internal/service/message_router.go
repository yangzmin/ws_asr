package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	pb "xiaozhi-grpc-proto/generated/go/ai_service"
	"xiaozhi-im-service/internal/model"

	"github.com/sirupsen/logrus"
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
func (mr *MessageRouter) RouteWebSocketMessage(connectionID string, data []byte) error {
	fmt.Println("333333333333")
	if !mr.isRunning {
		return ErrServiceStopped
	}

	// 获取连接信息
	conn, exists := mr.connMgr.GetConnection(connectionID)
	if !exists {
		return ErrConnectionNotFound
	}
	fmt.Println("44444444444", conn)

	// 从连接信息中获取必要的参数
	sessionID := conn.Info.SessionID
	deviceID := conn.Info.DeviceID
	clientID := conn.Info.ClientID

	// 如果SessionID为空，使用connectionID作为sessionID
	if sessionID == "" {
		sessionID = connectionID
	}

	mr.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"session_id":    sessionID,
		"device_id":     deviceID,
		"client_id":     clientID,
		"message_row":   string(data),
	}).Debug("准备发送WebSocket消息到AI服务端")

	// 发送JSON消息到AI服务端
	if err := mr.grpcClient.SendTextMessage(sessionID, deviceID, clientID, data); err != nil {
		mr.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"session_id":    sessionID,
			"message_row":   string(data),
			"error":         err,
		}).Error("发送JSON消息失败")
		return fmt.Errorf("发送JSON消息失败: %v", err)
	}

	mr.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"session_id":    sessionID,
		"message_row":   string(data),
	}).Debug("WebSocket消息已路由到AI服务端")

	return nil
}

// RouteGRPCMessage 路由gRPC消息到WebSocket
func (mr *MessageRouter) RouteGRPCMessage(connectionID string, grpcResp *pb.ChatResponse) error {
	if !mr.isRunning {
		return ErrServiceStopped
	}

	// 转换gRPC响应为WebSocket消息
	wsMsg, err := mr.convertGRPCToWebSocket(grpcResp)
	fmt.Println("wsMsgwsMsg", wsMsg)
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

	// 序列化消息 - 只发送wsMsg.Data部分
	data, err := json.Marshal(wsMsg.Data)
	fmt.Println("datadatadata", string(data))
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

// StartMessageLoop 启动消息循环
func (mr *MessageRouter) StartMessageLoop(connectionID string) {
	// 确保在启动消息循环前创建gRPC流
	if _, exists := mr.grpcClient.GetStream(connectionID); !exists {
		if _, err := mr.grpcClient.CreateStream(connectionID); err != nil {
			mr.logger.WithFields(logrus.Fields{
				"connection_id": connectionID,
				"error":         err,
			}).Error("创建gRPC流失败")
			return
		}
	}

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
	// 将WebSocket消息转换为JSON字符串
	jsonData, err := json.Marshal(wsMsg)
	if err != nil {
		return nil, fmt.Errorf("序列化WebSocket消息失败: %v", err)
	}

	req := &pb.ChatRequest{
		MessageType: 1, // 使用int32类型，1表示文本消息
		MessageData: jsonData,
		Timestamp:   wsMsg.Timestamp,
	}
	return req, nil
}

// convertGRPCToWebSocket 转换gRPC响应为WebSocket消息
func (mr *MessageRouter) convertGRPCToWebSocket(grpcResp *pb.ChatResponse) (*model.WebSocketMessage, error) {
	// 将gRPC响应的ResponseData解析为JSON
	var data map[string]interface{}
	if err := json.Unmarshal(grpcResp.ResponseData, &data); err != nil {
		return nil, fmt.Errorf("解析gRPC响应数据失败: %v", err)
	}

	wsMsg := &model.WebSocketMessage{
		Type:      "response",
		Data:      data,
		Timestamp: grpcResp.Timestamp,
	}

	return wsMsg, nil
}

// mapToStruct 将map转换为结构体
func (mr *MessageRouter) mapToStruct(data map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

// convertWebSocketToJSON 将WebSocket消息转换为AI服务端期望的JSON格式
func (mr *MessageRouter) convertWebSocketToJSON(wsMsg *model.WebSocketMessage) (string, error) {
	// 创建AI服务端期望的消息格式
	jsonMsg := map[string]interface{}{
		"type": wsMsg.Type,
	}

	// 根据消息类型添加相应的数据
	switch wsMsg.Type {
	case "hello":
		if wsMsg.Data != nil {
			// 将Data中的内容直接合并到根级别
			for key, value := range wsMsg.Data {
				jsonMsg[key] = value
			}
		}

	case "listen":
		if wsMsg.Data != nil {
			for key, value := range wsMsg.Data {
				jsonMsg[key] = value
			}
		}

	case "chat":
		if wsMsg.Data != nil {
			for key, value := range wsMsg.Data {
				jsonMsg[key] = value
			}
		}

	case "abort":
		if wsMsg.Data != nil {
			for key, value := range wsMsg.Data {
				jsonMsg[key] = value
			}
		}

	case "vision":
		if wsMsg.Data != nil {
			for key, value := range wsMsg.Data {
				jsonMsg[key] = value
			}
		}

	case "image":
		if wsMsg.Data != nil {
			for key, value := range wsMsg.Data {
				jsonMsg[key] = value
			}
		}

	case "mcp":
		if wsMsg.Data != nil {
			for key, value := range wsMsg.Data {
				jsonMsg[key] = value
			}
		}
	}

	// 序列化为JSON字符串
	jsonBytes, err := json.Marshal(jsonMsg)
	if err != nil {
		return "", fmt.Errorf("序列化JSON失败: %v", err)
	}

	return string(jsonBytes), nil
}

// 删除重复的RouteWebSocketMessage方法定义，保留原有的正确方法

// RouteBinaryMessage 路由二进制音频数据到gRPC
func (r *MessageRouter) RouteBinaryMessage(connectionID string, audioData []byte) error {
	// 直接发送二进制音频数据
	return r.grpcClient.SendBinaryMessage(connectionID, "", "", audioData)
}
