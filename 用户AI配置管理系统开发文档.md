# 用户AI配置管理系统开发文档

## 📋 项目概述

**项目名称**: 用户AI配置管理系统  
**项目类型**: 基于GORM的用户自定义AI功能配置与动态Function Call系统  
**开发框架**: Go + Gin + GORM + WebSocket  
**文档版本**: v1.0  
**创建日期**: 2025年1月

**重要说明**：经过对现有代码的深入分析，发现系统已经具备完善的LLM动态调用机制和MCP Function Call功能，本方案将基于现有实现进行扩展。

## 🎯 系统目标

本系统旨在为用户提供灵活的AI功能配置管理能力，支持用户自定义添加多个AI配置，并通过WebSocket连接实现动态Function Call机制，让用户能够通过自然语言描述配置AI功能，系统自动选择最佳AI执行任务。

## 📊 现有功能分析

### 已实现的LLM功能
- **LLM提供者工厂**：`src/core/providers/llm/llm.go`中的`Factory`支持动态创建LLM实例
- **多LLM支持**：已支持Qwen、ChatGLM、Ollama、Coze等多种LLM提供者
- **动态调用**：通过`createModelProvider`方法可根据模型名动态创建LLM实例
- **Function Call支持**：LLM提供者已实现`ResponseWithFunctions`方法

### 已实现的Function Call功能
- **MCP客户端**：完整的MCP(Model Context Protocol)客户端实现
- **本地MCP工具**：支持time、exit、change_role、play_music等本地工具
- **外部MCP工具**：支持连接外部MCP服务器
- **工具注册机制**：动态工具注册和调用机制
- **工具调用处理**：在`connection.go`中已实现完整的工具调用处理流程

## 🏗️ 系统架构

### 核心组件

1. **用户AI配置管理服务** (`UserAIConfigService`)
   - 基于现有GORM数据库操作
   - 提供用户AI配置的CRUD操作
   - 支持配置验证和默认值设置

2. **WebSocket连接增强** (`Enhanced WebSocket Handler`)
   - 扩展现有WebSocket连接处理器
   - 在连接建立时获取用户AI配置
   - 实现配置缓存机制

3. **动态Function Call路由器** (`Dynamic Function Call Router`)
   - 基于现有MCP Manager扩展
   - 根据用户配置动态选择LLM提供者
   - 集成现有工具调用机制

4. **消息处理增强** (`Enhanced Message Processing`)
   - 扩展现有ConnectionHandler
   - 支持用户配置的动态应用
   - 保持与现有消息处理流程的兼容性

### 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        WebSocket Client                         │
└─────────────────────────┬───────────────────────────────────────┘
                          │ JWT + Device-ID + User-Config
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Enhanced WebSocket Transport                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   JWT Auth      │  │  Config Cache   │  │  Session Mgmt   │  │
│  │   (existing)    │  │    (new)        │  │   (existing)    │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                Enhanced Connection Handler                       │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │  Message Proc   │  │  Dynamic Router │  │   MCP Manager   │  │
│  │   (existing)    │  │     (new)       │  │   (existing)    │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                    LLM Provider Factory                         │
│                        (existing)                               │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   Qwen LLM      │  │  ChatGLM LLM    │  │   Ollama LLM    │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## 📊 数据模型设计

### 1. 用户AI配置表 (user_ai_configs)

