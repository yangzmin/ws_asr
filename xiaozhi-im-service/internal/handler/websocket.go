package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"xiaozhi-im-service/internal/model"
	"xiaozhi-im-service/internal/service"
	"xiaozhi-im-service/pkg/auth"
)

// WebSocketHandler WebSocket处理器
type WebSocketHandler struct {
	upgrader   websocket.Upgrader
	connMgr    *service.ConnectionManager
	grpcClient *service.GRPCClient
	msgRouter  *service.MessageRouter
	jwtManager *auth.JWTManager
	logger     *logrus.Logger
}

// NewWebSocketHandler 创建WebSocket处理器
func NewWebSocketHandler(
	connMgr *service.ConnectionManager,
	grpcClient *service.GRPCClient,
	msgRouter *service.MessageRouter,
	jwtManager *auth.JWTManager,
	logger *logrus.Logger,
) *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// 在生产环境中应该检查Origin
				return true
			},
			ReadBufferSize:  1024 * 4, // 4KB
			WriteBufferSize: 1024 * 4, // 4KB
		},
		connMgr:    connMgr,
		grpcClient: grpcClient,
		msgRouter:  msgRouter,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// 验证JWT Token
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("token")
	}

	claims, err := h.jwtManager.ValidateToken(token)
	if err != nil {
		h.logger.WithError(err).Warn("JWT验证失败")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	// 升级为WebSocket连接
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("WebSocket升级失败")
		return
	}

	// 创建连接信息
	connectionID := uuid.New().String()
	connInfo := &model.ConnectionInfo{
		SessionID:     c.GetHeader("X-Session-ID"),
		DeviceID:      claims.DeviceID,
		ClientID:      claims.ClientID,
		UserID:        claims.UserID,
		Headers:       h.extractHeaders(c.Request),
		ConnectedAt:   time.Now(),
		LastActiveAt:  time.Now(),
		TransportType: "websocket",
	}

	// 创建连接对象
	wsConn := service.NewConnection(connectionID, conn, connInfo)

	// 添加到连接管理器
	h.connMgr.AddConnection(wsConn)

	// 创建gRPC流
	_, err = h.grpcClient.CreateStream(connectionID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"error":         err,
		}).Error("创建gRPC流失败")
		wsConn.Close()
		return
	}

	// 启动消息循环
	h.msgRouter.StartMessageLoop(connectionID)

	h.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"user_id":       claims.UserID,
		"device_id":     claims.DeviceID,
	}).Info("WebSocket连接已建立")

	// 启动连接处理
	go h.handleConnection(wsConn)
}

// handleConnection 处理WebSocket连接
func (h *WebSocketHandler) handleConnection(conn *service.Connection) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.WithFields(logrus.Fields{
				"connection_id": conn.ID,
				"panic":         r,
			}).Error("WebSocket连接处理发生panic")
		}
		h.cleanup(conn)
	}()

	// 启动读写协程
	go h.readPump(conn)
	go h.writePump(conn)

	// 等待连接关闭
	<-conn.CloseCh
}

// readPump 读取消息
func (h *WebSocketHandler) readPump(conn *service.Connection) {
	defer func() {
		conn.Close()
	}()

	// 设置读取超时和限制
	conn.Conn.SetReadLimit(1024 * 1024) // 1MB
	conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.UpdateLastActive()
		return nil
	})

	for {
		select {
		case <-conn.CloseCh:
			return
		default:
			// 读取消息，获取消息类型
			messageType, message, err := conn.Conn.ReadMessage()
			fmt.Println("读取到消息:", string(message), messageType)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.logger.WithFields(logrus.Fields{
						"connection_id": conn.ID,
						"error":         err,
					}).Error("WebSocket读取错误")
				}
				return
			}

			// 更新活跃时间
			conn.UpdateLastActive()

			// 根据消息类型处理
			switch messageType {
			case websocket.TextMessage:
				// 处理文本消息（JSON格式）
				if err := h.msgRouter.RouteWebSocketMessage(conn.ID, message); err != nil {
					h.logger.WithError(err).Error("处理文本消息失败")
				}
			case websocket.BinaryMessage:
				// 处理二进制消息（音频数据）
				if err := h.handleBinaryMessage(conn.Conn, conn.ID, message); err != nil {
					h.logger.WithError(err).Error("处理二进制消息失败")
				}
			default:
				h.logger.WithFields(logrus.Fields{
					"connection_id": conn.ID,
					"message_type":  messageType,
				}).Warn("不支持的消息类型")
			}
		}
	}
}

