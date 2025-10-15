package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionState 会话状态
type SessionState struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	LastActive  time.Time              `json:"last_active"`
	Status      string                 `json:"status"` // "active", "idle", "closed"
	Metadata    map[string]interface{} `json:"metadata"`
	MessageCount int64                 `json:"message_count"`
	mutex       sync.RWMutex
}

// SessionManager 会话管理器
type SessionManager struct {
	sessions    map[string]*SessionState
	mutex       sync.RWMutex
	cleanupTick *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
	config      *SessionConfig
}

// SessionConfig 会话配置
type SessionConfig struct {
	MaxSessions     int           `json:"max_sessions"`
	SessionTimeout  time.Duration `json:"session_timeout"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	EnableMetrics   bool          `json:"enable_metrics"`
}

// DefaultSessionConfig 默认会话配置
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		MaxSessions:     1000,
		SessionTimeout:  time.Hour * 2,
		CleanupInterval: time.Minute * 10,
		EnableMetrics:   true,
	}
}

// NewSessionManager 创建会话管理器
func NewSessionManager(config *SessionConfig) *SessionManager {
	if config == nil {
		config = DefaultSessionConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	sm := &SessionManager{
		sessions: make(map[string]*SessionState),
		ctx:      ctx,
		cancel:   cancel,
		config:   config,
	}

	// 启动清理协程
	sm.startCleanup()

	return sm
}

// CreateSession 创建新会话
func (sm *SessionManager) CreateSession(userID string, metadata map[string]interface{}) (*SessionState, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// 检查会话数量限制
	if len(sm.sessions) >= sm.config.MaxSessions {
		return nil, fmt.Errorf("maximum sessions limit reached: %d", sm.config.MaxSessions)
	}

	sessionID := uuid.New().String()
	now := time.Now()

	session := &SessionState{
		ID:          sessionID,
		UserID:      userID,
		CreatedAt:   now,
		LastActive:  now,
		Status:      "active",
		Metadata:    make(map[string]interface{}),
		MessageCount: 0,
	}

	// 复制元数据
	if metadata != nil {
		for k, v := range metadata {
			session.Metadata[k] = v
		}
	}

	sm.sessions[sessionID] = session
	return session, nil
}

// GetSession 获取会话
func (sm *SessionManager) GetSession(sessionID string) (*SessionState, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// UpdateSession 更新会话
func (sm *SessionManager) UpdateSession(sessionID string, metadata map[string]interface{}) error {
	sm.mutex.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.LastActive = time.Now()
	
	// 更新元数据
	if metadata != nil {
		for k, v := range metadata {
			session.Metadata[k] = v
		}
	}

	return nil
}

// IncrementMessageCount 增加消息计数
func (sm *SessionManager) IncrementMessageCount(sessionID string) error {
	sm.mutex.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.MessageCount++
	session.LastActive = time.Now()

	return nil
}

// SetSessionStatus 设置会话状态
func (sm *SessionManager) SetSessionStatus(sessionID string, status string) error {
	sm.mutex.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Status = status
	session.LastActive = time.Now()

	return nil
}

// CloseSession 关闭会话
func (sm *SessionManager) CloseSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mutex.Lock()
	session.Status = "closed"
	session.mutex.Unlock()

	delete(sm.sessions, sessionID)
	return nil
}

// ListSessions 列出所有会话
func (sm *SessionManager) ListSessions() []*SessionState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	sessions := make([]*SessionState, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// GetSessionsByUser 获取用户的所有会话
func (sm *SessionManager) GetSessionsByUser(userID string) []*SessionState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var sessions []*SessionState
	for _, session := range sm.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// GetStats 获取会话统计信息
func (sm *SessionManager) GetStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// 按状态统计
	statusCount := make(map[string]int)
	var totalMessages int64
	activeSessions := 0
	
	for _, session := range sm.sessions {
		session.mutex.RLock()
		statusCount[session.Status]++
		totalMessages += session.MessageCount
		if session.Status == "active" {
			activeSessions++
		}
		session.mutex.RUnlock()
	}

	stats := map[string]interface{}{
		"total_sessions":   len(sm.sessions),
		"active_sessions":  activeSessions,
		"max_sessions":     sm.config.MaxSessions,
		"session_timeout":  sm.config.SessionTimeout.String(),
		"cleanup_interval": sm.config.CleanupInterval.String(),
		"status_count":     statusCount,
		"total_messages":   totalMessages,
	}

	return stats
}

// startCleanup 启动清理协程
func (sm *SessionManager) startCleanup() {
	sm.cleanupTick = time.NewTicker(sm.config.CleanupInterval)
	
	go func() {
		for {
			select {
			case <-sm.ctx.Done():
				sm.cleanupTick.Stop()
				return
			case <-sm.cleanupTick.C:
				sm.cleanupExpiredSessions()
			}
		}
	}()
}

// cleanupExpiredSessions 清理过期会话
func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	expiredSessions := make([]string, 0)

	for sessionID, session := range sm.sessions {
		session.mutex.RLock()
		if now.Sub(session.LastActive) > sm.config.SessionTimeout {
			expiredSessions = append(expiredSessions, sessionID)
		}
		session.mutex.RUnlock()
	}

	// 删除过期会话
	for _, sessionID := range expiredSessions {
		if session, exists := sm.sessions[sessionID]; exists {
			session.mutex.Lock()
			session.Status = "expired"
			session.mutex.Unlock()
			delete(sm.sessions, sessionID)
		}
	}
}

// Shutdown 关闭会话管理器
func (sm *SessionManager) Shutdown() {
	sm.cancel()
	
	if sm.cleanupTick != nil {
		sm.cleanupTick.Stop()
	}

	// 关闭所有会话
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for sessionID, session := range sm.sessions {
		session.mutex.Lock()
		session.Status = "closed"
		session.mutex.Unlock()
		delete(sm.sessions, sessionID)
	}
}

// SessionState 方法

// ToJSON 转换为JSON
func (s *SessionState) ToJSON() ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return json.Marshal(s)
}

// UpdateMetadata 更新元数据
func (s *SessionState) UpdateMetadata(key string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	
	s.Metadata[key] = value
	s.LastActive = time.Now()
}

// GetMetadata 获取元数据
func (s *SessionState) GetMetadata(key string) (interface{}, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	if s.Metadata == nil {
		return nil, false
	}
	
	value, exists := s.Metadata[key]
	return value, exists
}

// IsActive 检查会话是否活跃
func (s *SessionState) IsActive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return s.Status == "active"
}

// GetAge 获取会话年龄
func (s *SessionState) GetAge() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return time.Since(s.CreatedAt)
}

// GetIdleTime 获取空闲时间
func (s *SessionState) GetIdleTime() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return time.Since(s.LastActive)
}