package device

import (
	"context"
	"net/http"
	"strings"
	"time"

	"angrymiao-ai-server/src/configs"
	"angrymiao-ai-server/src/core/auth"
	"angrymiao-ai-server/src/core/auth/am_token"
	"angrymiao-ai-server/src/core/utils"

	"github.com/gin-gonic/gin"
)

type DefaultDeviceService struct {
	logger    *utils.Logger
	config    *configs.Config
	authToken *auth.AuthToken
	deviceDB  *DeviceBindDB
}

// NewDefaultDeviceService 构造函数
func NewDefaultDeviceService(config *configs.Config, logger *utils.Logger) *DefaultDeviceService {
	// 初始化AuthToken
	authToken := auth.NewAuthToken(config.Server.Token)

	// 初始化设备绑定数据库操作
	deviceDB := NewDeviceBindDB()

	return &DefaultDeviceService{
		logger:    logger,
		config:    config,
		authToken: authToken,
		deviceDB:  deviceDB,
	}
}

// Start 注册 Device 相关路由
func (s *DefaultDeviceService) Start(ctx context.Context, engine *gin.Engine, apiGroup *gin.RouterGroup) error {
	apiGroup.OPTIONS("/device/", s.handleDeviceOptions)
	apiGroup.POST("/device/bind", s.handleDeviceBind)
	apiGroup.POST("/device/unbind", s.handleDeviceUnbind)
	apiGroup.GET("/device/refresh_token", s.handleDeviceRefToken)

	return nil
}

// addCORSHeaders 添加CORS头
func (s *DefaultDeviceService) addCORSHeaders(c *gin.Context) {
	c.Header("Access-Control-Allow-Headers", "client-id, content-type, device-id, authorization")
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

// @Summary Device 预检请求
// @Description 处理 Device 接口的 OPTIONS 预检请求，返回 200
// @Tags Device
// @Accept */*
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /device/ [options]
func (s *DefaultDeviceService) handleDeviceOptions(c *gin.Context) {
	s.addCORSHeaders(c)
	c.Status(http.StatusOK)
}

// @Summary 刷新token
// @Description 刷新设备的JWT token
// @Tags Device
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer <UserJWT>"
// @Param body body BindDeviceRequest true "请求体"
// @Success 200 {object} RefreshTokenResponse
// @Failure 400 {object} ErrorResponse
// @Router /device/refreshToken [get]
func (s *DefaultDeviceService) handleDeviceRefToken(c *gin.Context) {
	s.addCORSHeaders(c)

	// 获取get请求参数
	deviceID := c.Query("device_id")
	if deviceID == "" {
		s.respondError(c, http.StatusBadRequest, "设备ID不能为空")
		return
	}

	token := c.Query("token")
	if token == "" {
		s.respondError(c, http.StatusBadRequest, "token不能为空")
		return
	}

	bindKey := c.Query("bind_key")
	if bindKey == "" {
		s.respondError(c, http.StatusBadRequest, "bind_key不能为空")
		return
	}

	deviceInfo, err := s.deviceDB.GetDeviceBind(deviceID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "设备不存在")
		return
	}

	// 验证绑定密钥
	if deviceInfo.BindKey != bindKey {
		s.respondError(c, http.StatusInternalServerError, "绑定密钥验证失败")
		return
	}

	// 使用Casbin进行JWT token验证

	_, tokenDeviceID, tokenUserID, err := s.authToken.VerifyToken(token, true)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "token验证失败: "+err.Error())
		return
	}

	if tokenDeviceID != deviceID || tokenUserID != deviceInfo.UserID {
		s.respondError(c, http.StatusInternalServerError, "token设备ID与请求设备ID不匹配")
		return
	}

	// 生成新的7天有效期token
	sevenDays := 7 * 24 * time.Hour
	newToken, err := s.authToken.GenerateTokenWithExpiry(deviceInfo.UserID, deviceID, sevenDays)
	if err != nil {
		s.logger.Error("生成新token失败: %v", err)
		s.respondError(c, http.StatusInternalServerError, "生成新token失败")
		return
	}

	s.logger.Info("刷新设备token成功 - 用户ID: %d, 设备ID: %s", deviceInfo.UserID, deviceID)

	// 返回成功响应
	response := RefreshTokenResponse{
		Success: true,
		Token:   newToken,
	}
	c.JSON(http.StatusOK, response)
}

