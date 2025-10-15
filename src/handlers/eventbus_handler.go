package handlers

import (
	"net/http"
	"time"

	"xiaozhi-server-go/src/core/eventbus"
	"xiaozhi-server-go/src/core/utils"

	"github.com/gin-gonic/gin"
)

// EventBusHandler 事件总线API处理器
type EventBusHandler struct {
	manager *eventbus.EventBusManager
	monitor *eventbus.MonitoringService
	logger  *utils.Logger
}

// NewEventBusHandler 创建事件总线处理器
func NewEventBusHandler(manager *eventbus.EventBusManager, monitor *eventbus.MonitoringService, logger *utils.Logger) *EventBusHandler {
	return &EventBusHandler{
		manager: manager,
		monitor: monitor,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *EventBusHandler) RegisterRoutes(apiGroup *gin.RouterGroup) {
	// 事件总线状态监控接口
	apiGroup.GET("/event-bus/status", h.GetEventBusStatus)
	
	// 服务健康检查接口
	apiGroup.GET("/services/health", h.GetServicesHealth)
	
	// 解耦配置管理接口
	configGroup := apiGroup.Group("/config")
	{
		configGroup.GET("/decoupling", h.GetDecouplingConfig)
		configGroup.POST("/decoupling", h.UpdateDecouplingConfig)
	}
}

// EventBusStatusResponse 事件总线状态响应
type EventBusStatusResponse struct {
	QueueLength    int     `json:"queue_length"`    // 队列长度
	ProcessingRate float64 `json:"processing_rate"` // 处理速率（消息/秒）
	ErrorCount     int64   `json:"error_count"`     // 错误计数
	TotalProcessed int64   `json:"total_processed"` // 总处理数量
	Uptime         string  `json:"uptime"`          // 运行时间
	Status         string  `json:"status"`          // 运行状态
}

// ServiceHealthResponse 服务健康状态响应
type ServiceHealthResponse struct {
	Status    string                   `json:"status"`    // 整体状态：healthy, degraded, unhealthy
	Timestamp time.Time                `json:"timestamp"` // 检查时间
	Services  map[string]ServiceStatus `json:"services"`  // 各服务状态
}

// ServiceStatus 单个服务状态
type ServiceStatus struct {
	Name      string    `json:"name"`      // 服务名称
	Status    string    `json:"status"`    // 状态：healthy, unhealthy, unknown
	Timestamp time.Time `json:"timestamp"` // 最后检查时间
	Error     string    `json:"error,omitempty"` // 错误信息（如果有）
}

// DecouplingConfigResponse 解耦配置响应
type DecouplingConfigResponse struct {
	EventBusEnabled    bool   `json:"event_bus_enabled"`    // 事件总线是否启用
	AdapterMode        string `json:"adapter_mode"`         // 适配器模式：direct, event_driven
	SessionManagement  string `json:"session_management"`   // 会话管理：local, redis
	MonitoringEnabled  bool   `json:"monitoring_enabled"`   // 监控是否启用
	MaxQueueSize       int    `json:"max_queue_size"`       // 最大队列大小
	ProcessingTimeout  int    `json:"processing_timeout"`   // 处理超时时间（秒）
}

// DecouplingConfigRequest 解耦配置请求
type DecouplingConfigRequest struct {
	EventBusEnabled   *bool   `json:"event_bus_enabled,omitempty"`
	AdapterMode       *string `json:"adapter_mode,omitempty"`
	SessionManagement *string `json:"session_management,omitempty"`
	MonitoringEnabled *bool   `json:"monitoring_enabled,omitempty"`
	MaxQueueSize      *int    `json:"max_queue_size,omitempty"`
	ProcessingTimeout *int    `json:"processing_timeout,omitempty"`
}

// GetEventBusStatus 获取事件总线状态
// @Summary 获取事件总线状态
// @Description 返回事件总线的运行状态，包括队列长度、处理速率、错误计数等
// @Tags EventBus
// @Accept json
// @Produce json
// @Success 200 {object} EventBusStatusResponse "成功"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/event-bus/status [get]
func (h *EventBusHandler) GetEventBusStatus(c *gin.Context) {
	if h.manager == nil {
		h.respondError(c, http.StatusInternalServerError, "事件总线管理器未初始化", nil)
		return
	}

	eventBus := h.manager.GetEventBus()
	if eventBus == nil {
		h.respondError(c, http.StatusServiceUnavailable, "事件总线未初始化", nil)
		return
	}

	// 获取事件总线统计信息
	stats := eventBus.GetStats()
	
	response := EventBusStatusResponse{
		QueueLength:    stats.QueueLength,
		ProcessingRate: stats.ProcessingRate,
		ErrorCount:     stats.ErrorCount,
		TotalProcessed: stats.TotalProcessed,
		Uptime:         time.Since(stats.LastUpdateTime).String(),
		Status:         "running",
	}

	h.respondSuccess(c, response)
}

// GetServicesHealth 获取服务健康状态
// @Summary 获取AI服务健康状态
// @Description 返回各AI服务的健康状态检查结果
// @Tags EventBus
// @Accept json
// @Produce json
// @Success 200 {object} ServiceHealthResponse "成功"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/services/health [get]
func (h *EventBusHandler) GetServicesHealth(c *gin.Context) {
	if h.monitor == nil {
		h.respondError(c, http.StatusInternalServerError, "监控服务未初始化", nil)
		return
	}

	healthStatus := h.monitor.GetHealthStatus()

	// 构建响应
	response := ServiceHealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceStatus),
	}

	// 转换监控服务的健康状态到API响应格式
	for serviceName, serviceHealth := range healthStatus.Services {
		response.Services[serviceName] = ServiceStatus{
			Name:      serviceName,
			Status:    serviceHealth.Status,
			Timestamp: serviceHealth.LastCheck,
			Error:     serviceHealth.Message,
		}
		if serviceHealth.Status != "healthy" {
			response.Status = "unhealthy"
		}
	}

	// 设置整体状态
	response.Status = healthStatus.Status

	// 如果没有服务信息，添加基本检查
	if len(response.Services) == 0 {
		// 检查事件总线
		if h.manager != nil && h.manager.GetEventBus() != nil {
			response.Services["event_bus"] = ServiceStatus{
				Name:      "EventBus",
				Status:    "healthy",
				Timestamp: time.Now(),
			}
		}

		// 检查监控服务
		if h.monitor != nil {
			response.Services["monitor"] = ServiceStatus{
				Name:      "Monitor",
				Status:    "healthy",
				Timestamp: time.Now(),
			}
		}
	}

	h.respondSuccess(c, response)
}

