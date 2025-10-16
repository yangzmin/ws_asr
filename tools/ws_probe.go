package main

import (
    "context"
    "encoding/json"
    "bytes"
    "fmt"
    "log"
    "net/http"
    "time"

	"angrymiao-ai-server/src/configs"
	"angrymiao-ai-server/src/core/auth"

	"github.com/gorilla/websocket"
)

func main() {
	// Load config to get WS address and server token
	cfg, _, err := configs.LoadConfig(nil)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	// Prepare device & token
	deviceID := "web-client-12333"
	at := auth.NewAuthToken(cfg.Server.Token)
	token, err := at.GenerateToken(deviceID)
	if err != nil {
		log.Fatalf("generate token failed: %v", err)
	}

	// Build WS URL with browser-friendly query mapping
	wsURL := fmt.Sprintf("ws://%s:%d/?device-id=%s&client-id=test&transport-type=browser&token=%s",
		cfg.Transport.WebSocket.IP,
		cfg.Transport.WebSocket.Port,
		deviceID,
		token,
	)

	// Dial WS
	header := http.Header{}
	header.Set("Device-Id", deviceID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d := websocket.Dialer{}
	c, _, err := d.DialContext(ctx, wsURL, header)
	if err != nil {
		log.Fatalf("ws dial failed: %v", err)
	}
	defer c.Close()
	log.Printf("WS connected: %s", wsURL)

	// Send hello with audio params
	hello := map[string]interface{}{
		"type": "hello",
		"audio_params": map[string]interface{}{
			"format":         "pcm",
			"sample_rate":    24000,
			"channels":       1,
			"frame_duration": 20,
		},
	}
	if data, err := json.Marshal(hello); err == nil {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Fatalf("send hello failed: %v", err)
		}
	}

	// Send a simple chat
	time.Sleep(300 * time.Millisecond)
	chat := map[string]interface{}{
		"type":    "chat",
		"content": "你好！请用简短话回复",
	}
	if data, err := json.Marshal(chat); err == nil {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Fatalf("send chat failed: %v", err)
		}
	}

	// Read a few messages
    deadline := time.Now().Add(8 * time.Second)
    for time.Now().Before(deadline) {
        _ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
        mt, msg, err := c.ReadMessage()
        if err != nil {
            // Break on any error to avoid repeated reads on failed connection
            log.Printf("read stop: %v", err)
            break
        }
        if mt == websocket.TextMessage {
            fmt.Println("WS<-", string(msg))
            // Stop if TTS reported stop to avoid server closing first
            if string(msg) != "" && (string(msg) == string(msg)) {
                // naive check: if contains '"type":"tts"' and '"state":"stop"'
                if bytes.Contains(msg, []byte("\"type\":\"tts\"")) && bytes.Contains(msg, []byte("\"state\":\"stop\"")) {
                    break
                }
            }
        } else {
            fmt.Println("WS<- bin", len(msg))
        }
    }
}