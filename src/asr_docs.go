package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ASRDocHandler ASR文档处理器
type ASRDocHandler struct{}

// NewASRDocHandler 创建ASR文档处理器
func NewASRDocHandler() *ASRDocHandler {
	return &ASRDocHandler{}
}

// GetASRDocs 获取ASR流程文档
// @Summary 获取ASR处理流程文档
// @Description 获取ASR系统的完整处理流程文档，包括手动、自动、实时模式和消息类型说明
// @Tags ASR
// @Produce html
// @Success 200 {string} string "ASR流程文档HTML页面"
// @Router /api/asr/docs [get]
func (h *ASRDocHandler) GetASRDocs(c *gin.Context) {
	docHTML := h.generateASRDocsHTML()
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, docHTML)
}

// GetASRDocsJSON 获取ASR流程文档JSON格式
// @Summary 获取ASR处理流程文档(JSON)
// @Description 获取ASR系统的完整处理流程文档JSON格式数据
// @Tags ASR
// @Produce json
// @Success 200 {object} map[string]interface{} "ASR流程文档JSON数据"
// @Router /api/asr/docs/json [get]
func (h *ASRDocHandler) GetASRDocsJSON(c *gin.Context) {
	docData := h.generateASRDocsData()
	c.JSON(http.StatusOK, docData)
}

