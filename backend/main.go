package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"speech-recognition-backend/backend/websocket"
)

func main() {
	// 设置日志
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)

	// 创建WebSocket处理器
	hub := websocket.NewHub()
	go hub.Run()

	// 设置路由
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWS(hub, w, r)
	})

	// 健康检查接口
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		timestamp := time.Now().Unix()
		response := fmt.Sprintf(`{"status":"ok","timestamp":%d}`, timestamp)
		w.Write([]byte(response))
	})

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}