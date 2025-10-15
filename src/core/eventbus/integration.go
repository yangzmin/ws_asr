package eventbus

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"xiaozhi-server-go/src/core/providers"
	"xiaozhi-server-go/src/core/types"
)

// EventBusManager 事件总线管理器 - 用于集成到现有架构
type EventBusManager struct {
	eventBus       *EventBus
	sessionManager *SessionManager
	monitor        *MonitoringService
	adapters       map[string]ServiceAdapterInterface
	mutex          sync.RWMutex
	config         *IntegrationConfig
}

// IntegrationConfig 集成配置
type IntegrationConfig struct {
	EnableEventBus     bool `json:"enable_event_bus"`
	EnableSessionMgr   bool `json:"enable_session_mgr"`
	EnableMonitoring   bool `json:"enable_monitoring"`
	MonitoringPort     int  `json:"monitoring_port"`
	MigrationMode      bool `json:"migration_mode"`      // 迁移模式，逐步替换现有调用
	FallbackToOriginal bool `json:"fallback_to_original"` // 失败时回退到原始实现
}

// DefaultIntegrationConfig 默认集成配置
func DefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		EnableEventBus:     true,
		EnableSessionMgr:   true,
		EnableMonitoring:   true,
		MonitoringPort:     8080,
		MigrationMode:      true,
		FallbackToOriginal: true,
	}
}

// NewEventBusManager 创建事件总线管理器
func NewEventBusManager(config *IntegrationConfig) *EventBusManager {
	if config == nil {
		config = DefaultIntegrationConfig()
	}

	manager := &EventBusManager{
		adapters: make(map[string]ServiceAdapterInterface),
		config:   config,
	}

	// 初始化组件
	if config.EnableEventBus {
		manager.eventBus = NewEventBus(DefaultEventBusConfig())
	}

	if config.EnableSessionMgr {
		manager.sessionManager = NewSessionManager(DefaultSessionConfig())
	}

	if config.EnableMonitoring && manager.eventBus != nil {
		manager.monitor = NewMonitoringService(
			manager.eventBus,
			manager.sessionManager,
			config.MonitoringPort + 1, // 使用不同的端口避免冲突
		)
	}

	return manager
}

// RegisterProviders 注册现有提供者到事件总线
func (ebm *EventBusManager) RegisterProviders(
	asrProvider providers.ASRProvider,
	llmProvider types.LLMProvider,
	ttsProvider providers.TTSProvider,
) error {
	if ebm.eventBus == nil {
		return fmt.Errorf("event bus not initialized")
	}

	ebm.mutex.Lock()
	defer ebm.mutex.Unlock()

	// 注册ASR适配器
	if asrProvider != nil {
		asrAdapter := NewASRAdapter(asrProvider)
		if err := ebm.eventBus.RegisterHandler("asr", asrAdapter); err != nil {
			return fmt.Errorf("failed to register ASR adapter: %w", err)
		}
		ebm.adapters["asr"] = asrAdapter
	}

	// 注册LLM适配器
	if llmProvider != nil {
		llmAdapter := NewLLMAdapter(llmProvider)
		if err := ebm.eventBus.RegisterHandler("llm", llmAdapter); err != nil {
			return fmt.Errorf("failed to register LLM adapter: %w", err)
		}
		ebm.adapters["llm"] = llmAdapter
	}

	// 注册TTS适配器
	if ttsProvider != nil {
		ttsAdapter := NewTTSAdapter(ttsProvider)
		if err := ebm.eventBus.RegisterHandler("tts", ttsAdapter); err != nil {
			return fmt.Errorf("failed to register TTS adapter: %w", err)
		}
		ebm.adapters["tts"] = ttsAdapter
	}

	// 注册MCP适配器
	mcpAdapter := NewMCPAdapter()
	if err := ebm.eventBus.RegisterHandler("mcp", mcpAdapter); err != nil {
		return fmt.Errorf("failed to register MCP adapter: %w", err)
	}
	ebm.adapters["mcp"] = mcpAdapter

	return nil
}

// Start 启动事件总线管理器
func (ebm *EventBusManager) Start() error {
	if ebm.monitor != nil {
		if err := ebm.monitor.Start(); err != nil {
			return fmt.Errorf("failed to start monitoring service: %w", err)
		}
	}
	return nil
}

