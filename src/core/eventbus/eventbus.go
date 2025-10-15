package eventbus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// EventBus 事件总线结构
type EventBus struct {
	config   *EventBusConfig
	channels map[string]chan *EventMessage
	handlers map[string]EventHandler
	adapters map[string]*ServiceAdapter
	stats    *EventBusStats
	workers  []*Worker
	ctx      context.Context
	cancel   context.CancelFunc
	mutex    sync.RWMutex
	statsMux sync.RWMutex
	running  int32
}

// Worker 工作协程结构
type Worker struct {
	id       int
	eventBus *EventBus
	ctx      context.Context
}

// NewEventBus 创建新的事件总线
func NewEventBus(config *EventBusConfig) *EventBus {
	if config == nil {
		config = DefaultEventBusConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	eb := &EventBus{
		config:   config,
		channels: make(map[string]chan *EventMessage),
		handlers: make(map[string]EventHandler),
		adapters: make(map[string]*ServiceAdapter),
		stats: &EventBusStats{
			LastUpdateTime: time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// 初始化默认通道
	eb.initDefaultChannels()

	// 启动工作协程
	eb.startWorkers()

	// 启动统计协程
	go eb.statsCollector()

	atomic.StoreInt32(&eb.running, 1)
	return eb
}

// initDefaultChannels 初始化默认通道
func (eb *EventBus) initDefaultChannels() {
	defaultTypes := []string{"asr", "llm", "tts", "mcp"}
	for _, eventType := range defaultTypes {
		eb.channels[eventType] = make(chan *EventMessage, eb.config.ChannelBufferSize)
	}
}

// startWorkers 启动工作协程
func (eb *EventBus) startWorkers() {
	eb.workers = make([]*Worker, eb.config.WorkerCount)
	for i := 0; i < eb.config.WorkerCount; i++ {
		worker := &Worker{
			id:       i,
			eventBus: eb,
			ctx:      eb.ctx,
		}
		eb.workers[i] = worker
		go worker.run()
	}
}

// RegisterHandler 注册事件处理器
func (eb *EventBus) RegisterHandler(eventType string, handler EventHandler) error {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if _, exists := eb.handlers[eventType]; exists {
		return fmt.Errorf("handler for event type %s already exists", eventType)
	}

	eb.handlers[eventType] = handler

	// 如果通道不存在，创建新通道
	if _, exists := eb.channels[eventType]; !exists {
		eb.channels[eventType] = make(chan *EventMessage, eb.config.ChannelBufferSize)
	}

	// 注册服务适配器
	adapter := &ServiceAdapter{
		Name:      eventType,
		Type:      eventType,
		Status:    "active",
		LastCheck: time.Now(),
	}
	eb.adapters[eventType] = adapter

	return nil
}

// UnregisterHandler 注销事件处理器
func (eb *EventBus) UnregisterHandler(eventType string) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	delete(eb.handlers, eventType)
	if adapter, exists := eb.adapters[eventType]; exists {
		adapter.Status = "inactive"
	}
}

// Publish 发布事件消息
func (eb *EventBus) Publish(eventType string, payload []byte, metadata map[string]interface{}) error {
	fmt.Printf("开始发布事件，类型: %s\n", eventType)

	if atomic.LoadInt32(&eb.running) == 0 {
		fmt.Printf("事件总线未运行\n")
		return fmt.Errorf("event bus is not running")
	}

	eb.mutex.RLock()
	channel, exists := eb.channels[eventType]
	eb.mutex.RUnlock()

	if !exists {
		fmt.Printf("未找到事件类型 %s 的通道\n", eventType)
		return fmt.Errorf("no channel found for event type: %s", eventType)
	}

	fmt.Printf("找到通道，创建事件消息\n")
	msg := &EventMessage{
		ID:        uuid.New().String(),
		Type:      eventType,
		Payload:   payload,
		Metadata:  metadata,
		CreatedAt: time.Now(),
		Status:    "pending",
		Priority:  5, // 默认优先级
	}

	// 从metadata中提取session_id
	if sessionID, ok := metadata["session_id"].(string); ok {
		msg.SessionID = sessionID
	}

	// 从metadata中提取priority
	if priority, ok := metadata["priority"].(int); ok {
		msg.Priority = priority
	}

	fmt.Printf("尝试发送消息到通道，消息ID: %s\n", msg.ID)
	select {
	case channel <- msg:
		atomic.AddInt64(&eb.stats.TotalProcessed, 1)
		fmt.Printf("消息发送成功，消息ID: %s\n", msg.ID)
		return nil
	default:
		atomic.AddInt64(&eb.stats.ErrorCount, 1)
		fmt.Printf("通道已满，消息发送失败，消息ID: %s\n", msg.ID)
		return fmt.Errorf("channel for event type %s is full", eventType)
	}
}

// GetStats 获取统计信息
func (eb *EventBus) GetStats() *EventBusStats {
	eb.statsMux.RLock()
	defer eb.statsMux.RUnlock()

	// 计算当前队列长度
	queueLength := 0
	eb.mutex.RLock()
	for _, channel := range eb.channels {
		queueLength += len(channel)
	}
	eb.mutex.RUnlock()

	stats := *eb.stats
	stats.QueueLength = queueLength
	return &stats
}

// GetAdapters 获取服务适配器信息
func (eb *EventBus) GetAdapters() map[string]*ServiceAdapter {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	adapters := make(map[string]*ServiceAdapter)
	for k, v := range eb.adapters {
		adapters[k] = v
	}
	return adapters
}

// Shutdown 关闭事件总线
func (eb *EventBus) Shutdown() {
	if atomic.CompareAndSwapInt32(&eb.running, 1, 0) {
		eb.cancel()

		// 关闭所有通道
		eb.mutex.Lock()
		for _, channel := range eb.channels {
			close(channel)
		}
		eb.mutex.Unlock()
	}
}

// Worker.run 工作协程运行逻辑
func (w *Worker) run() {
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			w.processMessages()
		}
	}
}

