package eventbus

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"xiaozhi-server-go/src/core/providers"
	"xiaozhi-server-go/src/core/types"
	"github.com/sashabaranov/go-openai"
)

// TestEventBusIntegration 测试事件总线集成
func TestEventBusIntegration() error {
	fmt.Println("开始测试事件总线集成...")
	
	// 1. 测试事件总线基本功能
	if err := testEventBusBasics(); err != nil {
		return fmt.Errorf("事件总线基本功能测试失败: %v", err)
	}
	
	// 2. 测试服务适配器
	if err := testServiceAdapters(); err != nil {
		return fmt.Errorf("服务适配器测试失败: %v", err)
	}
	
	// 3. 测试会话管理器
	if err := testSessionManager(); err != nil {
		return fmt.Errorf("会话管理器测试失败: %v", err)
	}
	
	// 4. 测试监控服务
	if err := testMonitoringService(); err != nil {
		return fmt.Errorf("监控服务测试失败: %v", err)
	}
	
	// 5. 测试集成管理器
	if err := testIntegrationManager(); err != nil {
		return fmt.Errorf("集成管理器测试失败: %v", err)
	}
	
	// 6. 测试并发性能
	if err := testConcurrentPerformance(); err != nil {
		return fmt.Errorf("并发性能测试失败: %v", err)
	}
	
	fmt.Println("所有测试通过！")
	return nil
}

// testEventBusBasics 测试事件总线基本功能
func testEventBusBasics() error {
	fmt.Println("  测试事件总线基本功能...")
	
	eventBus := NewEventBus(DefaultEventBusConfig())
	defer eventBus.Shutdown()
	
	// 测试处理器注册
	testHandler := &TestEventHandler{processed: make(chan bool, 1)}
	if err := eventBus.RegisterHandler("test", testHandler); err != nil {
		return fmt.Errorf("注册处理器失败: %v", err)
	}
	
	// 测试消息发布
	payload := []byte(`{"test": "data"}`)
	metadata := map[string]interface{}{
		"session_id": "test-session",
		"priority":   5,
	}
	
	if err := eventBus.Publish("test", payload, metadata); err != nil {
		return fmt.Errorf("发布消息失败: %v", err)
	}
	
	// 等待处理完成
	select {
	case <-testHandler.processed:
		fmt.Println("    ✓ 消息处理成功")
	case <-time.After(time.Second * 2):
		return fmt.Errorf("消息处理超时")
	}
	
	// 测试统计信息
	stats := eventBus.GetStats()
	if stats.TotalProcessed == 0 {
		return fmt.Errorf("统计信息不正确")
	}
	
	fmt.Printf("    ✓ 统计信息正确 (处理数: %d)\n", stats.TotalProcessed)
	
	// 测试处理器注销
	eventBus.UnregisterHandler("test")
	
	fmt.Println("    ✓ 处理器注销成功")
	return nil
}

