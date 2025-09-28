# MCP调用链路和关键代码文档

## 1. 概述

MCP (Multi-Cloud Platform) 是项目中的工具调用系统，负责管理和执行各种工具函数。本文档详细分析了MCP的调用链路、关键代码结构和实现细节。

## 2. 目录结构

```
src/
├── core/
│   └── mcp/
│       ├── README.md              # MCP说明文档
│       ├── interface.go           # MCP接口定义
│       ├── manager.go             # MCP管理器
│       ├── client.go              # 通用MCP客户端
│       ├── local_client.go        # 本地工具客户端
│       ├── local_mcp_tools.go     # 本地工具实现
│       └── xiaozhi_client.go      # 小智MCP客户端
├── services/
│   └── mcp/
│       └── internal/
│           └── config/            # MCP服务配置
└── core/
    ├── types/
    │   └── llm.go                 # 动作类型定义
    ├── connection.go              # 连接处理器
    └── connection_handlemcp.go    # MCP结果处理
```

## 3. 核心接口和类型定义

### 3.1 MCPClient接口

```go
// MCPClient MCP客户端接口
type MCPClient interface {
    Start() error                                    // 启动客户端
    Stop() error                                     // 停止客户端
    HasTool(toolName string) bool                    // 检查工具是否存在
    GetAvailableTools() []Tool                       // 获取可用工具列表
    CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) // 调用工具
    IsReady() bool                                   // 检查客户端是否就绪
    ResetConnection() error                          // 重置连接
}
```

### 3.2 Tool结构体

```go
// Tool 表示一个MCP工具
type Tool struct {
    Name        string           `json:"name"`        // 工具名称
    Description string           `json:"description"` // 工具描述
    InputSchema ToolInputSchema  `json:"inputSchema"` // 输入参数模式
}

// ToolInputSchema 工具输入参数模式
type ToolInputSchema struct {
    Type       string                 `json:"type"`       // 参数类型
    Properties map[string]interface{} `json:"properties"` // 参数属性
    Required   []string               `json:"required"`   // 必需参数
}
```

### 3.3 动作类型定义

```go
// Action 动作类型
type Action int

const (
    ActionTypeError       Action = -1  // 错误
    ActionTypeNotFound    Action = 0   // 没有找到函数
    ActionTypeNone        Action = 1   // 啥也不干
    ActionTypeResponse    Action = 2   // 直接回复
    ActionTypeReqLLM      Action = 3   // 调用函数后再请求llm生成回复
    ActionTypeCallHandler Action = 4   // 调用处理器
)

// ActionResponse 动作响应
type ActionResponse struct {
    Action   Action      `json:"action"`   // 动作类型
    Result   interface{} `json:"result"`   // 动作产生的结果
    Response interface{} `json:"response"` // 直接回复的内容
}

// ActionResponseCall 动作响应调用
type ActionResponseCall struct {
    FuncName string      `json:"funcName"` // 函数名
    Args     interface{} `json:"args"`     // 函数参数
}
```

## 4. 核心组件实现

### 4.1 Manager - MCP管理器

```go
// Manager MCP服务管理器
type Manager struct {
    logger         *logger.Logger                    // 日志记录器
    conn           *websocket.Conn                   // WebSocket连接
    functionHandle func(string, map[string]interface{}) // 函数处理器
    configPath     string                            // 配置路径
    clients        map[string]MCPClient              // MCP客户端映射
    localClient    *LocalClient                     // 本地客户端
    xiaozhiClient  *XiaoZhiMCPClient                // 小智客户端
    // ... 其他字段
}

// 关键方法
func (m *Manager) IsMCPTool(toolName string) bool
func (m *Manager) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error)
```

### 4.2 LocalClient - 本地工具客户端

```go
// LocalClient 本地工具客户端
type LocalClient struct {
    logger   *logger.Logger
    tools    map[string]Tool
    handlers map[string]HandlerFunc
    ready    bool
}

// HandlerFunc 工具处理函数类型
type HandlerFunc func(args map[string]interface{}) types.ActionResponse

// 关键方法
func (lc *LocalClient) RegisterTools(tools []Tool, handlers map[string]HandlerFunc)
func (lc *LocalClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error)
func (lc *LocalClient) AddTool(tool Tool, handler HandlerFunc)
```

### 4.3 Client - 通用MCP客户端

```go
// Client MCP客户端实现
type Client struct {
    Config *Config
    client *mcpclient.Client
    // ... 其他字段
}

// 关键方法
func (c *Client) Start() error
func (c *Client) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error)
```