```go
type UserAIConfig struct {
    ID          uint      `json:"id" gorm:"primaryKey"`
    UserID      string    `json:"user_id" gorm:"index;not null"`
    ConfigName  string    `json:"config_name" gorm:"not null"`
    ConfigType  string    `json:"config_type" gorm:"not null"` // "llm", "function_call"
    
    // LLM配置
    LLMType     string    `json:"llm_type,omitempty"`     // "qwen", "chatglm", "ollama", "coze"
    ModelName   string    `json:"model_name,omitempty"`   // 模型名称
    APIKey      string    `json:"api_key,omitempty"`      // API密钥
    BaseURL     string    `json:"base_url,omitempty"`     // API基础URL
    MaxTokens   int       `json:"max_tokens,omitempty"`   // 最大token数
    Temperature float32   `json:"temperature,omitempty"`  // 温度参数
    
    // Function Call配置
    FunctionName string   `json:"function_name,omitempty"`    // 函数名称
    Description  string   `json:"description,omitempty"`      // 函数描述
    Parameters   string   `json:"parameters,omitempty"`       // JSON格式的参数定义
    MCPServerURL string   `json:"mcp_server_url,omitempty"`   // MCP服务器URL
    
    IsActive    bool      `json:"is_active" gorm:"default:true"`
    Priority    int       `json:"priority" gorm:"default:0"`   // 优先级，数字越大优先级越高
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**字段说明**:
- `UserID`: 用户ID（无外键约束，通过应用层维护关联）
- `Name`: AI配置名称
- `Description`: 功能描述文本，用于Function Call匹配
- `AIType`: AI类型（llm: LLM配置, function: 标准功能）
- `ModelName`: 对应的AI模型名称
- `Parameters`: 相关参数设置（JSON格式存储）
- `IsActive`: 是否启用

### 2. 用户会话配置表 (user_session_configs)

```go
type UserSessionConfig struct {
    ID              uint      `json:"id" gorm:"primaryKey"`
    UserID          string    `json:"user_id" gorm:"index;not null"`
    SessionID       string    `json:"session_id" gorm:"index;not null"`
    DefaultLLMID    uint      `json:"default_llm_id,omitempty"`     // 默认LLM配置ID
    EnabledFunctions string   `json:"enabled_functions,omitempty"`  // JSON数组，启用的函数列表
    SessionData     string    `json:"session_data,omitempty"`       // JSON格式的会话数据
    ExpiresAt       time.Time `json:"expires_at"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

**字段说明**:
- `UserID`: 用户ID
- `SessionID`: WebSocket会话ID
- `ConfigData`: 用户所有AI配置的缓存数据
- `UpdatedAt`: 最后更新时间



## 🔧 核心功能实现

### 1. 用户AI配置管理服务

#### 服务接口定义

```go
// UserAIConfigService 用户AI配置服务接口
type UserAIConfigService interface {
    // CRUD操作
    CreateConfig(ctx context.Context, config *UserAIConfig) error
    GetConfigByID(ctx context.Context, id uint) (*UserAIConfig, error)
    GetUserConfigs(ctx context.Context, userID string, configType string) ([]*UserAIConfig, error)
    UpdateConfig(ctx context.Context, config *UserAIConfig) error
    DeleteConfig(ctx context.Context, id uint) error
    
    // 业务逻辑
    GetActiveConfigs(ctx context.Context, userID string, configType string) ([]*UserAIConfig, error)
    SetConfigPriority(ctx context.Context, id uint, priority int) error
    ToggleConfigStatus(ctx context.Context, id uint, isActive bool) error
    
    // 配置验证
    ValidateLLMConfig(ctx context.Context, config *UserAIConfig) error
    ValidateFunctionConfig(ctx context.Context, config *UserAIConfig) error
    
    // 缓存用户配置到会话
    CacheUserConfigsToSession(userID string, sessionID string) error
    
    // 从缓存获取用户配置
    GetCachedUserConfigs(userID string) ([]*UserAIConfig, error)
}```

#### 服务实现

```go
// DefaultUserAIConfigService 默认用户AI配置服务实现
type DefaultUserAIConfigService struct {
    db     *gorm.DB
    logger *utils.Logger
}

func NewUserAIConfigService(db *gorm.DB, logger *utils.Logger) UserAIConfigService {
    return &DefaultUserAIConfigService{
        db:     db,
        logger: logger,
    }
}

func (s *DefaultUserAIConfigService) CreateConfig(userID uint, req *CreateAIConfigRequest) (*UserAIConfig, error) {
    config := &UserAIConfig{
        UserID:      userID,
        Name:        req.Name,
        Description: req.Description,
        AIType:      req.AIType,
        ModelName:   req.ModelName,
        Parameters:  req.Parameters,
        IsActive:    true,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    
    if err := s.db.Create(config).Error; err != nil {
        s.logger.Error("创建AI配置失败: %v", err)
        return nil, err
    }
    
    // 更新用户会话缓存
    s.CacheUserConfigsToSession(userID, "")
    
    return config, nil
}

func (s *DefaultUserAIConfigService) GetUserConfigs(userID uint) ([]*UserAIConfig, error) {
    var configs []*UserAIConfig
    err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&configs).Error
    return configs, err
}

func (s *DefaultUserAIConfigService) GetActiveConfigs(userID uint) ([]*UserAIConfig, error) {
    var configs []*UserAIConfig
    err := s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&configs).Error
    return configs, err
}

