package eventbus

import (
	"fmt"
	"log"
	"time"

	"xiaozhi-server-go/src/core/providers"
	"xiaozhi-server-go/src/core/types"
)

// ExampleUsage 演示如何使用事件总线系统
func ExampleUsage() {
	// 1. 创建事件总线管理器
	config := DefaultIntegrationConfig()
	config.MonitoringPort = 8081 // 使用不同端口避免冲突
	
	manager := NewEventBusManager(config)
	
	// 2. 假设我们有现有的提供者实例（这里用nil代替，实际使用时需要真实的提供者）
	var asrProvider providers.ASRProvider = nil
	var llmProvider types.LLMProvider = nil
	var ttsProvider providers.TTSProvider = nil
	
	// 3. 注册提供者到事件总线
	if err := manager.RegisterProviders(asrProvider, llmProvider, ttsProvider); err != nil {
		log.Printf("注册提供者失败: %v", err)
		return
	}
	
	// 4. 启动管理器
	if err := manager.Start(); err != nil {
		log.Printf("启动管理器失败: %v", err)
		return
	}
	defer manager.Stop()
	
	// 5. 创建会话
	sessionID := "example-session-123"
	if manager.sessionManager != nil {
		session, err := manager.sessionManager.CreateSession("user-123", map[string]interface{}{
			"client_type": "web",
			"language":    "zh-CN",
		})
		if err != nil {
			log.Printf("创建会话失败: %v", err)
			return
		}
		sessionID = session.ID
		fmt.Printf("创建会话成功: %s\n", sessionID)
	}
	
	// 6. 创建连接处理器集成
	integration := NewConnectionHandlerIntegration(manager, sessionID, nil)
	
	// 7. 使用事件总线处理请求
	demonstrateEventBusUsage(integration)
	
	// 8. 监控系统状态
	demonstrateMonitoring(integration)
	
	fmt.Println("事件总线演示完成")
}

// demonstrateEventBusUsage 演示事件总线使用
func demonstrateEventBusUsage(integration *ConnectionHandlerIntegration) {
	fmt.Println("\n=== 事件总线使用演示 ===")
	
	// ASR处理示例
	fmt.Println("1. 处理ASR请求...")
	audioData := []byte("fake_audio_data")
	result, err := integration.ProcessASR(audioData, time.Second*5)
	if err != nil {
		fmt.Printf("ASR处理失败: %v\n", err)
	} else {
		fmt.Printf("ASR结果: %s\n", result)
	}
	
	// LLM处理示例
	fmt.Println("2. 处理LLM请求...")
	messages := []types.Message{
		{Role: "user", Content: "你好"},
	}
	llmResult, err := integration.ProcessLLM(messages, nil)
	if err != nil {
		fmt.Printf("LLM处理失败: %v\n", err)
	} else {
		fmt.Printf("LLM结果: %v\n", llmResult)
	}
	
	// TTS处理示例
	fmt.Println("3. 处理TTS请求...")
	audioPath, err := integration.ProcessTTS("你好，这是TTS测试", "default")
	if err != nil {
		fmt.Printf("TTS处理失败: %v\n", err)
	} else {
		fmt.Printf("TTS音频路径: %s\n", audioPath)
	}
	
	// MCP处理示例
	fmt.Println("4. 处理MCP请求...")
	mcpRequest := map[string]interface{}{
		"method": "test",
		"params": map[string]interface{}{
			"message": "hello mcp",
		},
	}
	mcpResult, err := integration.ProcessMCP(mcpRequest)
	if err != nil {
		fmt.Printf("MCP处理失败: %v\n", err)
	} else {
		fmt.Printf("MCP结果: %v\n", mcpResult)
	}
}

// demonstrateMonitoring 演示监控功能
func demonstrateMonitoring(integration *ConnectionHandlerIntegration) {
	fmt.Println("\n=== 监控功能演示 ===")
	
	// 获取系统统计信息
	stats, err := integration.GetSystemStats()
	if err != nil {
		fmt.Printf("获取系统统计失败: %v\n", err)
		return
	}
	
	fmt.Printf("系统运行时间: %s\n", stats.Uptime)
	fmt.Printf("Go版本: %s\n", stats.GoVersion)
	fmt.Printf("协程数量: %d\n", stats.NumGoroutines)
	fmt.Printf("内存使用: %d MB\n", stats.MemoryUsage.HeapInuse/1024/1024)
	
	if stats.EventBusStats != nil {
		fmt.Printf("事件总线队列长度: %d\n", stats.EventBusStats.QueueLength)
		fmt.Printf("处理速率: %.2f msg/s\n", stats.EventBusStats.ProcessingRate)
		fmt.Printf("总处理数: %d\n", stats.EventBusStats.TotalProcessed)
		fmt.Printf("错误数: %d\n", stats.EventBusStats.ErrorCount)
	}
	
	// 获取会话状态
	sessionState, err := integration.GetSessionState()
	if err != nil {
		fmt.Printf("获取会话状态失败: %v\n", err)
	} else {
		fmt.Printf("会话ID: %s\n", sessionState.ID)
		fmt.Printf("会话状态: %s\n", sessionState.Status)
		fmt.Printf("消息数量: %d\n", sessionState.MessageCount)
		fmt.Printf("会话年龄: %s\n", sessionState.GetAge().String())
		fmt.Printf("空闲时间: %s\n", sessionState.GetIdleTime().String())
	}
}

