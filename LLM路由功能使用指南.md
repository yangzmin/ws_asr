# LLM路由功能使用指南

## 📋 目录

- [功能概述](#功能概述)
- [核心特性](#核心特性)
- [系统架构](#系统架构)
- [功能状态检查](#功能状态检查)
- [启用方法](#启用方法)
- [配置详解](#配置详解)
- [API调用方法](#api调用方法)
- [使用场景](#使用场景)
- [最佳实践](#最佳实践)
- [监控与调试](#监控与调试)
- [故障排除](#故障排除)
- [FAQ常见问题](#faq常见问题)

---

## 🎯 功能概述

LLM路由系统是一个智能的多AI大模型调度中心，能够根据用户请求的内容和意图，自动选择最适合的AI模型进行响应。系统支持关键词匹配、意图识别和混合路由策略，实现了AI能力的专业化分工和智能调度。

### 核心价值
- **🎯 智能路由**：自动识别请求类型，选择最适合的AI模型
- **⚡ 性能优化**：专用模型处理专门任务，提升响应质量和速度
- **💰 成本控制**：根据任务复杂度选择合适的模型，优化成本
- **🔧 灵活配置**：支持动态添加、删除和配置AI模型
- **📊 全面监控**：实时监控路由性能和模型状态

---

## ✨ 核心特性

### 1. 智能路由策略
- **关键词路由**：基于预定义关键词规则进行路由
- **意图识别路由**：使用AI模型识别用户意图
- **混合路由**：结合关键词和意图识别的综合策略

### 2. 专用LLM管理
- **动态注册**：运行时添加和删除专用LLM
- **负载均衡**：智能分配请求到不同模型
- **故障转移**：自动降级到备用模型

### 3. 性能优化
- **智能缓存**：缓存路由决策和响应结果
- **连接池**：复用HTTP连接提升性能
- **异步处理**：非阻塞的请求处理机制

### 4. 监控与管理
- **实时监控**：路由性能、模型状态、错误率统计
- **REST API**：完整的管理和监控接口
- **日志记录**：详细的路由决策和错误日志

---

## 🏗️ 系统架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   用户请求      │───▶│   路由器核心     │───▶│   专用LLM池     │
│  (WebSocket/    │    │  - 意图识别      │    │  - 联网查询LLM  │
│   REST API)     │    │  - 关键词匹配    │    │  - 代码助手LLM  │
└─────────────────┘    │  - 路由决策      │    │  - 通用对话LLM  │
                       │  - 缓存管理      │    │  - 自定义LLM    │
                       └──────────────────┘    └─────────────────┘
                                │
                       ┌──────────────────┐
                       │   监控与管理     │
                       │  - 性能指标      │
                       │  - 状态监控      │
                       │  - API管理       │
                       └──────────────────┘
```

---

## 🔍 功能状态检查

### 检查路由器注册状态
路由器已在系统中正确注册，可以通过以下方式验证：

```bash
# 检查服务器日志中的注册信息
grep "router" src/logs/server.log

# 或者启动服务器后查看注册的LLM提供者
curl -X GET http://localhost:8080/api/router
```

### 检查当前配置状态
当前 `config.yaml` 中的LLM配置：
- **当前模块**：`QwenLLM`（单一模型）
- **路由状态**：❌ 未启用
- **需要操作**：切换到 `RouterLLM` 并添加路由配置

---

## 🚀 启用方法

### 步骤1：修改模块选择

在 `config.yaml` 文件中找到 `selected_module` 部分（约第99行），将LLM模块改为RouterLLM：

```yaml
selected_module:
  ASR: DoubaoASR
  TTS: DoubaoTTS
  LLM: RouterLLM  # 从 QwenLLM 改为 RouterLLM
  VLLLM: ChatGLMVLLM
```

### 步骤2：添加RouterLLM配置

在 `config.yaml` 的LLM配置部分添加完整的RouterLLM配置：

```yaml
LLM:
  # 保留现有的其他LLM配置...
  
  RouterLLM:
    type: router
    router_config:
      # 路由器LLM（用于意图识别）
      router:
        type: openai
        model_name: qwen-flash-2025-07-28
        url: https://dashscope.aliyuncs.com/compatible-mode/v1
        api_key: sk-bb65b7b2775049d3827e76c3857027e5
        temperature: 0.1
        max_tokens: 500
      
      # 专用LLM配置
      specialized_llms:
        internet_llm:
          type: openai
          model_name: qwen-flash-2025-07-28
          url: https://dashscope.aliyuncs.com/compatible-mode/v1
          api_key: sk-bb65b7b2775049d3827e76c3857027e5
          temperature: 0.3
          max_tokens: 2000
        
        code_llm:
          type: openai
          model_name: glm-4-flash
          url: https://open.bigmodel.cn/api/paas/v4/
          api_key: 75cd6a6a64c64acab762fe8a3f571bd2.avQ3LKXmKVjj1v8F
          temperature: 0.1
          max_tokens: 4000
        
        general_llm:
          type: openai
          model_name: qwen-flash-2025-07-28
          url: https://dashscope.aliyuncs.com/compatible-mode/v1
          api_key: sk-bb65b7b2775049d3827e76c3857027e5
          temperature: 0.7
          max_tokens: 2000
      
      # 路由策略配置
      routing_strategy:
        type: hybrid  # keyword | intent | hybrid
        keyword_rules:
          - keywords: ["天气", "温度", "气候", "下雨", "晴天", "预报"]
            target_llm: internet_llm
            confidence: 0.9
          - keywords: ["代码", "编程", "程序", "算法", "bug", "函数", "Python", "JavaScript", "Go"]
            target_llm: code_llm
            confidence: 0.8
          - keywords: ["搜索", "查询", "最新", "新闻", "实时"]
            target_llm: internet_llm
            confidence: 0.7
        default_llm: general_llm
      
      # 基础配置
      default_llm: general_llm
      confidence_threshold: 0.7
      cache_enabled: true
      cache_ttl: 300
      max_cache_size: 1000
```

### 步骤3：重启服务器

```bash
# 停止当前服务器（如果正在运行）
# Ctrl+C 或者 kill 进程

# 重新编译并启动
go build -o bin/server.exe ./src/main.go
./bin/server.exe
```

---

## ⚙️ 配置详解

### 路由器LLM配置
```yaml
router:
  type: openai              # LLM类型
  model_name: qwen-flash    # 模型名称（建议使用快速、便宜的模型）
  temperature: 0.1          # 低温度确保意图识别的一致性
  max_tokens: 500           # 意图识别不需要太多token
```

### 专用LLM配置
```yaml
specialized_llms:
  internet_llm:             # 联网查询专用
    type: openai
    model_name: qwen-flash-2025-07-28
    temperature: 0.3        # 中等创造性
    max_tokens: 2000
  
  code_llm:                 # 代码助手专用
    type: openai
    model_name: glm-4-flash
    temperature: 0.1        # 低创造性，确保代码准确性
    max_tokens: 4000        # 代码生成需要更多token
  
  general_llm:              # 通用对话
    type: openai
    model_name: qwen-flash-2025-07-28
    temperature: 0.7        # 高创造性
    max_tokens: 2000
```

### 路由策略配置
```yaml
routing_strategy:
  type: hybrid              # 推荐使用混合策略
  keyword_rules:
    - keywords: ["关键词1", "关键词2"]
      target_llm: 目标LLM名称
      confidence: 0.8       # 匹配置信度
  default_llm: general_llm  # 默认LLM
```

### 缓存配置
```yaml
cache_enabled: true         # 启用缓存
cache_ttl: 300             # 缓存生存时间（秒）
max_cache_size: 1000       # 最大缓存条目数
confidence_threshold: 0.7   # 路由置信度阈值
```

---

## 📡 API调用方法

### WebSocket调用（推荐）
```javascript
// 建立WebSocket连接
const ws = new WebSocket('ws://localhost:8000');

// 发送消息（会自动路由）
ws.send(JSON.stringify({
  type: 'chat',
  message: '今天北京的天气怎么样？',  // 会路由到internet_llm
  session_id: 'user_session_123'
}));

// 接收响应
ws.onmessage = function(event) {
  const response = JSON.parse(event.data);
  console.log('AI回复:', response.message);
};
```

### REST API调用

#### 1. 路由器状态查询
```bash
curl -X GET http://localhost:8080/api/router
```

响应示例：
```json
{
  "status": "active",
  "strategy": "hybrid",
  "llm_count": 3,
  "cache_enabled": true
}
```

#### 2. 获取所有LLM状态
```bash
curl -X GET http://localhost:8080/api/router/llms
```

#### 3. 测试路由决策
```bash
curl -X POST http://localhost:8080/api/router/route \
  -H "Content-Type: application/json" \
  -d '{
    "message": "帮我写一个Python排序函数",
    "session_id": "test_session"
  }'
```

#### 4. 获取路由统计
```bash
curl -X GET http://localhost:8080/api/router/stats
```

#### 5. 动态LLM管理
```bash
# 注册新LLM
curl -X POST http://localhost:8080/api/router/llms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "translation_llm",
    "config": {
      "type": "openai",
      "model_name": "gpt-4",
      "api_key": "your_api_key",
      "temperature": 0.3
    }
  }'

# 注销LLM
curl -X DELETE http://localhost:8080/api/router/llms/translation_llm
```

---

## 🎯 使用场景

### 1. 联网查询场景
**触发关键词**：天气、新闻、股价、实时信息
**路由目标**：`internet_llm`
**示例请求**：
- "今天上海的天气如何？"
- "最新的科技新闻有哪些？"
- "比特币现在的价格是多少？"

### 2. 编程助手场景
**触发关键词**：代码、编程、算法、bug、函数
**路由目标**：`code_llm`
**示例请求**：
- "帮我写一个快速排序算法"
- "这段Python代码有什么问题？"
- "如何优化这个SQL查询？"

### 3. 通用对话场景
**触发条件**：不匹配特定关键词的请求
**路由目标**：`general_llm`
**示例请求**：
- "给我讲个笑话"
- "帮我写一首诗"
- "人生的意义是什么？"

---

## ⭐ 最佳实践

### 1. 关键词规则优化
```yaml
keyword_rules:
  # 高置信度规则（精确匹配）
  - keywords: ["天气预报", "气温", "降雨概率"]
    target_llm: internet_llm
    confidence: 0.9
  
  # 中等置信度规则（模糊匹配）
  - keywords: ["代码", "编程", "算法"]
    target_llm: code_llm
    confidence: 0.8
  
  # 低置信度规则（兜底匹配）
  - keywords: ["搜索", "查询"]
    target_llm: internet_llm
    confidence: 0.6
```

### 2. 模型选择建议
- **路由器LLM**：选择快速、便宜的模型（如qwen-flash）
- **代码LLM**：选择代码能力强的模型（如glm-4-flash、claude-3）
- **联网LLM**：选择信息检索能力强的模型
- **通用LLM**：选择平衡性能和成本的模型

### 3. 性能优化
```yaml
# 缓存配置优化
cache_enabled: true
cache_ttl: 600          # 根据业务需求调整
max_cache_size: 2000    # 根据内存情况调整

# 置信度阈值调整
confidence_threshold: 0.7  # 过低会误路由，过高会降级过多
```

### 4. 监控指标关注
- **路由准确率**：正确路由的请求比例
- **平均响应时间**：包含路由决策的总时间
- **缓存命中率**：缓存使用效率
- **错误率**：路由失败和LLM调用失败的比例

---

## 📊 监控与调试

### 1. 实时监控
```bash
# 查看路由统计
curl -X GET http://localhost:8080/api/router/stats

# 查看LLM状态
curl -X GET http://localhost:8080/api/router/llms

# 查看缓存状态
curl -X GET http://localhost:8080/api/router/cache
```

### 2. 日志分析
```bash
# 查看路由决策日志
grep "routing decision" src/logs/server.log

# 查看错误日志
grep "ERROR" src/logs/server.log | grep "router"

# 查看性能日志
grep "router metrics" src/logs/server.log
```

### 3. 调试模式
在配置中启用调试模式：
```yaml
router_config:
  debug_mode: true        # 启用详细日志
  log_routing_decisions: true  # 记录路由决策过程
```

---

## 🔧 故障排除

### 常见问题及解决方案

#### 1. 路由器无法启动
**症状**：服务器启动时报错，提示router相关错误
**解决方案**：
```bash
# 检查配置文件语法
go run src/main.go --check-config

# 检查LLM配置是否正确
curl -X GET http://localhost:8080/api/router/llms
```

#### 2. 路由决策不准确
**症状**：请求被路由到错误的LLM
**解决方案**：
- 调整关键词规则和置信度
- 检查意图识别LLM的配置
- 启用调试模式查看决策过程

#### 3. 性能问题
**症状**：响应时间过长
**解决方案**：
- 启用缓存并调整缓存参数
- 优化关键词规则，减少意图识别调用
- 检查各LLM的响应时间

#### 4. LLM调用失败
**症状**：特定LLM无法响应
**解决方案**：
```bash
# 检查LLM配置
curl -X GET http://localhost:8080/api/router/llms/[llm_name]

# 测试LLM连接
curl -X POST http://localhost:8080/api/router/test \
  -d '{"llm_name": "code_llm", "message": "test"}'
```

---

## ❓ FAQ常见问题

### Q1: 如何添加新的专用LLM？
**A**: 有两种方式：
1. **配置文件方式**：在`config.yaml`中添加新的LLM配置
2. **API方式**：使用REST API动态添加
```bash
curl -X POST http://localhost:8080/api/router/llms \
  -H "Content-Type: application/json" \
  -d '{"name": "new_llm", "config": {...}}'
```

### Q2: 路由策略可以动态修改吗？
**A**: 目前路由策略需要通过配置文件修改并重启服务器。动态修改功能在开发计划中。

### Q3: 如何处理多语言路由？
**A**: 在关键词规则中添加多语言关键词：
```yaml
keyword_rules:
  - keywords: ["weather", "天气", "날씨"]
    target_llm: internet_llm
    confidence: 0.8
```

### Q4: 缓存会影响实时性吗？
**A**: 可以通过调整`cache_ttl`参数控制缓存时间，或者为特定类型的请求禁用缓存。

### Q5: 如何监控成本？
**A**: 通过路由统计API可以查看各LLM的调用次数，结合各模型的定价计算成本。

### Q6: 支持哪些LLM提供商？
**A**: 目前支持OpenAI兼容的API，包括：
- OpenAI GPT系列
- 阿里云通义千问
- 智谱GLM系列
- 本地部署的开源模型（通过OpenAI兼容接口）

---

## 📞 技术支持

如果遇到问题，请按以下顺序寻求帮助：

1. **查看日志**：`src/logs/server.log`
2. **检查配置**：对比`config-router-example.yaml`
3. **API调试**：使用提供的curl命令测试
4. **GitHub Issues**：提交详细的问题描述和日志

---

**文档版本**: v1.0  
**最后更新**: 2025-01-27  
**适用版本**: ws_asr v2.0+