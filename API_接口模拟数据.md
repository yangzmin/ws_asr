# AI配置管理API接口模拟数据

## 接口列表
基于 `/h:/project/go/ws_asr/src/handlers/ai_config_handler.go#L39-45` 的API接口

所有接口都需要在请求头中包含JWT认证token：
```
Authorization: Bearer <your_jwt_token>
```

---

## 1. GET /api/ai-configs - 获取用户配置列表

**用途**: 获取当前用户的所有AI配置，支持按配置类型过滤

**请求参数** (Query Parameters):
```json
{
  "config_type": "llm"  // 可选，配置类型过滤 ("llm" | "function_call")
}
```

**请求示例**:
```bash
# 获取所有配置
GET /api/ai-configs

# 获取LLM类型配置
GET /api/ai-configs?config_type=llm

# 获取函数调用类型配置
GET /api/ai-configs?config_type=function_call
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "configs": [
      {
        "id": 1,
        "user_id": "user123",
        "config_name": "GPT-4配置",
        "config_type": "llm",
        "llm_type": "openai",
        "model_name": "gpt-4",
        "base_url": "https://api.openai.com/v1",
        "max_tokens": 4096,
        "temperature": 0.7,
        "is_active": true,
        "priority": 10,
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 1
  }
}
```

---

## 2. POST /api/ai-configs - 创建AI配置

**用途**: 创建新的AI配置，支持LLM配置和函数调用配置两种类型

**请求体** (JSON):

### LLM配置示例:
```json
{
  "config_name": "我的GPT-4配置",           // 必填，配置名称
  "config_type": "llm",                    // 必填，配置类型 ("llm" | "function_call")
  "llm_type": "openai",                    // LLM提供商类型 ("qwen" | "chatglm" | "ollama" | "coze" | "openai")
  "model_name": "gpt-4-turbo-preview",     // 模型名称
  "api_key": "sk-xxxxxxxxxxxxxxxx",        // API密钥
  "base_url": "https://api.openai.com/v1", // API基础URL
  "max_tokens": 4096,                      // 最大token数
  "temperature": 0.7,                      // 温度参数 (0.0-2.0)
  "priority": 10                           // 优先级，数字越大优先级越高
}
```

### 函数调用配置示例:
```json
{
  "config_name": "天气查询函数",             // 必填，配置名称
  "config_type": "function_call",          // 必填，配置类型
  "function_name": "get_weather",          // 函数名称
  "description": "获取指定城市的天气信息",    // 函数描述
  "mcp_server_url": "http://localhost:3000/mcp", // MCP服务器URL
  "parameters": {                          // 函数参数定义 (JSON Schema格式)
    "type": "object",
    "properties": {
      "city": {
        "type": "string",
        "description": "城市名称"
      },
      "unit": {
        "type": "string",
        "enum": ["celsius", "fahrenheit"],
        "description": "温度单位"
      }
    },
    "required": ["city"]
  },
  "priority": 5
}
```

---

## 3. GET /api/ai-configs/:id - 获取配置详情

**用途**: 根据配置ID获取特定配置的详细信息

**路径参数**:
- `id`: 配置ID (整数)

**请求示例**:
```bash
GET /api/ai-configs/1
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "config": {
      "id": 1,
      "user_id": "user123",
      "config_name": "GPT-4配置",
      "config_type": "llm",
      "llm_type": "openai",
      "model_name": "gpt-4",
      "api_key": "sk-xxxxxxxxxxxxxxxx",  // 注意：实际返回时可能会隐藏敏感信息
      "base_url": "https://api.openai.com/v1",
      "max_tokens": 4096,
      "temperature": 0.7,
      "is_active": true,
      "priority": 10,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

---

## 4. PUT /api/ai-configs/:id - 更新配置

**用途**: 更新指定ID的AI配置，支持部分字段更新

**路径参数**:
- `id`: 配置ID (整数)

**请求体** (JSON):

### 更新LLM配置示例:
```json
{
  "config_name": "更新后的GPT-4配置",       // 可选，配置名称
  "llm_type": "openai",                    // 可选，LLM类型
  "model_name": "gpt-4-turbo",             // 可选，模型名称
  "api_key": "sk-new-api-key",             // 可选，新的API密钥
  "base_url": "https://api.openai.com/v1", // 可选，API基础URL
  "max_tokens": 8192,                      // 可选，最大token数
  "temperature": 0.8,                      // 可选，温度参数
  "priority": 15,                          // 可选，优先级
  "is_active": true                        // 可选，是否激活
}
```

### 更新函数调用配置示例:
```json
{
  "function_name": "get_weather_v2",       // 可选，函数名称
  "description": "获取天气信息的增强版本",   // 可选，函数描述
  "mcp_server_url": "http://localhost:3001/mcp", // 可选，MCP服务器URL
  "parameters": {                          // 可选，更新参数定义
    "type": "object",
    "properties": {
      "city": {
        "type": "string",
        "description": "城市名称"
      },
      "unit": {
        "type": "string",
        "enum": ["celsius", "fahrenheit"],
        "default": "celsius",
        "description": "温度单位"
      },
      "forecast_days": {
        "type": "integer",
        "minimum": 1,
        "maximum": 7,
        "default": 1,
        "description": "预报天数"
      }
    },
    "required": ["city"]
  },
  "priority": 8,
  "is_active": false
}
```

---

## 5. DELETE /api/ai-configs/:id - 删除配置

**用途**: 删除指定ID的AI配置

**路径参数**:
- `id`: 配置ID (整数)

**请求示例**:
```bash
DELETE /api/ai-configs/1
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "配置删除成功"
  }
}
```

---

## 6. PATCH /api/ai-configs/:id/toggle - 切换配置状态

**用途**: 启用或禁用指定的AI配置

**路径参数**:
- `id`: 配置ID (整数)

**请求体** (JSON):
```json
{
  "is_active": true  // 必填，true=启用，false=禁用
}
```

**请求示例**:
```bash
# 启用配置
PATCH /api/ai-configs/1/toggle
Content-Type: application/json

{
  "is_active": true
}

# 禁用配置
PATCH /api/ai-configs/1/toggle
Content-Type: application/json

{
  "is_active": false
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "配置状态切换成功",
    "is_active": true
  }
}
```

---

## 7. PATCH /api/ai-configs/:id/priority - 设置配置优先级

**用途**: 设置指定AI配置的优先级，数字越大优先级越高

**路径参数**:
- `id`: 配置ID (整数)

**请求体** (JSON):
```json
{
  "priority": 20  // 必填，优先级数值 (整数)
}
```

**请求示例**:
```bash
PATCH /api/ai-configs/1/priority
Content-Type: application/json

{
  "priority": 20
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "优先级设置成功",
    "priority": 20
  }
}
```

---

## 错误响应格式

所有接口在出错时都会返回统一的错误格式：

```json
{
  "code": 400,  // HTTP状态码
  "message": "请求参数格式错误",  // 错误消息
  "error": "具体的错误详情"  // 可选，详细错误信息
}
```

常见错误码：
- `400`: 请求参数错误
- `401`: 未授权（token无效或过期）
- `404`: 资源不存在
- `500`: 服务器内部错误

---

## 注意事项

1. **认证要求**: 所有接口都需要在请求头中包含有效的JWT token
2. **配置类型**: 支持两种配置类型：
   - `llm`: 大语言模型配置
   - `function_call`: 函数调用配置
3. **优先级**: 数字越大优先级越高，用于多配置场景下的选择
4. **参数验证**: 创建和更新接口会验证必填字段和数据格式
5. **敏感信息**: API密钥等敏感信息在返回时可能会被隐藏或脱敏处理