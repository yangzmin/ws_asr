package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"xiaozhi-server-go/src/core/providers"
	"xiaozhi-server-go/src/core/types"
)

// ServiceAdapterInterface 服务适配器接口
type ServiceAdapterInterface interface {
	EventHandler
	GetServiceType() string
	GetProvider() interface{}
	UpdateStats(latency time.Duration, success bool)
}

// ASRAdapter ASR服务适配器
type ASRAdapter struct {
	provider providers.ASRProvider
	adapter  *ServiceAdapter
	mutex    sync.RWMutex
}

// NewASRAdapter 创建ASR适配器
func NewASRAdapter(provider providers.ASRProvider) *ASRAdapter {
	return &ASRAdapter{
		provider: provider,
		adapter: &ServiceAdapter{
			Name:      "asr",
			Type:      "asr",
			Status:    "active",
			LastCheck: time.Now(),
		},
	}
}

// Handle 处理ASR事件
func (a *ASRAdapter) Handle(msg *EventMessage) error {
	startTime := time.Now()

	// 解析音频数据 - 支持两种格式
	var audioData []byte
	var requestData struct {
		AudioData []byte `json:"audio_data"`
	}

	// 尝试解析为结构体格式
	if err := json.Unmarshal(msg.Payload, &requestData); err == nil && len(requestData.AudioData) > 0 {
		audioData = requestData.AudioData
	} else {
		// 尝试直接解析为字节数组
		if err := json.Unmarshal(msg.Payload, &audioData); err != nil {
			a.UpdateStats(time.Since(startTime), false)
			return fmt.Errorf("failed to unmarshal audio data: %w", err)
		}
	}

	// 调用ASR服务
	ctx := context.Background()
	if timeout, ok := msg.Metadata["timeout"].(time.Duration); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	result, err := a.provider.Transcribe(ctx, audioData)
	if err != nil {
		a.UpdateStats(time.Since(startTime), false)
		return fmt.Errorf("ASR transcription failed: %w", err)
	}

	// 更新消息结果
	msg.Metadata["result"] = result
	a.UpdateStats(time.Since(startTime), true)

	return nil
}

// GetType 获取事件类型
func (a *ASRAdapter) GetType() string {
	return "asr"
}

// GetStatus 获取状态
func (a *ASRAdapter) GetStatus() string {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.adapter.Status
}

// GetServiceType 获取服务类型
func (a *ASRAdapter) GetServiceType() string {
	return "asr"
}

// GetProvider 获取提供者
func (a *ASRAdapter) GetProvider() interface{} {
	return a.provider
}

// UpdateStats 更新统计信息
func (a *ASRAdapter) UpdateStats(latency time.Duration, success bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.adapter.AvgLatency = (a.adapter.AvgLatency + latency) / 2
	a.adapter.LastCheck = time.Now()

	if !success {
		a.adapter.ErrorCount++
	}
}

// LLMAdapter LLM服务适配器
type LLMAdapter struct {
	provider types.LLMProvider
	adapter  *ServiceAdapter
	mutex    sync.RWMutex
}

// NewLLMAdapter 创建LLM适配器
func NewLLMAdapter(provider types.LLMProvider) *LLMAdapter {
	return &LLMAdapter{
		provider: provider,
		adapter: &ServiceAdapter{
			Name:      "llm",
			Type:      "llm",
			Status:    "active",
			LastCheck: time.Now(),
		},
	}
}

// Handle 处理LLM事件
func (l *LLMAdapter) Handle(msg *EventMessage) error {
	startTime := time.Now()

	// 解析消息数据
	var requestData struct {
		Messages []types.Message `json:"messages"`
		Tools    []interface{}   `json:"tools,omitempty"`
	}

	if err := json.Unmarshal(msg.Payload, &requestData); err != nil {
		l.UpdateStats(time.Since(startTime), false)
		return fmt.Errorf("failed to unmarshal LLM request: %w", err)
	}

	// 调用LLM服务
	ctx := context.Background()
	if timeout, ok := msg.Metadata["timeout"].(time.Duration); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// 根据是否有工具调用选择不同的方法
	fmt.Println("requestData.Tools", requestData.Tools)
	if len(requestData.Tools) > 0 {
		// 有工具调用的情况
		responseChan, err := l.provider.ResponseWithFunctions(
			ctx,
			msg.SessionID,
			requestData.Messages,
			nil, // 需要转换工具格式
		)
		fmt.Println("responseChan111", responseChan)
		if err != nil {
			l.UpdateStats(time.Since(startTime), false)
			return fmt.Errorf("LLM response with functions failed: %w", err)
		}

		// 收集响应
		var responses []types.Response
		for response := range responseChan {
			responses = append(responses, response)
		}
		msg.Metadata["responses"] = responses
	} else {
		// 普通对话
		responseChan, err := l.provider.Response(ctx, msg.SessionID, requestData.Messages)
		fmt.Println("responseChan222", responseChan)
		if err != nil {
			l.UpdateStats(time.Since(startTime), false)
			return fmt.Errorf("LLM response failed: %w", err)
		}

		// 收集响应
		var result string
		for chunk := range responseChan {
			result += chunk
		}
		msg.Metadata["result"] = result
	}

	l.UpdateStats(time.Since(startTime), true)
	return nil
}

