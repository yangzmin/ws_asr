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

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置Gin模式
	if cfg.Server.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建连接管理服务
	connectionService := service.NewConnectionService(cfg)

	// 创建gRPC客户端服务
	grpcClientService, err := service.NewGRPCClientService(cfg)
	if err != nil {
		log.Fatalf("创建gRPC客户端服务失败: %v", err)
	}

	// 创建消息路由服务
	messageService := service.NewMessageService(cfg, grpcClientService)

	// 创建WebSocket处理器
	wsHandler := handler.NewWebSocketHandler(cfg, connectionService, messageService)

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
	router.GET("/ws", wsHandler.HandleWebSocket)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// 启动服务器
	go func() {
		log.Printf("IM服务启动在端口 %d", cfg.Server.Port)
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
	grpcClientService.Close()

	// 关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭失败: %v", err)
	}

	log.Println("服务器已关闭")
}