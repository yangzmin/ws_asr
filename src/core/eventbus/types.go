package eventbus

import (
	"sync"
	"time"
)

// EventMessage 事件消息结构
type EventMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`      // "asr", "llm", "tts", "mcp"
	SessionID string                 `json:"session_id"`
	Payload   []byte                 `json:"payload"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	Status    string                 `json:"status"` // "pending", "processing", "completed", "failed"
	Priority  int                    `json:"priority"` // 0-10, 10为最高优先级
}

// ServiceAdapter 服务适配器结构
type ServiceAdapter struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Status     string        `json:"status"`
	PoolSize   int          `json:"pool_size"`
	AvgLatency time.Duration `json:"avg_latency"`
	ErrorCount int64        `json:"error_count"`
	LastCheck  time.Time    `json:"last_check"`
	mutex      sync.RWMutex
}

// EventBusStats 事件总线统计信息
type EventBusStats struct {
	QueueLength     int     `json:"queue_length"`
	ProcessingRate  float64 `json:"processing_rate"`
	ErrorCount      int64   `json:"error_count"`
	TotalProcessed  int64   `json:"total_processed"`
	AverageLatency  float64 `json:"average_latency"`
	LastUpdateTime  time.Time `json:"last_update_time"`
}

// EventHandler 事件处理器接口
type EventHandler interface {
	Handle(msg *EventMessage) error
	GetType() string
	GetStatus() string
}

// EventBusConfig 事件总线配置
type EventBusConfig struct {
	ChannelBufferSize int           `json:"channel_buffer_size"`
	WorkerCount       int           `json:"worker_count"`
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	StatsInterval     time.Duration `json:"stats_interval"`
}

// DefaultEventBusConfig 默认配置
func DefaultEventBusConfig() *EventBusConfig {
	return &EventBusConfig{
		ChannelBufferSize: 1000,
		WorkerCount:       4,
		MaxRetries:        3,
		RetryDelay:        time.Millisecond * 100,
		StatsInterval:     time.Second * 5,
	}
}