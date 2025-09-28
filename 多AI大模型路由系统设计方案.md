# 多AI大模型路由系统设计方案

## 1. 系统概述

### 1.1 问题背景

当前ASR语音识别系统采用单一LLM处理所有用户请求，存在以下局限性：

- **能力局限**：单一模型无法满足所有场景需求（如实时信息查询、专业领域问答等）
- **效率问题**：通用模型处理简单任务时资源浪费，处理复杂任务时能力不足
- **扩展困难**：添加新功能需要重新训练或更换整个模型

### 1.2 解决方案概述

设计一个**智能路由系统**，通过默认大模型作为路由器，根据用户意图动态选择最适合的专用大模型进行处理。

**核心理念**：
- 🧠 **智能路由**：默认LLM分析用户意图，智能选择专用LLM
- 🎯 **专业分工**：不同LLM专注不同领域，提高处理质量
- 🔄 **无缝切换**：用户无感知的模型切换体验
- 📈 **易于扩展**：支持动态添加新的专用LLM

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   用户输入      │───▶│   路由器LLM      │───▶│   专用LLM池         │
│   (ASR文本)     │    │   (意图分析)     │    │                     │
└─────────────────┘    └──────────────────┘    │  ┌───────────────┐  │
                                ▲               │  │  联网LLM      │  │
                                │               │  └───────────────┘  │
                                │               │  ┌───────────────┐  │
                                │               │  │  代码LLM      │  │
                                │               │  └───────────────┘  │
                                │               │  ┌───────────────┐  │
                                │               │  │  图像LLM      │  │
                                │               │  └───────────────┘  │
                                │               │  ┌───────────────┐  │
                                │               │  │  通用LLM      │  │
                                │               │  └───────────────┘  │
                                │               └─────────────────────┘
                                │                         │
                                │                         ▼
                                │               ┌─────────────────────┐
                                └───────────────│   响应结果          │
                                                │   (返回用户)        │
                                                └─────────────────────┘
```

### 2.2 核心组件

#### 2.2.1 路由器LLM (RouterLLMProvider)
- **职责**：分析用户意图，决定使用哪个专用LLM
- **特点**：轻量级、快速响应、专注意图识别
- **输入**：用户原始请求 + 上下文信息
- **输出**：路由决策 + 处理后的请求

#### 2.2.2 专用LLM池 (SpecializedLLMPool)
- **联网LLM**：具备实时信息查询能力（天气、股价、新闻等）
- **代码LLM**：专门处理编程、技术问题
- **图像LLM**：处理视觉理解和图像相关任务
- **通用LLM**：处理日常对话和通用问题

#### 2.2.3 路由管理器 (RouterManager)
- **LLM注册与发现**：动态管理专用LLM实例
- **负载均衡**：分配请求到可用的LLM实例
- **健康检查**：监控LLM状态，自动故障转移

#### 2.2.4 意图分类器 (IntentClassifier)
- **关键词匹配**：基于预定义规则的快速分类
- **语义分析**：使用轻量级NLP模型进行意图识别
- **上下文感知**：结合对话历史进行智能判断

## 3. 技术实现

### 3.1 路由器实现

```go
// RouterLLMProvider 路由器LLM提供者
type RouterLLMProvider struct {
    *llm.BaseProvider
    routerLLM      llm.Provider           // 路由决策LLM
    specializedLLMs map[string]llm.Provider // 专用LLM池
    intentClassifier *IntentClassifier     // 意图分类器
    routingStrategy  RoutingStrategy       // 路由策略
}

// RoutingDecision 路由决策结果
type RoutingDecision struct {
    TargetLLM    string                 // 目标LLM名称
    Confidence   float64                // 置信度
    ProcessedMsg []types.Message        // 处理后的消息
    Metadata     map[string]interface{} // 元数据
}

