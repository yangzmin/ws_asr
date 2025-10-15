package eventbus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// MonitoringService 监控服务
type MonitoringService struct {
	eventBus       *EventBus
	sessionManager *SessionManager
	server         *http.Server
	metrics        *SystemMetrics
	mutex          sync.RWMutex
	startTime      time.Time
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	StartTime        time.Time `json:"start_time"`
	Uptime          string    `json:"uptime"`
	GoVersion       string    `json:"go_version"`
	NumGoroutines   int       `json:"num_goroutines"`
	MemoryUsage     MemoryStats `json:"memory_usage"`
	EventBusStats   *EventBusStats `json:"event_bus_stats"`
	SessionStats    map[string]interface{} `json:"session_stats"`
	AdapterStats    map[string]*ServiceAdapter `json:"adapter_stats"`
	LastUpdateTime  time.Time `json:"last_update_time"`
}

// MemoryStats 内存统计
type MemoryStats struct {
	Alloc        uint64 `json:"alloc"`         // 当前分配的内存
	TotalAlloc   uint64 `json:"total_alloc"`   // 总分配的内存
	Sys          uint64 `json:"sys"`           // 系统内存
	NumGC        uint32 `json:"num_gc"`        // GC次数
	HeapAlloc    uint64 `json:"heap_alloc"`    // 堆内存分配
	HeapSys      uint64 `json:"heap_sys"`      // 堆系统内存
	HeapInuse    uint64 `json:"heap_inuse"`    // 堆使用内存
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status      string                     `json:"status"`      // "healthy", "degraded", "unhealthy"
	Timestamp   time.Time                  `json:"timestamp"`
	Services    map[string]ServiceHealth   `json:"services"`
	Issues      []string                   `json:"issues,omitempty"`
	Uptime      string                     `json:"uptime"`
}

// ServiceHealth 服务健康状态
type ServiceHealth struct {
	Status      string    `json:"status"`
	LastCheck   time.Time `json:"last_check"`
	ErrorCount  int64     `json:"error_count"`
	Latency     string    `json:"latency"`
	Message     string    `json:"message,omitempty"`
}

// NewMonitoringService 创建监控服务
func NewMonitoringService(eventBus *EventBus, sessionManager *SessionManager, port int) *MonitoringService {
	ms := &MonitoringService{
		eventBus:       eventBus,
		sessionManager: sessionManager,
		metrics:        &SystemMetrics{},
		startTime:      time.Now(),
	}

	// 设置HTTP服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/health", ms.healthHandler)
	mux.HandleFunc("/metrics", ms.metricsHandler)
	mux.HandleFunc("/stats", ms.statsHandler)
	mux.HandleFunc("/sessions", ms.sessionsHandler)
	mux.HandleFunc("/adapters", ms.adaptersHandler)

	ms.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return ms
}

// Start 启动监控服务
func (ms *MonitoringService) Start() error {
	// 启动指标收集
	go ms.collectMetrics()
	
	// 启动HTTP服务器
	go func() {
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("监控服务启动失败: %v\n", err)
		}
	}()

	fmt.Printf("监控服务已启动，端口: %s\n", ms.server.Addr)
	return nil
}

// Stop 停止监控服务
func (ms *MonitoringService) Stop() error {
	if ms.server != nil {
		return ms.server.Close()
	}
	return nil
}

// collectMetrics 收集系统指标
func (ms *MonitoringService) collectMetrics() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for range ticker.C {
		ms.updateMetrics()
	}
}

// updateMetrics 更新指标
func (ms *MonitoringService) updateMetrics() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	ms.metrics.StartTime = ms.startTime
	ms.metrics.Uptime = time.Since(ms.startTime).String()
	ms.metrics.GoVersion = runtime.Version()
	ms.metrics.NumGoroutines = runtime.NumGoroutine()
	ms.metrics.MemoryUsage = MemoryStats{
		Alloc:      memStats.Alloc,
		TotalAlloc: memStats.TotalAlloc,
		Sys:        memStats.Sys,
		NumGC:      memStats.NumGC,
		HeapAlloc:  memStats.HeapAlloc,
		HeapSys:    memStats.HeapSys,
		HeapInuse:  memStats.HeapInuse,
	}
	ms.metrics.LastUpdateTime = time.Now()

	// 更新事件总线统计
	if ms.eventBus != nil {
		ms.metrics.EventBusStats = ms.eventBus.GetStats()
		ms.metrics.AdapterStats = ms.eventBus.GetAdapters()
	}

	// 更新会话统计
	if ms.sessionManager != nil {
		ms.metrics.SessionStats = ms.sessionManager.GetStats()
	}
}

// GetMetrics 获取系统指标
func (ms *MonitoringService) GetMetrics() *SystemMetrics {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	// 返回副本
	metrics := *ms.metrics
	return &metrics
}

