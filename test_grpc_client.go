package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "xiaozhi-grpc-proto/generated/go/ai_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 连接到gRPC服务器
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	client := pb.NewAIServiceClient(conn)

	// 测试健康检查
	fmt.Println("测试健康检查...")
	healthResp, err := client.HealthCheck(context.Background(), &pb.HealthCheckRequest{})
	if err != nil {
		log.Printf("健康检查失败: %v", err)
	} else {
		fmt.Printf("健康检查成功: %v\n", healthResp.Status)
	}

	// 测试聊天流
	fmt.Println("测试聊天流...")
	stream, err := client.ChatStream(context.Background())
	if err != nil {
		log.Fatalf("创建聊天流失败: %v", err)
	}

	// 发送测试消息
	testMessage := &pb.ChatRequest{
		SessionId:   "test_session_123",
		DeviceId:    "test_device_456",
		ClientId:    "test_client_789",
		MessageType: 1, // 假设1表示文本消息
		MessageData: []byte(`{"type": "chat", "content": "你好，这是一个测试消息"}`),
		Timestamp:   time.Now().Unix(),
	}

	fmt.Printf("发送消息: %s\n", string(testMessage.MessageData))
	err = stream.Send(testMessage)
	if err != nil {
		log.Printf("发送消息失败: %v", err)
		return
	}

	// 接收响应
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				fmt.Println("流已结束")
				return
			}
			if err != nil {
				log.Printf("接收消息失败: %v", err)
				return
			}
			fmt.Printf("收到响应: %s\n", string(resp.ResponseData))
		}
	}()

	// 等待一段时间以接收响应
	time.Sleep(5 * time.Second)

	// 关闭流
	err = stream.CloseSend()
	if err != nil {
		log.Printf("关闭流失败: %v", err)
	}

	fmt.Println("测试完成")
}
