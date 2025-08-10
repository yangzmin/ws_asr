package websocket

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"speech-recognition-backend/backend/models"
	"speech-recognition-backend/backend/speech"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512KB
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境需要限制
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Speech recognition client
	speechClient *speech.Client
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		if c.speechClient != nil {
			c.speechClient.Close()
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(message []byte) {
	var wsMsg models.WSMessage
	if err := json.Unmarshal(message, &wsMsg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	switch wsMsg.Type {
	case "start_recognition":
		c.handleStartRecognition(&wsMsg)
	case "audio_data":
		c.handleAudioData(&wsMsg)
	case "stop_recognition":
		c.handleStopRecognition()
	default:
		log.Printf("Unknown message type: %s", wsMsg.Type)
	}
}

// handleStartRecognition initializes speech recognition
func (c *Client) handleStartRecognition(msg *models.WSMessage) {
	log.Println("Starting speech recognition")

	// 创建语音识别客户端
	c.speechClient = speech.NewClient()

	// 启动识别会话
	if err := c.speechClient.StartRecognition(func(result *models.RecognitionResult) {
		// 发送识别结果到客户端
		if resultBytes, err := json.Marshal(result); err == nil {
			c.send <- resultBytes
		}
	}); err != nil {
		log.Printf("Error starting recognition: %v", err)
		return
	}

	// 发送确认消息
	response := models.WSMessage{
		Type: "recognition_started",
	}
	if responseBytes, err := json.Marshal(response); err == nil {
		c.send <- responseBytes
	}
}

// handleAudioData processes audio data
func (c *Client) handleAudioData(msg *models.WSMessage) {
	if c.speechClient == nil {
		log.Println("Speech client not initialized")
		return
	}

	// 调试日志：观察每段数据大小和序号
	if msg.Data != "" {
		log.Printf("WS audio_data: seq=%d final=%v bytes=%d", msg.Sequence, msg.IsFinal, len(msg.Data))
	} else {
		log.Printf("WS audio_data: seq=%d final=%v (empty data)", msg.Sequence, msg.IsFinal)
	}

	// 发送音频数据到语音识别服务
	if err := c.speechClient.SendAudioData(msg.Data, msg.Sequence, msg.IsFinal); err != nil {
		log.Printf("Error sending audio data: %v", err)
	}
}

// handleStopRecognition stops speech recognition
func (c *Client) handleStopRecognition() {
	log.Println("Stop requested: will wait for final ASR result; not closing upstream connection yet")

	// 不再立刻关闭 ByteDance 连接，避免丢失最终结果
	// 等待上游返回最后一个包 (IsLastPackage)，由 speech.Client 的 recvMessages 结束
	// 如需在前端展示状态，可在这里下发一个过渡状态消息
	// response := models.WSMessage{ Type: "recognition_stopping" }
	// if responseBytes, err := json.Marshal(response); err == nil { c.send <- responseBytes }
}

// ServeWS handles websocket requests from the peer.
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
