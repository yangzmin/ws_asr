package service

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"xiaozhi-im-service/internal/model"
)

// Connection WebSocket连接封装
type Connection struct {
	ID       string
	Conn     *websocket.Conn
	Info     *model.ConnectionInfo
	SendCh   chan []byte
	CloseCh  chan struct{}
	mu       sync.RWMutex
	isClosed bool
}

// NewConnection 创建新连接
func NewConnection(id string, conn *websocket.Conn, info *model.ConnectionInfo) *Connection {
	return &Connection{
		ID:       id,
		Conn:     conn,
		Info:     info,
		SendCh:   make(chan []byte, 256),
		CloseCh:  make(chan struct{}),
		isClosed: false,
	}
}

// Send 发送消息
func (c *Connection) Send(data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isClosed {
		return ErrConnectionClosed
	}

	select {
	case c.SendCh <- data:
		return nil
	default:
		return ErrSendChannelFull
	}
}

// Close 关闭连接
func (c *Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isClosed {
		c.isClosed = true
		close(c.CloseCh)
		close(c.SendCh)
		c.Conn.Close()
	}
}

// IsClosed 检查连接是否已关闭
func (c *Connection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isClosed
}

// UpdateLastActive 更新最后活跃时间
func (c *Connection) UpdateLastActive() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Info.LastActiveAt = time.Now()
}

// ConnectionManager 连接管理器
type ConnectionManager struct {
	connections map[string]*Connection
	mu          sync.RWMutex
	logger      *logrus.Logger
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(logger *logrus.Logger) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*Connection),
		logger:      logger,
	}
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(conn *Connection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.connections[conn.ID] = conn
	cm.logger.WithFields(logrus.Fields{
		"connection_id": conn.ID,
		"user_id":       conn.Info.UserID,
		"device_id":     conn.Info.DeviceID,
	}).Info("连接已添加")
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(connectionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, exists := cm.connections[connectionID]; exists {
		conn.Close()
		delete(cm.connections, connectionID)
		cm.logger.WithField("connection_id", connectionID).Info("连接已移除")
	}
}

// GetConnection 获取连接
func (cm *ConnectionManager) GetConnection(connectionID string) (*Connection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connections[connectionID]
	return conn, exists
}

// GetConnectionsByUserID 根据用户ID获取连接
func (cm *ConnectionManager) GetConnectionsByUserID(userID string) []*Connection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var connections []*Connection
	for _, conn := range cm.connections {
		if conn.Info.UserID == userID {
			connections = append(connections, conn)
		}
	}
	return connections
}

// GetAllConnections 获取所有连接
func (cm *ConnectionManager) GetAllConnections() []*Connection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	connections := make([]*Connection, 0, len(cm.connections))
	for _, conn := range cm.connections {
		connections = append(connections, conn)
	}
	return connections
}

// GetConnectionCount 获取连接数量
func (cm *ConnectionManager) GetConnectionCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.connections)
}

// CleanupInactiveConnections 清理不活跃的连接
func (cm *ConnectionManager) CleanupInactiveConnections(ctx context.Context, timeout time.Duration) {
	ticker := time.NewTicker(timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.cleanupInactive(timeout)
		}
	}
}

// cleanupInactive 清理不活跃的连接
func (cm *ConnectionManager) cleanupInactive(timeout time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for id, conn := range cm.connections {
		if now.Sub(conn.Info.LastActiveAt) > timeout {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		if conn, exists := cm.connections[id]; exists {
			conn.Close()
			delete(cm.connections, id)
			cm.logger.WithField("connection_id", id).Info("清理不活跃连接")
		}
	}
}

// Shutdown 关闭所有连接
func (cm *ConnectionManager) Shutdown() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for id, conn := range cm.connections {
		conn.Close()
		delete(cm.connections, id)
	}
	cm.logger.Info("所有连接已关闭")
}