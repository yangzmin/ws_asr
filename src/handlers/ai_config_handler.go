package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"xiaozhi-server-go/src/core/auth/casbin"
	"xiaozhi-server-go/src/core/utils"
	"xiaozhi-server-go/src/models"
	"xiaozhi-server-go/src/services"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AIConfigHandler AI配置处理器
type AIConfigHandler struct {
	configService services.UserAIConfigService
	logger        *utils.Logger
}

// NewAIConfigHandler 创建AI配置处理器
func NewAIConfigHandler(db *gorm.DB, logger *utils.Logger) *AIConfigHandler {
	return &AIConfigHandler{
		configService: services.NewUserAIConfigService(db, logger),
		logger:        logger,
	}
}

// RegisterRoutes 注册路由
func (h *AIConfigHandler) RegisterRoutes(apiGroup *gin.RouterGroup) {
	configGroup := apiGroup.Group("/ai-configs")
	// 添加JWT认证中间件
	configGroup.Use(h.authMiddleware())
	{
		configGroup.GET("", h.GetUserConfigs)
		configGroup.POST("", h.CreateConfig)
		configGroup.GET("/:id", h.GetConfigByID)
		configGroup.PUT("/:id", h.UpdateConfig)
		configGroup.DELETE("/:id", h.DeleteConfig)
		configGroup.PATCH("/:id/toggle", h.ToggleConfigStatus)
		configGroup.PATCH("/:id/priority", h.SetConfigPriority)
	}
}

// GetUserConfigs 获取用户配置列表
// @Summary 获取用户AI配置列表
// @Description 获取当前用户的所有AI配置
// @Tags AI配置管理
// @Accept json
// @Produce json
// @Param config_type query string false "配置类型" Enums(llm,function_call)
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/ai-configs [get]
func (h *AIConfigHandler) GetUserConfigs(c *gin.Context) {
	userID := h.getUserID(c)
	configType := c.Query("config_type")
	
	configs, err := h.configService.GetUserConfigs(c.Request.Context(), userID, configType)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "获取配置失败", err)
		return
	}
	
	// 转换为响应格式
	var responses []*models.AIConfigResponse
	for _, config := range configs {
		responses = append(responses, config.ToResponse())
	}
	
	h.respondSuccess(c, gin.H{
		"configs": responses,
		"total":   len(responses),
	})
}

// CreateConfig 创建AI配置
// @Summary 创建AI配置
// @Description 创建新的AI配置
// @Tags AI配置管理
// @Accept json
// @Produce json
// @Param config body models.CreateAIConfigRequest true "配置信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/ai-configs [post]
func (h *AIConfigHandler) CreateConfig(c *gin.Context) {
	userID := h.getUserID(c)
	
	var req models.CreateAIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "请求参数格式错误", err)
		return
	}
	
	// 构建配置对象
	config := &models.UserAIConfig{
		UserID:       userID,
		ConfigName:   req.ConfigName,
		ConfigType:   req.ConfigType,
		LLMType:      req.LLMType,
		ModelName:    req.ModelName,
		APIKey:       req.APIKey,
		BaseURL:      req.BaseURL,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		FunctionName: req.FunctionName,
		Description:  req.Description,
		MCPServerURL: req.MCPServerURL,
		Priority:     req.Priority,
		IsActive:     true,
	}
	
	// 处理参数JSON
	if req.Parameters != nil {
		parametersJSON, err := json.Marshal(req.Parameters)
		if err != nil {
			h.respondError(c, http.StatusBadRequest, "参数格式错误", err)
			return
		}
		config.Parameters = datatypes.JSON(parametersJSON)
	}
	
	if err := h.configService.CreateConfig(c.Request.Context(), config); err != nil {
		h.respondError(c, http.StatusInternalServerError, "创建配置失败", err)
		return
	}
	
	h.logger.Info("用户 %s 创建AI配置成功: %s (ID: %d)", userID, config.ConfigName, config.ID)
	
	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "配置创建成功",
		"data":    config.ToResponse(),
	})
}

