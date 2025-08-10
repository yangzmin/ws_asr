package speech

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"speech-recognition-backend/backend/config"
	"speech-recognition-backend/backend/models"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	seq             int
	segmentDuration int
	url             string
	connect         *websocket.Conn
	resultCallback  func(*models.RecognitionResult)
	ctx             context.Context
	cancel          context.CancelFunc
	messageChan     chan []byte
	audioSegments   [][]byte
	currentSegment  int
	ticker          *time.Ticker
	finalReceived   bool
	sendingStarted  bool
}

func NewClient() *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		seq:             1,
		url:             config.GetAPIEndpoint(),
		segmentDuration: config.DefaultSegmentDuration,
		ctx:             ctx,
		cancel:          cancel,
		messageChan:     make(chan []byte, 10),
		audioSegments:   make([][]byte, 0),
		currentSegment:  0,
		finalReceived:   false,
		sendingStarted:  false,
	}
}

func (c *Client) StartRecognition(callback func(*models.RecognitionResult)) error {
	c.resultCallback = callback

	// 创建WebSocket连接
	if err := c.createConnection(); err != nil {
		return fmt.Errorf("create connection err: %w", err)
	}

	// 发送全客户端请求
	if err := c.sendFullClientRequest(); err != nil {
		return fmt.Errorf("send full request err: %w", err)
	}

	// 启动消息发送goroutine
	go c.startMessageSender()

	// 启动消息接收
	go c.recvMessages()

	return nil
}

func (c *Client) createConnection() error {
	header := c.newAuthHeader()
	conn, resp, err := websocket.DefaultDialer.DialContext(c.ctx, c.url, header)
	if err != nil {
		return fmt.Errorf("dial websocket err: %w", err)
	}
	log.Printf("header: %v", header)
	log.Printf("logid: %s", resp.Header.Get("X-Tt-Logid"))
	c.connect = conn
	return nil
}

func (c *Client) newAuthHeader() http.Header {
	reqid := uuid.New().String()
	header := http.Header{}

	header.Add("X-Api-Resource-Id", "volc.bigasr.sauc.duration")
	header.Add("X-Api-Request-Id", reqid)
	// header.Add("X-Api-Connect-Id", reqid)
	header.Add("X-Api-Access-Key", config.GetAccessKey())
	header.Add("X-Api-App-Key", config.GetAppKey())
	return header
}

func (c *Client) sendFullClientRequest() error {
	fullClientRequest := c.newFullClientRequest()
	err := c.connect.WriteMessage(websocket.BinaryMessage, fullClientRequest)
	if err != nil {
		return fmt.Errorf("full client message write websocket err: %w", err)
	}
	_, resp, err := c.connect.ReadMessage()
	if err != nil {
		return fmt.Errorf("full client message read err: %w", err)
	}
	respStruct := c.parseResponse(resp)
	log.Printf("Full client response: %+v", respStruct)

	// 在发送完整客户端请求后递增序列号，参考demo
	c.seq++

	return nil
}

func (c *Client) newFullClientRequest() []byte {
	var request []byte

	// 添加协议头
	header := c.defaultHeader().withMessageTypeSpecificFlags(0b0001).toBytes()
	request = append(request, header...)

	// 创建payload
	payload := models.AsrRequestPayload{
		User: models.UserMeta{
			Uid: "demo_uid",
		},
		Audio: models.AudioMeta{
			Format:  config.DefaultFormat,
			Codec:   config.DefaultCodec,
			Rate:    config.DefaultSampleRate,
			Bits:    config.DefaultBits,
			Channel: config.DefaultChannels,
		},
		Request: models.RequestMeta{
			ModelName:       "bigmodel",
			EnableITN:       true,
			EnablePUNC:      true,
			EnableDDC:       true,
			ShowUtterances:  true,
			EnableNonstream: false,
		},
	}

	payloadArr, _ := sonic.Marshal(payload)
	payloadArr = c.gzipCompress(payloadArr)
	payloadSize := len(payloadArr)

	// 添加序列号
	seqBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(seqBytes, uint32(1))
	request = append(request, seqBytes...)

	// 添加payload大小
	payloadSizeArr := make([]byte, 4)
	binary.BigEndian.PutUint32(payloadSizeArr, uint32(payloadSize))
	request = append(request, payloadSizeArr...)

	// 添加payload
	request = append(request, payloadArr...)

	return request
}