func (s *DefaultUserAIConfigService) CacheUserConfigsToSession(userID string, sessionID string) error {
    configs, err := s.GetActiveConfigs(context.Background(), userID, "")
    if err != nil {
        return err
    }
    
    configData, _ := json.Marshal(configs)
    
    sessionConfig := &UserSessionConfig{
        UserID:          userID,
        SessionID:       sessionID,
        SessionData:     string(configData),
        ExpiresAt:       time.Now().Add(30 * time.Minute),
        UpdatedAt:       time.Now(),
    }
    
    return s.db.Save(sessionConfig).Error
}

func (s *DefaultUserAIConfigService) GetCachedUserConfigs(userID string) ([]*UserAIConfig, error) {
    var sessionConfig UserSessionConfig
    err := s.db.Where("user_id = ? AND expires_at > ?", userID, time.Now()).First(&sessionConfig).Error
    if err != nil {
        return nil, err
    }
    
    var configs []*UserAIConfig
    err = json.Unmarshal([]byte(sessionConfig.SessionData), &configs)
    return configs, err
}

### 2. 动态LLM调用服务 (DynamicLLMService)

```go
type DynamicLLMService interface {
    // 基于现有LLM Provider Factory扩展
    CreateUserLLMProvider(ctx context.Context, userID string, configID uint) (types.LLMProvider, error)
    GetUserDefaultLLM(ctx context.Context, userID string) (types.LLMProvider, error)
    
    // 与现有ResponseWithFunctions集成
    CallLLMWithFunctions(ctx context.Context, userID string, configID uint, messages []types.Message, functions []types.Function) (*types.Response, error)
    
    // 配置管理
    RefreshUserLLMCache(ctx context.Context, userID string) error
    GetAvailableLLMTypes() []string // 返回现有支持的LLM类型
}
```

### 3. 动态Function Call路由服务 (DynamicFunctionRouter)

```go
type DynamicFunctionRouter interface {
    // 基于现有MCP Manager扩展
    RegisterUserFunction(ctx context.Context, userID string, config *UserAIConfig) error
    UnregisterUserFunction(ctx context.Context, userID string, functionName string) error
    
    // 与现有MCP工具集成
    ExecuteUserFunction(ctx context.Context, userID string, functionName string, args map[string]interface{}) (*types.ActionResponse, error)
    GetUserFunctions(ctx context.Context, userID string) ([]mcp.Tool, error)
    
    // 会话管理
    InitializeUserSession(ctx context.Context, userID string, sessionID string) error
    CleanupUserSession(ctx context.Context, sessionID string) error
}
```

### 4. 会话配置管理服务 (SessionConfigService)

```go
type SessionConfigService interface {
    // 会话配置CRUD
    CreateSessionConfig(ctx context.Context, config *UserSessionConfig) error
    GetSessionConfig(ctx context.Context, userID string, sessionID string) (*UserSessionConfig, error)
    UpdateSessionConfig(ctx context.Context, config *UserSessionConfig) error
    DeleteSessionConfig(ctx context.Context, sessionID string) error
    
    // 会话生命周期管理
    RefreshSession(ctx context.Context, sessionID string, ttl time.Duration) error
    CleanupExpiredSessions(ctx context.Context) error
    
    // 配置应用
    ApplySessionConfig(ctx context.Context, sessionID string) error
}
```

### 2. WebSocket连接增强

#### 连接处理器增强

```go
// 在ConnectionHandler中添加用户AI配置相关字段
type ConnectionHandler struct {
    // ... 现有字段 ...
    
    userID           uint                    // 用户ID
    userAIConfigs    []*UserAIConfig        // 用户AI配置缓存
    aiConfigService  UserAIConfigService    // AI配置服务
    functionRouter   *FunctionCallRouter    // Function Call路由器
}

// 增强连接处理器初始化
func (h *ConnectionHandler) initializeUserConfigs() error {
    // 从请求头获取用户ID
    userIDStr := h.headers["User-Id"]
    if userIDStr == "" {
        return fmt.Errorf("缺少用户ID")
    }
    
    userID, err := strconv.ParseUint(userIDStr, 10, 32)
    if err != nil {
        return fmt.Errorf("无效的用户ID: %v", err)
    }
    
    h.userID = uint(userID)
    
    // 获取用户AI配置并缓存到会话
    err = h.aiConfigService.CacheUserConfigsToSession(h.userID, h.sessionID)
    if err != nil {
        h.logger.Warn("缓存用户AI配置失败: %v", err)
    }
    
    // 获取缓存的配置
    h.userAIConfigs, err = h.aiConfigService.GetCachedUserConfigs(h.userID)
    if err != nil {
        h.logger.Warn("获取用户AI配置失败: %v", err)
        h.userAIConfigs = []*UserAIConfig{} // 初始化为空切片
    }
    
    h.logger.Info("用户 %d 加载了 %d 个AI配置", h.userID, len(h.userAIConfigs))
    return nil
}
```