### 4.4 XiaoZhiMCPClient - 小智MCP客户端

```go
// XiaoZhiMCPClient 小智MCP客户端
type XiaoZhiMCPClient struct {
    logger    *logger.Logger
    conn      *websocket.Conn
    sessionID string
    tools     []Tool
    ready     bool
    ctx       context.Context
    // ... 其他字段
}
```

## 5. MCP调用链路分析

### 5.1 完整调用流程

```
1. 用户输入 → WebSocket消息
2. ConnectionHandler.genResponseByLLM()
3. LLM生成回复（包含工具调用）
4. 检测到工具调用 → mcpManager.IsMCPTool()
5. 执行工具 → mcpManager.ExecuteTool()
6. 路由到具体客户端 → LocalClient/Client/XiaoZhiMCPClient
7. 工具执行 → 返回ActionResponse
8. 结果处理 → handleFunctionResult()
9. 根据Action类型进行后续处理
```

### 5.2 关键调用代码

#### 5.2.1 工具调用检测和执行

```go
// connection.go - genResponseByLLM方法中
if h.mcpManager.IsMCPTool(functionName) {
    // 处理MCP函数调用
    result, err := h.mcpManager.ExecuteTool(ctx, functionName, arguments)
    if err != nil {
        h.LogError(fmt.Sprintf("MCP函数调用失败: %v", err))
        if result == nil {
            result = "MCP工具调用失败"
        }
    }
    // 判断result是否是types.ActionResponse类型
    if actionResult, ok := result.(types.ActionResponse); ok {
        h.handleFunctionResult(actionResult, functionCallData, textIndex)
    } else {
        h.LogInfo(fmt.Sprintf("MCP函数调用结果: %v", result))
        actionResult := types.ActionResponse{
            Action: types.ActionTypeReqLLM,
            Result: result,
        }
        h.handleFunctionResult(actionResult, functionCallData, textIndex)
    }
}
```

#### 5.2.2 结果处理分发

```go
// connection.go - handleFunctionResult方法
func (h *ConnectionHandler) handleFunctionResult(result types.ActionResponse, functionCallData map[string]interface{}, textIndex int) {
    switch result.Action {
    case types.ActionTypeError:
        h.LogError(fmt.Sprintf("函数调用错误: %v", result.Result))
    case types.ActionTypeNotFound:
        h.LogError(fmt.Sprintf("函数未找到: %v", result.Result))
    case types.ActionTypeNone:
        h.LogInfo(fmt.Sprintf("函数调用无操作: %v", result.Result))
    case types.ActionTypeResponse:
        h.LogInfo(fmt.Sprintf("函数调用直接回复: %v", result.Response))
        h.SystemSpeak(result.Response.(string))
    case types.ActionTypeCallHandler:
        resultStr := h.handleMCPResultCall(result)
        h.addToolCallMessage(resultStr, functionCallData)
    case types.ActionTypeReqLLM:
        h.LogInfo(fmt.Sprintf("函数调用后请求LLM: %v", result.Result))
        text, ok := result.Result.(string)
        if ok && len(text) > 0 {
            h.addToolCallMessage(text, functionCallData)
            h.genResponseByLLM(context.Background(), h.dialogueManager.GetLLMDialogue(), h.talkRound)
        }
    }
}
```

#### 5.2.3 MCP结果调用处理

```go
// connection_handlemcp.go - handleMCPResultCall方法
func (h *ConnectionHandler) handleMCPResultCall(result types.ActionResponse) string {
    if result.Action != types.ActionTypeCallHandler {
        return "调用工具失败"
    }
    
    if Caller, ok := result.Result.(types.ActionResponseCall); ok {
        if handler, exists := h.mcpResultHandlers[Caller.FuncName]; exists {
            handler(Caller.Args)
            return "调用工具成功: " + Caller.FuncName
        }
    }
    return "调用工具失败"
}
```

## 6. 本地工具实现示例

### 6.1 退出工具

```go
func AddToolExit(client *LocalClient) {
    tool := Tool{
        Name:        "exit",
        Description: "退出程序",
        InputSchema: ToolInputSchema{
            Type:       "object",
            Properties: map[string]interface{}{},
            Required:   []string{},
        },
    }
    
    handler := func(args map[string]interface{}) types.ActionResponse {
        return types.ActionResponse{
            Action: types.ActionTypeCallHandler,
            Result: types.ActionResponseCall{
                FuncName: "mcp_handler_exit",
                Args:     args,
            },
        }
    }
    
    client.AddTool(tool, handler)
}
```

