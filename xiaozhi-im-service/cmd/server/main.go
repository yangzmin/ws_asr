package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xiaozhi-im-service/internal/config"
	"xiaozhi-im-service/internal/handler"
	"xiaozhi-im-service/internal/service"
	"xiaozhi-im-service/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// 加载配置
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	logger := logrus.New()
	if config.Server.Debug {
		logger.SetLevel(logrus.DebugLevel)
	}

	// 创建JWT管理器
	jwtManager := auth.NewJWTManager(config.JWT.Secret)

	// 创建服务实例
	connectionManager := service.NewConnectionManager(logger)
	grpcClient := service.NewGRPCClient(&config.GRPC, logger)
	messageRouter := service.NewMessageRouter(grpcClient, connectionManager, logger)

	// 启动gRPC客户端
	ctx := context.Background()
	if err := grpcClient.Start(ctx); err != nil {
		log.Fatalf("启动gRPC客户端失败: %v", err)
	}
	logger.Info("gRPC客户端已启动")

	// 创建WebSocket处理器
	wsHandler := handler.NewWebSocketHandler(connectionManager, grpcClient, messageRouter, jwtManager, logger)

	// 创建Gin路由
	router := gin.Default()

	// 健康检查接口
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	// WebSocket连接接口
	router.GET("/", wsHandler.HandleWebSocket)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Server.Port),
		Handler: router,
	}

	// 启动服务器
	go func() {
		log.Printf("IM服务启动在端口 %d", config.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭gRPC连接
	grpcClient.Stop()

	// 关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭失败: %v", err)
	}

	log.Println("服务器已关闭")
}