// GetType 获取事件类型
func (l *LLMAdapter) GetType() string {
	return "llm"
}

// GetStatus 获取状态
func (l *LLMAdapter) GetStatus() string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.adapter.Status
}

// GetServiceType 获取服务类型
func (l *LLMAdapter) GetServiceType() string {
	return "llm"
}

// GetProvider 获取提供者
func (l *LLMAdapter) GetProvider() interface{} {
	return l.provider
}

// UpdateStats 更新统计信息
func (l *LLMAdapter) UpdateStats(latency time.Duration, success bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.adapter.AvgLatency = (l.adapter.AvgLatency + latency) / 2
	l.adapter.LastCheck = time.Now()

	if !success {
		l.adapter.ErrorCount++
	}
}

// TTSAdapter TTS服务适配器
type TTSAdapter struct {
	provider providers.TTSProvider
	adapter  *ServiceAdapter
	mutex    sync.RWMutex
}

// NewTTSAdapter 创建TTS适配器
func NewTTSAdapter(provider providers.TTSProvider) *TTSAdapter {
	return &TTSAdapter{
		provider: provider,
		adapter: &ServiceAdapter{
			Name:      "tts",
			Type:      "tts",
			Status:    "active",
			LastCheck: time.Now(),
		},
	}
}

// Handle 处理TTS事件
func (t *TTSAdapter) Handle(msg *EventMessage) error {
	startTime := time.Now()

	// 解析文本数据
	var requestData struct {
		Text  string `json:"text"`
		Voice string `json:"voice,omitempty"`
	}

	if err := json.Unmarshal(msg.Payload, &requestData); err != nil {
		t.UpdateStats(time.Since(startTime), false)
		return fmt.Errorf("failed to unmarshal TTS request: %w", err)
	}

	// 设置语音（如果指定）
	if requestData.Voice != "" {
		if err := t.provider.SetVoice(requestData.Voice); err != nil {
			t.UpdateStats(time.Since(startTime), false)
			return fmt.Errorf("failed to set voice: %w", err)
		}
	}

	// 调用TTS服务
	audioPath, err := t.provider.ToTTS(requestData.Text)
	if err != nil {
		t.UpdateStats(time.Since(startTime), false)
		return fmt.Errorf("TTS synthesis failed: %w", err)
	}

	// 更新消息结果
	msg.Metadata["audio_path"] = audioPath
	t.UpdateStats(time.Since(startTime), true)

	return nil
}

// GetType 获取事件类型
func (t *TTSAdapter) GetType() string {
	return "tts"
}

// GetStatus 获取状态
func (t *TTSAdapter) GetStatus() string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.adapter.Status
}

// GetServiceType 获取服务类型
func (t *TTSAdapter) GetServiceType() string {
	return "tts"
}

// GetProvider 获取提供者
func (t *TTSAdapter) GetProvider() interface{} {
	return t.provider
}

// UpdateStats 更新统计信息
func (t *TTSAdapter) UpdateStats(latency time.Duration, success bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.adapter.AvgLatency = (t.adapter.AvgLatency + latency) / 2
	t.adapter.LastCheck = time.Now()

	if !success {
		t.adapter.ErrorCount++
	}
}

// MCPAdapter MCP服务适配器
type MCPAdapter struct {
	// MCP相关的提供者接口需要根据实际实现定义
	adapter *ServiceAdapter
	mutex   sync.RWMutex
}

// NewMCPAdapter 创建MCP适配器
func NewMCPAdapter() *MCPAdapter {
	return &MCPAdapter{
		adapter: &ServiceAdapter{
			Name:      "mcp",
			Type:      "mcp",
			Status:    "active",
			LastCheck: time.Now(),
		},
	}
}

// Handle 处理MCP事件
func (m *MCPAdapter) Handle(msg *EventMessage) error {
	startTime := time.Now()

	// MCP处理逻辑需要根据实际MCP实现来完成
	// 这里提供基础框架

	// 解析MCP请求数据
	var requestData map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &requestData); err != nil {
		m.UpdateStats(time.Since(startTime), false)
		return fmt.Errorf("failed to unmarshal MCP request: %w", err)
	}

	// TODO: 实现具体的MCP调用逻辑
	// 这里需要根据实际的MCP接口来实现

	msg.Metadata["result"] = "mcp_processed"
	m.UpdateStats(time.Since(startTime), true)

	return nil
}

// GetType 获取事件类型
func (m *MCPAdapter) GetType() string {
	return "mcp"
}

// GetStatus 获取状态
func (m *MCPAdapter) GetStatus() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.adapter.Status
}

// GetServiceType 获取服务类型
func (m *MCPAdapter) GetServiceType() string {
	return "mcp"
}

// GetProvider 获取提供者
func (m *MCPAdapter) GetProvider() interface{} {
	return nil // MCP可能不需要传统的提供者模式
}

// UpdateStats 更新统计信息
func (m *MCPAdapter) UpdateStats(latency time.Duration, success bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.adapter.AvgLatency = (m.adapter.AvgLatency + latency) / 2
	m.adapter.LastCheck = time.Now()

	if !success {
		m.adapter.ErrorCount++
	}
}