// GetConfigByID 根据ID获取配置
// @Summary 获取配置详情
// @Description 根据配置ID获取配置详情
// @Tags AI配置管理
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "配置不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/ai-configs/{id} [get]
func (h *AIConfigHandler) GetConfigByID(c *gin.Context) {
	configID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "无效的配置ID", err)
		return
	}
	
	config, err := h.configService.GetConfigByID(c.Request.Context(), uint(configID))
	if err != nil {
		if err.Error() == "配置不存在" {
			h.respondError(c, http.StatusNotFound, "配置不存在", err)
		} else {
			h.respondError(c, http.StatusInternalServerError, "获取配置失败", err)
		}
		return
	}
	
	h.respondSuccess(c, gin.H{
		"config": config.ToResponse(),
	})
}

// UpdateConfig 更新配置
// @Summary 更新AI配置
// @Description 更新指定的AI配置
// @Tags AI配置管理
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Param config body models.UpdateAIConfigRequest true "更新信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "配置不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/ai-configs/{id} [put]
func (h *AIConfigHandler) UpdateConfig(c *gin.Context) {
	configID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "无效的配置ID", err)
		return
	}
	
	var req models.UpdateAIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "请求参数格式错误", err)
		return
	}
	
	// 获取现有配置
	config, err := h.configService.GetConfigByID(c.Request.Context(), uint(configID))
	if err != nil {
		if err.Error() == "配置不存在" {
			h.respondError(c, http.StatusNotFound, "配置不存在", err)
		} else {
			h.respondError(c, http.StatusInternalServerError, "获取配置失败", err)
		}
		return
	}
	
	// 更新字段
	if req.ConfigName != nil {
		config.ConfigName = *req.ConfigName
	}
	if req.LLMType != nil {
		config.LLMType = *req.LLMType
	}
	if req.ModelName != nil {
		config.ModelName = *req.ModelName
	}
	if req.APIKey != nil {
		config.APIKey = *req.APIKey
	}
	if req.BaseURL != nil {
		config.BaseURL = *req.BaseURL
	}
	if req.MaxTokens != nil {
		config.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		config.Temperature = *req.Temperature
	}
	if req.FunctionName != nil {
		config.FunctionName = *req.FunctionName
	}
	if req.Description != nil {
		config.Description = *req.Description
	}
	if req.MCPServerURL != nil {
		config.MCPServerURL = *req.MCPServerURL
	}
	if req.Priority != nil {
		config.Priority = *req.Priority
	}
	if req.IsActive != nil {
		config.IsActive = *req.IsActive
	}
	
	// 处理参数JSON
	if req.Parameters != nil {
		parametersJSON, err := json.Marshal(req.Parameters)
		if err != nil {
			h.respondError(c, http.StatusBadRequest, "参数格式错误", err)
			return
		}
		config.Parameters = datatypes.JSON(parametersJSON)
	}
	
	if err := h.configService.UpdateConfig(c.Request.Context(), config); err != nil {
		h.respondError(c, http.StatusInternalServerError, "更新配置失败", err)
		return
	}
	
	h.logger.Info("用户 %s 更新AI配置成功: %s (ID: %d)", config.UserID, config.ConfigName, config.ID)
	
	h.respondSuccess(c, gin.H{
		"config": config.ToResponse(),
	})
}

// DeleteConfig 删除配置
// @Summary 删除AI配置
// @Description 删除指定的AI配置
// @Tags AI配置管理
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "配置不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/ai-configs/{id} [delete]
func (h *AIConfigHandler) DeleteConfig(c *gin.Context) {
	configID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "无效的配置ID", err)
		return
	}
	
	if err := h.configService.DeleteConfig(c.Request.Context(), uint(configID)); err != nil {
		if err.Error() == "配置不存在" {
			h.respondError(c, http.StatusNotFound, "配置不存在", err)
		} else {
			h.respondError(c, http.StatusInternalServerError, "删除配置失败", err)
		}
		return
	}
	
	h.logger.Info("删除AI配置成功 (ID: %d)", configID)
	
	h.respondSuccess(c, gin.H{
		"message": "配置删除成功",
	})
}

