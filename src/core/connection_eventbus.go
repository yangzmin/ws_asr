package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"xiaozhi-server-go/src/configs"
	"xiaozhi-server-go/src/core/eventbus"
	"xiaozhi-server-go/src/core/pool"
	"xiaozhi-server-go/src/core/utils"
)

// EventBusConnectionHandler 集成事件总线的连接处理器
type EventBusConnectionHandler struct {
	*ConnectionHandler // 嵌入原有的ConnectionHandler

	// 事件总线相关组件
	eventBus       *eventbus.EventBus
	sessionManager *eventbus.SessionManager

	// 配置和状态
	eventBusConfig  *configs.Config
	isEventBusMode  bool
	migrationStatus map[string]bool // 记录各服务的迁移状态
	mu              sync.RWMutex
}

// NewEventBusConnectionHandler 创建集成事件总线的连接处理器
func NewEventBusConnectionHandler(
	config *configs.Config,
	providerSet *pool.ProviderSet,
	logger *utils.Logger,
	req *http.Request,
	ctx context.Context,
) *EventBusConnectionHandler {
	// 首先创建原有的ConnectionHandler
	baseHandler := NewConnectionHandler(config, providerSet, logger, req, ctx)

	handler := &EventBusConnectionHandler{
		ConnectionHandler: baseHandler,
		eventBusConfig:    config,
		isEventBusMode:    config.EventBus.Enabled,
		migrationStatus:   make(map[string]bool),
	}

	// 如果启用了事件总线模式，初始化事件总线组件
	if handler.isEventBusMode {
		handler.initEventBusComponents()
	}

	return handler
}

// initEventBusComponents 初始化事件总线组件
func (h *EventBusConnectionHandler) initEventBusComponents() {
	h.logger.Info("初始化事件总线组件")

	// 创建事件总线配置
	busConfig := &eventbus.EventBusConfig{
		ChannelBufferSize: h.eventBusConfig.EventBus.Bus.BufferSize,
		WorkerCount:       h.eventBusConfig.EventBus.Bus.WorkerCount,
		MaxRetries:        3,
		RetryDelay:        time.Millisecond * 100,
		StatsInterval:     time.Second * 5,
	}

	// 初始化事件总线
	h.eventBus = eventbus.NewEventBus(busConfig)

	// 初始化会话管理器
	sessionConfig := &eventbus.SessionConfig{
		MaxSessions:     1000,
		SessionTimeout:  time.Duration(h.eventBusConfig.EventBus.Session.Timeout) * time.Minute,
		CleanupInterval: time.Duration(h.eventBusConfig.EventBus.Session.CleanupInterval) * time.Minute,
		EnableMetrics:   true,
	}
	h.sessionManager = eventbus.NewSessionManager(sessionConfig)

	// 注册服务适配器到事件总线
	h.registerEventBusHandlers()

	// 初始化服务适配器迁移状态
	h.initMigrationStatus()

	h.logger.Info("事件总线组件初始化完成")
}