// processMessages 处理消息
func (w *Worker) processMessages() {
	types := []string{"asr", "llm", "tts", "mcp"}

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		// 轮询检查各个通道
		for _, eventType := range types {
			w.eventBus.mutex.RLock()
			channel, exists := w.eventBus.channels[eventType]
			w.eventBus.mutex.RUnlock()

			if !exists {
				continue
			}

			select {
			case msg := <-channel:
				w.handleMessage(msg)
			default:
				// 通道为空，继续下一个
			}
		}
	}

	// 短暂休眠避免CPU占用过高
	time.Sleep(time.Millisecond)
}

// handleMessage 处理单个消息
func (w *Worker) handleMessage(msg *EventMessage) {
	fmt.Printf("Worker %d 开始处理消息，ID: %s, 类型: %s\n", w.id, msg.ID, msg.Type)

	start := time.Now()
	msg.Status = "processing"

	w.eventBus.mutex.RLock()
	handler, exists := w.eventBus.handlers[msg.Type]
	w.eventBus.mutex.RUnlock()

	if !exists {
		fmt.Printf("未找到类型 %s 的处理器\n", msg.Type)
		msg.Status = "failed"
		atomic.AddInt64(&w.eventBus.stats.ErrorCount, 1)
		return
	}

	fmt.Printf("找到处理器，开始调用处理方法\n")
	fmt.Println("msg", msg)
	err := handler.Handle(msg)

	latency := time.Since(start)
	fmt.Printf("消息处理完成，耗时: %v\n", latency)

	if err != nil {
		fmt.Printf("处理消息时出错: %v\n", err)
		msg.Status = "failed"
		atomic.AddInt64(&w.eventBus.stats.ErrorCount, 1)
	} else {
		msg.Status = "completed"
		atomic.AddInt64(&w.eventBus.stats.TotalProcessed, 1)
	}

	// 更新适配器统计信息
	w.eventBus.mutex.RLock()
	adapter, exists := w.eventBus.adapters[msg.Type]
	w.eventBus.mutex.RUnlock()

	if exists {
		adapter.mutex.Lock()
		adapter.AvgLatency = (adapter.AvgLatency + latency) / 2
		if err != nil {
			adapter.ErrorCount++
		}
		adapter.LastCheck = time.Now()
		adapter.mutex.Unlock()
	}

	fmt.Printf("消息处理结束，最终状态: %s\n", msg.Status)
}

// statsCollector 统计信息收集器
func (eb *EventBus) statsCollector() {
	ticker := time.NewTicker(eb.config.StatsInterval)
	defer ticker.Stop()

	var lastProcessed int64
	var lastTime time.Time = time.Now()

	for {
		select {
		case <-eb.ctx.Done():
			return
		case <-ticker.C:
			eb.statsMux.Lock()

			currentProcessed := atomic.LoadInt64(&eb.stats.TotalProcessed)
			currentTime := time.Now()

			if !lastTime.IsZero() {
				duration := currentTime.Sub(lastTime).Seconds()
				if duration > 0 {
					eb.stats.ProcessingRate = float64(currentProcessed-lastProcessed) / duration
				}
			}

			lastProcessed = currentProcessed
			lastTime = currentTime
			eb.stats.LastUpdateTime = currentTime

			eb.statsMux.Unlock()
		}
	}
}
