package device

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"xiaozhi-server-go/src/configs"
	"xiaozhi-server-go/src/core/auth/casbin"
	"xiaozhi-server-go/src/core/utils"

	"github.com/gin-gonic/gin"
)

type DefaultDeviceService struct {
	logger *utils.Logger
	config *configs.Config
}

// NewDefaultDeviceService 构造函数
func NewDefaultDeviceService(config *configs.Config, logger *utils.Logger) *DefaultDeviceService {
	return &DefaultDeviceService{logger: logger, config: config}
}

// Start 注册 Device 相关路由
func (s *DefaultDeviceService) Start(ctx context.Context, engine *gin.Engine, apiGroup *gin.RouterGroup) error {
	apiGroup.OPTIONS("/device/", s.handleDeviceOptions)
	apiGroup.POST("/device/bind", s.handleDeviceBind)

	// engine.GET("/device_bin/:filename", handleDeviceBinDownload)

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
	claims, err := casbin.ParseToken(token)
	if err != nil {
		s.respondError(c, http.StatusUnauthorized, "token验证失败: "+err.Error())
		return
	}

	fmt.Println("claims", claims)

	var body BindDeviceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		s.respondError(c, http.StatusBadRequest, err.Error())
		return
	}
}

// respondError 返回错误响应
func (s *DefaultDeviceService) respondError(c *gin.Context, statusCode int, message string) {
	response := BindDeviceResponse{
		Success: false,
		Message: message,
	}
	c.JSON(statusCode, response)
}