// generateASRDocsHTML 生成ASR文档HTML
func (h *ASRDocHandler) generateASRDocsHTML() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ASR语音识别系统处理流程文档</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #2c3e50;
            text-align: center;
            border-bottom: 3px solid #3498db;
            padding-bottom: 10px;
        }
        h2 {
            color: #34495e;
            border-left: 4px solid #3498db;
            padding-left: 15px;
            margin-top: 30px;
        }
        h3 {
            color: #2980b9;
            margin-top: 25px;
        }
        .mode-section {
            background: #ecf0f1;
            padding: 20px;
            margin: 15px 0;
            border-radius: 8px;
            border-left: 5px solid #3498db;
        }
        .message-type {
            background: #fff;
            border: 1px solid #bdc3c7;
            padding: 15px;
            margin: 10px 0;
            border-radius: 5px;
        }
        .code {
            background: #2c3e50;
            color: #ecf0f1;
            padding: 15px;
            border-radius: 5px;
            font-family: 'Courier New', monospace;
            overflow-x: auto;
            margin: 10px 0;
        }
        .highlight {
            background: #f39c12;
            color: white;
            padding: 2px 6px;
            border-radius: 3px;
        }
        .flow-step {
            background: #e8f5e8;
            border-left: 4px solid #27ae60;
            padding: 10px;
            margin: 8px 0;
        }
        .warning {
            background: #fdf2e9;
            border-left: 4px solid #e67e22;
            padding: 10px;
            margin: 10px 0;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 15px 0;
        }
        th, td {
            border: 1px solid #bdc3c7;
            padding: 12px;
            text-align: left;
        }
        th {
            background: #3498db;
            color: white;
        }
        tr:nth-child(even) {
            background: #f8f9fa;
        }
        .toc {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 5px;
            margin-bottom: 30px;
        }
        .toc ul {
            list-style-type: none;
            padding-left: 0;
        }
        .toc li {
            margin: 5px 0;
        }
        .toc a {
            color: #2980b9;
            text-decoration: none;
        }
        .toc a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎙️ ASR语音识别系统处理流程文档</h1>
        
        <div class="toc">
            <h3>📋 目录</h3>
            <ul>
                <li><a href="#overview">1. 系统概述</a></li>
                <li><a href="#modes">2. ASR处理模式</a></li>
                <li><a href="#messages">3. WebSocket消息类型</a></li>
                <li><a href="#flow">4. 完整处理流程</a></li>
                <li><a href="#examples">5. 使用示例</a></li>
                <li><a href="#config">6. 配置说明</a></li>
            </ul>
        </div>

        <h2 id="overview">🔍 1. 系统概述</h2>
        <p>ASR语音识别系统是一个基于WebSocket的实时语音处理系统，支持语音识别(ASR)、大语言模型对话(LLM)和语音合成(TTS)的完整语音对话流程。</p>
        
        <div class="warning">
            <strong>⚠️ 注意：</strong> 系统通过WebSocket协议进行通信，默认运行在8000端口。所有消息均为JSON格式。
        </div>

        <h3>🏗️ 系统架构</h3>
        <ul>
            <li><strong>WebSocket服务器：</strong> 处理客户端连接和消息路由</li>
            <li><strong>ASR提供者：</strong> 支持DoubaoASR、GoSherpaASR、DeepgramSST</li>
            <li><strong>LLM提供者：</strong> 支持QwenLLM、OpenAI、Ollama等</li>
            <li><strong>TTS提供者：</strong> 支持DoubaoTTS、EdgeTTS、GoSherpaTTS等</li>
            <li><strong>连接管理器：</strong> 管理客户端会话和状态</li>
        </ul>

        <h2 id="modes">⚙️ 2. ASR处理模式</h2>
        
        <div class="mode-section">
            <h3>🔧 手动模式 (Manual)</h3>
            <p><span class="highlight">clientListenMode = "manual"</span></p>
            <ul>
                <li>用户手动控制录音开始和停止</li>
                <li>通过发送 <code>listen</code> 消息的 <code>start</code> 和 <code>stop</code> 状态控制</li>
                <li>只有在收到 <code>stop</code> 状态且有ASR文本时才处理对话</li>
                <li>适合需要精确控制录音时机的场景</li>
            </ul>
            
            <div class="flow-step">
                <strong>处理流程：</strong><br>
                1. 发送 listen(start) → 开始录音和ASR识别<br>
                2. 持续积累ASR识别结果<br>
                3. 发送 listen(stop) → 停止录音<br>
                4. 如果有完整ASR文本，则发送给LLM处理
            </div>
        </div>

        <div class="mode-section">
            <h3>🤖 自动模式 (Auto)</h3>
            <p><span class="highlight">clientListenMode = "auto"</span></p>
            <ul>
                <li>系统自动检测语音结束点</li>
                <li>一旦ASR识别到完整语句立即处理</li>
                <li>无需手动控制，适合连续对话场景</li>
                <li>检测到连续两次静音会自动结束对话</li>
            </ul>
            
            <div class="flow-step">
                <strong>处理流程：</strong><br>
                1. ASR持续监听语音输入<br>
                2. 识别到完整语句时立即返回true停止识别<br>
                3. 直接发送识别结果给LLM处理<br>
                4. 连续两次静音时自动结束对话
            </div>
        </div>

        <div class="mode-section">
            <h3>⚡ 实时模式 (Realtime)</h3>
            <p><span class="highlight">clientListenMode = "realtime"</span></p>
            <ul>
                <li>实时响应，打断式对话</li>
                <li>识别到语音时立即停止当前TTS播放</li>
                <li>重置ASR状态准备下一次识别</li>
                <li>适合需要快速响应的交互场景</li>
            </ul>
            
            <div class="flow-step">
                <strong>处理流程：</strong><br>
                1. ASR持续监听语音输入<br>
                2. 识别到语音时立即停止服务器语音播放<br>
                3. 重置ASR状态准备下一次识别<br>
                4. 发送识别结果给LLM处理
            </div>
        </div>

        <h2 id="messages">📨 3. WebSocket消息类型</h2>
        
        <h3>📤 客户端发送消息</h3>
        
        <div class="message-type">
            <h4>🤝 hello - 建立连接</h4>
            <div class="code">{
  "type": "hello",
  "device_id": "设备ID",
  "audio_params": {
    "sample_rate": 16000,
    "channels": 1,
    "format": "pcm"
  }
}</div>
            <p><strong>功能：</strong> 建立WebSocket会话，获取session_id</p>
        </div>

        <div class="message-type">
            <h4>🎙️ listen - 语音控制</h4>
            <div class="code">{
  "type": "listen",
  "state": "start|stop|detect",
  "mode": "manual|auto|realtime",
  "text": "文本内容(仅detect状态)"
}</div>
            <p><strong>功能：</strong> 控制ASR监听状态和模式</p>
            <ul>
                <li><strong>start：</strong> 开始ASR监听</li>
                <li><strong>stop：</strong> 停止ASR监听</li>
                <li><strong>detect：</strong> 文本检测模式，直接处理文本</li>
            </ul>
        </div>

        <div class="message-type">
            <h4>💬 chat - 文本对话</h4>
            <div class="code">{
  "type": "chat",
  "text": "用户输入的文本内容"
}</div>
            <p><strong>功能：</strong> 直接发送文本消息进行对话</p>
        </div>

        <div class="message-type">
            <h4>🛑 abort - 中止对话</h4>
            <div class="code">{
  "type": "abort"
}</div>
            <p><strong>功能：</strong> 中止当前对话，重置所有状态</p>
        </div>

        <h3>📥 服务器返回消息</h3>
        
        <div class="message-type">
            <h4>🎯 stt - 语音识别结果</h4>
            <div class="code">{
  "type": "stt",
  "text": "识别到的文本内容"
}</div>
            <p><strong>功能：</strong> 返回ASR语音识别的结果</p>
        </div>

        <div class="message-type">
            <h4>🤖 llm - AI回复</h4>
            <div class="code">{
  "type": "llm",
  "text": "AI回复内容",
  "emotion": "情绪状态"
}</div>
            <p><strong>功能：</strong> 返回大语言模型的回复内容</p>
        </div>

        <div class="message-type">
            <h4>🔊 tts - 语音合成状态</h4>
            <div class="code">{
  "type": "tts",
  "state": "start|text|audio|end|error",
  "text": "合成的文本",
  "text_index": 0
}</div>
            <p><strong>功能：</strong> TTS语音合成的状态更新</p>
            <ul>
                <li><strong>start：</strong> TTS服务启动</li>
                <li><strong>text：</strong> 开始合成指定文本</li>
                <li><strong>audio：</strong> 音频数据准备就绪</li>
                <li><strong>end：</strong> TTS合成完成</li>
                <li><strong>error：</strong> TTS合成错误</li>
            </ul>
        </div>

        <div class="message-type">
            <h4>❌ error - 错误信息</h4>
            <div class="code">{
  "type": "error",
  "message": "错误描述",
  "code": "错误代码"
}</div>
            <p><strong>功能：</strong> 返回系统错误信息</p>
        </div>

        <div class="message-type">
            <h4>📊 status - 状态更新</h4>
            <div class="code">{
  "type": "status",
  "status": "connecting|connected|disconnected|processing|ready",
  "message": "状态描述"
}</div>
            <p><strong>功能：</strong> 系统状态更新通知</p>
        </div>

        <h2 id="flow">🔄 4. 完整处理流程</h2>
        
        <h3>📋 标准语音对话流程</h3>
        <table>
            <tr>
                <th>步骤</th>
                <th>客户端操作</th>
                <th>服务器响应</th>
                <th>说明</th>
            </tr>
            <tr>
                <td>1</td>
                <td>发送 hello 消息</td>
                <td>返回 session_id</td>
                <td>建立会话连接</td>
            </tr>
            <tr>
                <td>2</td>
                <td>发送 listen(start) + mode</td>
                <td>开始ASR监听</td>
                <td>设置监听模式并开始录音</td>
            </tr>
            <tr>
                <td>3</td>
                <td>发送音频数据流</td>
                <td>实时ASR识别</td>
                <td>持续语音识别处理</td>
            </tr>
            <tr>
                <td>4</td>
                <td>发送 listen(stop) 或自动检测</td>
                <td>返回 stt 消息</td>
                <td>完成语音识别</td>
            </tr>
            <tr>
                <td>5</td>
                <td>等待AI处理</td>
                <td>返回 llm 消息</td>
                <td>大语言模型生成回复</td>
            </tr>
            <tr>
                <td>6</td>
                <td>接收音频流</td>
                <td>发送 tts 状态 + 音频数据</td>
                <td>语音合成并播放</td>
            </tr>
        </table>

        <h2 id="examples">💡 5. 使用示例</h2>
        
        <h3>🔧 手动模式示例</h3>
        <div class="code">// 1. 建立连接