func (c *Client) SendAudioData(audioData string, sequence int, isFinal bool) error {
	if c.connect == nil {
		return fmt.Errorf("websocket connection not established")
	}

	// 解码base64音频数据
	audioBytes, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		return fmt.Errorf("decode audio data err: %w", err)
	}

	// 添加到音频段列表
	c.audioSegments = append(c.audioSegments, audioBytes)

	// 首次收到音频就启动定时发送
	if !c.sendingStarted {
		c.startTimedSending()
	}

	// 如果是最后一个音频包，标记最终
	if isFinal {
		c.finalReceived = true
	}

	log.Printf("Added audio data: size=%d, total_segments=%d, final=%v", len(audioBytes), len(c.audioSegments), isFinal)
	return nil
}

// startMessageSender 启动消息发送goroutine
func (c *Client) startMessageSender() {
	for {
		select {
		case message := <-c.messageChan:
			// 修正：音频帧必须使用二进制帧发送
			err := c.connect.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				log.Printf("write message err: %s", err)
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// startTimedSending 按固定间隔发送已收到的音频段，直到收到最终包并发送完所有段
func (c *Client) startTimedSending() {
	if c.ticker != nil {
		c.sendingStarted = true
		return // 已经在发送中
	}

	c.ticker = time.NewTicker(time.Duration(c.segmentDuration) * time.Millisecond)
	c.sendingStarted = true
	go func() {
		defer c.ticker.Stop()

		for {
			select {
			case <-c.ticker.C:
				if c.currentSegment < len(c.audioSegments) {
					segment := c.audioSegments[c.currentSegment]

					// 使用当前序列号发送本段（首段应为2），发送后再自增
					seqToSend := c.seq
					isLastThisSegment := c.finalReceived && (c.currentSegment == len(c.audioSegments)-1)
					if isLastThisSegment {
						seqToSend = -seqToSend
					}

					message := c.newAudioOnlyRequest(seqToSend, segment)
					c.messageChan <- message
					log.Printf("send message: seq: %d, segment: %d/%d", seqToSend, c.currentSegment+1, len(c.audioSegments))

					c.currentSegment++
					if isLastThisSegment { // 已发送最后一段
						return
					}
					// 非最后一段，递增序列号
					c.seq++
				} else if c.finalReceived { // 已经没有新段并且收到最终标志
					return
				}
			case <-c.ctx.Done():
				return
			}
		}
	}()
}

func (c *Client) newAudioOnlyRequest(seq int, segment []byte) []byte {
	var request []byte

	// 设置消息类型标志
	flags := byte(0b0001) // POS_SEQUENCE
	if seq < 0 {
		flags = byte(0b0011) // NEG_WITH_SEQUENCE
	}

	header := c.defaultHeader().withMessageType(0b0010).withMessageTypeSpecificFlags(flags).toBytes()
	request = append(request, header...)

	// 添加序列号 (使用binary.Write确保正确处理负数)
	var seqBuffer bytes.Buffer
	binary.Write(&seqBuffer, binary.BigEndian, int32(seq))
	request = append(request, seqBuffer.Bytes()...)

	// 压缩音频数据
	payload := c.gzipCompress(segment)

	// 添加payload大小 (使用binary.Write确保正确处理)
	var payloadSizeBuffer bytes.Buffer
	binary.Write(&payloadSizeBuffer, binary.BigEndian, int32(len(payload)))
	request = append(request, payloadSizeBuffer.Bytes()...)

	// 添加payload
	request = append(request, payload...)

	return request
}

func (c *Client) recvMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.connect.ReadMessage()
			if err != nil {
				log.Printf("Read message error: %v", err)
				return
			}

			resp := c.parseResponse(message)
			log.Printf("Received response: code=%d, isLast=%v, payloadSize=%d", resp.Code, resp.IsLastPackage, resp.PayloadSize)

			if resp.Code != 0 {
				log.Printf("Recognition error: code=%d", resp.Code)
				if resp.PayloadMsg != nil && resp.PayloadMsg.Error != "" {
					log.Printf("Error message: %s", resp.PayloadMsg.Error)
				}
				return
			}

			fmt.Println("resp.PayloadMsg", resp.PayloadMsg)
			fmt.Println("--------")
			if resp.PayloadMsg != nil && resp.PayloadMsg.Result.Text != "" {
				// 转换为识别结果
				result := &models.RecognitionResult{
					Type:       "recognition_result",
					Text:       resp.PayloadMsg.Result.Text,
					IsFinal:    resp.IsLastPackage,
					Confidence: 0.95, // 默认置信度
					Timestamp:  time.Now().UnixMilli(),
				}

				log.Printf("Recognition result: %s (final: %v)", result.Text, result.IsFinal)
				if c.resultCallback != nil {
					c.resultCallback(result)
				}
			}

			if resp.IsLastPackage {
				return
			}
		}
	}
}