// testServiceAdapters 测试服务适配器
func testServiceAdapters() error {
	fmt.Println("  测试服务适配器...")
	
	// 创建模拟提供者
	mockASR := &MockASRProvider{}
	mockLLM := &MockLLMProvider{}
	mockTTS := &MockTTSProvider{}
	
	// 创建适配器
	asrAdapter := NewASRAdapter(mockASR)
	llmAdapter := NewLLMAdapter(mockLLM)
	ttsAdapter := NewTTSAdapter(mockTTS)
	
	// 测试ASR适配器
	asrMsg := &EventMessage{
		ID:        "asr-test",
		Type:      "asr",
		SessionID: "test-session",
		Payload:   []byte(`{"audio_data": [1,2,3,4,5]}`), // 修改为正确的JSON格式
		Metadata:  make(map[string]interface{}),
	}
	
	if err := asrAdapter.Handle(asrMsg); err != nil {
		return fmt.Errorf("ASR适配器处理失败: %v", err)
	}
	
	if !mockASR.called {
		return fmt.Errorf("ASR提供者未被调用")
	}
	
	fmt.Println("    ✓ ASR适配器测试通过")
	
	// 测试LLM适配器
	llmMsg := &EventMessage{
		ID:        "llm-test",
		Type:      "llm",
		SessionID: "test-session",
		Payload:   []byte(`{"messages": [{"role": "user", "content": "hello"}]}`),
		Metadata:  make(map[string]interface{}),
	}
	
	if err := llmAdapter.Handle(llmMsg); err != nil {
		return fmt.Errorf("LLM适配器处理失败: %v", err)
	}
	
	if !mockLLM.called {
		return fmt.Errorf("LLM提供者未被调用")
	}
	
	fmt.Println("    ✓ LLM适配器测试通过")
	
	// 测试TTS适配器
	ttsMsg := &EventMessage{
		ID:        "tts-test",
		Type:      "tts",
		SessionID: "test-session",
		Payload:   []byte(`{"text": "hello world", "voice": "default"}`),
		Metadata:  make(map[string]interface{}),
	}
	
	if err := ttsAdapter.Handle(ttsMsg); err != nil {
		return fmt.Errorf("TTS适配器处理失败: %v", err)
	}
	
	if !mockTTS.called {
		return fmt.Errorf("TTS提供者未被调用")
	}
	
	fmt.Println("    ✓ TTS适配器测试通过")
	
	return nil
}

// testSessionManager 测试会话管理器
func testSessionManager() error {
	fmt.Println("  测试会话管理器...")
	
	config := DefaultSessionConfig()
	config.MaxSessions = 10
	config.SessionTimeout = time.Minute * 5
	
	manager := NewSessionManager(config)
	defer manager.Shutdown()
	
	// 测试创建会话
	session, err := manager.CreateSession("user-123", map[string]interface{}{
		"client_type": "web",
		"language":    "zh-CN",
	})
	if err != nil {
		return fmt.Errorf("创建会话失败: %v", err)
	}
	
	fmt.Printf("    ✓ 会话创建成功: %s\n", session.ID)
	
	// 测试获取会话
	retrievedSession, err := manager.GetSession(session.ID)
	if err != nil {
		return fmt.Errorf("获取会话失败: %v", err)
	}
	
	if retrievedSession.ID != session.ID {
		return fmt.Errorf("会话ID不匹配")
	}
	
	fmt.Println("    ✓ 会话获取成功")
	
	// 测试更新会话
	if err := manager.UpdateSession(session.ID, map[string]interface{}{
		"last_activity": time.Now(),
		"message_count": 5,
	}); err != nil {
		return fmt.Errorf("更新会话失败: %v", err)
	}
	
	fmt.Println("    ✓ 会话更新成功")
	
	// 测试会话统计
	stats := manager.GetStats()
	if activeSessions, ok := stats["active_sessions"].(int); !ok || activeSessions != 1 {
		return fmt.Errorf("会话统计不正确: 期望1个活跃会话，实际%v", stats["active_sessions"])
	}
	
	fmt.Printf("    ✓ 会话统计正确 (活跃会话: %v)\n", stats["active_sessions"])
	
	// 测试关闭会话
	if err := manager.CloseSession(session.ID); err != nil {
		return fmt.Errorf("关闭会话失败: %v", err)
	}
	
	fmt.Println("    ✓ 会话关闭成功")
	
	return nil
}