// RoutingStrategy 路由策略接口
type RoutingStrategy interface {
    Route(ctx context.Context, messages []types.Message) (*RoutingDecision, error)
    RegisterLLM(name string, llm llm.Provider) error
    GetAvailableLLMs() []string
}
```

### 3.2 配置结构扩展

```yaml
# 路由器LLM配置
RouterLLM:
  # 路由器配置
  router:
    type: openai
    model_name: gpt-3.5-turbo  # 轻量级模型用于路由决策
    url: https://api.openai.com/v1
    api_key: your_router_api_key
    temperature: 0.1  # 低温度确保稳定的路由决策
  
  # 专用LLM配置
  specialized_llms:
    # 联网LLM
    internet_llm:
      type: openai
      model_name: gpt-4
      url: https://api.openai.com/v1
      api_key: your_internet_api_key
      capabilities: ["web_search", "real_time_info"]
      
    # 代码LLM  
    code_llm:
      type: openai
      model_name: gpt-4-code
      url: https://api.openai.com/v1
      api_key: your_code_api_key
      capabilities: ["programming", "debugging", "code_review"]
      
    # 通用LLM
    general_llm:
      type: openai
      model_name: gpt-3.5-turbo
      url: https://api.openai.com/v1
      api_key: your_general_api_key
      capabilities: ["conversation", "general_qa"]

  # 路由规则配置
  routing_rules:
    # 关键词路由规则
    keyword_rules:
      - keywords: ["天气", "温度", "下雨", "晴天"]
        target_llm: "internet_llm"
        confidence: 0.9
      - keywords: ["股价", "股票", "行情", "涨跌"]
        target_llm: "internet_llm"
        confidence: 0.9
      - keywords: ["代码", "编程", "bug", "函数", "算法"]
        target_llm: "code_llm"
        confidence: 0.8
        
    # 意图路由规则
    intent_rules:
      - intent: "weather_query"
        target_llm: "internet_llm"
      - intent: "programming_help"
        target_llm: "code_llm"
      - intent: "general_chat"
        target_llm: "general_llm"
        
    # 默认路由
    default_llm: "general_llm"
```

### 3.3 路由策略实现

#### 3.3.1 关键词匹配策略

```go
type KeywordRoutingStrategy struct {
    rules []KeywordRule
}

type KeywordRule struct {
    Keywords   []string
    TargetLLM  string
    Confidence float64
}

func (s *KeywordRoutingStrategy) Route(ctx context.Context, messages []types.Message) (*RoutingDecision, error) {
    lastMessage := messages[len(messages)-1].Content
    
    for _, rule := range s.rules {
        for _, keyword := range rule.Keywords {
            if strings.Contains(strings.ToLower(lastMessage), strings.ToLower(keyword)) {
                return &RoutingDecision{
                    TargetLLM:    rule.TargetLLM,
                    Confidence:   rule.Confidence,
                    ProcessedMsg: messages,
                }, nil
            }
        }
    }
    
    // 默认路由
    return &RoutingDecision{
        TargetLLM:    "general_llm",
        Confidence:   0.5,
        ProcessedMsg: messages,
    }, nil
}
```

#### 3.3.2 智能意图识别策略

```go
type IntentRoutingStrategy struct {
    routerLLM llm.Provider
    intentRules map[string]string
}

func (s *IntentRoutingStrategy) Route(ctx context.Context, messages []types.Message) (*RoutingDecision, error) {
    // 构造意图识别提示词
    intentPrompt := s.buildIntentPrompt(messages)
    
    // 调用路由器LLM进行意图识别
    responseChan, err := s.routerLLM.Response(ctx, "intent_analysis", intentPrompt)
    if err != nil {
        return nil, err
    }
    
    // 解析意图识别结果
    intent := s.parseIntentResponse(responseChan)
    
    // 根据意图选择目标LLM
    targetLLM, exists := s.intentRules[intent]
    if !exists {
        targetLLM = "general_llm"
    }
    
    return &RoutingDecision{
        TargetLLM:    targetLLM,
        Confidence:   0.8,
        ProcessedMsg: messages,
        Metadata:     map[string]interface{}{"intent": intent},
    }, nil
}
```

## 4. 路由决策流程

### 4.1 决策流程图

```
用户输入
    │
    ▼