// ToggleConfigStatus 切换配置状态
// @Summary 切换配置状态
// @Description 启用或禁用指定的AI配置
// @Tags AI配置管理
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Param status body map[string]bool true "状态信息"
// @Success 200 {object} map[string]interface{} "操作成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "配置不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/ai-configs/{id}/toggle [patch]
func (h *AIConfigHandler) ToggleConfigStatus(c *gin.Context) {
	configID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "无效的配置ID", err)
		return
	}
	
	var req struct {
		IsActive bool `json:"is_active" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "请求参数格式错误", err)
		return
	}
	
	if err := h.configService.ToggleConfigStatus(c.Request.Context(), uint(configID), req.IsActive); err != nil {
		if err.Error() == "配置不存在" {
			h.respondError(c, http.StatusNotFound, "配置不存在", err)
		} else {
			h.respondError(c, http.StatusInternalServerError, "切换配置状态失败", err)
		}
		return
	}
	
	status := "禁用"
	if req.IsActive {
		status = "启用"
	}
	
	h.logger.Info("配置状态切换成功 (ID: %d, 状态: %s)", configID, status)
	
	h.respondSuccess(c, gin.H{
		"message":   "配置状态切换成功",
		"is_active": req.IsActive,
	})
}

// SetConfigPriority 设置配置优先级
// @Summary 设置配置优先级
// @Description 设置指定AI配置的优先级
// @Tags AI配置管理
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Param priority body map[string]int true "优先级信息"
// @Success 200 {object} map[string]interface{} "操作成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "配置不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/ai-configs/{id}/priority [patch]
func (h *AIConfigHandler) SetConfigPriority(c *gin.Context) {
	configID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "无效的配置ID", err)
		return
	}
	
	var req struct {
		Priority int `json:"priority" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "请求参数格式错误", err)
		return
	}
	
	if err := h.configService.SetConfigPriority(c.Request.Context(), uint(configID), req.Priority); err != nil {
		if err.Error() == "配置不存在" {
			h.respondError(c, http.StatusNotFound, "配置不存在", err)
		} else {
			h.respondError(c, http.StatusInternalServerError, "设置优先级失败", err)
		}
		return
	}
	
	h.logger.Info("配置优先级设置成功 (ID: %d, 优先级: %d)", configID, req.Priority)
	
	h.respondSuccess(c, gin.H{
		"message":  "优先级设置成功",
		"priority": req.Priority,
	})
}

// 辅助方法

// authMiddleware JWT认证中间件
func (h *AIConfigHandler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 验证认证
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			h.respondError(c, http.StatusUnauthorized, "无效的认证token或token已过期", nil)
			c.Abort()
			return
		}

		token := authHeader[7:] // 移除"Bearer "前缀

		// 使用Casbin进行JWT token验证
		claims, err := casbin.ParseToken(token)
		if err != nil {
			h.respondError(c, http.StatusUnauthorized, "token验证失败: "+err.Error(), err)
			c.Abort()
			return
		}

		// 将用户ID存储到上下文中
		c.Set("user_id", uint(claims.UserID))
		c.Set("jwt_claims", claims)
		
		c.Next()
	}
}

// getUserID 从上下文获取用户ID
func (h *AIConfigHandler) getUserID(c *gin.Context) string {
	// 从JWT认证中间件设置的上下文中获取用户ID
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uint); ok {
			return strconv.FormatUint(uint64(uid), 10)
		}
	}
	
	// 如果没有找到用户ID，返回空字符串（这种情况不应该发生，因为有认证中间件）
	h.logger.Error("无法从上下文中获取用户ID")
	return ""
}

// respondSuccess 返回成功响应
func (h *AIConfigHandler) respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "操作成功",
		"data":    data,
	})
}

// respondError 返回错误响应
func (h *AIConfigHandler) respondError(c *gin.Context, statusCode int, message string, err error) {
	h.logger.Error("%s: %v", message, err)
	
	response := gin.H{
		"code":    statusCode,
		"message": message,
	}
	
	if err != nil {
		response["error"] = err.Error()
	}
	
	c.JSON(statusCode, response)
}