// handleTextMessage 处理文本消息
func (h *WebSocketHandler) handleTextMessage(conn *websocket.Conn, connectionID string, data []byte) error {
	// 解析JSON消息
	// var msgJSON interface{}
	// if err := json.Unmarshal(data, &msgJSON); err != nil {
	// 	return conn.WriteMessage(1, data)
	// }

	// // 检查是否为整数类型
	// if _, ok := msgJSON.(float64); ok {
	// 	return conn.WriteMessage(1, data)
	// }

	// // 解析为map类型处理具体消息
	// msgMap, ok := msgJSON.(map[string]interface{})
	// if !ok {
	// 	return fmt.Errorf("消息格式错误")
	// }

	// 解析WebSocket消息
	// var msg model.WebSocketMessage
	// if err := json.Unmarshal(data, &msg); err != nil {
	// 	h.logger.WithError(err).Error("解析WebSocket消息失败")
	// 	return err
	// }

	// 设置时间戳
	// msg.Timestamp = time.Now().UnixNano() / int64(time.Millisecond)

	// 路由消息到gRPC
	if err := h.msgRouter.RouteWebSocketMessage(connectionID, data); err != nil {
		h.logger.WithError(err).Error("路由WebSocket消息失败")
		return err
	}

	return nil
}

// handleBinaryMessage 处理二进制消息（音频数据）
func (h *WebSocketHandler) handleBinaryMessage(conn *websocket.Conn, connectionID string, data []byte) error {
	h.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"data_size":     len(data),
	}).Debug("接收到二进制音频数据")

	// 直接路由音频数据到gRPC
	if err := h.msgRouter.RouteBinaryMessage(connectionID, data); err != nil {
		h.logger.WithError(err).Error("路由二进制音频数据失败")
		return err
	}

	return nil
}

// writePump 写入消息
func (h *WebSocketHandler) writePump(conn *service.Connection) {
	ticker := time.NewTicker(54 * time.Second) // ping间隔
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case <-conn.CloseCh:
			return
		case message, ok := <-conn.SendCh:
			fmt.Println("messagemessagemessageconn.SendCh", message)
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				h.logger.WithFields(logrus.Fields{
					"connection_id": conn.ID,
					"error":         err,
				}).Error("WebSocket写入失败")
				return
			}

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.logger.WithFields(logrus.Fields{
					"connection_id": conn.ID,
					"error":         err,
				}).Error("发送ping失败")
				return
			}
		}
	}
}

// sendErrorMessage 发送错误消息
func (h *WebSocketHandler) sendErrorMessage(conn *service.Connection, errorCode, errorMessage string) {
	errMsg := &model.WebSocketMessage{
		Type:      "error_response",
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Data: map[string]interface{}{
			"error_code":    errorCode,
			"error_message": errorMessage,
		},
	}

	data, err := json.Marshal(errMsg)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"connection_id": conn.ID,
			"error":         err,
		}).Error("错误消息序列化失败")
		return
	}

	conn.Send(data)
}

// cleanup 清理连接资源
func (h *WebSocketHandler) cleanup(conn *service.Connection) {
	// 移除gRPC流
	h.grpcClient.RemoveStream(conn.ID)

	// 移除连接
	h.connMgr.RemoveConnection(conn.ID)

	h.logger.WithField("connection_id", conn.ID).Info("WebSocket连接已清理")
}

// extractHeaders 提取请求头
func (h *WebSocketHandler) extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}