┌─────────────────┐
│  预处理消息     │
│  (清理、格式化) │
└─────────────────┘
    │
    ▼
┌─────────────────┐
│  关键词匹配     │
│  (快速路由)     │
└─────────────────┘
    │
    ▼
┌─────────────────┐    高置信度    ┌─────────────────┐
│  置信度检查     │──────────────▶│  执行路由决策   │
└─────────────────┘                └─────────────────┘
    │ 低置信度                           │
    ▼                                    ▼
┌─────────────────┐                ┌─────────────────┐
│  智能意图识别   │                │  调用专用LLM   │
│  (路由器LLM)    │                └─────────────────┘
└─────────────────┘                           │
    │                                         ▼
    ▼                                ┌─────────────────┐
┌─────────────────┐                │  返回处理结果   │
│  最终路由决策   │                └─────────────────┘
└─────────────────┘
    │
    ▼
┌─────────────────┐
│  调用专用LLM   │
└─────────────────┘
```

### 4.2 错误处理和回退机制

```go
func (r *RouterLLMProvider) ResponseWithFunctions(
    ctx context.Context, 
    sessionID string, 
    messages []types.Message, 
    tools []openai.Tool,
) (<-chan types.Response, error) {
    
    // 1. 路由决策
    decision, err := r.routingStrategy.Route(ctx, messages)
    if err != nil {
        // 路由失败，使用默认LLM
        return r.fallbackToDefault(ctx, sessionID, messages, tools)
    }
    
    // 2. 获取目标LLM
    targetLLM, exists := r.specializedLLMs[decision.TargetLLM]
    if !exists {
        // 目标LLM不存在，使用默认LLM
        return r.fallbackToDefault(ctx, sessionID, messages, tools)
    }
    
    // 3. 调用专用LLM
    responseChan, err := targetLLM.ResponseWithFunctions(ctx, sessionID, decision.ProcessedMsg, tools)
    if err != nil {
        // 专用LLM调用失败，使用默认LLM
        return r.fallbackToDefault(ctx, sessionID, messages, tools)
    }
    
    // 4. 包装响应，添加路由信息
    return r.wrapResponse(responseChan, decision), nil
}

func (r *RouterLLMProvider) fallbackToDefault(
    ctx context.Context, 
    sessionID string, 
    messages []types.Message, 
    tools []openai.Tool,
) (<-chan types.Response, error) {
    defaultLLM := r.specializedLLMs["general_llm"]
    if defaultLLM == nil {
        return nil, fmt.Errorf("默认LLM不可用")
    }
    return defaultLLM.ResponseWithFunctions(ctx, sessionID, messages, tools)
}
```

## 5. 实施计划

### 5.1 阶段一：基础路由器实现 (2-3周)

**目标**：实现基本的关键词路由功能

**任务清单**：
- [ ] 创建RouterLLMProvider结构体
- [ ] 实现KeywordRoutingStrategy
- [ ] 配置文件结构扩展
- [ ] 基础的专用LLM管理
- [ ] 单元测试和集成测试

**交付物**：
- 支持2-3个专用LLM的基础路由系统
- 配置文件模板和文档
- 基础测试用例

### 5.2 阶段二：智能路由决策 (3-4周)

**目标**：集成意图识别，提高路由准确性

**任务清单**：
- [ ] 实现IntentRoutingStrategy
- [ ] 集成轻量级意图识别模型
- [ ] 上下文感知路由逻辑
- [ ] 路由决策日志和监控
- [ ] 性能优化和缓存机制

**交付物**：
- 智能意图识别路由系统
- 路由决策监控面板
- 性能测试报告

### 5.3 阶段三：高级功能和优化 (2-3周)

**目标**：完善系统功能，提升用户体验

**任务清单**：
- [ ] 多轮对话中的LLM切换
- [ ] LLM协作机制
- [ ] 动态LLM注册和发现
- [ ] 高级监控和分析
- [ ] 用户界面和管理工具

**交付物**：
- 完整的多AI路由系统
- 管理和监控工具
- 用户使用指南

## 6. 配置示例

### 6.1 完整配置文件示例

```yaml
# 选择使用路由器LLM
SelectedModule:
  LLM: "RouterLLM"