### 6.2 时间工具

```go
func AddToolTime(client *LocalClient) {
    tool := Tool{
        Name:        "get_current_time",
        Description: "获取当前时间",
        InputSchema: ToolInputSchema{
            Type:       "object",
            Properties: map[string]interface{}{},
            Required:   []string{},
        },
    }
    
    handler := func(args map[string]interface{}) types.ActionResponse {
        currentTime := time.Now().Format("2006-01-02 15:04:05")
        return types.ActionResponse{
            Action: types.ActionTypeReqLLM,
            Result: fmt.Sprintf("当前时间是: %s", currentTime),
        }
    }
    
    client.AddTool(tool, handler)
}
```

## 7. MCP处理器映射

### 7.1 处理器初始化

```go
// connection_handlemcp.go - initMCPResultHandlers方法
func (h *ConnectionHandler) initMCPResultHandlers() {
    h.mcpResultHandlers = map[string]func(interface{}){
        "mcp_handler_exit":       h.handleExit,
        "mcp_handler_take_photo": h.handleTakePhoto,
        "mcp_handler_play_music": h.handlePlayMusic,
        // ... 更多处理器
    }
}
```

### 7.2 具体处理器实现

```go
// 退出处理器
func (h *ConnectionHandler) handleExit(args interface{}) {
    h.LogInfo("执行退出操作")
    // 执行退出逻辑
}

// 拍照处理器
func (h *ConnectionHandler) handleTakePhoto(args interface{}) {
    h.LogInfo("执行拍照操作")
    // 执行拍照逻辑
}

// 播放音乐处理器
func (h *ConnectionHandler) handlePlayMusic(args interface{}) {
    h.LogInfo("执行播放音乐操作")
    // 执行播放音乐逻辑
}
```

## 8. 工具类型分类

### 8.1 ToolType枚举

```go
type ToolType int

const (
    ToolNone            ToolType = iota + 1 // 调用完工具后，不做其他操作
    ToolWait                                // 调用工具，等待函数返回
    ToolChangeSysPrompt                     // 修改系统提示词，切换角色性格或职责
    ToolSystemCtl                           // 系统控制，影响正常的对话流程
    ToolIotCtl                              // IOT设备控制，需要传递conn参数
    ToolMcpClient                           // MCP客户端
)
```

## 9. 错误处理机制

### 9.1 错误类型

- **ActionTypeError**: 工具执行错误
- **ActionTypeNotFound**: 工具未找到
- **连接错误**: MCP客户端连接失败
- **参数错误**: 工具调用参数不正确

### 9.2 错误处理流程

```go
// 错误处理示例
if err != nil {
    h.LogError(fmt.Sprintf("MCP函数调用失败: %v", err))
    if result == nil {
        result = "MCP工具调用失败"
    }
}
```

## 10. 配置和扩展

### 10.1 工具注册

```go
// 注册本地工具
func (lc *LocalClient) RegisterTools(tools []Tool, handlers map[string]HandlerFunc) {
    for i, tool := range tools {
        lc.tools[tool.Name] = tool
        if i < len(handlers) {
            for name, handler := range handlers {
                if name == tool.Name {
                    lc.handlers[tool.Name] = handler
                    break
                }
            }
        }
    }
}
```

### 10.2 动态添加工具

```go
// 动态添加工具
func (lc *LocalClient) AddTool(tool Tool, handler HandlerFunc) {
    lc.tools[tool.Name] = tool
    lc.handlers[tool.Name] = handler
}
```

## 11. 性能和监控

### 11.1 日志记录

- 工具调用开始和结束时间
- 工具执行结果
- 错误信息记录
- 性能指标统计

### 11.2 连接管理

- 连接状态检查 (`IsReady()`)
- 连接重置 (`ResetConnection()`)
- 自动重连机制

## 12. 总结

MCP系统通过以下关键组件实现了灵活的工具调用机制：

1. **统一接口**: `MCPClient`接口提供了标准化的工具调用方式
2. **多客户端支持**: 支持本地工具、远程MCP服务和小智客户端
3. **动作分发**: 通过`ActionResponse`实现不同类型的结果处理
4. **错误处理**: 完善的错误处理和日志记录机制
5. **扩展性**: 支持动态添加工具和处理器

整个调用链路从WebSocket消息接收开始，经过LLM处理、工具识别、执行和结果处理，最终返回给用户，形成了完整的闭环。