// registerEventBusHandlers 注册事件总线处理器
func (h *EventBusConnectionHandler) registerEventBusHandlers() {
	h.logger.Info("开始注册事件总线处理器")

	// 注册LLM适配器
	if h.providers.llm != nil {
		llmAdapter := eventbus.NewLLMAdapter(h.providers.llm)
		if err := h.eventBus.RegisterHandler("llm", llmAdapter); err != nil {
			h.logger.Error("注册LLM适配器失败: %v", err)
		} else {
			h.logger.Info("LLM适配器注册成功")
		}
	} else {
		h.logger.Warn("LLM提供者为空，跳过LLM适配器注册")
	}

	// 注册ASR适配器
	if h.providers.asr != nil {
		asrAdapter := eventbus.NewASRAdapter(h.providers.asr)
		if err := h.eventBus.RegisterHandler("asr", asrAdapter); err != nil {
			h.logger.Error("注册ASR适配器失败: %v", err)
		} else {
			h.logger.Info("ASR适配器注册成功")
		}
	} else {
		h.logger.Warn("ASR提供者为空，跳过ASR适配器注册")
	}

	// 注册TTS适配器
	if h.providers.tts != nil {
		ttsAdapter := eventbus.NewTTSAdapter(h.providers.tts)
		if err := h.eventBus.RegisterHandler("tts", ttsAdapter); err != nil {
			h.logger.Error("注册TTS适配器失败: %v", err)
		} else {
			h.logger.Info("TTS适配器注册成功")
		}
	} else {
		h.logger.Warn("TTS提供者为空，跳过TTS适配器注册")
	}

	// 注册MCP适配器
	mcpAdapter := eventbus.NewMCPAdapter()
	if err := h.eventBus.RegisterHandler("mcp", mcpAdapter); err != nil {
		h.logger.Error("注册MCP适配器失败: %v", err)
	} else {
		h.logger.Info("MCP适配器注册成功")
	}

	h.logger.Info("事件总线处理器注册完成")
}

// initMigrationStatus 初始化服务迁移状态
func (h *EventBusConnectionHandler) initMigrationStatus() {
	h.logger.Info("初始化服务迁移状态")

	h.migrationStatus = make(map[string]bool)
	h.migrationStatus["ASR"] = h.eventBusConfig.EventBus.Adapters.ASR.Enabled
	h.migrationStatus["LLM"] = h.eventBusConfig.EventBus.Adapters.LLM.Enabled
	h.migrationStatus["TTS"] = h.eventBusConfig.EventBus.Adapters.TTS.Enabled
	h.migrationStatus["MCP"] = h.eventBusConfig.EventBus.Adapters.MCP.Enabled

	h.logger.Info("服务迁移状态初始化完成", "status", h.migrationStatus)
}

// Handle 重写Handle方法以支持事件总线模式
func (h *EventBusConnectionHandler) Handle(conn Connection) {
	if !h.isEventBusMode {
		// 如果未启用事件总线，使用原有逻辑
		h.ConnectionHandler.Handle(conn)
		return
	}

	// 事件总线模式处理逻辑
	h.handleWithEventBus(conn)
}

// handleWithEventBus 使用事件总线处理连接
func (h *EventBusConnectionHandler) handleWithEventBus(conn Connection) {
	defer conn.Close()

	h.conn = conn
	h.logger.Info(fmt.Sprintf("使用事件总线模式处理连接: %s", h.sessionID))

	// 注册会话到会话管理器
	metadata := map[string]interface{}{
		"transport_type": h.transportType,
		"headers":        h.headers,
		"device_id":      h.deviceID,
		"client_id":      h.clientId,
	}
	h.sessionManager.CreateSession(h.userID, metadata)

	// 启动事件总线
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动消息处理协程（使用事件总线版本）
	go h.processClientAudioMessagesWithEventBus()
	go h.processClientTextMessagesWithEventBus()
	go h.processTTSQueueWithEventBus()
	go h.sendAudioMessageWithEventBus()

	// 在WebSocket连接建立后加载用户AI配置
	if h.request != nil {
		h.loadUserAIConfigurations(h.request)
	}

	// 绑定MCP管理器（如果使用事件总线模式）
	if h.migrationStatus["MCP"] {
		params := map[string]interface{}{
			"session_id": h.sessionID,
			"vision_url": h.config.Web.VisionURL,
			"device_id":  h.deviceID,
			"client_id":  h.clientId,
			"token":      h.config.Server.Token,
		}

		// 通过事件总线发送MCP绑定事件
		payloadBytes, _ := json.Marshal(params)
		metadata := map[string]interface{}{
			"timestamp":  time.Now(),
			"session_id": h.sessionID,
			"priority":   1,
		}
		h.eventBus.Publish("mcp", payloadBytes, metadata)
	}

	// 主消息循环
	for {
		select {
		case <-h.stopChan:
			h.logger.Info("收到停止信号，退出事件总线消息循环")
			return
		default:
			messageType, message, err := conn.ReadMessage(h.stopChan)
			if err != nil {
				h.LogError(fmt.Sprintf("读取消息失败: %v, 退出主消息循环", err))
				return
			}

			if err := h.handleMessageWithEventBus(messageType, message); err != nil {
				h.LogError(fmt.Sprintf("处理消息失败: %v", err))
			}
		}
	}
}