### 3. 动态Function Call路由器

#### 路由器接口定义

```go
// FunctionCallRouter Function Call路由器接口
type FunctionCallRouter interface {
    // 处理用户问题，返回执行结果
    ProcessQuestion(userID uint, sessionID string, question string, userConfigs []*UserAIConfig) (*FunctionCallResult, error)
    
    // 注册标准Function Call处理器
    RegisterStandardFunction(name string, handler StandardFunctionHandler) error
    
    // 执行特定AI配置
    ExecuteAIConfig(config *UserAIConfig, question string) (*FunctionCallResult, error)
}

// FunctionCallResult Function Call执行结果
type FunctionCallResult struct {
    ConfigID     uint   `json:"config_id"`
    ConfigName   string `json:"config_name"`
    Result       string `json:"result"`
    ExecuteTime  int64  `json:"execute_time"`
    Status       string `json:"status"`
    Error        string `json:"error,omitempty"`
}

// StandardFunctionHandler 标准功能处理器
type StandardFunctionHandler func(parameters map[string]interface{}) (*FunctionCallResult, error)
```

#### 路由器实现

```go
// DefaultFunctionCallRouter 默认Function Call路由器实现
type DefaultFunctionCallRouter struct {
    config              *configs.Config
    logger              *utils.Logger
    llmProvider         providers.LLMProvider
    standardFunctions   map[string]StandardFunctionHandler
}

func NewFunctionCallRouter(
    config *configs.Config,
    logger *utils.Logger,
    llmProvider providers.LLMProvider,
) FunctionCallRouter {
    return &DefaultFunctionCallRouter{
        config:            config,
        logger:            logger,
        llmProvider:       llmProvider,
        standardFunctions: make(map[string]StandardFunctionHandler),
    }
}

func (r *DefaultFunctionCallRouter) ProcessQuestion(
    userID uint, 
    sessionID string, 
    question string, 
    userConfigs []*UserAIConfig,
) (*FunctionCallResult, error) {
    startTime := time.Now()
    
    // 构建Function Call提示词
    prompt := r.buildFunctionCallPrompt(question, userConfigs)
    
    // 调用系统默认LLM进行Function Call决策
    response, err := r.llmProvider.Chat(context.Background(), &types.ChatRequest{
        Messages: []types.Message{
            {
                Role:    "user",
                Content: prompt,
            },
        },
        Stream: false,
    })
    
    if err != nil {
        return nil, fmt.Errorf("LLM调用失败: %v", err)
    }
    
    // 解析Function Call响应
    if len(response.ToolCalls) > 0 {
        toolCall := response.ToolCalls[0]
        return r.executeFunctionCall(userID, sessionID, question, toolCall, userConfigs, startTime)
    }
    
    // 没有Function Call，返回直接回答
    result := &FunctionCallResult{
        ConfigID:    0,
        ConfigName:  "direct_answer",
        Result:      response.Content,
        ExecuteTime: time.Since(startTime).Milliseconds(),
        Status:      "success",
    }
    
    return result, nil
}

func (r *DefaultFunctionCallRouter) buildFunctionCallPrompt(question string, userConfigs []*UserAIConfig) string {
    var functions []string
    
    // 添加用户自定义LLM配置
    for _, config := range userConfigs {
        if config.AIType == "llm" {
            functionDef := fmt.Sprintf(`{
    "name": "use_ai_config_%d",
    "description": "%s",
    "parameters": {
        "type": "object",
        "properties": {
            "question": {
                "type": "string",
                "description": "用户的问题"
            }
        },
        "required": ["question"]
    }
}`, config.ID, config.Description)
            functions = append(functions, functionDef)
        }
    }
    
    // 添加标准Function Call
    for name, _ := range r.standardFunctions {
        functionDef := fmt.Sprintf(`{
    "name": "%s",
    "description": "标准功能: %s",
    "parameters": {
        "type": "object",
        "properties": {
            "parameters": {
                "type": "object",
                "description": "功能参数"
            }
        }
    }
}`, name, name)
        functions = append(functions, functionDef)
    }
    
    prompt := fmt.Sprintf(`你是一个智能助手，需要根据用户的问题选择合适的功能来处理。