# 路由器LLM配置
LLM:
  RouterLLM:
    type: router
    
    # 路由器LLM配置
    router:
      type: openai
      model_name: gpt-3.5-turbo
      url: https://api.openai.com/v1
      api_key: sk-your-router-api-key
      temperature: 0.1
      max_tokens: 500
    
    # 专用LLM配置
    specialized_llms:
      # 联网查询LLM
      internet_llm:
        type: openai
        model_name: gpt-4
        url: https://api.openai.com/v1
        api_key: sk-your-internet-api-key
        temperature: 0.7
        max_tokens: 1000
        capabilities: ["web_search", "real_time_info", "weather", "news"]
        
      # 编程助手LLM
      code_llm:
        type: openai
        model_name: gpt-4-code
        url: https://api.openai.com/v1
        api_key: sk-your-code-api-key
        temperature: 0.2
        max_tokens: 2000
        capabilities: ["programming", "debugging", "code_review", "algorithm"]
        
      # 通用对话LLM
      general_llm:
        type: openai
        model_name: gpt-3.5-turbo
        url: https://api.openai.com/v1
        api_key: sk-your-general-api-key
        temperature: 0.8
        max_tokens: 1000
        capabilities: ["conversation", "general_qa", "creative_writing"]
    
    # 路由策略配置
    routing_strategy:
      type: "hybrid"  # keyword | intent | hybrid
      
      # 关键词路由规则
      keyword_rules:
        - keywords: ["天气", "温度", "下雨", "晴天", "气温", "降雨"]
          target_llm: "internet_llm"
          confidence: 0.9
          
        - keywords: ["股价", "股票", "行情", "涨跌", "市值", "财经"]
          target_llm: "internet_llm"
          confidence: 0.9
          
        - keywords: ["新闻", "资讯", "最新", "今天发生", "热点"]
          target_llm: "internet_llm"
          confidence: 0.8
          
        - keywords: ["代码", "编程", "bug", "函数", "算法", "开发"]
          target_llm: "code_llm"
          confidence: 0.8
          
        - keywords: ["Python", "JavaScript", "Java", "Go", "C++"]
          target_llm: "code_llm"
          confidence: 0.9
      
      # 意图路由规则
      intent_rules:
        weather_query: "internet_llm"
        stock_query: "internet_llm"
        news_query: "internet_llm"
        programming_help: "code_llm"
        code_review: "code_llm"
        debugging: "code_llm"
        general_chat: "general_llm"
        creative_writing: "general_llm"
      
      # 默认路由
      default_llm: "general_llm"
      
      # 置信度阈值
      confidence_threshold: 0.7
      
      # 缓存配置
      cache:
        enabled: true
        ttl: 300  # 5分钟
        max_size: 1000
```

### 6.2 路由器提示词模板

```yaml
# 路由器系统提示词
router_system_prompt: |
  你是一个智能路由器，负责分析用户意图并选择最适合的AI助手。
  
  可用的AI助手：
  1. internet_llm - 联网查询助手，擅长实时信息查询（天气、新闻、股价等）
  2. code_llm - 编程助手，擅长代码编写、调试、算法问题
  3. general_llm - 通用对话助手，擅长日常对话、创意写作
  
  请根据用户输入，返回JSON格式的路由决策：
  {
    "target_llm": "助手名称",
    "confidence": 0.0-1.0,
    "reason": "选择理由"
  }
  
  示例：
  用户："今天珠海天气怎么样？"
  回复：{"target_llm": "internet_llm", "confidence": 0.9, "reason": "需要查询实时天气信息"}