// testMonitoringService 测试监控服务
func testMonitoringService() error {
	fmt.Println("  测试监控服务...")
	
	// 创建临时的事件总线和会话管理器用于监控
	eventBus := NewEventBus(DefaultEventBusConfig())
	defer eventBus.Shutdown()
	
	sessionManager := NewSessionManager(DefaultSessionConfig())
	defer sessionManager.Shutdown()
	
	monitor := NewMonitoringService(eventBus, sessionManager, 8083)
	
	// 启动监控服务
	if err := monitor.Start(); err != nil {
		return fmt.Errorf("启动监控服务失败: %v", err)
	}
	defer monitor.Stop()
	
	// 等待服务启动
	time.Sleep(time.Millisecond * 100)
	
	// 测试健康状态
	health := monitor.GetHealthStatus()
	if health.Status != "healthy" {
		return fmt.Errorf("健康状态不正确: %s", health.Status)
	}
	
	fmt.Printf("    ✓ 健康状态正确: %s\n", health.Status)
	
	// 测试系统指标
	metrics := monitor.collectSystemMetrics()
	if metrics.GoVersion == "" {
		return fmt.Errorf("系统指标不完整")
	}
	
	fmt.Printf("    ✓ 系统指标正确 (Go版本: %s)\n", metrics.GoVersion)
	
	return nil
}

// testIntegrationManager 测试集成管理器
func testIntegrationManager() error {
	fmt.Println("  测试集成管理器...")
	
	config := DefaultIntegrationConfig()
	config.MonitoringPort = 8084
	
	manager := NewEventBusManager(config)
	
	// 测试启动
	if err := manager.Start(); err != nil {
		return fmt.Errorf("启动集成管理器失败: %v", err)
	}
	defer manager.Stop()
	
	fmt.Println("    ✓ 集成管理器启动成功")
	
	// 测试注册提供者
	mockASR := &MockASRProvider{}
	mockLLM := &MockLLMProvider{}
	mockTTS := &MockTTSProvider{}
	
	if err := manager.RegisterProviders(mockASR, mockLLM, mockTTS); err != nil {
		return fmt.Errorf("注册提供者失败: %v", err)
	}
	
	fmt.Println("    ✓ 提供者注册成功")
	
	// 测试连接处理器集成
	integration := NewConnectionHandlerIntegration(manager, "test-session", nil)
	
	if !integration.IsEventBusEnabled() {
		return fmt.Errorf("事件总线未启用")
	}
	
	fmt.Println("    ✓ 连接处理器集成成功")
	
	return nil
}

// testConcurrentPerformance 测试并发性能
func testConcurrentPerformance() error {
	fmt.Println("  测试并发性能...")
	
	eventBus := NewEventBus(DefaultEventBusConfig())
	defer eventBus.Shutdown()
	
	// 注册测试处理器
	testHandler := &ConcurrentTestHandler{
		processed: make(map[string]bool),
		mutex:     sync.RWMutex{},
	}
	
	if err := eventBus.RegisterHandler("concurrent", testHandler); err != nil {
		return fmt.Errorf("注册处理器失败: %v", err)
	}
	
	// 并发发送消息
	const numMessages = 100
	const numWorkers = 10
	
	var wg sync.WaitGroup
	errChan := make(chan error, numWorkers)
	
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < numMessages/numWorkers; j++ {
				msgID := fmt.Sprintf("worker-%d-msg-%d", workerID, j)
				payload := []byte(fmt.Sprintf(`{"id": "%s", "data": "test"}`, msgID))
				metadata := map[string]interface{}{
					"session_id": "perf-test",
					"worker_id":  workerID,
					"msg_id":     msgID,
				}
				
				if err := eventBus.Publish("concurrent", payload, metadata); err != nil {
					errChan <- fmt.Errorf("发布消息失败: %v", err)
					return
				}
			}
		}(i)
	}
	
	wg.Wait()
	close(errChan)
	
	// 检查错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	
	// 等待所有消息处理完成
	time.Sleep(time.Second * 2)
	
	// 检查处理结果
	testHandler.mutex.RLock()
	processedCount := len(testHandler.processed)
	testHandler.mutex.RUnlock()
	
	if processedCount != numMessages {
		return fmt.Errorf("并发处理不完整: 期望%d条消息，实际处理%d条", numMessages, processedCount)
	}
	
	fmt.Printf("    ✓ 并发性能测试通过 (处理%d条消息)\n", processedCount)
	
	// 检查统计信息
	stats := eventBus.GetStats()
	fmt.Printf("    ✓ 处理速率: %.2f msg/s\n", stats.ProcessingRate)
	fmt.Printf("    ✓ 错误率: %.2f%%\n", float64(stats.ErrorCount)/float64(stats.TotalProcessed)*100)
	
	return nil
}