// Stop 停止事件总线管理器
func (ebm *EventBusManager) Stop() error {
	if ebm.monitor != nil {
		if err := ebm.monitor.Stop(); err != nil {
			return fmt.Errorf("failed to stop monitoring service: %w", err)
		}
	}

	if ebm.eventBus != nil {
		ebm.eventBus.Shutdown()
	}

	if ebm.sessionManager != nil {
		ebm.sessionManager.Shutdown()
	}

	return nil
}

// GetMonitor 获取监控服务
func (ebm *EventBusManager) GetMonitor() *MonitoringService {
	ebm.mutex.RLock()
	defer ebm.mutex.RUnlock()
	return ebm.monitor
}

// GetEventBus 获取事件总线
func (ebm *EventBusManager) GetEventBus() *EventBus {
	ebm.mutex.RLock()
	defer ebm.mutex.RUnlock()
	return ebm.eventBus
}

// GetAdapters 获取所有适配器
func (ebm *EventBusManager) GetAdapters() map[string]ServiceAdapterInterface {
	ebm.mutex.RLock()
	defer ebm.mutex.RUnlock()
	
	// 返回副本以避免并发问题
	adapters := make(map[string]ServiceAdapterInterface)
	for k, v := range ebm.adapters {
		adapters[k] = v
	}
	return adapters
}

// ConnectionHandlerIntegration 连接处理器集成接口
type ConnectionHandlerIntegration struct {
	manager         *EventBusManager
	sessionID       string
	originalHandler interface{} // 原始处理器的引用
	mutex           sync.RWMutex
}

// NewConnectionHandlerIntegration 创建连接处理器集成
func NewConnectionHandlerIntegration(manager *EventBusManager, sessionID string, originalHandler interface{}) *ConnectionHandlerIntegration {
	return &ConnectionHandlerIntegration{
		manager:         manager,
		sessionID:       sessionID,
		originalHandler: originalHandler,
	}
}

// ProcessASR 处理ASR请求 - 事件总线版本
func (chi *ConnectionHandlerIntegration) ProcessASR(audioData []byte, timeout time.Duration) (string, error) {
	if !chi.manager.config.EnableEventBus || chi.manager.eventBus == nil {
		return "", fmt.Errorf("event bus not available")
	}

	// 准备请求数据
	payload, err := json.Marshal(audioData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal audio data: %w", err)
	}

	metadata := map[string]interface{}{
		"session_id": chi.sessionID,
		"timeout":    timeout,
		"timestamp":  time.Now(),
	}

	// 发布ASR事件
	if err := chi.manager.eventBus.Publish("asr", payload, metadata); err != nil {
		if chi.manager.config.FallbackToOriginal {
			// 回退到原始实现
			return chi.fallbackASR(audioData, timeout)
		}
		return "", fmt.Errorf("failed to publish ASR event: %w", err)
	}

	// 等待处理结果 (简化版本，实际应该使用回调或通道)
	// 这里需要根据实际需求实现结果获取机制
	time.Sleep(time.Millisecond * 100) // 临时等待

	return "asr_result_placeholder", nil
}

// ProcessLLM 处理LLM请求 - 事件总线版本
func (chi *ConnectionHandlerIntegration) ProcessLLM(messages []types.Message, tools []interface{}) (interface{}, error) {
	if !chi.manager.config.EnableEventBus || chi.manager.eventBus == nil {
		return nil, fmt.Errorf("event bus not available")
	}

	// 准备请求数据
	requestData := struct {
		Messages []types.Message `json:"messages"`
		Tools    []interface{}   `json:"tools,omitempty"`
	}{
		Messages: messages,
		Tools:    tools,
	}

	payload, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LLM request: %w", err)
	}

	metadata := map[string]interface{}{
		"session_id": chi.sessionID,
		"timestamp":  time.Now(),
	}

	// 发布LLM事件
	if err := chi.manager.eventBus.Publish("llm", payload, metadata); err != nil {
		if chi.manager.config.FallbackToOriginal {
			// 回退到原始实现
			return chi.fallbackLLM(messages, tools)
		}
		return nil, fmt.Errorf("failed to publish LLM event: %w", err)
	}

	// 等待处理结果
	time.Sleep(time.Millisecond * 100) // 临时等待

	return "llm_result_placeholder", nil
}