```

## 7. API设计

### 7.1 路由器管理API

```go
// RouterManager API接口
type RouterManagerAPI interface {
    // 注册专用LLM
    RegisterLLM(name string, config *llm.Config) error
    
    // 注销专用LLM
    UnregisterLLM(name string) error
    
    // 获取所有LLM状态
    GetLLMStatus() map[string]LLMStatus
    
    // 更新路由规则
    UpdateRoutingRules(rules *RoutingRules) error
    
    // 获取路由统计
    GetRoutingStats() *RoutingStats
    
    // 测试路由决策
    TestRouting(message string) (*RoutingDecision, error)
}

// LLM状态
type LLMStatus struct {
    Name         string    `json:"name"`
    Type         string    `json:"type"`
    Status       string    `json:"status"`       // online | offline | error
    LastUsed     time.Time `json:"last_used"`
    RequestCount int64     `json:"request_count"`
    ErrorCount   int64     `json:"error_count"`
    AvgLatency   float64   `json:"avg_latency"`
}

// 路由统计
type RoutingStats struct {
    TotalRequests    int64                    `json:"total_requests"`
    RoutingAccuracy  float64                  `json:"routing_accuracy"`
    LLMUsageStats    map[string]int64         `json:"llm_usage_stats"`
    AvgRoutingTime   float64                  `json:"avg_routing_time"`
    ErrorRate        float64                  `json:"error_rate"`
    TopIntents       []IntentStat             `json:"top_intents"`
}
```

### 7.2 REST API端点

```yaml
# 路由器管理API
/api/router:
  get:
    summary: 获取路由器状态
    responses:
      200:
        description: 路由器状态信息
        
  post:
    summary: 测试路由决策
    requestBody:
      content:
        application/json:
          schema:
            type: object
            properties:
              message:
                type: string
                description: 测试消息
    responses:
      200:
        description: 路由决策结果

/api/router/llms:
  get:
    summary: 获取所有LLM状态
    responses:
      200:
        description: LLM状态列表
        
  post:
    summary: 注册新的LLM
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/LLMConfig'

/api/router/llms/{name}:
  delete:
    summary: 注销指定LLM
    parameters:
      - name: name
        in: path
        required: true
        schema:
          type: string

/api/router/rules:
  get:
    summary: 获取路由规则
    responses:
      200:
        description: 当前路由规则
        
  put:
    summary: 更新路由规则
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/RoutingRules'

/api/router/stats:
  get:
    summary: 获取路由统计信息
    responses:
      200:
        description: 路由统计数据