ws.send(JSON.stringify({
  "type": "hello",
  "device_id": "client_001"
}));

// 2. 开始手动录音
ws.send(JSON.stringify({
  "type": "listen",
  "state": "start",
  "mode": "manual"
}));

// 3. 发送音频数据...

// 4. 停止录音
ws.send(JSON.stringify({
  "type": "listen",
  "state": "stop"
}));</div>

        <h3>🤖 自动模式示例</h3>
        <div class="code">// 1. 建立连接
ws.send(JSON.stringify({
  "type": "hello",
  "device_id": "client_002"
}));

// 2. 开始自动模式
ws.send(JSON.stringify({
  "type": "listen",
  "state": "start",
  "mode": "auto"
}));

// 3. 发送音频数据，系统自动检测结束点</div>

        <h3>💬 文本对话示例</h3>
        <div class="code">// 直接文本对话
ws.send(JSON.stringify({
  "type": "chat",
  "text": "你好，今天天气怎么样？"
}));</div>

        <h2 id="config">⚙️ 6. 配置说明</h2>
        
        <h3>📝 ASR提供者配置</h3>
        <div class="code">selected_module:
  ASR: DoubaoASR  # 可选: DoubaoASR, GoSherpaASR, DeepgramSST
  TTS: DoubaoTTS  # 可选: DoubaoTTS, EdgeTTS, GoSherpaTTS
  LLM: QwenLLM    # 可选: QwenLLM, OpenAI, Ollama