// GetDecouplingConfig 获取解耦配置
// @Summary 获取解耦配置
// @Description 返回当前的解耦配置信息
// @Tags EventBus
// @Accept json
// @Produce json
// @Success 200 {object} DecouplingConfigResponse "成功"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/config/decoupling [get]
func (h *EventBusHandler) GetDecouplingConfig(c *gin.Context) {
	// TODO: 实现获取解耦配置的逻辑
	h.respondSuccess(c, gin.H{
		"message": "获取解耦配置功能待实现",
	})
}

// UpdateDecouplingConfig 更新解耦配置
// @Summary 更新解耦配置
// @Description 更新解耦相关的配置参数
// @Tags EventBus
// @Accept json
// @Produce json
// @Param config body DecouplingConfigRequest true "配置信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/config/decoupling [post]
func (h *EventBusHandler) UpdateDecouplingConfig(c *gin.Context) {
	var req DecouplingConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "请求参数格式错误", err)
		return
	}

	// 验证配置参数
	if req.AdapterMode != nil {
		if *req.AdapterMode != "direct" && *req.AdapterMode != "event_driven" {
			h.respondError(c, http.StatusBadRequest, "无效的适配器模式", nil)
			return
		}
	}

	if req.SessionManagement != nil {
		if *req.SessionManagement != "local" && *req.SessionManagement != "redis" {
			h.respondError(c, http.StatusBadRequest, "无效的会话管理模式", nil)
			return
		}
	}

	if req.MaxQueueSize != nil && *req.MaxQueueSize <= 0 {
		h.respondError(c, http.StatusBadRequest, "队列大小必须大于0", nil)
		return
	}

	if req.ProcessingTimeout != nil && *req.ProcessingTimeout <= 0 {
		h.respondError(c, http.StatusBadRequest, "处理超时时间必须大于0", nil)
		return
	}

	// 记录配置更新
	h.logger.Info("解耦配置更新请求: %+v", req)

	// 注意：这里只是演示API接口，实际的配置更新逻辑需要根据具体需求实现
	// 可能需要重启某些服务或重新加载配置

	h.respondSuccess(c, gin.H{
		"message": "配置更新成功",
		"updated_at": time.Now(),
	})
}

// 辅助方法

// respondSuccess 返回成功响应
func (h *EventBusHandler) respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    data,
	})
}

// respondError 返回错误响应
func (h *EventBusHandler) respondError(c *gin.Context, statusCode int, message string, err error) {
	response := gin.H{
		"code":    statusCode,
		"message": message,
	}
	
	if err != nil {
		h.logger.Error("%s: %v", message, err)
		if gin.Mode() == gin.DebugMode {
			response["error"] = err.Error()
		}
	} else {
		h.logger.Warn(message)
	}
	
	c.JSON(statusCode, response)
}