```

## 8. 监控和优化

### 8.1 性能监控指标

```go
// 监控指标
type RouterMetrics struct {
    // 路由性能指标
    RoutingLatency    prometheus.Histogram // 路由决策延迟
    RoutingAccuracy   prometheus.Gauge     // 路由准确率
    RoutingErrors     prometheus.Counter   // 路由错误计数
    
    // LLM使用指标
    LLMRequestCount   prometheus.CounterVec   // 各LLM请求计数
    LLMLatency        prometheus.HistogramVec // 各LLM响应延迟
    LLMErrorRate      prometheus.GaugeVec     // 各LLM错误率
    
    // 系统资源指标
    MemoryUsage       prometheus.Gauge     // 内存使用量
    CPUUsage          prometheus.Gauge     // CPU使用率
    ActiveConnections prometheus.Gauge     // 活跃连接数
}
```

### 8.2 日志记录

```go
// 路由决策日志
type RoutingLog struct {
    Timestamp    time.Time              `json:"timestamp"`
    SessionID    string                 `json:"session_id"`
    UserMessage  string                 `json:"user_message"`
    Intent       string                 `json:"intent"`
    TargetLLM    string                 `json:"target_llm"`
    Confidence   float64                `json:"confidence"`
    RoutingTime  time.Duration          `json:"routing_time"`
    Success      bool                   `json:"success"`
    Error        string                 `json:"error,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// 日志记录器
type RoutingLogger struct {
    logger *logrus.Logger
    buffer chan *RoutingLog
}

func (rl *RoutingLogger) LogRouting(log *RoutingLog) {
    select {
    case rl.buffer <- log:
    default:
        rl.logger.Warn("路由日志缓冲区已满，丢弃日志")
    }
}
```

### 8.3 优化建议

#### 8.3.1 性能优化
- **缓存机制**：缓存常见查询的路由决策结果
- **预热策略**：预先加载常用的专用LLM
- **连接池**：复用LLM连接，减少建立连接的开销
- **异步处理**：非关键路径使用异步处理

#### 8.3.2 准确性优化
- **A/B测试**：对比不同路由策略的效果
- **用户反馈**：收集用户对路由结果的满意度
- **机器学习**：使用历史数据训练更好的意图识别模型
- **规则优化**：定期分析日志，优化路由规则

#### 8.3.3 可靠性优化
- **健康检查**：定期检查各LLM的可用性
- **故障转移**：自动切换到备用LLM
- **限流保护**：防止单个LLM过载
- **熔断机制**：在LLM持续失败时暂时停用

## 9. 使用示例

### 9.1 用户对话示例

**示例1：天气查询**
```
用户：帮我查一下珠海今天的天气
路由器分析：检测到"天气"关键词 → 路由到internet_llm
internet_llm：正在为您查询珠海今天的天气...
[调用天气API]
internet_llm：珠海今天多云，气温22-28℃，东南风3-4级，适合外出。
```

**示例2：编程问题**
```
用户：帮我写一个Python函数，计算斐波那契数列
路由器分析：检测到"Python"和"函数"关键词 → 路由到code_llm
code_llm：我来为您编写一个计算斐波那契数列的Python函数...
[生成代码和解释]
```

**示例3：日常对话**
```
用户：今天心情不太好，聊聊天吧
路由器分析：意图识别为"general_chat" → 路由到general_llm
general_llm：我理解您的感受，有什么特别的事情让您心情不好吗？...
```

### 9.2 配置和部署示例

```bash
# 1. 更新配置文件
cp config.yaml .config.yaml
# 编辑.config.yaml，添加RouterLLM配置

# 2. 启动服务
go run ./src/main.go

# 3. 测试路由功能
curl -X POST http://localhost:8080/api/router \
  -H "Content-Type: application/json" \
  -d '{"message": "今天天气怎么样？"}'

# 4. 查看路由统计
curl http://localhost:8080/api/router/stats
```

## 10. 总结

本设计方案提供了一个完整的多AI大模型路由系统解决方案，具有以下优势：

### 10.1 核心优势
- 🎯 **智能路由**：根据用户意图自动选择最适合的LLM
- 🚀 **性能提升**：专业分工提高处理质量和效率
- 🔧 **易于扩展**：支持动态添加新的专用LLM
- 🛡️ **高可靠性**：完善的错误处理和回退机制
- 📊 **可观测性**：全面的监控和日志记录

### 10.2 实施价值
- **用户体验**：更准确、更专业的AI响应
- **资源优化**：合理分配计算资源，降低成本
- **业务扩展**：支持更多专业领域的AI服务
- **技术演进**：为未来的AI能力扩展奠定基础

### 10.3 下一步行动
1. 根据实施计划开始阶段一的开发工作
2. 准备测试数据和评估指标
3. 搭建开发和测试环境
4. 开始编码实现核心组件

通过这个路由系统，您的ASR语音识别项目将具备更强的AI能力和更好的用户体验，为未来的功能扩展提供坚实的技术基础。