ASR:
  DoubaoASR:
    type: doubao
    appid: "your_app_id"
    access_token: "your_access_token"
    output_dir: tmp/</div>

        <div class="warning">
            <strong>⚠️ 重要提示：</strong>
            <ul>
                <li>确保WebSocket连接稳定，网络中断会影响实时性能</li>
                <li>音频格式建议使用PCM 16kHz 单声道</li>
                <li>大文件传输时注意分片处理</li>
                <li>生产环境建议配置SSL/TLS加密</li>
            </ul>
        </div>

        <hr>
        <p style="text-align: center; color: #7f8c8d; margin-top: 30px;">
            📚 ASR语音识别系统文档 | 版本: 1.0 | 更新时间: 2024年
        </p>
    </div>
</body>
</html>`
}

// generateASRDocsData 生成ASR文档数据
func (h *ASRDocHandler) generateASRDocsData() map[string]interface{} {
	return map[string]interface{}{
		"title": "ASR语音识别系统处理流程文档",
		"version": "1.0",
		"overview": map[string]interface{}{
			"description": "ASR语音识别系统是一个基于WebSocket的实时语音处理系统，支持语音识别(ASR)、大语言模型对话(LLM)和语音合成(TTS)的完整语音对话流程。",
			"websocket_port": 8000,
			"protocol": "WebSocket",
			"message_format": "JSON",
		},
		"modes": map[string]interface{}{
			"manual": map[string]interface{}{
				"name": "手动模式",
				"description": "用户手动控制录音开始和停止",
				"control_method": "通过listen消息的start/stop状态控制",
				"use_case": "需要精确控制录音时机的场景",
				"flow": []string{
					"发送 listen(start) → 开始录音和ASR识别",
					"持续积累ASR识别结果",
					"发送 listen(stop) → 停止录音",
					"如果有完整ASR文本，则发送给LLM处理",
				},
			},
			"auto": map[string]interface{}{
				"name": "自动模式",
				"description": "系统自动检测语音结束点",
				"control_method": "ASR自动检测完整语句",
				"use_case": "连续对话场景",
				"flow": []string{
					"ASR持续监听语音输入",
					"识别到完整语句时立即返回true停止识别",
					"直接发送识别结果给LLM处理",
					"连续两次静音时自动结束对话",
				},
			},
			"realtime": map[string]interface{}{
				"name": "实时模式",
				"description": "实时响应，打断式对话",
				"control_method": "实时检测并打断当前播放",
				"use_case": "需要快速响应的交互场景",
				"flow": []string{
					"ASR持续监听语音输入",
					"识别到语音时立即停止服务器语音播放",
					"重置ASR状态准备下一次识别",
					"发送识别结果给LLM处理",
				},
			},
		},
		"message_types": map[string]interface{}{
			"client_messages": map[string]interface{}{
				"hello": map[string]interface{}{
					"description": "建立WebSocket会话",
					"fields": map[string]string{
						"type": "消息类型，固定为hello",
						"device_id": "设备ID",
						"audio_params": "音频参数配置",
					},
					"example": `{"type": "hello", "device_id": "client_001", "audio_params": {"sample_rate": 16000, "channels": 1, "format": "pcm"}}`,
				},
				"listen": map[string]interface{}{
					"description": "控制ASR监听状态和模式",
					"fields": map[string]string{
						"type": "消息类型，固定为listen",
						"state": "状态：start|stop|detect",
						"mode": "模式：manual|auto|realtime",
						"text": "文本内容(仅detect状态使用)",
					},
					"example": `{"type": "listen", "state": "start", "mode": "manual"}`,
				},
				"chat": map[string]interface{}{
					"description": "直接发送文本消息进行对话",
					"fields": map[string]string{
						"type": "消息类型，固定为chat",
						"text": "用户输入的文本内容",
					},
					"example": `{"type": "chat", "text": "你好，今天天气怎么样？"}`,
				},
				"abort": map[string]interface{}{
					"description": "中止当前对话，重置所有状态",
					"fields": map[string]string{
						"type": "消息类型，固定为abort",
					},
					"example": `{"type": "abort"}`,
				},
			},
			"server_messages": map[string]interface{}{
				"stt": map[string]interface{}{
					"description": "返回ASR语音识别的结果",
					"fields": map[string]string{
						"type": "消息类型，固定为stt",
						"text": "识别到的文本内容",
					},
					"example": `{"type": "stt", "text": "你好，今天天气怎么样？"}`,
				},
				"llm": map[string]interface{}{
					"description": "返回大语言模型的回复内容",
					"fields": map[string]string{
						"type": "消息类型，固定为llm",
						"text": "AI回复内容",
						"emotion": "情绪状态(可选)",
					},
					"example": `{"type": "llm", "text": "今天天气很好，阳光明媚！", "emotion": "happy"}`,
				},
				"tts": map[string]interface{}{
					"description": "TTS语音合成的状态更新",
					"fields": map[string]string{
						"type": "消息类型，固定为tts",
						"state": "状态：start|text|audio|end|error",
						"text": "合成的文本内容",
						"text_index": "文本索引",
					},
					"example": `{"type": "tts", "state": "text", "text": "今天天气很好", "text_index": 0}`,
				},
				"error": map[string]interface{}{
					"description": "返回系统错误信息",
					"fields": map[string]string{
						"type": "消息类型，固定为error",
						"message": "错误描述",
						"code": "错误代码(可选)",
					},
					"example": `{"type": "error", "message": "ASR服务连接失败", "code": "ASR_001"}`,
				},
				"status": map[string]interface{}{
					"description": "系统状态更新通知",
					"fields": map[string]string{
						"type": "消息类型，固定为status",
						"status": "状态：connecting|connected|disconnected|processing|ready",
						"message": "状态描述(可选)",
					},
					"example": `{"type": "status", "status": "connected", "message": "服务连接成功"}`,
				},
			},
		},
		"workflow": []map[string]interface{}{
			{
				"step": 1,
				"client_action": "发送 hello 消息",
				"server_response": "返回 session_id",
				"description": "建立会话连接",
			},
			{
				"step": 2,
				"client_action": "发送 listen(start) + mode",
				"server_response": "开始ASR监听",
				"description": "设置监听模式并开始录音",
			},
			{
				"step": 3,
				"client_action": "发送音频数据流",
				"server_response": "实时ASR识别",
				"description": "持续语音识别处理",
			},
			{
				"step": 4,
				"client_action": "发送 listen(stop) 或自动检测",
				"server_response": "返回 stt 消息",
				"description": "完成语音识别",
			},
			{
				"step": 5,
				"client_action": "等待AI处理",
				"server_response": "返回 llm 消息",
				"description": "大语言模型生成回复",
			},
			{
				"step": 6,
				"client_action": "接收音频流",
				"server_response": "发送 tts 状态 + 音频数据",
				"description": "语音合成并播放",
			},
		},
		"configuration": map[string]interface{}{
			"websocket_url": "ws://localhost:8000/",
			"supported_asr_providers": []string{"DoubaoASR", "GoSherpaASR", "DeepgramSST"},
			"supported_tts_providers": []string{"DoubaoTTS", "EdgeTTS", "GoSherpaTTS"},
			"supported_llm_providers": []string{"QwenLLM", "OpenAI", "Ollama"},
			"audio_format": map[string]interface{}{
				"recommended": "PCM 16kHz 单声道",
				"sample_rate": 16000,
				"channels": 1,
				"format": "pcm",
			},
		},
		"notes": []string{
			"确保WebSocket连接稳定，网络中断会影响实时性能",
			"音频格式建议使用PCM 16kHz 单声道",
			"大文件传输时注意分片处理",
			"生产环境建议配置SSL/TLS加密",
		},
	}
}

// RegisterASRDocsRoutes 注册ASR文档路由
func RegisterASRDocsRoutes(router *gin.RouterGroup) {
	handler := NewASRDocHandler()
	
	// HTML格式文档
	router.GET("/asr/docs", handler.GetASRDocs)
	
	// JSON格式文档
	router.GET("/asr/docs/json", handler.GetASRDocsJSON)
}