# xiaozhi-im-service

小智AI服务的IM（即时通讯）服务项目，负责处理WebSocket连接和gRPC长连接双向流通信。

## 项目概述

本项目是小智AI服务抽象化架构的核心组件，主要功能包括：

- **WebSocket处理器**：接收和处理来自客户端的WebSocket连接
- **gRPC长连接客户端**：与AI服务建立gRPC双向流连接
- **消息路由和转换**：在WebSocket JSON格式和gRPC protobuf格式之间进行消息转换
- **连接池管理**：管理WebSocket连接和gRPC流的生命周期
- **心跳检测和自动重连**：确保连接的稳定性和可靠性

## 项目结构

```
xiaozhi-im-service/
├── cmd/
│   └── server/
│       └── main.go              # 应用程序入口
├── internal/
│   ├── config/
│   │   └── config.go            # 配置管理
│   ├── handler/
│   │   ├── websocket.go         # WebSocket处理器
│   │   └── health.go            # 健康检查处理器
│   ├── model/
│   │   └── message.go           # 消息模型定义
│   └── service/
│       ├── connection_manager.go # 连接管理器
│       ├── grpc_client.go       # gRPC客户端
│       ├── message_router.go    # 消息路由器
│       └── errors.go            # 错误定义
├── pkg/
│   └── auth/
│       └── jwt.go               # JWT认证
├── go.mod                       # Go模块定义
└── README.md                    # 项目文档
```

## 功能特性

### 1. WebSocket连接管理
- 支持JWT认证
- 连接生命周期管理
- 自动清理不活跃连接
- 支持多设备同时连接

### 2. gRPC长连接
- 双向流通信
- 连接池管理
- 自动重连机制
- 心跳检测

### 3. 消息处理
- 完整的消息类型支持（hello, listen, chat, abort, vision, image, mcp, audio）
- WebSocket JSON ↔ gRPC protobuf 格式转换
- 实时消息路由
- 错误处理和状态反馈

### 4. 音频数据支持
- 支持PCM和Opus音频格式
- 可配置采样率、声道数、帧时长
- 实时音频流传输

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `IM_SERVER_PORT` | 8081 | HTTP服务器端口 |
| `IM_SERVER_DEBUG` | true | 调试模式 |
| `AI_SERVICE_GRPC_ADDR` | localhost:50051 | AI服务gRPC地址 |
| `GRPC_MAX_CONNECTIONS` | 10 | gRPC最大连接数 |
| `GRPC_HEARTBEAT_SECONDS` | 30 | 心跳间隔（秒） |
| `GRPC_RECONNECT_SECONDS` | 5 | 重连间隔（秒） |
| `JWT_SECRET` | xiaozhi-im-service-secret | JWT密钥 |

## API接口

### WebSocket连接
```
GET /ws?token=<JWT_TOKEN>
```

### 健康检查
```
GET /health        # 健康检查
GET /ready         # 就绪检查
GET /alive         # 存活检查
```

## 消息格式

### WebSocket消息格式
```json
{
  "type": "hello|listen|chat|abort|vision|image|mcp|audio",
  "data": {
    // 消息数据，根据type不同而不同
  },
  "timestamp": 1640995200000
}
```

### 支持的消息类型

1. **hello** - 初始化连接
2. **listen** - 语音识别控制
3. **chat** - 文本对话
4. **abort** - 中止当前操作
5. **vision** - 视觉相关命令
6. **image** - 图片处理
7. **mcp** - MCP协议消息
8. **audio** - 音频数据

## 使用方法

### 1. 启动服务
```bash
cd cmd/server
go run main.go
```

### 2. 客户端连接
```javascript
const ws = new WebSocket('ws://localhost:8081/ws?token=YOUR_JWT_TOKEN');

ws.onopen = function() {
    // 发送hello消息初始化连接
    ws.send(JSON.stringify({
        type: 'hello',
        data: {
            audio_params: {
                format: 'opus',
                sample_rate: 16000,
                channels: 1,
                frame_duration: 20
            }
        },
        timestamp: Date.now()
    }));
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('收到消息:', message);
};
```

## 依赖项

- **xiaozhi-grpc-proto**: gRPC协议定义
- **gin**: HTTP框架
- **gorilla/websocket**: WebSocket支持
- **grpc**: gRPC客户端
- **jwt**: JWT认证
- **logrus**: 日志记录
- **uuid**: UUID生成

## 兼容性

- 完全兼容现有WebSocket消息格式
- 支持所有现有音频格式和参数
- 保持与现有客户端的向后兼容性

## 开发和测试

### 运行测试
```bash
go test ./...
```

### 构建
```bash
go build -o xiaozhi-im-service cmd/server/main.go
```

## 注意事项

1. 确保AI服务的gRPC端口已开启
2. JWT Token必须包含有效的用户信息
3. 生产环境中需要配置适当的CORS策略
4. 建议配置负载均衡和高可用部署

## 版本信息

- Go版本: 1.21+
- gRPC版本: v1.60.0+
- 协议版本: 与xiaozhi-grpc-proto保持同步