package model

import "time"

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	SessionID     string            `json:"session_id"`
	DeviceID      string            `json:"device_id"`
	ClientID      string            `json:"client_id"`
	UserID        uint              `json:"user_id"`
	Headers       map[string]string `json:"headers"`
	ConnectedAt   time.Time         `json:"connected_at"`
	LastActiveAt  time.Time         `json:"last_active_at"`
	TransportType string            `json:"transport_type"`
}

// AudioParams 音频参数
type AudioParams struct {
	Format        string `json:"format"`
	SampleRate    int32  `json:"sample_rate"`
	Channels      int32  `json:"channels"`
	FrameDuration int32  `json:"frame_duration"`
}

// HelloMessage Hello消息
type HelloMessage struct {
	AudioParams *AudioParams `json:"audio_params,omitempty"`
}

// ListenMessage Listen消息
type ListenMessage struct {
	State string `json:"state"` // start, stop, detect
	Mode  string `json:"mode"`  // auto, manual
	Text  string `json:"text,omitempty"`
}

// ChatMessage Chat消息
type ChatMessage struct {
	Text string `json:"text"`
}

// AbortMessage Abort消息
type AbortMessage struct {
	Reason string `json:"reason,omitempty"`
}

// VisionMessage Vision消息
type VisionMessage struct {
	Cmd    string            `json:"cmd"` // gen_pic, gen_video, read_img
	Params map[string]string `json:"params,omitempty"`
}

// ImageMessage Image消息
type ImageMessage struct {
	Text      string     `json:"text"`
	ImageData *ImageData `json:"image_data"`
}

// ImageData 图片数据
type ImageData struct {
	URL    string `json:"url,omitempty"`
	Data   string `json:"data,omitempty"`   // base64编码
	Format string `json:"format,omitempty"` // 图片格式
}

// MCPMessage MCP消息
type MCPMessage struct {
	Method string            `json:"method"`
	Params map[string]string `json:"params,omitempty"`
}

// HelloResponse Hello响应
type HelloResponse struct {
	ServerAudioParams *AudioParams `json:"server_audio_params"`
	Status            string       `json:"status"`
}

// STTResponse STT响应
type STTResponse struct {
	Text    string `json:"text"`
	IsFinal bool   `json:"is_final"`
}

// TTSResponse TTS响应
type TTSResponse struct {
	State     string `json:"state"` // start, end
	Text      string `json:"text"`
	TextIndex int32  `json:"text_index"`
}

// EmotionResponse 情绪响应
type EmotionResponse struct {
	Emotion string `json:"emotion"` // thinking, speaking, listening
}

// AudioResponse 音频响应
type AudioResponse struct {
	AudioData []byte `json:"audio_data"`
	Text      string `json:"text"`
	Round     int32  `json:"round"`
	TextIndex int32  `json:"text_index"`
	Format    string `json:"format"` // opus, pcm
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// StatusResponse 状态响应
type StatusResponse struct {
	Status  string            `json:"status"`
	Details map[string]string `json:"details,omitempty"`
}