// 测试用的模拟类型和处理器

type TestEventHandler struct {
	processed chan bool
}

func (t *TestEventHandler) Handle(msg *EventMessage) error {
	// 模拟处理时间
	time.Sleep(time.Millisecond * 10)
	t.processed <- true
	return nil
}

func (t *TestEventHandler) GetType() string {
	return "test"
}

func (t *TestEventHandler) GetStatus() string {
	return "active"
}

type ConcurrentTestHandler struct {
	processed map[string]bool
	mutex     sync.RWMutex
}

func (c *ConcurrentTestHandler) Handle(msg *EventMessage) error {
	// 模拟处理时间
	time.Sleep(time.Millisecond * 5)
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if msgID, ok := msg.Metadata["msg_id"].(string); ok {
		c.processed[msgID] = true
	}
	
	return nil
}

func (c *ConcurrentTestHandler) GetType() string {
	return "concurrent"
}

func (c *ConcurrentTestHandler) GetStatus() string {
	return "active"
}

// 模拟提供者

type MockASRProvider struct {
	called bool
}

func (m *MockASRProvider) Initialize() error {
	return nil
}

func (m *MockASRProvider) Cleanup() error {
	return nil
}

func (m *MockASRProvider) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	m.called = true
	return "mock transcription", nil
}

func (m *MockASRProvider) AddAudio(data []byte) error {
	m.called = true
	return nil
}

func (m *MockASRProvider) SetListener(listener providers.AsrEventListener) {
	// Mock implementation
}

func (m *MockASRProvider) Reset() error {
	return nil
}

func (m *MockASRProvider) GetSilenceCount() int {
	return 0
}

func (m *MockASRProvider) ResetStartListenTime() {
	// Mock implementation
}

type MockLLMProvider struct {
	called bool
}

func (m *MockLLMProvider) Initialize() error {
	return nil
}

func (m *MockLLMProvider) Cleanup() error {
	return nil
}

func (m *MockLLMProvider) Response(ctx context.Context, sessionID string, messages []types.Message) (<-chan string, error) {
	m.called = true
	ch := make(chan string, 1)
	ch <- "mock response"
	close(ch)
	return ch, nil
}

func (m *MockLLMProvider) ResponseWithFunctions(
	ctx context.Context,
	sessionID string,
	messages []types.Message,
	tools []openai.Tool,
) (<-chan types.Response, error) {
	m.called = true
	ch := make(chan types.Response, 1)
	ch <- types.Response{
		Content: "mock response with functions",
	}
	close(ch)
	return ch, nil
}

func (m *MockLLMProvider) GetSessionID() string {
	return "mock-session"
}

func (m *MockLLMProvider) SetIdentityFlag(idType string, flag string) {
	// Mock implementation
}

type MockTTSProvider struct {
	called bool
}

func (m *MockTTSProvider) Initialize() error {
	return nil
}

func (m *MockTTSProvider) Cleanup() error {
	return nil
}

func (m *MockTTSProvider) SetVoice(voice string) error {
	m.called = true
	return nil
}

func (m *MockTTSProvider) ToTTS(text string) (string, error) {
	m.called = true
	return "/mock/audio/path.wav", nil
}

// RunIntegrationTest 运行集成测试的主函数
func RunIntegrationTest() {
	fmt.Println("=== WebSocket AI服务解耦方案集成测试 ===\n")
	
	if err := TestEventBusIntegration(); err != nil {
		log.Fatalf("集成测试失败: %v", err)
	}
	
	fmt.Println("\n=== 所有测试通过！解耦方案验证成功 ===")
}