// ProcessTTS 处理TTS请求 - 事件总线版本
func (chi *ConnectionHandlerIntegration) ProcessTTS(text string, voice string) (string, error) {
	if !chi.manager.config.EnableEventBus || chi.manager.eventBus == nil {
		return "", fmt.Errorf("event bus not available")
	}

	// 准备请求数据
	requestData := struct {
		Text  string `json:"text"`
		Voice string `json:"voice,omitempty"`
	}{
		Text:  text,
		Voice: voice,
	}

	payload, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal TTS request: %w", err)
	}

	metadata := map[string]interface{}{
		"session_id": chi.sessionID,
		"timestamp":  time.Now(),
	}

	// 发布TTS事件
	if err := chi.manager.eventBus.Publish("tts", payload, metadata); err != nil {
		if chi.manager.config.FallbackToOriginal {
			// 回退到原始实现
			return chi.fallbackTTS(text, voice)
		}
		return "", fmt.Errorf("failed to publish TTS event: %w", err)
	}

	// 等待处理结果
	time.Sleep(time.Millisecond * 100) // 临时等待

	return "audio_path_placeholder", nil
}

// ProcessMCP 处理MCP请求 - 事件总线版本
func (chi *ConnectionHandlerIntegration) ProcessMCP(request map[string]interface{}) (interface{}, error) {
	if !chi.manager.config.EnableEventBus || chi.manager.eventBus == nil {
		return nil, fmt.Errorf("event bus not available")
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP request: %w", err)
	}

	metadata := map[string]interface{}{
		"session_id": chi.sessionID,
		"timestamp":  time.Now(),
	}

	// 发布MCP事件
	if err := chi.manager.eventBus.Publish("mcp", payload, metadata); err != nil {
		if chi.manager.config.FallbackToOriginal {
			// 回退到原始实现
			return chi.fallbackMCP(request)
		}
		return nil, fmt.Errorf("failed to publish MCP event: %w", err)
	}

	// 等待处理结果
	time.Sleep(time.Millisecond * 100) // 临时等待

	return "mcp_result_placeholder", nil
}

// 回退方法 - 调用原始实现
func (chi *ConnectionHandlerIntegration) fallbackASR(audioData []byte, timeout time.Duration) (string, error) {
	// 这里需要调用原始的ASR处理逻辑
	// 具体实现需要根据原始ConnectionHandler的结构来完成
	return "fallback_asr_result", nil
}

func (chi *ConnectionHandlerIntegration) fallbackLLM(messages []types.Message, tools []interface{}) (interface{}, error) {
	// 这里需要调用原始的LLM处理逻辑
	return "fallback_llm_result", nil
}

func (chi *ConnectionHandlerIntegration) fallbackTTS(text string, voice string) (string, error) {
	// 这里需要调用原始的TTS处理逻辑
	return "fallback_audio_path", nil
}

func (chi *ConnectionHandlerIntegration) fallbackMCP(request map[string]interface{}) (interface{}, error) {
	// 这里需要调用原始的MCP处理逻辑
	return "fallback_mcp_result", nil
}

// GetSessionState 获取会话状态
func (chi *ConnectionHandlerIntegration) GetSessionState() (*SessionState, error) {
	if chi.manager.sessionManager == nil {
		return nil, fmt.Errorf("session manager not available")
	}

	return chi.manager.sessionManager.GetSession(chi.sessionID)
}

// UpdateSessionMetadata 更新会话元数据
func (chi *ConnectionHandlerIntegration) UpdateSessionMetadata(metadata map[string]interface{}) error {
	if chi.manager.sessionManager == nil {
		return fmt.Errorf("session manager not available")
	}

	return chi.manager.sessionManager.UpdateSession(chi.sessionID, metadata)
}

// GetSystemStats 获取系统统计信息
func (chi *ConnectionHandlerIntegration) GetSystemStats() (*SystemMetrics, error) {
	if chi.manager.monitor == nil {
		return nil, fmt.Errorf("monitoring service not available")
	}

	return chi.manager.monitor.GetMetrics(), nil
}

// IsEventBusEnabled 检查事件总线是否启用
func (chi *ConnectionHandlerIntegration) IsEventBusEnabled() bool {
	return chi.manager.config.EnableEventBus && chi.manager.eventBus != nil
}

// IsMigrationMode 检查是否为迁移模式
func (chi *ConnectionHandlerIntegration) IsMigrationMode() bool {
	return chi.manager.config.MigrationMode
}