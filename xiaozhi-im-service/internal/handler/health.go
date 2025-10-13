package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"xiaozhi-im-service/internal/service"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	connMgr    *service.ConnectionManager
	grpcClient *service.GRPCClient
	logger     *logrus.Logger
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(
	connMgr *service.ConnectionManager,
	grpcClient *service.GRPCClient,
	logger *logrus.Logger,
) *HealthHandler {
	return &HealthHandler{
		connMgr:    connMgr,
		grpcClient: grpcClient,
		logger:     logger,
	}
}

// HealthCheck 健康检查接口
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	status := "healthy"
	httpStatus := http.StatusOK

	// 检查gRPC客户端状态
	grpcReady := h.grpcClient.IsReady()
	if !grpcReady {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	// 获取连接统计信息
	connectionCount := h.connMgr.GetConnectionCount()
	streamCount := h.grpcClient.GetStreamCount()

	response := gin.H{
		"status": status,
		"services": gin.H{
			"grpc_client": gin.H{
				"ready":        grpcReady,
				"stream_count": streamCount,
			},
			"connection_manager": gin.H{
				"connection_count": connectionCount,
			},
		},
		"timestamp": c.GetHeader("X-Request-Time"),
	}

	h.logger.WithFields(logrus.Fields{
		"status":           status,
		"grpc_ready":       grpcReady,
		"connection_count": connectionCount,
		"stream_count":     streamCount,
	}).Debug("健康检查完成")

	c.JSON(httpStatus, response)
}

// ReadinessCheck 就绪检查接口
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	ready := h.grpcClient.IsReady()
	
	if ready {
		c.JSON(http.StatusOK, gin.H{
			"ready": true,
			"message": "服务已就绪",
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready": false,
			"message": "服务未就绪",
		})
	}
}

// LivenessCheck 存活检查接口
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"alive": true,
		"message": "服务运行中",
	})
}