// @Summary 上传设备信息获取最新固件
// @Description 设备上传信息后，返回最新固件版本和下载地址
// @Tags Device
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer <UserJWT>"
// @Param body body BindDeviceRequest true "请求体"
// @Success 200 {object} OtaFirmwareResponse
// @Failure 400 {object} ErrorResponse
// @Router /device/bind [post]
func (s *DefaultDeviceService) handleDeviceBind(c *gin.Context) {
	s.addCORSHeaders(c)
	c.Status(http.StatusOK)

	// 验证认证
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		s.respondError(c, http.StatusUnauthorized, "无效的认证token或token已过期")
		return
	}

	token := authHeader[7:] // 移除"Bearer "前缀

	// 使用Casbin进行JWT token验证
	claims, err := am_token.ParseToken(token)
	if err != nil {
		s.respondError(c, http.StatusUnauthorized, "token验证失败: "+err.Error())
		return
	}

	// 从claims中获取用户ID
	userID := uint(claims.UserID)

	// 解析请求体
	var body BindDeviceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		s.respondError(c, http.StatusBadRequest, "请求参数格式错误: "+err.Error())
		return
	}

	// 验证设备ID
	if !ValidateDeviceID(body.DeviceID) {
		s.respondError(c, http.StatusBadRequest, "设备ID格式无效")
		return
	}

	// 是否绑定过设备
	exists, err := s.deviceDB.IsDeviceBound(body.DeviceID)
	if err != nil {
		s.logger.Error("检查设备绑定失败: %v", err)
		s.respondError(c, http.StatusInternalServerError, "检查设备绑定失败")
		return
	}
	if exists {
		s.respondError(c, http.StatusBadRequest, "设备已绑定")
		return
	}

	// 生成绑定密钥，使用HMAC(device_id, server_secret)算法
	bindKey := GenerateBindKey(body.DeviceID, s.config.Server.Token, userID)

	// 生成7天有效期的token
	sevenDays := 7 * 24 * time.Hour
	deviceToken, err := s.authToken.GenerateTokenWithExpiry(userID, body.DeviceID, sevenDays)
	if err != nil {
		s.logger.Error("生成设备token失败: %v", err)
		s.respondError(c, http.StatusInternalServerError, "生成设备token失败")
		return
	}

	// 保存绑定信息到数据库
	if err := s.deviceDB.SaveDeviceBind(body.DeviceID, userID, bindKey); err != nil {
		s.logger.Error("保存设备绑定信息失败: %v", err)
		s.respondError(c, http.StatusInternalServerError, "保存绑定信息失败")
		return
	}

	s.logger.Info("设备绑定成功 - 用户ID: %d, 设备ID: %s", userID, body.DeviceID)

	// 返回成功响应
	response := BindDeviceResponse{
		Success:   true,
		DeviceKey: bindKey,
		Token:     deviceToken,
	}
	c.JSON(http.StatusOK, response)
}

// handleDeviceUnbind 处理设备解绑请求
// @Summary 设备解绑
// @Description 解绑指定设备，将数据库中is_active字段设置为false
// @Tags Device
// @Accept json
// @Produce json
// @Param request body UnbindDeviceRequest true "解绑请求"
// @Success 200 {object} UnbindDeviceResponse "解绑成功"
// @Failure 400 {object} UnbindDeviceResponse "请求参数错误"
// @Failure 404 {object} UnbindDeviceResponse "设备未找到"
// @Failure 500 {object} UnbindDeviceResponse "服务器内部错误"
// @Router /api/device/unbind [post]
func (s *DefaultDeviceService) handleDeviceUnbind(c *gin.Context) {
	// 添加CORS头
	s.addCORSHeaders(c)

	// 解析请求体
	var body UnbindDeviceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		s.logger.Error("解绑请求参数解析失败: %v", err)
		s.respondUnbindError(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}

	// 验证设备ID格式
	if !ValidateDeviceID(body.DeviceID) {
		s.logger.Error("无效的设备ID格式: %s", body.DeviceID)
		s.respondUnbindError(c, http.StatusBadRequest, "设备ID格式无效")
		return
	}

	// 检查设备是否存在且已绑定
	existingBind, err := s.deviceDB.GetDeviceBind(body.DeviceID)
	if err != nil {
		s.logger.Error("查询设备绑定信息失败: %v", err)
		s.respondUnbindError(c, http.StatusNotFound, "设备未找到或未绑定")
		return
	}

	if existingBind.BindKey != body.BindKey {
		s.respondUnbindError(c, http.StatusBadRequest, "绑定key错误")
		return
	}

	// 执行解绑操作
	if err := s.deviceDB.UnbindDevice(body.DeviceID, body.BindKey); err != nil {
		s.logger.Error("设备解绑失败: %v", err)
		s.respondUnbindError(c, http.StatusInternalServerError, "解绑操作失败")
		return
	}

	s.logger.Info("设备解绑成功: %s (用户ID: %d)", body.DeviceID, existingBind.UserID)

	// 返回成功响应
	response := UnbindDeviceResponse{
		Success: true,
		Message: "设备解绑成功",
	}
	c.JSON(http.StatusOK, response)
}

// respondUnbindError 返回解绑错误响应
func (s *DefaultDeviceService) respondUnbindError(c *gin.Context, statusCode int, message string) {
	response := UnbindDeviceResponse{
		Success: false,
		Message: message,
	}
	c.JSON(statusCode, response)
}

// respondError 返回错误响应
func (s *DefaultDeviceService) respondError(c *gin.Context, statusCode int, message string) {
	response := BindDeviceResponse{
		Success: false,
		Message: message,
	}
	c.JSON(statusCode, response)
}