用户问题: %s

可用功能:
%s

请分析用户问题，如果需要使用特定功能，请调用相应的function call。如果不需要特殊功能，请直接回答。`, 
        question, 
        strings.Join(functions, ",\n"))
    
    return prompt
}

func (r *DefaultFunctionCallRouter) executeFunctionCall(
    userID uint,
    sessionID string,
    question string,
    toolCall types.ToolCall,
    userConfigs []*UserAIConfig,
    startTime time.Time,
) (*FunctionCallResult, error) {
    
    // 解析Function Call名称
    if strings.HasPrefix(toolCall.Function.Name, "use_ai_config_") {
        // 执行用户AI配置
        configIDStr := strings.TrimPrefix(toolCall.Function.Name, "use_ai_config_")
        configID, err := strconv.ParseUint(configIDStr, 10, 32)
        if err != nil {
            return nil, fmt.Errorf("无效的配置ID: %v", err)
        }
        
        // 查找对应配置
        var targetConfig *UserAIConfig
        for _, config := range userConfigs {
            if config.ID == uint(configID) {
                targetConfig = config
                break
            }
        }
        
        if targetConfig == nil {
            return nil, fmt.Errorf("未找到配置ID: %d", configID)
        }
        
        return r.ExecuteAIConfig(targetConfig, question)
        
    } else if handler, exists := r.standardFunctions[toolCall.Function.Name]; exists {
        // 执行标准功能
        var parameters map[string]interface{}
        json.Unmarshal([]byte(toolCall.Function.Arguments), &parameters)
        
        result, err := handler(parameters)
        if err != nil {
            result = &FunctionCallResult{
                ConfigName:  toolCall.Function.Name,
                Result:      "",
                ExecuteTime: time.Since(startTime).Milliseconds(),
                Status:      "failed",
                Error:       err.Error(),
            }
        }
        
        return result, err
    }
    
    return nil, fmt.Errorf("未知的Function Call: %s", toolCall.Function.Name)
}

func (r *DefaultFunctionCallRouter) ExecuteAIConfig(config *UserAIConfig, question string) (*FunctionCallResult, error) {
    startTime := time.Now()
    
    // 根据配置类型执行不同逻辑
    switch config.AIType {
    case "llm":
        return r.executeLLMConfig(config, question, startTime)
    default:
        return nil, fmt.Errorf("不支持的AI类型: %s", config.AIType)
    }
}

func (r *DefaultFunctionCallRouter) executeLLMConfig(config *UserAIConfig, question string, startTime time.Time) (*FunctionCallResult, error) {
    // 解析配置参数
    var parameters map[string]interface{}
    json.Unmarshal(config.Parameters, &parameters)
    
    // 构建LLM请求
    chatRequest := &types.ChatRequest{
        Messages: []types.Message{
            {
                Role:    "user",
                Content: question,
            },
        },
        Stream: false,
    }
    
    // 应用配置参数
    if temperature, ok := parameters["temperature"].(float64); ok {
        chatRequest.Temperature = &temperature
    }
    if maxTokens, ok := parameters["max_tokens"].(float64); ok {
        maxTokensInt := int(maxTokens)
        chatRequest.MaxTokens = &maxTokensInt
    }
    
    // 创建特定模型的LLM提供者
    modelProvider, err := r.createModelProvider(config.ModelName, parameters)
    if err != nil {
        return &FunctionCallResult{
            ConfigID:    config.ID,
            ConfigName:  config.Name,
            Result:      "",
            ExecuteTime: time.Since(startTime).Milliseconds(),
            Status:      "failed",
            Error:       err.Error(),
        }, err
    }
    
    // 执行LLM调用
    response, err := modelProvider.Chat(context.Background(), chatRequest)
    if err != nil {
        return &FunctionCallResult{
            ConfigID:    config.ID,
            ConfigName:  config.Name,
            Result:      "",
            ExecuteTime: time.Since(startTime).Milliseconds(),
            Status:      "failed",
            Error:       err.Error(),
        }, err
    }
    
    return &FunctionCallResult{
        ConfigID:    config.ID,
        ConfigName:  config.Name,
        Result:      response.Content,
        ExecuteTime: time.Since(startTime).Milliseconds(),
        Status:      "success",
    }, nil
}

func (r *DefaultFunctionCallRouter) createModelProvider(modelName string, parameters map[string]interface{}) (providers.LLMProvider, error) {
    // 根据模型名称创建对应的LLM提供者
    // 这里需要根据实际的提供者实现来创建
    // 示例实现：
    switch {
    case strings.Contains(modelName, "qwen"):
        return providers.NewQwenProvider(parameters)
    case strings.Contains(modelName, "gpt"):
        return providers.NewOpenAIProvider(parameters)
    case strings.Contains(modelName, "claude"):
        return providers.NewClaudeProvider(parameters)
    default:
        return nil, fmt.Errorf("不支持的模型: %s", modelName)
    }
}

func (r *DefaultFunctionCallRouter) RegisterStandardFunction(name string, handler StandardFunctionHandler) error {
    r.standardFunctions[name] = handler
    r.logger.Info("注册标准功能: %s", name)
    return nil
}
```