// GetHealthStatus 获取健康状态
func (ms *MonitoringService) GetHealthStatus() *HealthStatus {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	health := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceHealth),
		Issues:    make([]string, 0),
		Uptime:    time.Since(ms.startTime).String(),
	}

	// 检查事件总线健康状态
	if ms.eventBus != nil {
		stats := ms.eventBus.GetStats()
		ebHealth := ServiceHealth{
			Status:     "healthy",
			LastCheck:  time.Now(),
			ErrorCount: stats.ErrorCount,
			Latency:    fmt.Sprintf("%.2fms", stats.AverageLatency),
		}

		// 检查错误率
		if stats.TotalProcessed > 0 {
			errorRate := float64(stats.ErrorCount) / float64(stats.TotalProcessed)
			if errorRate > 0.1 { // 错误率超过10%
				ebHealth.Status = "degraded"
				ebHealth.Message = fmt.Sprintf("高错误率: %.2f%%", errorRate*100)
				health.Issues = append(health.Issues, ebHealth.Message)
			}
		}

		// 检查队列长度
		if stats.QueueLength > 500 { // 队列长度超过500
			ebHealth.Status = "degraded"
			ebHealth.Message = fmt.Sprintf("队列积压: %d", stats.QueueLength)
			health.Issues = append(health.Issues, ebHealth.Message)
		}

		health.Services["eventbus"] = ebHealth
	}

	// 检查会话管理器健康状态
	if ms.sessionManager != nil {
		sessionStats := ms.sessionManager.GetStats()
		smHealth := ServiceHealth{
			Status:    "healthy",
			LastCheck: time.Now(),
		}

		// 检查会话数量
		if totalSessions, ok := sessionStats["total_sessions"].(int); ok {
			if maxSessions, ok := sessionStats["max_sessions"].(int); ok {
				if float64(totalSessions)/float64(maxSessions) > 0.8 { // 使用率超过80%
					smHealth.Status = "degraded"
					smHealth.Message = fmt.Sprintf("会话使用率高: %d/%d", totalSessions, maxSessions)
					health.Issues = append(health.Issues, smHealth.Message)
				}
			}
		}

		health.Services["session_manager"] = smHealth
	}

	// 检查适配器健康状态
	if ms.eventBus != nil {
		adapters := ms.eventBus.GetAdapters()
		for name, adapter := range adapters {
			adapterHealth := ServiceHealth{
				Status:     "healthy",
				LastCheck:  adapter.LastCheck,
				ErrorCount: adapter.ErrorCount,
				Latency:    adapter.AvgLatency.String(),
			}

			if adapter.Status != "active" {
				adapterHealth.Status = "unhealthy"
				adapterHealth.Message = fmt.Sprintf("适配器状态: %s", adapter.Status)
				health.Issues = append(health.Issues, fmt.Sprintf("%s: %s", name, adapterHealth.Message))
			}

			health.Services[fmt.Sprintf("adapter_%s", name)] = adapterHealth
		}
	}

	// 检查内存使用
	memUsage := ms.metrics.MemoryUsage
	if memUsage.HeapInuse > 1024*1024*1024 { // 堆内存使用超过1GB
		health.Issues = append(health.Issues, fmt.Sprintf("内存使用过高: %d MB", memUsage.HeapInuse/1024/1024))
	}

	// 根据问题数量确定整体健康状态
	if len(health.Issues) > 0 {
		if len(health.Issues) > 3 {
			health.Status = "unhealthy"
		} else {
			health.Status = "degraded"
		}
	}

	return health
}

// HTTP处理器

// healthHandler 健康检查处理器
func (ms *MonitoringService) healthHandler(w http.ResponseWriter, r *http.Request) {
	health := ms.GetHealthStatus()
	
	w.Header().Set("Content-Type", "application/json")
	
	// 根据健康状态设置HTTP状态码
	switch health.Status {
	case "healthy":
		w.WriteHeader(http.StatusOK)
	case "degraded":
		w.WriteHeader(http.StatusOK) // 降级但仍可用
	case "unhealthy":
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(health)
}

// metricsHandler 指标处理器
func (ms *MonitoringService) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := ms.GetMetrics()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	json.NewEncoder(w).Encode(metrics)
}

// statsHandler 统计信息处理器
func (ms *MonitoringService) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})
	
	if ms.eventBus != nil {
		stats["eventbus"] = ms.eventBus.GetStats()
	}
	
	if ms.sessionManager != nil {
		stats["sessions"] = ms.sessionManager.GetStats()
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	json.NewEncoder(w).Encode(stats)
}

// sessionsHandler 会话信息处理器
func (ms *MonitoringService) sessionsHandler(w http.ResponseWriter, r *http.Request) {
	if ms.sessionManager == nil {
		http.Error(w, "Session manager not available", http.StatusServiceUnavailable)
		return
	}
	
	sessions := ms.sessionManager.ListSessions()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	json.NewEncoder(w).Encode(sessions)
}

// adaptersHandler 适配器信息处理器
func (ms *MonitoringService) adaptersHandler(w http.ResponseWriter, r *http.Request) {
	if ms.eventBus == nil {
		http.Error(w, "Event bus not available", http.StatusServiceUnavailable)
		return
	}
	
	adapters := ms.eventBus.GetAdapters()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	json.NewEncoder(w).Encode(adapters)
}

// collectSystemMetrics 收集系统指标
func (m *MonitoringService) collectSystemMetrics() SystemMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemMetrics{
		StartTime:     m.startTime,
		Uptime:        time.Since(m.startTime).String(),
		GoVersion:     runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		MemoryUsage: MemoryStats{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
			HeapAlloc:  memStats.HeapAlloc,
			HeapSys:    memStats.HeapSys,
			HeapInuse:  memStats.HeapInuse,
		},
		LastUpdateTime: time.Now(),
	}
}

// GetSystemMetrics 获取系统指标
func (m *MonitoringService) GetSystemMetrics() SystemMetrics {
	return m.collectSystemMetrics()
}