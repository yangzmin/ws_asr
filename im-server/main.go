package main

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "angrymiao-ai-server/src/configs"
    dbinit "angrymiao-ai-server/src/configs/database"
    "angrymiao-ai-server/src/core/utils"
    "angrymiao-ai-server/im-server/bus"
)

// Application for im-server
type Application struct {
    config *configs.Config
    logger *utils.Logger
    grpc   *bus.IMBusServer
    http   *http.Server
}

func (app *Application) init() error {
    // load config
    cfg, cfgPath, err := configs.LoadConfig(dbinit.GetServerConfigDB())
    if err != nil {
        return fmt.Errorf("加载配置失败: %w", err)
    }
    app.config = cfg

    // init logger
    logger, err := utils.NewLogger((*utils.LogCfg)(&cfg.Log))
    if err != nil {
        return fmt.Errorf("初始化日志失败: %w", err)
    }
    app.logger = logger
    utils.DefaultLogger = logger
    app.logger.Info("im-server 配置初始化完成: %s", cfgPath)

    return nil
}

func (app *Application) start() error {
    // Start gRPC bus server on websocket port + 1
    grpcPort := app.config.Transport.WebSocket.Port + 1
    grpcAddr := fmt.Sprintf("%s:%d", app.config.Transport.WebSocket.IP, grpcPort)
    lis, err := net.Listen("tcp", grpcAddr)
    if err != nil {
        return fmt.Errorf("监听 gRPC 地址失败: %w", err)
    }

    app.grpc = bus.NewIMBusServer(app.config, app.logger)
    if err := app.grpc.Start(lis); err != nil {
        return fmt.Errorf("启动 gRPC 总线失败: %w", err)
    }
    app.logger.Info("im-server gRPC 总线已启动: %s", grpcAddr)

    // Start WS server
    wsAddr := fmt.Sprintf("%s:%d", app.config.Transport.WebSocket.IP, app.config.Transport.WebSocket.Port)
    mux := http.NewServeMux()
    mux.HandleFunc("/", app.grpc.HandleWebSocket)
    app.http = &http.Server{Addr: wsAddr, Handler: mux}

    go func() {
        if err := app.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            app.logger.Error("im-server WebSocket 服务启动失败: %v", err)
        }
    }()
    app.logger.Info("im-server WebSocket 服务已启动: %s", wsAddr)

    return nil
}

func (app *Application) stop() {
    if app.http != nil {
        _ = app.http.Shutdown(context.Background())
    }
    if app.grpc != nil {
        app.grpc.Stop()
    }
}

func main() {
    app := &Application{}
    if err := app.init(); err != nil {
        fmt.Println("im-server 初始化失败:", err)
        os.Exit(1)
    }
    if err := app.start(); err != nil {
        fmt.Println("im-server 启动失败:", err)
        os.Exit(1)
    }

    // Graceful shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    app.logger.Info("im-server 收到退出信号，正在关闭...")
    app.stop()
    app.logger.Info("im-server 已退出")
}