// handleMessageWithEventBus 使用事件总线处理消息
func (h *EventBusConnectionHandler) handleMessageWithEventBus(messageType int, message []byte) error {
	// 更新会话活跃时间
	h.sessionManager.IncrementMessageCount(h.sessionID)

	// 创建消息处理事件
	payload := map[string]interface{}{
		"message_type": messageType,
		"message":      message,
	}
	payloadBytes, _ := json.Marshal(payload)
	metadata := map[string]interface{}{
		"timestamp":  time.Now(),
		"session_id": h.sessionID,
		"priority":   1,
	}

	// 发布事件到事件总线
	h.eventBus.Publish("message", payloadBytes, metadata)

	// 同时保持原有的消息处理逻辑作为后备
	return h.ConnectionHandler.handleMessage(messageType, message)
}

// processClientAudioMessagesWithEventBus 使用事件总线处理音频消息
func (h *EventBusConnectionHandler) processClientAudioMessagesWithEventBus() {
	for {
		select {
		case <-h.stopChan:
			return
		case audioData := <-h.clientAudioQueue:
			if h.closeAfterChat {
				continue
			}

			// 如果ASR服务已迁移到事件总线
			if h.migrationStatus["ASR"] {
				payload := map[string]interface{}{
					"audio_data": audioData,
				}
				payloadBytes, _ := json.Marshal(payload)
				metadata := map[string]interface{}{
					"timestamp":  time.Now(),
					"session_id": h.sessionID,
					"priority":   1,
				}
				h.eventBus.Publish("asr", payloadBytes, metadata)
			} else {
				// 使用原有逻辑
				if err := h.providers.asr.AddAudio(audioData); err != nil {
					h.LogError(fmt.Sprintf("处理音频数据失败: %v", err))
				}
			}
		}
	}
}

// processClientTextMessagesWithEventBus 使用事件总线处理文本消息
func (h *EventBusConnectionHandler) processClientTextMessagesWithEventBus() {
	for {
		select {
		case <-h.stopChan:
			return
		case text := <-h.clientTextQueue:
			h.logger.Info("收到文本消息: %s", text)
			
			// 解析消息类型，决定如何处理
			if err := h.processTextMessageWithEventBus(context.Background(), text); err != nil {
				h.LogError(fmt.Sprintf("处理文本数据失败: %v", err))
			}
		}
	}
}