### 4. 消息处理增强

#### 在现有的消息处理中集成Function Call

```go
// 在ConnectionHandler的文本消息处理中添加Function Call逻辑
func (h *ConnectionHandler) processTextMessage(message string) error {
    h.logger.Info("处理文本消息: %s", message)
    
    // 使用Function Call路由器处理问题
    result, err := h.functionRouter.ProcessQuestion(
        h.userID,
        h.sessionID,
        message,
        h.userAIConfigs,
    )
    
    if err != nil {
        h.logger.Error("Function Call处理失败: %v", err)
        // 发送错误消息给客户端
        h.sendErrorMessage(fmt.Sprintf("处理失败: %v", err))
        return err
    }
    
    // 发送结果给客户端
    response := map[string]interface{}{
        "type":         "function_call_result",
        "config_id":    result.ConfigID,
        "config_name":  result.ConfigName,
        "result":       result.Result,
        "execute_time": result.ExecuteTime,
        "status":       result.Status,
    }
    
    if result.Error != "" {
        response["error"] = result.Error
    }
    
    return h.sendJSONMessage(response)
}

func (h *ConnectionHandler) sendErrorMessage(message string) error {
    response := map[string]interface{}{
        "type":    "error",
        "message": message,
    }
    return h.sendJSONMessage(response)
}

func (h *ConnectionHandler) sendJSONMessage(data interface{}) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }
    return h.conn.WriteMessage(1, jsonData) // 1 = TextMessage
}
```

## 🔌 REST API设计

### 1. 用户AI配置管理API

#### 创建AI配置

```http
POST /api/v1/ai-configs
Authorization: Bearer <token>
Content-Type: application/json

{
    "name": "天气查询助手",
    "description": "帮助用户查询天气信息，包括当前天气、未来几天预报等",
    "ai_type": "llm",
    "model_name": "qwen-plus",
    "parameters": {
        "temperature": 0.7,
        "max_tokens": 1000,
        "api_key": "your-api-key",
        "base_url": "https://api.example.com"
    }
}
```

**响应**:
```json
{
    "code": 200,
    "message": "创建成功",
    "data": {
        "id": 1,
        "user_id": 123,
        "name": "天气查询助手",
        "description": "帮助用户查询天气信息，包括当前天气、未来几天预报等",
        "ai_type": "llm",
        "model_name": "qwen-plus",
        "parameters": {
            "temperature": 0.7,
            "max_tokens": 1000,
            "api_key": "your-api-key",
            "base_url": "https://api.example.com"
        },
        "is_active": true,
        "created_at": "2025-01-15T10:30:00Z",
        "updated_at": "2025-01-15T10:30:00Z"
    }
}
```

#### 获取用户所有AI配置

```http
GET /api/v1/ai-configs
Authorization: Bearer <token>
```

