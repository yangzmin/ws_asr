# xiaozhi-grpc-proto

小智AI服务的gRPC协议库项目，定义了IM服务与AI服务之间的通信协议。

## 项目结构

```
xiaozhi-grpc-proto/
├── go.mod                          # Go模块定义
├── go.sum                          # 依赖版本锁定
├── README.md                       # 项目说明
├── proto/
│   └── ai_service.proto            # gRPC服务定义
├── generated/
│   └── go/
│       └── proto/
│           ├── ai_service.pb.go    # 生成的protobuf代码
│           └── ai_service_grpc.pb.go # 生成的gRPC代码
└── scripts/
    ├── generate.sh                 # Linux/Mac代码生成脚本
    └── generate.bat                # Windows代码生成脚本
```

## 功能特性

- **双向流通信**：支持gRPC双向流，实现实时通信
- **消息类型完整**：支持hello、listen、chat、abort、vision、image、mcp、audio等所有消息类型
- **音频数据支持**：支持PCM、Opus等音频格式的传输
- **图片数据支持**：支持base64编码的图片数据传输
- **错误处理**：完善的错误响应机制
- **健康检查**：内置服务健康检查接口

## 使用方法

### 1. 代码生成

在Windows环境下：
```bash
scripts\generate.bat
```

在Linux/Mac环境下：
```bash
chmod +x scripts/generate.sh
./scripts/generate.sh
```

### 2. 作为依赖引入

在其他Go项目中引入此协议库：

```go
import "xiaozhi-grpc-proto/generated/go/proto"
```

### 3. 服务端实现示例

```go
type AIServiceServer struct {
    proto.UnimplementedAIServiceServer
}

func (s *AIServiceServer) ChatStream(stream proto.AIService_ChatStreamServer) error {
    // 实现双向流处理逻辑
    for {
        req, err := stream.Recv()
        if err != nil {
            return err
        }
        
        // 处理请求
        resp := &proto.ChatResponse{
            SessionId: req.SessionId,
            // ... 其他响应字段
        }
        
        if err := stream.Send(resp); err != nil {
            return err
        }
    }
}
```

### 4. 客户端实现示例

```go
conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := proto.NewAIServiceClient(conn)
stream, err := client.ChatStream(context.Background())
if err != nil {
    log.Fatal(err)
}

// 发送请求
req := &proto.ChatRequest{
    SessionId: "session-123",
    DeviceId: "device-456",
    MessageType: proto.MessageType_MESSAGE_TYPE_HELLO,
    // ... 其他字段
}

if err := stream.Send(req); err != nil {
    log.Fatal(err)
}

// 接收响应
resp, err := stream.Recv()
if err != nil {
    log.Fatal(err)
}
```

## 消息类型说明

### 请求消息类型
- `MESSAGE_TYPE_HELLO`: 客户端连接握手
- `MESSAGE_TYPE_LISTEN`: 语音监听控制
- `MESSAGE_TYPE_CHAT`: 文本聊天消息
- `MESSAGE_TYPE_ABORT`: 中止当前操作
- `MESSAGE_TYPE_VISION`: 视觉相关操作
- `MESSAGE_TYPE_IMAGE`: 图片处理请求
- `MESSAGE_TYPE_MCP`: MCP协议消息
- `MESSAGE_TYPE_AUDIO`: 音频数据传输

### 响应消息类型
- `RESPONSE_TYPE_HELLO`: 握手响应
- `RESPONSE_TYPE_STT`: 语音转文本结果
- `RESPONSE_TYPE_TTS`: 文本转语音状态
- `RESPONSE_TYPE_EMOTION`: 情绪状态
- `RESPONSE_TYPE_AUDIO`: 音频数据响应
- `RESPONSE_TYPE_ERROR`: 错误响应
- `RESPONSE_TYPE_STATUS`: 状态响应

## 兼容性

- 完全兼容现有WebSocket消息格式
- 支持现有音频格式（PCM、Opus）
- 支持现有图片格式（base64编码）
- 保持现有消息结构不变

## 依赖要求

- Go 1.21+
- Protocol Buffers 3.19+
- gRPC 1.60+

## 版本信息

- 当前版本：v1.0.0
- gRPC版本：v1.60.0
- Protobuf版本：v1.31.0