// processTextMessageWithEventBus 使用事件总线处理文本消息，根据消息类型分发
func (h *EventBusConnectionHandler) processTextMessageWithEventBus(ctx context.Context, text string) error {
	// 解析JSON消息
	var msgJSON interface{}
	if err := json.Unmarshal([]byte(text), &msgJSON); err != nil {
		return h.conn.WriteMessage(1, []byte(text))
	}

	// 检查是否为整数类型
	if _, ok := msgJSON.(float64); ok {
		return h.conn.WriteMessage(1, []byte(text))
	}

	// 解析为map类型处理具体消息
	msgMap, ok := msgJSON.(map[string]interface{})
	if !ok {
		return fmt.Errorf("消息格式错误")
	}

	// 根据消息类型分发处理
	h.logger.Info("解析消息类型，msgMap: %v", msgMap)
	msgType, ok := msgMap["type"].(string)
	if !ok {
		return fmt.Errorf("消息类型错误")
	}

	// 创建通用的元数据
	metadata := map[string]interface{}{
		"timestamp":  time.Now(),
		"session_id": h.sessionID,
		"priority":   1,
		"msg_type":   msgType,
	}

	switch msgType {
	case "hello":
		// hello消息通常不需要通过事件总线，直接处理
		return h.handleHelloMessage(msgMap)
		
	case "abort":
		// abort消息直接处理
		return h.clientAbortChat()
		
	case "listen":
		// listen消息直接处理
		return h.handleListenMessage(msgMap)
		
	case "chat":
		// chat消息发布到LLM事件总线（如果已迁移）
		if h.migrationStatus["LLM"] && h.eventBus != nil {
			h.logger.Info("发布chat消息到LLM事件总线")
			payload := map[string]interface{}{
				"text":     text,
				"msg_map":  msgMap,
				"msg_type": msgType,
			}
			payloadBytes, _ := json.Marshal(payload)
			
			if err := h.eventBus.Publish("llm", payloadBytes, metadata); err != nil {
				h.logger.Error("发布LLM消息失败: %v", err)
				return err
			}
			h.logger.Info("LLM消息发布成功")
			return nil
		} else {
			// 使用原有逻辑
			return h.handleChatMessage(ctx, text)
		}
		
	case "vision":
		// vision消息发布到vision事件总线（如果已迁移）
		if h.migrationStatus["Vision"] && h.eventBus != nil {
			h.logger.Info("发布vision消息到事件总线")
			payload := map[string]interface{}{
				"msg_map":  msgMap,
				"msg_type": msgType,
			}
			payloadBytes, _ := json.Marshal(payload)
			
			if err := h.eventBus.Publish("vision", payloadBytes, metadata); err != nil {
				h.logger.Error("发布Vision消息失败: %v", err)
				return err
			}
			h.logger.Info("Vision消息发布成功")
			return nil
		} else {
			// 使用原有逻辑
			return h.handleVisionMessage(msgMap)
		}
		
	case "image":
		// image消息发布到image事件总线（如果已迁移）
		if h.migrationStatus["Image"] && h.eventBus != nil {
			h.logger.Info("发布image消息到事件总线")
			payload := map[string]interface{}{
				"msg_map":  msgMap,
				"msg_type": msgType,
			}
			payloadBytes, _ := json.Marshal(payload)
			
			if err := h.eventBus.Publish("image", payloadBytes, metadata); err != nil {
				h.logger.Error("发布Image消息失败: %v", err)
				return err
			}
			h.logger.Info("Image消息发布成功")
			return nil
		} else {
			// 使用原有逻辑
			return h.handleImageMessage(ctx, msgMap)
		}
		
	case "mcp":
		// MCP消息发布到MCP事件总线（如果已迁移）
		if h.migrationStatus["MCP"] && h.eventBus != nil {
			h.logger.Info("发布MCP消息到事件总线")
			payload := map[string]interface{}{
				"msg_map":  msgMap,
				"msg_type": msgType,
			}
			payloadBytes, _ := json.Marshal(payload)
			
			if err := h.eventBus.Publish("mcp", payloadBytes, metadata); err != nil {
				h.logger.Error("发布MCP消息失败: %v", err)
				return err
			}
			h.logger.Info("MCP消息发布成功")
			return nil
		} else {
			// 使用原有逻辑
			return h.mcpManager.HandleXiaoZhiMCPMessage(msgMap)
		}
		
	default:
		h.logger.Warn("=== 未知消息类型 ===", map[string]interface{}{
			"unknown_type": msgType,
			"full_message": msgMap,
		})
		return fmt.Errorf("未知的消息类型: %s", msgType)
	}
}

