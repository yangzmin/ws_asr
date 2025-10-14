package grpc

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	pb "xiaozhi-grpc-proto/generated/go/ai_service"
)

// MessageData 表示消息数据
type MessageData struct {
	Type int
	Data []byte
}

// GRPCConnectionAdapter 实现Connection接口，适配gRPC流
type GRPCConnectionAdapter struct {
	stream       pb.AIService_ChatStreamServer
	id           string
	closed       bool
	mu           sync.RWMutex
	lastActive   time.Time
	messageQueue chan MessageData
	stopChan     chan struct{}
	// 新增：响应通道，用于从ConnectionHandler接收响应
	responseChan chan *pb.ChatResponse
	// 新增：消息发送回调，用于处理不同类型的消息
	messageHandlers map[string]func(interface{}) error
}

// NewGRPCConnectionAdapter 创建新的gRPC连接适配器
func NewGRPCConnectionAdapter(stream pb.AIService_ChatStreamServer, id string) *GRPCConnectionAdapter {
	adapter := &GRPCConnectionAdapter{
		stream:          stream,
		id:              id,
		closed:          false,
		lastActive:      time.Now(),
		messageQueue:    make(chan MessageData, 100),
		stopChan:        make(chan struct{}),
		responseChan:    make(chan *pb.ChatResponse, 100),
		messageHandlers: make(map[string]func(interface{}) error),
	}
	
	// 初始化消息处理器
	adapter.initMessageHandlers()
	
	return adapter
}

// initMessageHandlers 初始化消息处理器
func (g *GRPCConnectionAdapter) initMessageHandlers() {
	// Hello消息处理器
	g.messageHandlers["hello"] = func(data interface{}) error {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		return g.WriteMessage(1, jsonData)
	}
	
	// TTS消息处理器
	g.messageHandlers["tts"] = func(data interface{}) error {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		return g.WriteMessage(1, jsonData)
	}
	
	// STT消息处理器
	g.messageHandlers["stt"] = func(data interface{}) error {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		return g.WriteMessage(1, jsonData)
	}
	
	// LLM消息处理器
	g.messageHandlers["llm"] = func(data interface{}) error {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		return g.WriteMessage(1, jsonData)
	}
	
	// 音频消息处理器
	g.messageHandlers["audio"] = func(data interface{}) error {
		if audioData, ok := data.([]byte); ok {
			return g.WriteMessage(2, audioData) // 音频数据使用消息类型2
		}
		return fmt.Errorf("音频数据格式错误")
	}
}

// ReadMessage 读取消息
func (g *GRPCConnectionAdapter) ReadMessage(stopChan <-chan struct{}) (messageType int, p []byte, err error) {
	select {
	case <-stopChan:
		return 0, nil, fmt.Errorf("连接已停止")
	case <-g.stopChan:
		return 0, nil, fmt.Errorf("连接已关闭")
	case msg := <-g.messageQueue:
		g.mu.Lock()
		g.lastActive = time.Now()
		g.mu.Unlock()
		return msg.Type, msg.Data, nil
	}
}

// WriteMessage 写入消息 - 兼容现有的WebSocket接口
func (g *GRPCConnectionAdapter) WriteMessage(messageType int, data []byte) error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	if g.closed {
		return fmt.Errorf("连接已关闭")
	}
	
	response := &pb.ChatResponse{
		ResponseType: int32(messageType),
		ResponseData: data,
		Timestamp:    time.Now().UnixNano() / int64(time.Millisecond),
	}
	
	return g.stream.Send(response)
}

// SendJSONMessage 发送JSON消息 - 新增方法，兼容现有的消息发送逻辑
func (g *GRPCConnectionAdapter) SendJSONMessage(data interface{}) error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	if g.closed {
		return fmt.Errorf("连接已关闭")
	}
	
	// 检查是否是特定类型的消息
	if msgMap, ok := data.(map[string]interface{}); ok {
		if msgType, exists := msgMap["type"]; exists {
			if handler, found := g.messageHandlers[msgType.(string)]; found {
				return handler(data)
			}
		}
	}
	
	// 默认处理：序列化为JSON并发送
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化JSON消息失败: %v", err)
	}
	
	return g.WriteMessage(1, jsonData)
}

// SendHelloMessage 发送Hello消息 - 兼容现有的消息发送接口
func (g *GRPCConnectionAdapter) SendHelloMessage(hello map[string]interface{}) error {
	return g.SendJSONMessage(hello)
}

// SendTTSMessage 发送TTS消息 - 兼容现有的消息发送接口
func (g *GRPCConnectionAdapter) SendTTSMessage(state string, text string, textIndex int, sessionID string, audioFormat string) error {
	ttsMsg := map[string]interface{}{
		"type":        "tts",
		"state":       state,
		"session_id":  sessionID,
		"text":        text,
		"index":       textIndex,
		"audio_codec": audioFormat,
	}
	return g.SendJSONMessage(ttsMsg)
}

// SendSTTMessage 发送STT消息 - 兼容现有的消息发送接口
func (g *GRPCConnectionAdapter) SendSTTMessage(text string, sessionID string) error {
	sttMsg := map[string]interface{}{
		"type":       "stt",
		"text":       text,
		"session_id": sessionID,
	}
	return g.SendJSONMessage(sttMsg)
}

// SendEmotionMessage 发送情绪消息 - 兼容现有的消息发送接口
func (g *GRPCConnectionAdapter) SendEmotionMessage(emotion string, emoji string, sessionID string) error {
	emotionMsg := map[string]interface{}{
		"type":       "llm",
		"text":       emoji,
		"emotion":    emotion,
		"session_id": sessionID,
	}
	return g.SendJSONMessage(emotionMsg)
}

// SendAudioData 发送音频数据 - 兼容现有的音频发送接口
func (g *GRPCConnectionAdapter) SendAudioData(audioData []byte) error {
	return g.WriteMessage(2, audioData)
}

// Close 关闭连接
func (g *GRPCConnectionAdapter) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if !g.closed {
		g.closed = true
		close(g.stopChan)
		close(g.messageQueue)
		close(g.responseChan)
	}
	
	return nil
}

// GetID 获取连接ID
func (g *GRPCConnectionAdapter) GetID() string {
	return g.id
}

// GetType 获取连接类型
func (g *GRPCConnectionAdapter) GetType() string {
	return "grpc"
}

// IsClosed 检查连接是否已关闭
func (g *GRPCConnectionAdapter) IsClosed() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.closed
}

// GetLastActiveTime 获取最后活跃时间
func (g *GRPCConnectionAdapter) GetLastActiveTime() time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastActive
}

// IsStale 检查连接是否过期
func (g *GRPCConnectionAdapter) IsStale(timeout time.Duration) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return time.Since(g.lastActive) > timeout
}

// GetResponseChannel 获取响应通道 - 用于handleOutgoingMessages
func (g *GRPCConnectionAdapter) GetResponseChannel() <-chan *pb.ChatResponse {
	return g.responseChan
}

// GetStopChannel 获取停止通道 - 用于外部监听连接关闭
func (g *GRPCConnectionAdapter) GetStopChannel() <-chan struct{} {
	return g.stopChan
}

// PutMessage 将消息放入队列 - 用于接收gRPC请求
func (g *GRPCConnectionAdapter) PutMessage(messageType int, data []byte) error {
	select {
	case g.messageQueue <- MessageData{Type: messageType, Data: data}:
		return nil
	default:
		return fmt.Errorf("消息队列已满")
	}
}