**响应**:
```json
{
    "code": 200,
    "message": "获取成功",
    "data": [
        {
            "id": 1,
            "name": "天气查询助手",
            "description": "帮助用户查询天气信息",
            "ai_type": "llm",
            "model_name": "qwen-plus",
            "is_active": true,
            "created_at": "2025-01-15T10:30:00Z"
        },
        {
            "id": 2,
            "name": "代码生成助手",
            "description": "帮助用户生成和优化代码",
            "ai_type": "llm",
            "model_name": "claude-3",
            "is_active": true,
            "created_at": "2025-01-15T11:00:00Z"
        }
    ]
}
```

#### 更新AI配置

```http
PUT /api/v1/ai-configs/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
    "name": "增强天气查询助手",
    "description": "提供详细的天气信息查询，包括空气质量、紫外线指数等",
    "parameters": {
        "temperature": 0.8,
        "max_tokens": 1500
    }
}
```

#### 删除AI配置

```http
DELETE /api/v1/ai-configs/{id}
Authorization: Bearer <token>
```

#### 启用/禁用AI配置

```http
PATCH /api/v1/ai-configs/{id}/toggle
Authorization: Bearer <token>
Content-Type: application/json

{
    "is_active": false
}
```

### 2. API实现代码示例

```go
// GET /api/v1/users/{userID}/ai-configs
// 获取用户AI配置列表
func GetUserAIConfigs(c *gin.Context) {
    userID := c.Param("userID")
    configType := c.Query("type") // "llm" 或 "function_call"
    
    configs, err := configService.GetUserConfigs(c.Request.Context(), userID, configType)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"data": configs})
}

// POST /api/v1/users/{userID}/ai-configs
// 创建用户AI配置
func CreateUserAIConfig(c *gin.Context) {
    userID := c.Param("userID")
    
    var config UserAIConfig
    if err := c.ShouldBindJSON(&config); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    config.UserID = userID
    
    // 验证配置
    if config.ConfigType == "llm" {
        if err := configService.ValidateLLMConfig(c.Request.Context(), &config); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
    } else if config.ConfigType == "function_call" {
        if err := configService.ValidateFunctionConfig(c.Request.Context(), &config); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
    }
    
    if err := configService.CreateConfig(c.Request.Context(), &config); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, gin.H{"data": config})
}

// PUT /api/v1/users/{userID}/ai-configs/{configID}
// 更新用户AI配置
func UpdateUserAIConfig(c *gin.Context) {
    configID, _ := strconv.ParseUint(c.Param("configID"), 10, 32)
    
    var config UserAIConfig
    if err := c.ShouldBindJSON(&config); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    config.ID = uint(configID)
    
    if err := configService.UpdateConfig(c.Request.Context(), &config); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"data": config})
}

// DELETE /api/v1/users/{userID}/ai-configs/{configID}
// 删除用户AI配置
func DeleteUserAIConfig(c *gin.Context) {
    configID, _ := strconv.ParseUint(c.Param("configID"), 10, 32)
    
    if err := configService.DeleteConfig(c.Request.Context(), uint(configID)); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "配置删除成功"})
}
```

## 📋 配置说明

### 1. 无需修改config.yaml

基于现有代码分析，**不需要**在`config.yaml`中添加额外的LLM或Function Call配置：

#### LLM配置说明
- **默认LLM已配置**：`config.yaml#L112`已指定默认LLM为`QwenLLM`
- **多LLM支持完善**：`config.yaml#L151-330`已包含完整的LLM提供者配置
- **动态创建机制**：现有`LLM Provider Factory`支持根据配置动态创建LLM实例

#### Function Call配置说明
- **系统级Function Call**：`config.yaml#L99-105`已定义基础系统级函数(`time`, `exit`, `play_music`等)
- **MCP工具支持**：现有MCP Manager已支持本地和远程MCP工具
- **用户级配置**：通过数据库存储，无需在系统配置中定义

### 2. 现有架构优势

```yaml
# 现有配置结构已经足够完善
llm:
  selected: "QwenLLM"  # 系统默认LLM
  QwenLLM:
    type: "qwen"
    model_name: "qwen-turbo"
    # ... 其他配置

local_mcp_functions:
  - name: "time"        # 系统级基础函数
  - name: "exit"
  - name: "play_music"
  # 用户自定义函数通过数据库管理，不在此配置
```

### 3. 扩展策略

本系统采用**配置分层**策略：

- **系统级配置**：保持在`config.yaml`中，包括默认LLM和基础Function Call
- **用户级配置**：存储在数据库中，支持个性化定制
- **会话级配置**：通过Redis缓存，提供动态配置能力