// ExampleMigrationStrategy 演示渐进式迁移策略
func ExampleMigrationStrategy() {
	fmt.Println("\n=== 渐进式迁移策略演示 ===")
	
	// 创建迁移模式的配置
	config := DefaultIntegrationConfig()
	config.MigrationMode = true
	config.FallbackToOriginal = true
	
	manager := NewEventBusManager(config)
	defer manager.Stop()
	
	// 模拟原始ConnectionHandler
	originalHandler := &MockConnectionHandler{}
	
	// 创建集成层
	integration := NewConnectionHandlerIntegration(manager, "migration-session", originalHandler)
	
	fmt.Printf("事件总线启用状态: %t\n", integration.IsEventBusEnabled())
	fmt.Printf("迁移模式状态: %t\n", integration.IsMigrationMode())
	
	// 在迁移模式下，可以选择性地使用事件总线或原始实现
	if integration.IsEventBusEnabled() && integration.IsMigrationMode() {
		fmt.Println("使用事件总线处理请求（带回退机制）")
		// 处理请求...
	} else {
		fmt.Println("使用原始实现处理请求")
		// 使用原始处理逻辑...
	}
}

// MockConnectionHandler 模拟原始连接处理器
type MockConnectionHandler struct {
	// 原始处理器的字段...
}

// ExampleHealthCheck 演示健康检查
func ExampleHealthCheck() {
	fmt.Println("\n=== 健康检查演示 ===")
	
	config := DefaultIntegrationConfig()
	config.MonitoringPort = 8082
	
	manager := NewEventBusManager(config)
	if err := manager.Start(); err != nil {
		log.Printf("启动失败: %v", err)
		return
	}
	defer manager.Stop()
	
	// 等待一段时间让系统运行
	time.Sleep(time.Second * 2)
	
	if manager.monitor != nil {
		health := manager.monitor.GetHealthStatus()
		fmt.Printf("系统健康状态: %s\n", health.Status)
		fmt.Printf("运行时间: %s\n", health.Uptime)
		
		for serviceName, serviceHealth := range health.Services {
			fmt.Printf("服务 %s: %s (错误数: %d, 延迟: %s)\n", 
				serviceName, 
				serviceHealth.Status, 
				serviceHealth.ErrorCount, 
				serviceHealth.Latency)
		}
		
		if len(health.Issues) > 0 {
			fmt.Println("发现的问题:")
			for _, issue := range health.Issues {
				fmt.Printf("  - %s\n", issue)
			}
		}
		
		fmt.Printf("\n监控API可通过以下端点访问:\n")
		fmt.Printf("  健康检查: http://localhost:%d/health\n", config.MonitoringPort)
		fmt.Printf("  系统指标: http://localhost:%d/metrics\n", config.MonitoringPort)
		fmt.Printf("  统计信息: http://localhost:%d/stats\n", config.MonitoringPort)
		fmt.Printf("  会话信息: http://localhost:%d/sessions\n", config.MonitoringPort)
		fmt.Printf("  适配器信息: http://localhost:%d/adapters\n", config.MonitoringPort)
	}
}

// ExampleCustomEventHandler 演示自定义事件处理器
func ExampleCustomEventHandler() {
	fmt.Println("\n=== 自定义事件处理器演示 ===")
	
	// 创建事件总线
	eventBus := NewEventBus(DefaultEventBusConfig())
	defer eventBus.Shutdown()
	
	// 创建自定义处理器
	customHandler := &CustomEventHandler{
		name: "custom-processor",
	}
	
	// 注册自定义处理器
	if err := eventBus.RegisterHandler("custom", customHandler); err != nil {
		log.Printf("注册自定义处理器失败: %v", err)
		return
	}
	
	// 发布自定义事件
	metadata := map[string]interface{}{
		"session_id": "custom-session",
		"priority":   8,
	}
	
	payload := []byte(`{"message": "这是自定义事件", "data": {"key": "value"}}`)
	
	if err := eventBus.Publish("custom", payload, metadata); err != nil {
		log.Printf("发布自定义事件失败: %v", err)
		return
	}
	
	// 等待处理完成
	time.Sleep(time.Millisecond * 200)
	
	fmt.Println("自定义事件处理完成")
}

// CustomEventHandler 自定义事件处理器示例
type CustomEventHandler struct {
	name string
}

func (c *CustomEventHandler) Handle(msg *EventMessage) error {
	fmt.Printf("自定义处理器 %s 处理消息: ID=%s, Type=%s, SessionID=%s\n", 
		c.name, msg.ID, msg.Type, msg.SessionID)
	
	// 处理自定义逻辑
	fmt.Printf("消息内容: %s\n", string(msg.Payload))
	
	// 设置处理结果
	msg.Metadata["processed_by"] = c.name
	msg.Metadata["processed_at"] = time.Now()
	
	return nil
}

func (c *CustomEventHandler) GetType() string {
	return "custom"
}

func (c *CustomEventHandler) GetStatus() string {
	return "active"
}