func (c *Client) Close() {
	if c.ticker != nil {
		c.ticker.Stop()
	}
	if c.cancel != nil {
		c.cancel()
	}
	if c.connect != nil {
		c.connect.Close()
	}
}

// Helper methods from demo code
type asrRequestHeader struct {
	messageType              byte
	messageTypeSpecificFlags byte
	serializationType        byte
	compressionType          byte
	reservedData             []byte
}

func (c *Client) defaultHeader() *asrRequestHeader {
	return &asrRequestHeader{
		messageType:              0b0001, // CLIENT_FULL_REQUEST
		messageTypeSpecificFlags: 0b0001, // POS_SEQUENCE
		serializationType:        0b0001, // JSON
		compressionType:          0b0001, // GZIP
		reservedData:             []byte{0x00},
	}
}

func (h *asrRequestHeader) withMessageType(messageType byte) *asrRequestHeader {
	h.messageType = messageType
	return h
}

func (h *asrRequestHeader) withMessageTypeSpecificFlags(flags byte) *asrRequestHeader {
	h.messageTypeSpecificFlags = flags
	return h
}

func (h *asrRequestHeader) toBytes() []byte {
	header := make([]byte, 0, 4)
	header = append(header, byte(config.ProtocolVersion<<4|1))
	header = append(header, byte(h.messageType<<4)|h.messageTypeSpecificFlags)
	header = append(header, byte(h.serializationType<<4)|h.compressionType)
	header = append(header, h.reservedData...)
	return header
}

func (c *Client) parseResponse(msg []byte) *models.AsrResponse {
	var result models.AsrResponse

	headerSize := msg[0] & 0x0f
	messageType := msg[1] >> 4
	messageTypeSpecificFlags := msg[1] & 0x0f
	serializationMethod := msg[2] >> 4
	messageCompression := msg[2] & 0x0f
	payload := msg[headerSize*4:]

	// 解析messageTypeSpecificFlags
	if messageTypeSpecificFlags&0x01 != 0 {
		result.PayloadSequence = int32(binary.BigEndian.Uint32(payload[:4]))
		payload = payload[4:]
	}
	if messageTypeSpecificFlags&0x02 != 0 {
		result.IsLastPackage = true
	}
	if messageTypeSpecificFlags&0x04 != 0 {
		result.Event = int(binary.BigEndian.Uint32(payload[:4]))
		payload = payload[4:]
	}

	// 解析messageType
	switch messageType {
	case 0b1001: // SERVER_FULL_RESPONSE
		result.PayloadSize = int(binary.BigEndian.Uint32(payload[:4]))
		payload = payload[4:]
	case 0b1111: // SERVER_ERROR_RESPONSE
		result.Code = int(binary.BigEndian.Uint32(payload[:4]))
		result.PayloadSize = int(binary.BigEndian.Uint32(payload[4:8]))
		payload = payload[8:]
	}

	if len(payload) == 0 {
		return &result
	}

	// 是否压缩
	if messageCompression == 0b0001 { // GZIP
		payload = c.gzipDecompress(payload)
	}

	// 解析payload
	var asrResponse models.AsrResponsePayload
	switch serializationMethod {
	case 0b0001: // JSON
		_ = json.Unmarshal(payload, &asrResponse)
	case 0b0000: // NO_SERIALIZATION
	}
	result.PayloadMsg = &asrResponse
	return &result
}