这种设计既保持了系统配置的稳定性，又提供了用户配置的灵活性。

### 第一阶段：数据模型和基础服务

1. **创建数据模型文件**
   - 在 `src/models/` 目录下创建 `user_ai_config.go`
   - 定义 `UserAIConfig`、`UserSessionConfig` 模型

2. **数据库迁移**
   - 在 `src/configs/database/init.go` 中添加新模型的自动迁移
   - 确保数据库表结构正确创建

3. **创建服务层**
   - 在 `src/services/` 目录下创建 `user_ai_config_service.go`
   - 实现用户AI配置的CRUD操作

### 第二阶段：Function Call路由器

1. **创建路由器模块**
   - 在 `src/core/` 目录下创建 `function_call/` 子目录
   - 实现 `FunctionCallRouter` 接口和默认实现

2. **集成LLM提供者**
   - 扩展现有的LLM提供者支持动态模型创建
   - 实现参数化的LLM调用

3. **标准Function Call支持**
   - 设计标准功能注册机制
   - 预留扩展接口

### 第三阶段：WebSocket集成

1. **增强连接处理器**
   - 修改 `src/core/connection.go`
   - 添加用户配置加载和缓存逻辑

2. **消息处理增强**
   - 集成Function Call路由器到文本消息处理
   - 实现结果返回机制

### 第四阶段：REST API实现

1. **创建API控制器**
   - 在 `src/handlers/` 目录下创建 `ai_config_handler.go`
   - 实现所有CRUD接口

2. **路由注册**
   - 在 `src/routes/` 中注册新的API路由
   - 确保JWT认证中间件正确应用

## 📝 配置示例

### 数据库初始化示例

用户AI配置数据示例：

```json
{
  "id": "config_001",
  "user_id": "user_123",
  "config_name": "我的专属AI助手",
  "config_type": "llm",
  "config_data": {
    "provider": "QwenLLM",
    "model_name": "qwen-turbo",
    "temperature": 0.7,
    "max_tokens": 2000,
    "system_prompt": "你是一个专业的编程助手"
  },
  "is_active": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

Function Call配置示例：

```json
{
  "id": "func_001",
  "user_id": "user_123",
  "config_name": "开发工具集",
  "config_type": "function_call",
  "config_data": {
    "enabled_functions": ["time", "weather", "code_search"],
    "custom_functions": [
      {
        "name": "get_project_info",
        "description": "获取项目信息",
        "parameters": {
          "project_name": "string"
        }
      }
    ]
  },
  "is_active": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}

## 🔒 安全考虑

### 1. 认证和授权

- 所有API接口都需要JWT认证
- 用户只能管理自己的AI配置
- WebSocket连接需要有效的JWT token

### 2. 参数安全

- 敏感参数（如API密钥）需要加密存储
- 限制用户可配置的参数范围
- 防止参数注入攻击

### 3. 执行安全

- 设置Function Call执行超时
- 限制并发执行数量
- 监控异常执行模式

## 📊 性能优化

### 1. 缓存策略

- 用户配置缓存到会话中，避免频繁数据库查询
- LLM提供者实例复用
- 结果缓存（可选）

### 2. 异步处理

- Function Call异步执行
- 结果通过WebSocket实时返回
- 长时间执行的任务支持进度回调

### 3. 资源管理

- 连接池管理
- 垃圾回收优化

## 🧪 扩展性设计

### 1. 插件化架构

- 标准Function Call支持插件式扩展
- 新的AI提供者可以通过插件方式添加
- 支持第三方功能集成

### 2. 多租户支持

- 用户隔离的配置管理
- 资源配额控制
- 多级权限管理

### 3. 分布式部署

- 支持多实例部署
- 会话状态共享
- 负载均衡支持

## 📋 总结

本系统通过以下核心特性实现了用户自定义AI功能配置和动态调用：

1. **灵活的配置管理**: 用户可以自主添加和管理多个AI配置
2. **智能路由机制**: 系统自动选择最适合的AI配置执行任务
3. **实时交互体验**: 基于WebSocket的实时通信
4. **可扩展架构**: 支持标准Function Call和插件化扩展

系统遵循SOLID原则，采用模块化设计，确保代码的可维护性和可扩展性。通过合理的缓存策略和异步处理，保证了系统的高性能表现。

---

**注意**: 本文档提供了完整的实现方案，开发时请根据实际项目需求进行适当调整和优化。