// processTTSQueueWithEventBus 使用事件总线处理TTS队列
func (h *EventBusConnectionHandler) processTTSQueueWithEventBus() {
	for {
		select {
		case <-h.stopChan:
			return
		case task := <-h.ttsQueue:
			// 如果TTS服务已迁移到事件总线
			if h.migrationStatus["TTS"] {
				payload := map[string]interface{}{
					"text":       task.text,
					"round":      task.round,
					"text_index": task.textIndex,
					"filepath":   task.filepath,
				}
				payloadBytes, _ := json.Marshal(payload)
				metadata := map[string]interface{}{
					"timestamp":  time.Now(),
					"session_id": h.sessionID,
					"priority":   1,
				}
				h.eventBus.Publish("tts", payloadBytes, metadata)
			} else {
				// 使用原有逻辑
				h.processTTSTask(task.text, task.round, task.textIndex, task.filepath)
			}
		}
	}
}

// sendAudioMessageWithEventBus 使用事件总线发送音频消息
func (h *EventBusConnectionHandler) sendAudioMessageWithEventBus() {
	for {
		select {
		case <-h.stopChan:
			return
		case task := <-h.audioMessagesQueue:
			// 创建音频发送事件
			payload := map[string]interface{}{
				"filepath":   task.filepath,
				"text":       task.text,
				"text_index": task.textIndex,
				"round":      task.round,
			}
			payloadBytes, _ := json.Marshal(payload)
			metadata := map[string]interface{}{
				"timestamp":  time.Now(),
				"session_id": h.sessionID,
				"priority":   1,
			}
			h.eventBus.Publish("audio", payloadBytes, metadata)

			// 同时执行原有逻辑
			h.sendAudioMessage(task.filepath, task.text, task.textIndex, task.round)
		}
	}
}

// GetEventBusStatus 获取事件总线状态
func (h *EventBusConnectionHandler) GetEventBusStatus() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status := map[string]interface{}{
		"enabled":          h.isEventBusMode,
		"session_id":       h.sessionID,
		"migration_status": h.migrationStatus,
	}

	if h.isEventBusMode && h.eventBus != nil {
		// 获取事件总线统计信息
		stats := h.eventBus.GetStats()
		status["event_bus"] = map[string]interface{}{
			"active":          true,
			"total_processed": stats.TotalProcessed,
			"error_count":     stats.ErrorCount,
			"queue_length":    stats.QueueLength,
			"last_update":     stats.LastUpdateTime,
		}

		// 获取会话管理器状态
		if h.sessionManager != nil {
			// 会话管理器没有GetStatus方法，使用基本信息
			stats := h.eventBus.GetStats()
			status["session_manager"] = map[string]interface{}{
				"active":          true,
				"total_processed": stats.TotalProcessed,
				"error_count":     stats.ErrorCount,
				"queue_length":    stats.QueueLength,
				"last_update":     stats.LastUpdateTime,
			}
		}
	}

	return status
}

// ToggleServiceMigration 切换服务迁移状态
func (h *EventBusConnectionHandler) ToggleServiceMigration(serviceType string, enabled bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.isEventBusMode {
		return fmt.Errorf("事件总线模式未启用")
	}

	switch serviceType {
	case "ASR":
		h.migrationStatus["ASR"] = enabled
	case "LLM":
		h.migrationStatus["LLM"] = enabled
	case "TTS":
		h.migrationStatus["TTS"] = enabled
	case "MCP":
		h.migrationStatus["MCP"] = enabled
	default:
		return fmt.Errorf("不支持的服务类型: %s", serviceType)
	}

	h.logger.Info(fmt.Sprintf("服务 %s 迁移状态已更新为: %v", serviceType, enabled))
	return nil
}

// Close 重写Close方法以清理事件总线资源
func (h *EventBusConnectionHandler) Close() {
	// 清理事件总线资源
	if h.isEventBusMode && h.sessionManager != nil {
		h.sessionManager.CloseSession(h.sessionID)
		h.logger.Info(fmt.Sprintf("清理事件总线资源: %s", h.sessionID))
	}

	// 调用原有的Close方法
	h.ConnectionHandler.Close()
}
