<template>
  <div class="asr-tts-demo">
    <div class="header">
      <h1>ASR-TTS 语音对话演示</h1>
      <p class="description">实时语音识别与语音合成对话系统</p>
    </div>

    <div class="main-content">
      <!-- 连接状态 -->
      <div class="connection-status" :class="{ connected: wsConnected, disconnected: !wsConnected }">
        <div class="status-indicator"></div>
        <span>{{ wsConnected ? 'WebSocket 已连接' : 'WebSocket 未连接' }}</span>
      </div>

      <!-- WebSocket连接配置 -->
      <div class="connection-config" v-if="!wsConnected">
        <h3>连接配置</h3>
        <div class="config-form">
          <div class="config-row">
            <label for="deviceId">设备ID:</label>
            <input 
              id="deviceId"
              type="text" 
              v-model="headerConfig.deviceId" 
              placeholder="设备唯一标识"
              class="config-input"
            />
          </div>
          <div class="config-row">
            <label for="clientId">客户端ID:</label>
            <input 
              id="clientId"
              type="text" 
              v-model="headerConfig.clientId" 
              placeholder="客户端标识"
              class="config-input"
            />
          </div>
          <div class="config-row">
            <label for="sessionId">会话ID:</label>
            <input 
              id="sessionId"
              type="text" 
              v-model="headerConfig.sessionId" 
              placeholder="会话标识（可选）"
              class="config-input"
            />
          </div>
          <div class="config-row">
            <label for="transportType">传输类型:</label>
            <select id="transportType" v-model="headerConfig.transportType" class="config-select">
              <option value="websocket">WebSocket</option>
              <option value="http">HTTP</option>
            </select>
          </div>
          <div class="config-row">
            <label for="token">访问令牌:</label>
            <input 
              id="token"
              type="text" 
              v-model="headerConfig.token" 
              placeholder="请输入访问令牌"
              class="config-input"
            />
          </div>
          <div class="config-actions">
            <button @click="connectWithHeaders" class="btn-connect">连接服务器</button>
            <button @click="resetHeaderConfig" class="btn-reset">重置配置</button>
          </div>
        </div>
      </div>

      <!-- 会话信息 -->
      <div class="session-info" v-if="sessionId">
        <p><strong>会话ID:</strong> {{ sessionId }}</p>
        <p><strong>设备ID:</strong> {{ headerConfig.deviceId }}</p>
        <p><strong>客户端ID:</strong> {{ headerConfig.clientId }}</p>
        <p><strong>传输类型:</strong> {{ headerConfig.transportType }}</p>
      </div>

      <!-- 音频控制区域 -->
      <div class="audio-controls">
        <div class="recording-section">
          <h3>语音输入</h3>
          
          <!-- 麦克风选择 -->
          <div class="device-selector">
            <label for="audioDevice">选择麦克风:</label>
            <select id="audioDevice" v-model="selectedDevice" @change="updateAudioDevice" :disabled="isRecording">
              <option value="">默认设备</option>
              <option v-for="device in audioInputs" :key="device.deviceId" :value="device.deviceId">
                {{ device.label || `麦克风 ${device.deviceId.slice(0, 8)}` }}
              </option>
            </select>
          </div>
          
          <div class="recording-controls">
            <button 
              @click="toggleRecording" 
              :disabled="!wsConnected"
              :class="{ 
                'btn-record': !isRecording, 
                'btn-stop': isRecording,
                'disabled': !wsConnected 
              }"
            >
              {{ isRecording ? '停止录音' : '开始录音' }}
            </button>
            
            <!-- ASR监听控制按钮 -->
            <div class="listen-controls" v-if="isRecording">
              <button 
                @click="startListening" 
                :disabled="!wsConnected || !isRecording || isListening"
                class="btn-listen"
              >
                开始监听
              </button>
              <button 
                @click="stopListening" 
                :disabled="!wsConnected || !isRecording || !isListening"
                class="btn-stop-listen"
              >
                终止监听
              </button>
            </div>
            
            <button 
              @click="abortChat" 
              :disabled="!wsConnected || (!isRecording && ttsStatus === 'idle')"
              class="btn-abort"
            >
              中止对话
            </button>
            <div class="recording-status" v-if="isRecording">
              <div class="recording-indicator"></div>
              <span>正在录音...</span>
              <span v-if="isListening" class="listening-status">（监听中）</span>
            </div>
          </div>
          
          <!-- ASR 结果显示 -->
          <div class="asr-result" v-if="asrText">
            <h4>识别结果:</h4>
            <p class="asr-text">{{ asrText }}</p>
          </div>
        </div>

        <div class="playback-section">
          <h3>语音输出</h3>
          <div class="tts-status">
            <div class="status-item">
              <span class="label">TTS状态:</span>
              <span class="value" :class="ttsStatus">{{ getTtsStatusText() }}</span>
            </div>
            <div class="status-item" v-if="currentTtsText">
              <span class="label">当前合成:</span>
              <span class="value">{{ currentTtsText }}</span>
            </div>
          </div>
          
          <!-- LLM 回复结果显示 -->
          <div class="llm-result" v-if="llmText">
            <h4>AI回复:</h4>
            <p class="llm-text">{{ llmText }}</p>
          </div>
          
          <!-- 音频播放控制 -->
          <div class="audio-player" v-if="hasAudio || isAudioPlaying || ttsStatus === 'paused'">
            <div class="audio-controls">
              <button @click="toggleAudio" class="play-toggle-btn">
                <span v-if="!currentAudio">▶️ 播放</span>
                <span v-else-if="isAudioPlaying">⏸️ 暂停</span>
                <span v-else>▶️ 继续</span>
              </button>
              <button @click="stopAudio" :disabled="!currentAudio" class="stop-btn">
                ⏹️ 停止
              </button>
              <button @click="testAudioPlayback" class="test-btn">
                🔊 测试音频
              </button>
            </div>
            
            <!-- 音量控制 -->
            <div class="volume-control">
              <label>音量:</label>
              <input 
                type="range" 
                min="0" 
                max="1" 
                step="0.1" 
                :value="currentAudio ? currentAudio.volume : 0.8"
                @input="setAudioVolume($event.target.value)"
                class="volume-slider"
              />
              <span class="volume-value">{{ Math.round((currentAudio ? currentAudio.volume : 0.8) * 100) }}%</span>
            </div>
            
            <!-- 播放状态显示 -->
            <div class="audio-status">
              <span class="status-text">
                状态: {{ ttsStatus === 'playing' ? '播放中' : ttsStatus === 'paused' ? '已暂停' : ttsStatus === 'loading' ? '加载中' : '就绪' }}
              </span>
              <span v-if="audioChunks.length > 0" class="chunks-info">
                音频块: {{ audioChunks.length }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- MCP 工具管理区域 -->
      <div class="mcp-section">
        <h3>MCP 工具管理</h3>
        
        <!-- MCP 状态显示 -->
        <div class="mcp-status">
          <div class="status-item">
            <span class="label">MCP状态:</span>
            <span class="value" :class="mcpStatus">{{ getMcpStatusText() }}</span>
          </div>
          <div class="status-item" v-if="mcpTools.length > 0">
            <span class="label">可用工具:</span>
            <span class="value">{{ mcpTools.length }} 个</span>
          </div>
        </div>

        <!-- MCP 工具列表 -->
        <div class="mcp-tools-list" v-if="mcpTools.length > 0">
          <h4>可用工具列表</h4>
          <div class="tools-grid">
            <div 
              v-for="(tool, index) in mcpTools" 
              :key="index" 
              class="tool-card"
              :class="{ 'tool-selected': selectedTool === tool }"
              @click="selectTool(tool)"
            >
              <div class="tool-header">
                <span class="tool-name">{{ tool.name }}</span>
                <button 
                  @click.stop="callTool(tool)" 
                  :disabled="!wsConnected || mcpStatus !== 'ready'"
                  class="btn-call-tool"
                >
                  调用
                </button>
              </div>
              <div class="tool-description">{{ tool.description }}</div>
              <div class="tool-params" v-if="tool.inputSchema && tool.inputSchema.properties">
                <span class="params-label">参数:</span>
                <span class="params-list">
                  {{ Object.keys(tool.inputSchema.properties).join(', ') }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- MCP 工具调用界面 -->
        <div class="mcp-tool-call" v-if="selectedTool">
          <h4>调用工具: {{ selectedTool.name }}</h4>
          <div class="tool-call-form">
            <div class="tool-description">{{ selectedTool.description }}</div>
            
            <!-- 动态参数输入 -->
            <div class="tool-params-input" v-if="selectedTool.inputSchema && selectedTool.inputSchema.properties">
              <div 
                v-for="(param, paramName) in selectedTool.inputSchema.properties" 
                :key="paramName"
                class="param-input-group"
              >
                <label :for="'param-' + paramName">{{ paramName }}:</label>
                <input 
                  :id="'param-' + paramName"
                  type="text" 
                  v-model="toolCallParams[paramName]" 
                  :placeholder="param.description || '请输入' + paramName"
                  :required="selectedTool.inputSchema.required && selectedTool.inputSchema.required.includes(paramName)"
                  class="param-input"
                />
                <span class="param-description" v-if="param.description">{{ param.description }}</span>
              </div>
            </div>
            
            <!-- 调用按钮 -->
            <div class="tool-call-actions">
              <button 
                @click="callSelectedTool" 
                :disabled="!wsConnected || mcpStatus !== 'ready' || isCallingTool"
                class="btn-call-selected-tool"
              >
                {{ isCallingTool ? '调用中...' : '执行工具' }}
              </button>
              <button @click="clearSelectedTool" class="btn-clear-tool">
                清除选择
              </button>
            </div>
          </div>
        </div>

        <!-- MCP 工具调用结果 -->
        <div class="mcp-tool-result" v-if="lastToolResult">
          <h4>工具调用结果</h4>
          <div class="result-content">
            <div class="result-header">
              <span class="result-tool">工具: {{ lastToolResult.toolName }}</span>
              <span class="result-time">时间: {{ formatTime(lastToolResult.timestamp) }}</span>
            </div>
            <div class="result-data">
              <pre>{{ JSON.stringify(lastToolResult.result, null, 2) }}</pre>
            </div>
          </div>
        </div>

        <!-- MCP 快捷操作 -->
        <div class="mcp-quick-actions">
          <h4>快捷操作</h4>
          <div class="quick-actions-grid">
            <button 
              @click="sendMcpQuickCommand('帮我拍照')"
              :disabled="!wsConnected || mcpStatus !== 'ready'"
              class="btn-quick-action"
            >
              📷 拍照
            </button>
            <button 
              @click="sendMcpQuickCommand('调整音量到50%')"
              :disabled="!wsConnected || mcpStatus !== 'ready'"
              class="btn-quick-action"
            >
              🔊 调整音量
            </button>
            <button 
              @click="sendMcpQuickCommand('调整屏幕亮度')"
              :disabled="!wsConnected || mcpStatus !== 'ready'"
              class="btn-quick-action"
            >
              💡 调整亮度
            </button>
            <button 
              @click="sendMcpQuickCommand('切换主题')"
              :disabled="!wsConnected || mcpStatus !== 'ready'"
              class="btn-quick-action"
            >
              🎨 切换主题
            </button>
          </div>
        </div>
      </div>

      <!-- 文本聊天区域 -->
      <div class="text-chat-section">
        <h3>文本对话</h3>
        <div class="chat-input-area">
          <div class="input-group">
            <input 
              type="text" 
              v-model="chatInput" 
              @keyup.enter="sendChatMessage"
              :disabled="!wsConnected || ttsStatus === 'loading'"
              placeholder="输入消息并按回车发送..."
              class="chat-input"
            />
            <button 
              @click="sendChatMessage" 
              :disabled="!wsConnected || !chatInput.trim() || ttsStatus === 'loading'"
              class="btn-send"
            >
              发送
            </button>
          </div>
          <div class="chat-options">
            <label>
              <input type="checkbox" v-model="enableTextDetect" />
              启用文本检测模式
            </label>
          </div>
        </div>
      </div>

      <!-- 图片上传区域 
      <div class="image-upload-section">
        <h3>图片上传</h3>
        <div class="upload-area">
          <input 
            type="file" 
            ref="imageInput"
            @change="handleImageSelect"
            accept="image/*"
            style="display: none;"
          />
          <div class="upload-controls">
            <button 
              @click="selectImage" 
              :disabled="!wsConnected || ttsStatus === 'loading'"
              class="btn-select-image"
            >
              选择图片
            </button>
            <button 
              @click="sendImageMessage" 
              :disabled="!wsConnected || !selectedImage || ttsStatus === 'loading'"
              class="btn-send-image"
            >
              发送图片
            </button>
          </div>
          <div v-if="selectedImage" class="image-preview">
            <img :src="imagePreviewUrl" alt="预览图片" class="preview-img" />
            <div class="image-info">
              <p>文件名: {{ selectedImage.name }}</p>
              <p>大小: {{ formatFileSize(selectedImage.size) }}</p>
            </div>
          </div>
        </div>
      </div>-->

      <!-- 视觉功能区域
      <div class="vision-section">
        <h3>视觉功能</h3>
        <div class="vision-controls">
          <div class="vision-buttons">
            <button 
              @click="sendVisionMessage('gen_pic')"
              :disabled="!wsConnected || ttsStatus === 'loading'"
              class="btn-vision"
            >
              生成图片
            </button>
            <button 
              @click="sendVisionMessage('gen_video')"
              :disabled="!wsConnected || ttsStatus === 'loading'"
              class="btn-vision"
            >
              生成视频
            </button>
            <button 
              @click="sendVisionMessage('read_img')"
              :disabled="!wsConnected || ttsStatus === 'loading'"
              class="btn-vision"
            >
              读取图片
            </button>
          </div>
          <div class="vision-input">
            <input 
              type="text" 
              v-model="visionPrompt" 
              @keyup.enter="sendVisionWithPrompt"
              :disabled="!wsConnected || ttsStatus === 'loading'"
              placeholder="输入视觉相关的提示词..."
              class="vision-prompt-input"
            />
            <button 
              @click="sendVisionWithPrompt" 
              :disabled="!wsConnected || !visionPrompt.trim() || ttsStatus === 'loading'"
              class="btn-send-vision"
            >
              发送
            </button>
          </div>
        </div>
      </div> -->

      <!-- IoT设备控制区域
      <div class="iot-section">
        <h3>IoT设备控制</h3>
        <div class="iot-controls">
          <div class="iot-input-group">
            <div class="input-row">
              <label>设备描述符:</label>
              <textarea 
                v-model="iotDescriptors" 
                :disabled="!wsConnected || ttsStatus === 'loading'"
                placeholder="输入设备描述符JSON数组，例如: [{'device_id': 'light1', 'type': 'light'}]"
                class="iot-textarea"
                rows="3"
              ></textarea>
            </div>
            <div class="input-row">
              <label>设备状态:</label>
              <textarea 
                v-model="iotStates" 
                :disabled="!wsConnected || ttsStatus === 'loading'"
                placeholder="输入设备状态JSON数组，例如: [{'device_id': 'light1', 'state': 'on'}]"
                class="iot-textarea"
                rows="3"
              ></textarea>
            </div>
          </div>
          <div class="iot-buttons">
            <button 
              @click="sendIotMessage" 
              :disabled="!wsConnected || (!iotDescriptors.trim() && !iotStates.trim()) || ttsStatus === 'loading'"
              class="btn-send-iot"
            >
              发送IoT消息
            </button>
            <button 
              @click="clearIotInputs" 
              :disabled="!iotDescriptors.trim() && !iotStates.trim()"
              class="btn-clear-iot"
            >
              清空输入
            </button>
          </div>
        </div>
      </div> -->

      <!-- 对话历史 -->
      <div class="conversation-history">
        <h3>对话历史</h3>
        <div class="messages" ref="messagesContainer">
          <div 
            v-for="(message, index) in messages" 
            :key="index" 
            :class="['message', message.type]"
          >
            <div class="message-header">
              <span class="message-type">{{ getMessageTypeText(message.type) }}</span>
              <span class="message-time">{{ formatTime(message.timestamp) }}</span>
            </div>
            <div class="message-content">{{ message.content }}</div>
          </div>
        </div>
      </div>

      <!-- 调试信息 -->
      <div class="debug-info" v-if="showDebug">
        <h3>调试信息</h3>
        <div class="debug-content">
          <p><strong>音频格式:</strong> {{ audioFormat }}</p>
          <p><strong>采样率:</strong> {{ sampleRate }}Hz</p>
          <p><strong>声道数:</strong> {{ channels }}</p>
          <p><strong>帧时长:</strong> {{ frameDuration }}ms</p>
          <p><strong>接收到的音频块数:</strong> {{ audioChunksCount }}</p>
        </div>
      </div>
    </div>

    <!-- 设置面板 -->
    <div class="settings-panel">
      <button @click="showDebug = !showDebug" class="debug-toggle">
        {{ showDebug ? '隐藏调试' : '显示调试' }}
      </button>
      <button @click="clearHistory" class="clear-history">
        清空历史
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import PCMPlayer from 'pcm-player'

// WebSocket相关
const wsRef = ref(null)
const wsConnected = ref(false)
const sessionId = ref('')
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 5
const reconnectDelay = ref(3000) // 初始重连延迟3秒

// WebSocket Header配置
const headerConfig = reactive({
  deviceId: 'web-client-12333',
  clientId: 'web-' + Math.random().toString(36).substr(2, 9),
  sessionId: '',
  transportType: 'websocket',
  token: ''
})

// 音频相关
const isRecording = ref(false)
const mediaRecorder = ref(null)
const audioStream = ref(null)
const audioContext = ref(null)
const audioProcessor = ref(null)

// 音频设备相关
const audioInputs = ref([])
const selectedDevice = ref('')

// 音频参数（严格按照后端协议）
const audioFormat = ref('pcm')
const sampleRate = ref(16000)  // 匹配后端AudioToPCMData的目标采样率
const channels = ref(1)
const frameDuration = ref(60)

// ASR 相关
const asrText = ref('')
const isListening = ref(false) // ASR监听状态

// TTS 相关
const ttsStatus = ref('idle') // idle, loading, playing, error, paused
const currentTtsText = ref('')
const hasAudio = ref(false)
const isAudioPlaying = ref(false)
const audioChunks = ref([])
const currentAudio = ref(null)
const audioChunksCount = ref(0)

// PCM播放器实例
const pcmPlayer = ref(null)

// LLM 相关
const llmText = ref('')

// 文本聊天相关
const chatInput = ref('')
const enableTextDetect = ref(false)

// 图片上传相关
const imageInput = ref(null)
const selectedImage = ref(null)
const imagePreviewUrl = ref('')

// 视觉功能相关
const visionPrompt = ref('')

// IoT设备控制相关
const iotDescriptors = ref('')
const iotStates = ref('')

// MCP 相关
const mcpStatus = ref('disconnected') // disconnected, connecting, ready, error
const mcpTools = ref([]) // 可用的MCP工具列表
const selectedTool = ref(null) // 当前选中的工具
const toolCallParams = ref({}) // 工具调用参数
const isCallingTool = ref(false) // 是否正在调用工具
const lastToolResult = ref(null) // 最后一次工具调用结果

// 对话历史
const messages = ref([])
const messagesContainer = ref(null)

// 调试
const showDebug = ref(false)

/**
 * 连接WebSocket服务器
 */
const connectWebSocket = () => {
  // 检查重连次数限制
  if (reconnectAttempts.value >= maxReconnectAttempts) {
    console.error('已达到最大重连次数，停止重连')
    addMessage('error', `连接失败，已尝试 ${maxReconnectAttempts} 次重连`)
    return
  }
  
  // 构建WebSocket URL，通过查询参数传递header信息
  // const baseUrl = 'ws://localhost:8000/'
  const baseUrl = 'ws://localhost:8081/'
  const params = new URLSearchParams()
  
  // 添加header参数
  if (headerConfig.deviceId) params.append('device-id', headerConfig.deviceId)
  if (headerConfig.clientId) params.append('client-id', headerConfig.clientId)
  if (headerConfig.sessionId) params.append('session-id', headerConfig.sessionId)
  if (headerConfig.transportType) params.append('transport-type', headerConfig.transportType)
  if (headerConfig.token) params.append('token', headerConfig.token)
  
  const wsUrl = params.toString() ? `${baseUrl}?${params.toString()}` : baseUrl
  console.log('连接WebSocket URL:', wsUrl)
  
  const ws = new WebSocket(wsUrl)
  wsRef.value = ws // 将 WebSocket 实例保存到 ref 中
  
  // 设置请求头（通过URL参数或连接后发送）
  ws.onopen = () => {
    console.log('WebSocket连接已建立')
    wsConnected.value = true
    
    // 重置重连计数
    reconnectAttempts.value = 0
    reconnectDelay.value = 3000 // 重置延迟时间
    
    // 发送hello消息，建立会话
    sendHelloMessage()
    
    addMessage('system', 'WebSocket连接已建立')
    message.success('WebSocket连接成功')
  }
  
  ws.onmessage = (event) => {
    if (event.data instanceof ArrayBuffer || event.data instanceof Blob) {
      // 处理二进制音频数据
      handleAudioData(event.data)
    } else {
      // 处理文本消息
      handleWebSocketMessage(event.data)
    }
  }
  
  ws.onclose = (event) => {
    console.log('WebSocket连接已关闭', event)
    wsConnected.value = false
    
    const reason = event.reason || '未知原因'
    const code = event.code || 0
    
    addMessage('system', `WebSocket连接已关闭 (代码: ${code}, 原因: ${reason})`)
    
    // 根据关闭代码判断是否需要重连
     if (code !== 1000 && code !== 1001) { // 非正常关闭
       console.log('检测到异常关闭，准备重连...')
       addMessage('warning', '连接异常断开，将尝试重连')
       
       // 增加重连计数
       reconnectAttempts.value++
       
       // 检查是否超过最大重连次数
       if (reconnectAttempts.value >= maxReconnectAttempts) {
         addMessage('error', `已达到最大重连次数 (${maxReconnectAttempts})，停止重连`)
         return
       }
       
       // 使用指数退避算法计算延迟时间
       const delay = Math.min(reconnectDelay.value * Math.pow(2, reconnectAttempts.value - 1), 30000) // 最大30秒
       
       // 延迟重连
       setTimeout(() => {
         if (!wsConnected.value) {
           console.log(`尝试重新连接WebSocket... (第${reconnectAttempts.value}次)`)
           addMessage('system', `正在尝试重新连接... (第${reconnectAttempts.value}次)`)
           connectWebSocket()
         }
       }, delay)
     } else {
       addMessage('system', 'WebSocket连接正常关闭')
       // 正常关闭时重置重连计数
       reconnectAttempts.value = 0
     }
  }
  
  ws.onerror = (error) => {
    console.error('WebSocket错误:', error)
    wsConnected.value = false
    
    addMessage('error', 'WebSocket连接发生错误')
    
    // 使用UI库显示错误提示
    if (window.ElMessage) {
      window.ElMessage.error('WebSocket连接失败，请检查网络连接')
    } else if (message) {
      message.error('WebSocket连接失败，请检查网络连接')
    }
    
    // 增加重连计数
     reconnectAttempts.value++
     
     // 检查是否超过最大重连次数
     if (reconnectAttempts.value >= maxReconnectAttempts) {
       addMessage('error', `已达到最大重连次数 (${maxReconnectAttempts})，停止重连`)
       return
     }
     
     // 使用指数退避算法计算延迟时间
     const delay = Math.min(reconnectDelay.value * Math.pow(2, reconnectAttempts.value - 1), 30000) // 最大30秒
     
     // 延迟重连
     setTimeout(() => {
       if (!wsConnected.value) {
         console.log(`WebSocket错误后尝试重连... (第${reconnectAttempts.value}次)`)
         addMessage('system', `连接错误，正在重试... (第${reconnectAttempts.value}次)`)
         connectWebSocket()
       }
     }, delay)
  }
  
  wsRef.value = ws
}

/**
 * 发送Hello消息，建立会话
 */
const sendHelloMessage = () => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  const helloMessage = {
    type: 'hello',
    audio_params: {
      format: audioFormat.value,
      sample_rate: sampleRate.value,
      channels: channels.value,
      frame_duration: frameDuration.value
    }
  }
  
  wsRef.value.send(JSON.stringify(helloMessage))
  console.log('发送Hello消息:', helloMessage)
}

/**
 * 为PCM数据添加WAV头
 */
const addWavHeader = (pcmData) => {
  const currentSampleRate = sampleRate.value // 使用动态采样率配置
  const numChannels = channels.value // 使用动态声道配置
  const bitsPerSample = 16 // 16位深度
  const byteRate = currentSampleRate * numChannels * bitsPerSample / 8
  const blockAlign = numChannels * bitsPerSample / 8
  const dataSize = pcmData.length
  const fileSize = 36 + dataSize
  
  const header = new ArrayBuffer(44)
  const view = new DataView(header)
  
  // RIFF header
  view.setUint32(0, 0x46464952, true) // "RIFF"
  view.setUint32(4, fileSize, true) // File size
  view.setUint32(8, 0x45564157, true) // "WAVE"
  
  // fmt chunk
  view.setUint32(12, 0x20746d66, true) // "fmt "
  view.setUint32(16, 16, true) // Chunk size
  view.setUint16(20, 1, true) // Audio format (PCM)
  view.setUint16(22, numChannels, true) // Number of channels
  view.setUint32(24, currentSampleRate, true) // Sample rate
  view.setUint32(28, byteRate, true) // Byte rate
  view.setUint16(32, blockAlign, true) // Block align
  view.setUint16(34, bitsPerSample, true) // Bits per sample
  
  // data chunk
  view.setUint32(36, 0x61746164, true) // "data"
  view.setUint32(40, dataSize, true) // Data size
  
  // 合并头部和PCM数据
  const wavData = new Uint8Array(44 + dataSize)
  wavData.set(new Uint8Array(header), 0)
  wavData.set(pcmData, 44)
  
  return wavData.buffer
}

/**
 * 将原始Opus数据包装成简单的OGG容器
 * 注意：这是一个简化的实现，主要用于处理后端发送的Opus数据包
 */


/**
 * 处理音频数据 - 直接处理PCM格式
 */
/**
 * 处理接收到的PCM音频数据
 * 后端发送的是完整的PCM数据块，前端需要直接播放
 */
/**
 * 处理接收到的PCM音频数据
 * 使用PCMPlayer直接播放，无需转换为WAV格式
 */
const handleAudioData = async (data) => {
  try {
    console.log('收到PCM音频数据:', {
      type: data.constructor.name,
      size: data.byteLength || data.size
    })
    
    // 检查PCMPlayer是否已初始化
    if (!pcmPlayer.value) {
      console.error('PCM播放器未初始化')
      addMessage('error', 'PCM播放器未初始化')
      return
    }
    
    // 检查数据是否有效
    const dataSize = data.byteLength || data.size || 0
    if (dataSize === 0) {
      console.warn('收到空的音频数据')
      addMessage('warning', '收到空的音频数据')
      return
    }
    
    // 检查数据大小是否合理（避免过大的数据）
    const maxSize = 10 * 1024 * 1024 // 10MB
    if (dataSize > maxSize) {
      console.error('音频数据过大:', dataSize)
      addMessage('error', `音频数据过大: ${(dataSize / 1024 / 1024).toFixed(2)}MB`)
      return
    }
    
    // 处理PCM数据并直接播放
    if (data instanceof ArrayBuffer) {
      console.log('使用PCMPlayer播放PCM数据，大小:', dataSize, '字节')
      console.log('PCM数据前16字节:', Array.from(new Uint8Array(data.slice(0, 16))).map(b => b.toString(16).padStart(2, '0')).join(' '))
      addMessage('debug', `PCMPlayer播放PCM数据: ${dataSize} 字节`)
      
      // 检查数据长度是否合理（应该是偶数，因为16位PCM每个样本2字节）
      if (dataSize % 2 !== 0) {
        console.warn('PCM数据长度不是偶数，可能有问题')
        addMessage('warning', 'PCM数据长度异常')
      }
      
      // 直接将PCM数据喂给PCMPlayer
      pcmPlayer.value.feed(data)
      
      console.log('PCM数据已发送到播放器')
      addMessage('success', 'PCM音频播放中...')
      
    } else if (data instanceof Blob) {
      // 如果是Blob，需要转换为ArrayBuffer
      console.log('将Blob转换为ArrayBuffer后播放')
      const arrayBuffer = await data.arrayBuffer()
      pcmPlayer.value.feed(arrayBuffer)
      addMessage('success', 'PCM音频播放中...')
    } else {
      console.error('不支持的音频数据类型:', typeof data)
      addMessage('error', `不支持的音频数据类型: ${typeof data}`)
      return
    }
    
    // 更新音频状态
    hasAudio.value = true
    isAudioPlaying.value = true
    ttsStatus.value = 'playing'
    
  } catch (error) {
    console.error('处理音频数据时出错:', error)
    addMessage('error', `音频处理失败: ${error?.message || '未知错误'}`)
    ttsStatus.value = 'error'
  }
}

// 音频播放队列
const audioQueue = ref([])
const isProcessingQueue = ref(false)

/**
 * 生成测试音频数据（440Hz正弦波）
 */
const generateTestAudio = () => {
  const sampleRate = 16000
  const duration = 1 // 1秒
  const frequency = 440 // A4音符
  const samples = sampleRate * duration
  
  const pcmData = new Uint8Array(samples * 2) // 16位PCM，每个样本2字节
  
  for (let i = 0; i < samples; i++) {
    const sample = Math.sin(2 * Math.PI * frequency * i / sampleRate) * 0.3 // 30%音量
    const intSample = Math.round(sample * 32767) // 转换为16位整数
    
    // 小端序存储
    pcmData[i * 2] = intSample & 0xFF // 低字节
    pcmData[i * 2 + 1] = (intSample >> 8) & 0xFF // 高字节
  }
  
  return pcmData
}

/**
 * 测试音频播放
 */
/**
 * 测试PCM音频播放
 * 使用PCMPlayer直接播放生成的测试PCM数据
 */
const testAudioPlayback = async () => {
  try {
    console.log('生成测试PCM音频...')
    
    // 检查PCMPlayer是否已初始化
    if (!pcmPlayer.value) {
      console.error('PCM播放器未初始化，无法测试')
      addMessage('error', 'PCM播放器未初始化，无法测试')
      return
    }
    
    const testPcmData = generateTestAudio()
    
    console.log('测试PCM数据大小:', testPcmData.length, '字节')
    console.log('测试PCM数据前16字节:', Array.from(testPcmData.slice(0, 16)).map(b => b.toString(16).padStart(2, '0')).join(' '))
    addMessage('debug', `测试PCM音频生成完成: ${testPcmData.length} 字节`)
    
    // 使用PCMPlayer直接播放PCM数据
    console.log('使用PCMPlayer播放测试音频...')
    pcmPlayer.value.feed(testPcmData.buffer)
    
    // 更新播放状态
    isAudioPlaying.value = true
    ttsStatus.value = 'playing'
    hasAudio.value = true
    
    console.log('测试PCM音频播放开始')
    addMessage('success', '测试PCM音频播放开始')
    
  } catch (error) {
    console.error('测试音频播放失败:', error)
    addMessage('error', `测试音频播放失败: ${error?.message || '未知错误'}`)
    ttsStatus.value = 'error'
  }
}

/**
 * 直接播放音频Blob
 * 实现队列式音频播放，避免播放冲突
 */
const playAudioBlob = async (audioBlob) => {
  // 将音频添加到队列
  audioQueue.value.push(audioBlob)
  
  // 如果没有正在处理队列，开始处理
  if (!isProcessingQueue.value) {
    await processAudioQueue()
  }
}

/**
 * 处理音频播放队列
 */
const processAudioQueue = async () => {
  if (isProcessingQueue.value || audioQueue.value.length === 0) {
    return
  }
  
  isProcessingQueue.value = true
  
  while (audioQueue.value.length > 0) {
    const audioBlob = audioQueue.value.shift()
    await playAudioBlobDirect(audioBlob)
  }
  
  isProcessingQueue.value = false
}

/**
 * 直接播放单个音频Blob
 */
const playAudioBlobDirect = async (audioBlob) => {
  return new Promise((resolve, reject) => {
    try {
      // 创建新的音频对象
      const audio = new Audio()
      const audioUrl = URL.createObjectURL(audioBlob)
      audio.src = audioUrl
    
      // 设置音频事件监听器
      audio.addEventListener('loadstart', () => {
        console.log('开始加载WAV音频')
        addMessage('debug', '开始加载WAV音频')
      })
      
      audio.addEventListener('canplay', () => {
        console.log('WAV音频可以播放')
        addMessage('debug', 'WAV音频可以播放')
      })
      
      audio.addEventListener('play', () => {
        console.log('WAV音频开始播放')
        addMessage('audio', 'WAV音频开始播放')
        isAudioPlaying.value = true
        ttsStatus.value = 'playing'
      })
      
      audio.addEventListener('ended', () => {
        console.log('WAV音频播放完成')
        addMessage('audio', 'WAV音频播放完成')
        isAudioPlaying.value = false
        ttsStatus.value = 'idle'
        
        // 清理资源
        URL.revokeObjectURL(audioUrl)
        resolve()
      })
      
      audio.addEventListener('error', (e) => {
        console.error('WAV音频播放错误:', e)
        const errorMsg = e.error?.message || e.message || '音频格式不支持或文件损坏'
        addMessage('error', `WAV音频播放错误: ${errorMsg}`)
        isAudioPlaying.value = false
        ttsStatus.value = 'idle'
        
        // 清理资源
        URL.revokeObjectURL(audioUrl)
        reject(new Error(errorMsg))
      })
      
      // 设置当前音频引用
      currentAudio.value = audio
      
      // 开始播放
      audio.play().catch(error => {
        console.error('播放音频失败:', error)
        const errorMsg = error?.message || '播放失败'
        addMessage('error', `播放音频失败: ${errorMsg}`)
        URL.revokeObjectURL(audioUrl)
        reject(new Error(errorMsg))
      })
      
    } catch (error) {
      console.error('创建音频对象失败:', error)
      addMessage('error', `创建音频对象失败: ${error.message}`)
      reject(error)
    }
  })
}

/**
 * 处理WebSocket消息
 */
const handleWebSocketMessage = (data) => {
  try {
    const message = JSON.parse(data)
    console.log('收到消息:', message)
    
    switch (message.type) {
      case 'hello':
        handleHelloResponse(message)
        break
      case 'stt':
        handleSttMessage(message)
        break
      case 'tts':
        handleTtsMessage(message)
        break
      case 'llm':
        handleLlmMessage(message)
        break
      case 'mcp':
        handleMcpMessage(message)
        break
      case 'error':
        handleErrorMessage(message)
        break
      case 'status':
        handleStatusMessage(message)
        break
      default:
        console.log('未知消息类型:', message.type, message)
        handleUnknownMessage(message)
    }
  } catch (error) {
    console.error('解析WebSocket消息失败:', error)
    addMessage('error', `消息解析失败: ${error.message}`)
  }
}

/**
 * 处理Hello响应
 */
const handleHelloResponse = (message) => {
  sessionId.value = message.session_id || ''
  
  // 更新服务端音频参数
  if (message.audio_params) {
    console.log('服务端音频参数:', message.audio_params)
  }
  
  addMessage('system', `会话已建立，ID: ${sessionId.value}`)
}

/**
 * 处理STT消息（语音识别结果）
 */
const handleSttMessage = (message) => {
  asrText.value = message.text || ''
  addMessage('asr', message.text || '')
}

/**
 * 处理TTS消息
 */
const handleTtsMessage = (message) => {
  const state = message.state
  const text = message.text || ''
  const textIndex = message.text_index || 0
  
  console.log('收到TTS消息:', { state, text, textIndex })
  
  switch (state) {
    case 'start':
      // TTS服务整体启动
      ttsStatus.value = 'loading'
      currentTtsText.value = ''
      audioChunks.value = []
      hasAudio.value = false
      audioChunksCount.value = 0
      addMessage('tts_start', 'TTS服务启动')
      console.log('TTS服务启动')
      break
      
    case 'sentence_start':
      // 单句合成开始
      ttsStatus.value = 'loading'
      currentTtsText.value = text
      // 不清空audioChunks，因为可能有多句话需要连续播放
      addMessage('tts_sentence_start', `开始合成第${textIndex}句: ${text}`)
      console.log(`开始合成第${textIndex}句:`, text)
      break
      
    case 'sentence_end':
      // 单句合成完成
      addMessage('tts_sentence_end', `第${textIndex}句合成完成: ${text}`)
      console.log(`第${textIndex}句合成完成:`, text)
      
      // 如果有音频数据，立即播放当前句子
      if (audioChunks.value.length > 0) {
        console.log(`准备播放第${textIndex}句音频，当前音频块数:`, audioChunks.value.length)
        // 延迟一点时间确保音频数据完整接收
        setTimeout(() => {
          playCurrentSentence()
        }, 50)
      } else {
        console.warn(`第${textIndex}句没有音频数据`)
      }
      break
      
    case 'stop':
      // TTS服务整体停止
      ttsStatus.value = 'idle'
      currentTtsText.value = ''
      addMessage('tts_stop', 'TTS服务停止')
      console.log('TTS服务停止')
      
      // 停止当前播放的音频
      if (currentAudio.value && !currentAudio.value.paused) {
        currentAudio.value.pause()
        isAudioPlaying.value = false
      }
      break
      
    default:
      console.log('未知TTS状态:', state, message)
      addMessage('tts_unknown', `未知TTS状态: ${state}`)
  }
}

/**
 * 处理LLM消息
 */
const handleLlmMessage = (message) => {
  const text = message.text || ''
  const emotion = message.emotion || ''
  
  // 检查是否为thinking表情消息
  if (text === '🤔' || emotion === 'thinking' || (text.includes('🤔') && text.length <= 5)) {
    console.log('收到thinking表情消息，不作为音频数据处理')
    addMessage('thinking', '正在思考...')
    return
  }
  
  llmText.value = text
  addMessage('llm', text)
  
  // 处理情绪状态
  if (emotion) {
    console.log('收到LLM情绪状态:', emotion)
    addMessage('emotion', `情绪: ${emotion}`)
  }
  
  console.log('收到LLM回复:', text)
}

/**
 * 处理错误消息
 */
const handleErrorMessage = (message) => {
  const errorText = message.message || message.text || '未知错误'
  const errorCode = message.code || ''
  
  console.error('收到错误消息:', message)
  
  // 显示错误信息
  addMessage('error', `错误: ${errorText}${errorCode ? ` (${errorCode})` : ''}`)
  
  // 使用UI库显示错误提示
  if (window.ElMessage) {
    window.ElMessage.error(errorText)
  } else if (message) {
    message.error(errorText)
  }
  
  // 重置相关状态
  if (message.type === 'tts_error') {
    ttsStatus.value = 'idle'
    currentTtsText.value = ''
  }
}

/**
 * 处理状态消息
 */
const handleStatusMessage = (message) => {
  const status = message.status || ''
  const statusText = message.message || message.text || ''
  
  console.log('收到状态消息:', message)
  addMessage('status', `状态: ${status}${statusText ? ` - ${statusText}` : ''}`)
  
  // 根据状态类型进行相应处理
  switch (status) {
    case 'connecting':
      addMessage('system', '正在连接服务...')
      break
    case 'connected':
      addMessage('system', '服务连接成功')
      break
    case 'disconnected':
      addMessage('system', '服务连接断开')
      wsConnected.value = false
      break
    case 'processing':
      addMessage('system', '正在处理请求...')
      break
    case 'ready':
      addMessage('system', '服务就绪')
      break
    default:
      addMessage('status', `状态更新: ${status}`)
  }
}

/**
 * 处理未知消息类型
 */
const handleUnknownMessage = (message) => {
  console.warn('收到未知消息类型:', message)
  addMessage('unknown', `未知消息: ${JSON.stringify(message)}`)
  
  // 尝试从消息中提取有用信息
  if (message.text) {
    addMessage('info', message.text)
  }
  
  if (message.error) {
    addMessage('error', message.error)
  }
}

/**
 * 开始/停止录音
 */
const toggleRecording = async () => {
  if (isRecording.value) {
    stopRecording()
  } else {
    await startRecording()
  }
}

/**
 * 获取音频设备列表
 */
const getAudioDevices = async () => {
  try {
    const devices = await navigator.mediaDevices.enumerateDevices()
    audioInputs.value = devices.filter(device => device.kind === 'audioinput')
    
    // 如果没有选择设备且有可用设备，选择第一个
    if (!selectedDevice.value && audioInputs.value.length > 0) {
      selectedDevice.value = audioInputs.value[0].deviceId
    }
    
    console.log('可用音频输入设备:', audioInputs.value)
  } catch (error) {
    console.error('获取音频设备失败:', error)
  }
}

/**
 * 更新音频设备
 */
const updateAudioDevice = () => {
  console.log('切换音频设备:', selectedDevice.value)
  // 如果正在录音，需要重新启动录音以使用新设备
  if (isRecording.value) {
    stopRecording()
    nextTick(() => {
      startRecording()
    })
  }
}

/**
 * 开始录音
 */
const startRecording = async () => {
  try {
    // 构建音频约束
    const audioConstraints = {
      sampleRate: sampleRate.value,
      channelCount: channels.value,
      echoCancellation: true,
      noiseSuppression: true,
      autoGainControl: true
    }
    
    // 如果选择了特定设备，添加设备ID约束
    if (selectedDevice.value) {
      audioConstraints.deviceId = { exact: selectedDevice.value }
    }
    
    // 获取音频流
    const stream = await navigator.mediaDevices.getUserMedia({
      audio: audioConstraints
    })
    
    audioStream.value = stream
    isRecording.value = true
    asrText.value = ''
    
    // 发送listen start消息
    sendListenMessage('start')
    
    // 创建音频处理器
    createAudioProcessor(stream)
    
    addMessage('system', '开始录音')
    
  } catch (error) {
    console.error('启动录音失败:', error)
    message.error('无法访问麦克风')
  }
}

/**
 * 停止录音
 */
const stopRecording = () => {
  if (audioStream.value) {
    audioStream.value.getTracks().forEach(track => track.stop())
    audioStream.value = null
  }
  
  if (audioProcessor.value) {
    audioProcessor.value.disconnect()
    audioProcessor.value = null
  }
  
  if (audioContext.value) {
    audioContext.value.close()
    audioContext.value = null
  }
  
  isRecording.value = false
  
  // 停止ASR监听
  if (isListening.value) {
    isListening.value = false
  }
  
  // 发送listen stop消息
  sendListenMessage('stop')
  
  addMessage('system', '停止录音')
}

/**
 * 创建音频处理器
 */
const createAudioProcessor = (stream) => {
  audioContext.value = new (window.AudioContext || window.webkitAudioContext)({
    sampleRate: sampleRate.value
  })
  
  const source = audioContext.value.createMediaStreamSource(stream)
  
  // 创建ScriptProcessor处理音频数据
  const bufferSize = 4096
  audioProcessor.value = audioContext.value.createScriptProcessor(bufferSize, channels.value, channels.value)
  
  audioProcessor.value.onaudioprocess = (event) => {
    if (!isRecording.value) return
    
    const inputBuffer = event.inputBuffer
    const inputData = inputBuffer.getChannelData(0)
    
    // 转换为16位PCM
    const pcmData = convertToPCM16(inputData)
    
    // 发送音频数据
    if (wsRef.value && wsRef.value.readyState === WebSocket.OPEN) {
      wsRef.value.send(pcmData)
    }
  }
  
  source.connect(audioProcessor.value)
  audioProcessor.value.connect(audioContext.value.destination)
}

/**
 * 转换为16位PCM格式
 */
const convertToPCM16 = (float32Array) => {
  const buffer = new ArrayBuffer(float32Array.length * 2)
  const view = new DataView(buffer)
  
  for (let i = 0; i < float32Array.length; i++) {
    const sample = Math.max(-1, Math.min(1, float32Array[i]))
    view.setInt16(i * 2, sample * 0x7FFF, true)
  }
  
  return buffer
}

/**
 * 发送Listen消息
 */
const sendListenMessage = (state) => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  const listenMessage = {
    type: 'listen',
    state: state,
    mode: 'auto'
  }
  
  wsRef.value.send(JSON.stringify(listenMessage))
  console.log('发送Listen消息:', listenMessage)
}

/**
 * 开始ASR监听
 */
const startListening = () => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN || !isRecording.value) {
    return
  }
  
  isListening.value = true
  sendListenMessage('start')
  addMessage('system', '开始ASR监听')
}

/**
 * 停止ASR监听
 */
const stopListening = () => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  isListening.value = false
  sendListenMessage('stop')
  // sendListenMessage('start')
  addMessage('system', '停止ASR监听')
}

/**
 * 中止当前对话
 */
const abortChat = () => {
  // 停止录音
  if (isRecording.value) {
    stopRecording()
  }
  
  // 停止音频播放
  if (isAudioPlaying.value || ttsStatus.value !== 'idle') {
    stopAudio()
  }
  
  // 发送abort消息
  sendAbortMessage()
  
  // 重置状态
  asrText.value = ''
  llmText.value = ''
  currentTtsText.value = ''
  ttsStatus.value = 'idle'
  
  addMessage('system', '已中止当前对话')
}

/**
 * 发送Abort消息
 */
const sendAbortMessage = () => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  const abortMessage = {
    type: 'abort'
  }
  
  wsRef.value.send(JSON.stringify(abortMessage))
  console.log('发送Abort消息:', abortMessage)
}

/**
 * 发送文本聊天消息
 */
const sendChatMessage = () => {
  const text = chatInput.value.trim()
  if (!text || !wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  // 根据是否启用文本检测模式选择消息类型
  if (enableTextDetect.value) {
    // 使用listen消息的detect状态
    sendTextDetectMessage(text)
  } else {
    // 使用chat消息类型
    sendDirectChatMessage(text)
  }
  
  // 清空输入框
  chatInput.value = ''
  
  // 添加到消息历史
  addMessage('user', text)
}

/**
 * 发送直接聊天消息
 */
const sendDirectChatMessage = (text) => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  const chatMessage = {
    type: 'chat',
    text: text
  }
  
  wsRef.value.send(JSON.stringify(chatMessage))
  console.log('发送Chat消息:', chatMessage)
}

/**
 * 发送文本检测消息
 */
const sendTextDetectMessage = (text) => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  const detectMessage = {
    type: 'listen',
    state: 'detect',
    text: text
  }
  
  wsRef.value.send(JSON.stringify(detectMessage))
  console.log('发送Text Detect消息:', detectMessage)
}

/**
 * 选择图片
 */
const selectImage = () => {
  if (imageInput.value) {
    imageInput.value.click()
  }
}

/**
 * 处理图片选择
 */
const handleImageSelect = (event) => {
  const file = event.target.files[0]
  if (!file) {
    return
  }
  
  // 检查文件类型
  if (!file.type.startsWith('image/')) {
    alert('请选择图片文件')
    return
  }
  
  // 检查文件大小 (限制为10MB)
  const maxSize = 10 * 1024 * 1024
  if (file.size > maxSize) {
    alert('图片文件大小不能超过10MB')
    return
  }
  
  selectedImage.value = file
  
  // 创建预览URL
  if (imagePreviewUrl.value) {
    URL.revokeObjectURL(imagePreviewUrl.value)
  }
  imagePreviewUrl.value = URL.createObjectURL(file)
}

/**
 * 发送图片消息
 */
const sendImageMessage = async () => {
  if (!selectedImage.value || !wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  try {
    // 将图片转换为base64
    const base64Data = await fileToBase64(selectedImage.value)
    
    const imageMessage = {
      type: 'image',
      image_data: base64Data,
      filename: selectedImage.value.name,
      mime_type: selectedImage.value.type
    }
    
    wsRef.value.send(JSON.stringify(imageMessage))
    console.log('发送Image消息:', { ...imageMessage, image_data: '[base64 data]' })
    
    // 添加到消息历史
    addMessage('user', `[图片] ${selectedImage.value.name}`)
    
    // 清除选择的图片
    clearSelectedImage()
    
  } catch (error) {
    console.error('发送图片失败:', error)
    alert('发送图片失败，请重试')
  }
}

/**
 * 清除选择的图片
 */
const clearSelectedImage = () => {
  if (imagePreviewUrl.value) {
    URL.revokeObjectURL(imagePreviewUrl.value)
  }
  selectedImage.value = null
  imagePreviewUrl.value = ''
  if (imageInput.value) {
    imageInput.value.value = ''
  }
}

/**
 * 将文件转换为base64
 */
const fileToBase64 = (file) => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      // 移除data:image/xxx;base64,前缀
      const base64 = reader.result.split(',')[1]
      resolve(base64)
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

/**
 * 格式化文件大小
 */
const formatFileSize = (bytes) => {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

/**
 * 发送视觉消息
 */
const sendVisionMessage = (cmd) => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  const visionMessage = {
    type: 'vision',
    cmd: cmd
  }
  
  wsRef.value.send(JSON.stringify(visionMessage))
  console.log('发送Vision消息:', visionMessage)
  
  // 添加到消息历史
  const cmdNames = {
    'gen_pic': '生成图片',
    'gen_video': '生成视频',
    'read_img': '读取图片'
  }
  addMessage('user', `[视觉功能] ${cmdNames[cmd] || cmd}`)
}

/**
 * 发送带提示词的视觉消息
 */
const sendVisionWithPrompt = () => {
  const prompt = visionPrompt.value.trim()
  if (!prompt || !wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  const visionMessage = {
    type: 'vision',
    cmd: 'gen_pic', // 默认使用生成图片命令
    prompt: prompt
  }
  
  wsRef.value.send(JSON.stringify(visionMessage))
  console.log('发送Vision消息:', visionMessage)
  
  // 添加到消息历史
  addMessage('user', `[视觉功能] ${prompt}`)
  
  // 清空输入框
  visionPrompt.value = ''
}

/**
 * 发送IoT消息
 */
const sendIotMessage = () => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    return
  }
  
  const iotMessage = {
    type: 'iot'
  }
  
  // 处理设备描述符
  if (iotDescriptors.value.trim()) {
    try {
      const descriptors = JSON.parse(iotDescriptors.value.trim())
      iotMessage.descriptors = descriptors
    } catch (error) {
      alert('设备描述符JSON格式错误，请检查输入')
      return
    }
  }
  
  // 处理设备状态
  if (iotStates.value.trim()) {
    try {
      const states = JSON.parse(iotStates.value.trim())
      iotMessage.states = states
    } catch (error) {
      alert('设备状态JSON格式错误，请检查输入')
      return
    }
  }
  
  // 检查是否至少有一个字段
  if (!iotMessage.descriptors && !iotMessage.states) {
    alert('请至少输入设备描述符或设备状态')
    return
  }
  
  wsRef.value.send(JSON.stringify(iotMessage))
  console.log('发送IoT消息:', iotMessage)
  
  // 添加到消息历史
  const messageText = []
  if (iotMessage.descriptors) {
    messageText.push(`描述符: ${JSON.stringify(iotMessage.descriptors)}`)
  }
  if (iotMessage.states) {
    messageText.push(`状态: ${JSON.stringify(iotMessage.states)}`)
  }
  addMessage('user', `[IoT设备] ${messageText.join(', ')}`)
}

/**
 * 清空IoT输入
 */
const clearIotInputs = () => {
  iotDescriptors.value = ''
  iotStates.value = ''
}

/**
 * 播放当前句子的音频
 */
const playCurrentSentence = async () => {
  try {
    if (audioChunks.value.length === 0) {
      console.warn('没有音频数据可播放')
      return
    }
    
    console.log('播放当前句子，音频块数量:', audioChunks.value.length)
    
    // 创建当前句子的音频
    await createAndPlayAudio()
    
    // 播放完成后清空当前句子的音频数据，为下一句做准备
    audioChunks.value = []
    audioChunksCount.value = 0
    hasAudio.value = false
    
  } catch (error) {
    console.error('播放当前句子失败:', error)
    addMessage('error', `播放失败: ${error.message}`)
  }
}

/**
 * 创建并播放音频 - 直接播放WAV格式
 */
const createAndPlayAudio = async () => {
  if (audioChunks.value.length === 0) {
    console.warn('没有音频数据可播放')
    return
  }
  
  try {
    console.log('开始播放PCM音频，数据块数量:', audioChunks.value.length)
    
    // 检查是否有有效的音频数据
    if (audioChunks.value.length === 0) {
      throw new Error('没有音频数据可播放')
    }
    
    const totalSize = audioChunks.value.reduce((total, blob) => total + blob.size, 0)
    if (totalSize === 0) {
      throw new Error('音频数据为空')
    }
    
    console.log('音频数据总大小:', totalSize, '字节')
    
    // 直接合并所有WAV格式的音频块
    console.log('合并WAV音频块')
    const combinedBlob = new Blob(audioChunks.value, { type: 'audio/wav' })
    const audioUrl = URL.createObjectURL(combinedBlob)
    const audio = new Audio(audioUrl)
    audio.preload = 'auto'
    
    console.log('创建WAV音频对象，大小:', combinedBlob.size, '字节')
    
    // 设置音频事件监听器
    audio.onloadstart = () => {
      console.log('WAV音频开始加载')
    }
    
    audio.onloadeddata = () => {
      console.log('WAV音频数据加载完成')
    }
    
    audio.oncanplay = () => {
      console.log('WAV音频可以开始播放')
    }
    
    audio.onplay = () => {
      isAudioPlaying.value = true
      ttsStatus.value = 'playing'
      console.log('WAV音频开始播放')
      addMessage('system', '开始播放PCM-WAV音频')
    }
    
    audio.onended = () => {
      isAudioPlaying.value = false
      ttsStatus.value = 'idle'
      URL.revokeObjectURL(audioUrl)
      console.log('WAV音频播放完成')
      addMessage('system', 'PCM-WAV音频播放完成')
    }
    
    audio.onerror = (error) => {
      console.error('WAV音频播放错误:', error)
      console.error('音频错误详情:', {
        error: audio.error,
        networkState: audio.networkState,
        readyState: audio.readyState,
        src: audio.src
      })
      isAudioPlaying.value = false
      ttsStatus.value = 'error'
      URL.revokeObjectURL(audioUrl)
      
      const errorMsg = audio.error ? `WAV音频播放失败 (错误代码: ${audio.error.code})` : 'WAV音频播放失败'
      message.error(errorMsg)
      addMessage('error', errorMsg)
    }
    
    audio.onpause = () => {
      console.log('WAV音频暂停')
    }
    
    // 设置音量和其他属性
    audio.volume = 0.8
    audio.preload = 'auto'
    
    currentAudio.value = audio
    
    // 尝试播放音频
    try {
      console.log('开始播放音频...')
      await audio.play()
      console.log('音频播放命令执行成功')
    } catch (playError) {
      console.error('播放音频时出错:', playError)
      throw new Error(`播放失败: ${playError.message}`)
    }
    
  } catch (error) {
    console.error('创建音频失败:', error)
    ttsStatus.value = 'error'
    message.error(`音频处理失败: ${error.message}`)
    addMessage('error', `音频处理失败: ${error.message}`)
  }
}

/**
 * 播放音频
 */
const playAudio = () => {
  // 检查是否有队列中的音频或已缓存的音频
  if (audioQueue.value.length > 0) {
    console.log('播放队列中的音频')
    addMessage('system', '开始播放队列音频')
    if (!isProcessingQueue.value) {
      processAudioQueue()
    }
  } else if (audioChunks.value.length > 0) {
    console.log('播放缓存的音频')
    createAndPlayAudio()
  } else {
    console.warn('没有音频数据可播放')
    addMessage('warning', '没有音频数据可播放')
  }
}

/**
 * 停止音频播放
 */
/**
 * 停止音频播放
 * 支持传统Audio对象和PCMPlayer
 */
const stopAudio = () => {
  // 清空音频队列
  audioQueue.value = []
  isProcessingQueue.value = false
  
  // 停止PCMPlayer
  if (pcmPlayer.value) {
    try {
      pcmPlayer.value.pause()
      console.log('PCM播放器已停止')
      addMessage('system', 'PCM播放器已停止')
    } catch (error) {
      console.warn('停止PCM播放器时出错:', error)
    }
  }
  
  // 停止传统Audio对象
  if (currentAudio.value) {
    currentAudio.value.pause()
    currentAudio.value.currentTime = 0
    console.log('传统音频播放已停止')
    addMessage('system', '传统音频播放已停止')
    
    // 清理音频资源
    if (currentAudio.value.src) {
      URL.revokeObjectURL(currentAudio.value.src)
    }
    currentAudio.value = null
  }
  
  // 更新状态
  isAudioPlaying.value = false
  ttsStatus.value = 'idle'
}

/**
 * 暂停音频播放
 * 支持传统Audio对象和PCMPlayer
 */
const pauseAudio = () => {
  let paused = false
  
  // 暂停PCMPlayer
  if (pcmPlayer.value) {
    try {
      pcmPlayer.value.pause()
      paused = true
      console.log('PCM播放器已暂停')
      addMessage('system', 'PCM播放器已暂停')
    } catch (error) {
      console.warn('暂停PCM播放器时出错:', error)
    }
  }
  
  // 暂停传统Audio对象
  if (currentAudio.value && !currentAudio.value.paused) {
    currentAudio.value.pause()
    paused = true
    console.log('传统音频播放已暂停')
    addMessage('system', '传统音频播放已暂停')
  }
  
  if (paused) {
    isAudioPlaying.value = false
    ttsStatus.value = 'paused'
  }
}

/**
 * 恢复音频播放
 * 支持传统Audio对象和PCMPlayer
 */
const resumeAudio = () => {
  let resumed = false
  
  // 恢复PCMPlayer播放
  if (pcmPlayer.value) {
    try {
      pcmPlayer.value.continue()
      resumed = true
      console.log('PCM播放器已恢复')
      addMessage('system', 'PCM播放器已恢复')
    } catch (error) {
      console.warn('恢复PCM播放器时出错:', error)
    }
  }
  
  // 恢复传统Audio对象播放
  if (currentAudio.value && currentAudio.value.paused) {
    // 检查音频是否有有效的源
    if (!currentAudio.value.src || currentAudio.value.src === '') {
      console.warn('音频没有有效的源，无法恢复播放')
      if (!resumed) {
        addMessage('warning', '没有可播放的音频')
      }
      return
    }
    
    currentAudio.value.play().then(() => {
      resumed = true
      console.log('传统音频播放已恢复')
      addMessage('system', '传统音频播放已恢复')
    }).catch(error => {
      console.error('恢复播放失败:', error)
      const errorMsg = error?.message || '播放失败'
      if (!resumed) {
        addMessage('error', `恢复播放失败: ${errorMsg}`)
      }
    })
  }
  
  if (resumed) {
    isAudioPlaying.value = true
    ttsStatus.value = 'playing'
  } else if (!pcmPlayer.value && !currentAudio.value) {
    console.warn('没有可恢复的音频')
    addMessage('warning', '没有可恢复的音频')
  }
}

/**
 * 切换音频播放状态
 */
const toggleAudio = () => {
  if (!currentAudio.value) {
    playAudio()
  } else if (currentAudio.value.paused) {
    resumeAudio()
  } else {
    pauseAudio()
  }
}

/**
 * 设置音频音量
 */
const setAudioVolume = (volume) => {
  if (currentAudio.value) {
    currentAudio.value.volume = Math.max(0, Math.min(1, volume))
    console.log('音频音量设置为:', currentAudio.value.volume)
  }
}

/**
 * 获取音频播放进度
 */
const getAudioProgress = () => {
  if (currentAudio.value) {
    return {
      currentTime: currentAudio.value.currentTime,
      duration: currentAudio.value.duration,
      progress: currentAudio.value.duration > 0 ? currentAudio.value.currentTime / currentAudio.value.duration : 0
    }
  }
  return { currentTime: 0, duration: 0, progress: 0 }
}

/**
 * 添加消息到历史记录
 */
const addMessage = (type, content) => {
  messages.value.push({
    type,
    content,
    timestamp: new Date()
  })
  
  // 滚动到底部
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

/**
 * 清空历史记录
 */
const clearHistory = () => {
  messages.value = []
  asrText.value = ''
  llmText.value = ''
  currentTtsText.value = ''
  audioChunks.value = []
  hasAudio.value = false
  audioChunksCount.value = 0
  console.log('历史记录已清空')
}

/**
 * 获取TTS状态文本
 */
const getTtsStatusText = () => {
  switch (ttsStatus.value) {
    case 'idle': return '空闲'
    case 'loading': return '合成中'
    case 'playing': return '播放中'
    case 'paused': return '已暂停'
    case 'error': return '错误'
    default: return '未知'
  }
}

/**
 * 获取消息类型文本
 */
const getMessageTypeText = (type) => {
  switch (type) {
    case 'system': return '系统'
    case 'asr': return 'ASR识别'
    case 'llm': return 'LLM回复'
    case 'thinking': return '思考中'
    case 'tts_start': return 'TTS开始'
    case 'tts_end': return 'TTS完成'
    case 'error': return '错误'
    default: return type
  }
}

/**
 * 格式化时间
 */
const formatTime = (date) => {
  return date.toLocaleTimeString()
}

// ==================== MCP 相关方法 ====================

/**
 * 处理 MCP 消息
 * 注意：服务端(xiaozhi-server-go)是 MCP 客户端，设备端是 MCP 服务器
 */
const handleMcpMessage = (message) => {
  console.log('收到MCP消息:', message)
  
  const payload = message.payload || {}
  const method = payload.method || ''
  const id = payload.id || ''
  
  // 根据消息类型处理
  if (payload.result !== undefined || payload.error !== undefined) {
    // 这是响应消息（来自设备端 MCP 服务器的响应）
    handleMcpResponse(payload)
  } else if (method) {
    // 这是请求消息（服务端 MCP 客户端发送的请求）
    switch (method) {
      case 'initialize':
        handleMcpInitializeRequest(payload)
        break
      case 'tools/list':
        handleMcpToolsListRequest(payload)
        break
      case 'tools/call':
        handleMcpToolCallRequest(payload)
        break
      default:
        console.log('未知MCP请求方法:', method)
        addMessage('mcp', `未知MCP请求: ${method}`)
    }
  } else {
    console.log('未知MCP消息格式:', message)
    addMessage('mcp', `未知MCP消息: ${JSON.stringify(message)}`)
  }
}

/**
 * 处理 MCP 初始化请求（服务端发送给设备端）
 */
const handleMcpInitializeRequest = (payload) => {
  console.log('服务端发送MCP初始化请求:', payload)
  mcpStatus.value = 'connecting'
  addMessage('mcp', '收到服务端初始化请求，正在发送响应...')
  
  // 发送初始化响应给服务端
  const initializeResponse = {
    type: 'mcp',
    session_id: payload.session_id || sessionId.value,
    payload: {
      jsonrpc: '2.0',
      id: payload.id, // 使用请求的ID
      result: {
        protocolVersion: '2024-11-05',
        capabilities: {
          tools: {
            listChanged: true
          },
          logging: {}
        },
        serverInfo: {
          name: 'xiaozhi-device',
          version: '1.0.0'
        }
      }
    }
  }
  
  if (wsRef.value && wsRef.value.readyState === WebSocket.OPEN) {
    wsRef.value.send(JSON.stringify(initializeResponse))
    console.log('已发送MCP初始化响应:', initializeResponse)
    addMessage('mcp', '已向服务端发送初始化响应')
  } else {
    console.error('WebSocket连接未就绪，无法发送初始化响应')
    addMessage('mcp', 'WebSocket连接未就绪，无法发送初始化响应')
  }
}

/**
 * 处理 MCP 工具列表请求（服务端发送给设备端）
 */
const handleMcpToolsListRequest = (payload) => {
  console.log('服务端请求工具列表:', payload)
  addMessage('mcp', '收到服务端工具列表请求，正在发送响应...')
  
  // 发送工具列表响应给服务端
  const toolsListResponse = {
    type: 'mcp',
    session_id: payload.session_id || sessionId.value,
    payload: {
      jsonrpc: '2.0',
      id: payload.id, // 使用请求的ID
      result: {
        tools: [
        {
          name: 'echo',
          description: '回显输入的文本',
          inputSchema: {
            type: 'object',
            properties: {
              text: {
                type: 'string',
                description: '要回显的文本'
              }
            },
            required: ['text']
          }
        },
        {
          name: 'get_time',
          description: '获取当前时间',
          inputSchema: {
            type: 'object',
            properties: {},
            required: []
          }
        },
        {
          name: 'calculate',
          description: '执行数学计算',
          inputSchema: {
            type: 'object',
            properties: {
              expression: {
                type: 'string',
                description: '数学表达式，如 "2 + 3 * 4"'
              }
            },
            required: ['expression']
          }
        },
        {
          name: 'self.camera.take_photo',
          description: '拍照并分析图像内容',
          inputSchema: {
            type: 'object',
            properties: {
              question: {
                type: 'string',
                description: '对图像的问题或分析要求，如"这是什么？"、"描述一下这个场景"'
              }
            },
            required: ['question']
          }
        }
      ]
    }
  }
}
  
  if (wsRef.value && wsRef.value.readyState === WebSocket.OPEN) {
    wsRef.value.send(JSON.stringify(toolsListResponse))
    console.log('已发送MCP工具列表响应:', toolsListResponse)
    addMessage('mcp', '已向服务端发送工具列表响应')
  } else {
    console.error('WebSocket连接未就绪，无法发送工具列表响应')
    addMessage('mcp', 'WebSocket连接未就绪，无法发送工具列表响应')
  }
}

/**
 * 处理 MCP 工具调用请求（服务端发送给设备端）
 */
const handleMcpToolCallRequest = (payload) => {
  console.log('服务端调用工具:', payload)
  const toolName = payload.params?.name || '未知工具'
  const toolArgs = payload.params?.arguments || {}
  
  isCallingTool.value = true
  addMessage('mcp', `收到服务端工具调用请求: ${toolName}`)
  
  // 模拟工具执行并发送响应
  setTimeout(async () => {
    let result = null
    let error = null
    
    try {
      // 根据工具名称执行相应逻辑
      switch (toolName) {
        case 'echo':
          result = {
            content: [
              {
                type: 'text',
                text: `回显: ${toolArgs.text || '空文本'}`
              }
            ]
          }
          break
        case 'get_time':
          result = {
            content: [
              {
                type: 'text',
                text: `当前时间: ${new Date().toLocaleString()}`
              }
            ]
          }
          break
        case 'calculate':
          try {
            // 简单的数学表达式计算（仅用于演示）
            const expression = toolArgs.expression || '0'
            const calcResult = eval(expression) // 注意：实际应用中不应使用eval
            result = {
              content: [
                {
                  type: 'text',
                  text: `计算结果: ${expression} = ${calcResult}`
                }
              ]
            }
          } catch (e) {
            error = {
              code: -32000,
              message: `计算错误: ${e.message}`
            }
          }
          break
        case 'self.camera.take_photo':
          // 拍照工具处理
          const question = toolArgs.question || '这是什么？'
          addMessage('mcp', `正在拍照并分析: ${question}`)
          
          // 尝试获取摄像头权限并拍照
          try {
            const photoResult = await takeCameraPhoto(question)
            result = {
              content: [
                {
                  type: 'text',
                  text: JSON.stringify(photoResult)
                }
              ]
            }
          } catch (e) {
            console.error('拍照失败:', e)
            result = {
              content: [
                {
                  type: 'text',
                  text: JSON.stringify({
                    success: false,
                    result: '',
                    message: `拍照失败: ${e.message}`
                  })
                }
              ]
            }
          }
          break
        default:
          error = {
            code: -32601,
            message: `未知工具: ${toolName}`
          }
      }
    } catch (e) {
      error = {
        code: -32000,
        message: `工具执行错误: ${e.message}`
      }
    }
    
    // 发送工具调用响应
    const toolCallResponse = {
      type: 'mcp',
      session_id: payload.session_id || sessionId.value,
      payload: {
        jsonrpc: '2.0',
        id: payload.id, // 使用请求的ID
        ...(error ? { error } : { result })
      }
    }
    
    if (wsRef.value && wsRef.value.readyState === WebSocket.OPEN) {
       wsRef.value.send(JSON.stringify(toolCallResponse))
       console.log('已发送MCP工具调用响应:', toolCallResponse)
       addMessage('mcp', `已向服务端发送工具调用响应: ${toolName}`)
     } else {
       console.error('WebSocket连接未就绪，无法发送工具调用响应')
       addMessage('mcp', 'WebSocket连接未就绪，无法发送工具调用响应')
     }
    
    isCallingTool.value = false
  }, 1000) // 模拟1秒的工具执行时间
}

/**
 * 处理 MCP 响应消息（设备端返回的响应）
 */
const handleMcpResponse = (payload) => {
  console.log('收到设备端MCP响应:', payload)
  
  const id = payload.id
  
  if (payload.error) {
    // 错误响应
    handleMcpError(payload)
    return
  }
  
  // 成功响应
  if (id === 1) {
    // 初始化响应
    mcpStatus.value = 'ready'
    addMessage('mcp', '设备端MCP初始化完成')
    console.log('MCP初始化成功，设备信息:', payload.result?.serverInfo)
  } else if (id === 2) {
    // 工具列表响应
    if (payload.result && payload.result.tools) {
      mcpTools.value = payload.result.tools
      mcpStatus.value = 'ready'
      addMessage('mcp', `设备端返回 ${mcpTools.value.length} 个MCP工具`)
      console.log('设备端工具列表:', mcpTools.value)
    } else {
      console.warn('设备端工具列表格式异常:', payload)
      addMessage('mcp', '设备端工具列表格式异常')
    }
  } else {
    // 工具调用响应
    const result = {
      toolName: '未知工具',
      result: payload.result,
      timestamp: new Date()
    }
    
    lastToolResult.value = result
    isCallingTool.value = false
    addMessage('mcp', '设备端工具调用完成')
    console.log('工具调用结果:', result)
  }
}

/**
 * 处理 MCP 错误响应
 */
const handleMcpError = (payload) => {
  console.error('设备端MCP错误:', payload.error)
  
  const errorMsg = payload.error?.message || '未知MCP错误'
  mcpStatus.value = 'error'
  isCallingTool.value = false
  
  addMessage('error', `设备端MCP错误: ${errorMsg}`)
}

/**
 * 获取 MCP 状态文本
 */
const getMcpStatusText = () => {
  switch (mcpStatus.value) {
    case 'disconnected': return '未连接'
    case 'connecting': return '连接中'
    case 'ready': return '就绪'
    case 'error': return '错误'
    default: return '未知'
  }
}

/**
 * 选择工具
 */
const selectTool = (tool) => {
  selectedTool.value = tool
  toolCallParams.value = {}
  
  // 初始化参数
  if (tool.inputSchema && tool.inputSchema.properties) {
    Object.keys(tool.inputSchema.properties).forEach(paramName => {
      toolCallParams.value[paramName] = ''
    })
  }
  
  console.log('选择工具:', tool.name)
  addMessage('mcp', `选择工具: ${tool.name}`)
}

/**
 * 清除选择的工具
 */
const clearSelectedTool = () => {
  selectedTool.value = null
  toolCallParams.value = {}
  addMessage('mcp', '清除工具选择')
}

/**
 * 调用工具（快速调用）
 * 注意：前端不直接发送 MCP 消息，而是通过服务端转发
 */
const callTool = (tool) => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    message.error('WebSocket未连接')
    return
  }
  
  if (mcpStatus.value !== 'ready') {
    message.error('MCP未就绪')
    return
  }
  
  isCallingTool.value = true
  
  // 发送工具调用请求给服务端，服务端会转发给设备端
  const toolCallMessage = {
    type: 'mcp_tool_call',
    session_id: sessionId.value,
    tool_name: tool.name,
    arguments: {}
  }
  
  wsRef.value.send(JSON.stringify(toolCallMessage))
  console.log('请求服务端调用MCP工具:', toolCallMessage)
  addMessage('mcp', `请求调用工具: ${tool.name}`)
}

/**
 * 调用选中的工具（带参数）
 * 注意：前端不直接发送 MCP 消息，而是通过服务端转发
 */
const callSelectedTool = () => {
  if (!selectedTool.value) {
    message.error('请先选择工具')
    return
  }
  
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    message.error('WebSocket未连接')
    return
  }
  
  if (mcpStatus.value !== 'ready') {
    message.error('MCP未就绪')
    return
  }
  
  // 验证必需参数
  if (selectedTool.value.inputSchema && selectedTool.value.inputSchema.required) {
    for (const requiredParam of selectedTool.value.inputSchema.required) {
      if (!toolCallParams.value[requiredParam] || !toolCallParams.value[requiredParam].trim()) {
        message.error(`请填写必需参数: ${requiredParam}`)
        return
      }
    }
  }
  
  isCallingTool.value = true
  
  // 构建参数对象
  const args = {}
  Object.keys(toolCallParams.value).forEach(key => {
    if (toolCallParams.value[key] && toolCallParams.value[key].trim()) {
      args[key] = toolCallParams.value[key].trim()
    }
  })
  
  // 发送工具调用请求给服务端，服务端会转发给设备端
  const toolCallMessage = {
    type: 'mcp_tool_call',
    session_id: sessionId.value,
    tool_name: selectedTool.value.name,
    arguments: args
  }
  
  wsRef.value.send(JSON.stringify(toolCallMessage))
  console.log('请求服务端调用MCP工具:', toolCallMessage)
  addMessage('mcp', `请求调用工具: ${selectedTool.value.name}`)
}

/**
 * 发送 MCP 快捷命令
 */
const sendMcpQuickCommand = (command) => {
  if (!wsRef.value || wsRef.value.readyState !== WebSocket.OPEN) {
    message.error('WebSocket未连接')
    return
  }
  
  // 发送文本消息，让服务端的LLM处理并调用相应的MCP工具
  const textMessage = {
    type: 'text',
    text: command,
    session_id: sessionId.value
  }
  
  wsRef.value.send(JSON.stringify(textMessage))
  console.log('发送MCP快捷命令:', command)
  addMessage('user', command)
}

/**
 * 拍照并分析图像
 */
const takeCameraPhoto = async (question) => {
  try {
    // 获取摄像头权限
    const stream = await navigator.mediaDevices.getUserMedia({ 
      video: { 
        width: { ideal: 1280 },
        height: { ideal: 720 }
      } 
    })
    
    // 创建video元素
    const video = document.createElement('video')
    video.srcObject = stream
    video.autoplay = true
    
    // 等待视频加载
    await new Promise((resolve) => {
      video.onloadedmetadata = resolve
    })
    
    // 创建canvas进行截图
    const canvas = document.createElement('canvas')
    canvas.width = video.videoWidth
    canvas.height = video.videoHeight
    const ctx = canvas.getContext('2d')
    ctx.drawImage(video, 0, 0)
    
    // 停止摄像头
    stream.getTracks().forEach(track => track.stop())
    
    // 转换为blob
    const blob = await new Promise(resolve => {
      canvas.toBlob(resolve, 'image/jpeg', 0.8)
    })
    
    // 发送到视觉分析服务
    const formData = new FormData()
    formData.append('image', blob, 'camera_photo.jpg')
    formData.append('question', question)
    
    const response = await fetch('/api/vision/analyze', {
      method: 'POST',
      headers: {
        'Device-ID': headerConfig.deviceId,
        'Client-ID': headerConfig.clientId,
        'Authorization': headerConfig.token ? `Bearer ${headerConfig.token}` : ''
      },
      body: formData
    })
    
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }
    
    const result = await response.json()
    return result
    
  } catch (error) {
    console.error('拍照失败:', error)
    throw error
  }
}

/**
 * 使用配置的header信息连接WebSocket
 */
const connectWithHeaders = () => {
  if (!headerConfig.deviceId.trim()) {
    message.error('请输入设备ID')
    return
  }
  if (!headerConfig.clientId.trim()) {
    message.error('请输入客户端ID')
    return
  }
  
  // 重置重连计数
  reconnectAttempts.value = 0
  connectWebSocket()
}

/**
 * 重置header配置
 */
const resetHeaderConfig = () => {
  headerConfig.deviceId = 'web-client-' + Date.now()
  headerConfig.clientId = 'web-' + Math.random().toString(36).substr(2, 9)
  headerConfig.sessionId = ''
  headerConfig.transportType = 'websocket'
  headerConfig.token = ''
  message.success('配置已重置')
}

// 生命周期
onMounted(() => {
  // 初始化PCM播放器
  pcmPlayer.value = new PCMPlayer({
    inputCodec: 'Int16',    // 16位整数PCM
    channels: 1,           // 单声道
    sampleRate: 16000,     // 24kHz采样率，与后端配置一致
    flushTime: 1000        // 1秒缓冲时间
  })
  
  console.log('PCM播放器初始化完成:', pcmPlayer.value)
  
  // 不自动连接，等待用户手动连接
  // connectWebSocket()
  getAudioDevices()
})

onUnmounted(() => {
  if (wsRef.value) {
    wsRef.value.close()
  }
  
  if (isRecording.value) {
    stopRecording()
  }
  
  if (currentAudio.value) {
    currentAudio.value.pause()
  }
  
  // 清理PCM播放器
  if (pcmPlayer.value) {
    pcmPlayer.value.destroy()
    pcmPlayer.value = null
  }
  
  // 清理图片预览URL
  if (imagePreviewUrl.value) {
    URL.revokeObjectURL(imagePreviewUrl.value)
  }
})
</script>

<style scoped>
.asr-tts-demo {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
}

.header {
  text-align: center;
  margin-bottom: 30px;
}

.header h1 {
  color: #2c3e50;
  margin-bottom: 10px;
}

.description {
  color: #7f8c8d;
  font-size: 16px;
}

.connection-status {
  display: flex;
  align-items: center;
  padding: 10px 15px;
  border-radius: 8px;
  margin-bottom: 20px;
  font-weight: 500;
}

.connection-status.connected {
  background-color: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.connection-status.disconnected {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.status-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 10px;
}

.connected .status-indicator {
  background-color: #28a745;
}

.disconnected .status-indicator {
  background-color: #dc3545;
}

/* 连接配置样式 */
.connection-config {
  margin-bottom: 20px;
  padding: 20px;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  background-color: #f9f9f9;
}

.connection-config h3 {
  margin: 0 0 15px 0;
  color: #333;
  font-size: 16px;
}

.config-form {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.config-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.config-row label {
  min-width: 80px;
  font-weight: 500;
  color: #555;
  font-size: 14px;
}

.config-input,
.config-select {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #ccc;
  border-radius: 4px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.3s ease;
}

.config-input:focus,
.config-select:focus {
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
}

.config-actions {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}

.btn-connect {
  background-color: #007bff;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: background-color 0.3s ease;
}

.btn-connect:hover {
  background-color: #0056b3;
}

.btn-reset {
  background-color: #6c757d;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: background-color 0.3s ease;
}

.btn-reset:hover {
  background-color: #545b62;
}

.session-info {
  background-color: #f8f9fa;
  padding: 15px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.session-info p {
  margin: 5px 0;
  font-size: 14px;
}

.audio-controls {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 30px;
  margin-bottom: 30px;
}

.recording-section,
.playback-section {
  background-color: #ffffff;
  padding: 20px;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

.recording-section h3,
.playback-section h3 {
  margin-top: 0;
  color: #2c3e50;
  border-bottom: 2px solid #3498db;
  padding-bottom: 10px;
}

.device-selector {
  margin-bottom: 15px;
}

.device-selector label {
  display: block;
  margin-bottom: 5px;
  font-weight: 500;
  color: #495057;
}

.device-selector select {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #ced4da;
  border-radius: 4px;
  font-size: 14px;
  background-color: #fff;
}

.device-selector select:disabled {
  background-color: #e9ecef;
  cursor: not-allowed;
}

.recording-controls {
  display: flex;
  align-items: center;
  gap: 15px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.listen-controls {
  display: flex;
  gap: 10px;
  align-items: center;
}

.btn-record,
.btn-stop,
.btn-listen,
.btn-stop-listen {
  padding: 12px 24px;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.btn-listen,
.btn-stop-listen {
  padding: 8px 16px;
  font-size: 14px;
}

.btn-listen {
  background-color: #007bff;
  color: white;
}

.btn-listen:hover:not(:disabled) {
  background-color: #0056b3;
}

.btn-stop-listen {
  background-color: #ffc107;
  color: #212529;
}

.btn-stop-listen:hover:not(:disabled) {
  background-color: #e0a800;
}

.btn-listen:disabled,
.btn-stop-listen:disabled {
  background-color: #6c757d;
  cursor: not-allowed;
  opacity: 0.6;
}

.btn-record {
  background-color: #28a745;
  color: white;
}

.btn-record:hover:not(.disabled) {
  background-color: #218838;
}

.btn-stop {
  background-color: #dc3545;
  color: white;
}

.btn-stop:hover {
  background-color: #c82333;
}

.btn-record.disabled,
.btn-stop.disabled {
  background-color: #6c757d;
  cursor: not-allowed;
}

.recording-status {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #dc3545;
  font-weight: 500;
}

.listening-status {
  color: #007bff;
  font-weight: 600;
  margin-left: 5px;
}

.recording-indicator {
  width: 12px;
  height: 12px;
  background-color: #dc3545;
  border-radius: 50%;
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0% { opacity: 1; }
  50% { opacity: 0.3; }
  100% { opacity: 1; }
}

.asr-result {
  background-color: #e8f5e8;
  padding: 15px;
  border-radius: 8px;
  border-left: 4px solid #28a745;
}

.asr-result h4 {
  margin: 0 0 10px 0;
  color: #155724;
}

.asr-text {
  margin: 0;
  font-size: 16px;
  line-height: 1.5;
}

.llm-result {
  background-color: #e3f2fd;
  padding: 15px;
  border-radius: 8px;
  border-left: 4px solid #2196f3;
  margin-top: 15px;
}

.llm-result h4 {
  margin: 0 0 10px 0;
  color: #0d47a1;
}

.llm-text {
  margin: 0;
  font-size: 16px;
  line-height: 1.5;
  color: #1565c0;
}

.tts-status {
  margin-bottom: 20px;
}

.status-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.label {
  font-weight: 500;
  color: #495057;
}

.value {
  font-weight: 600;
}

.value.idle {
  color: #6c757d;
}

.value.loading {
  color: #ffc107;
}

.value.playing {
  color: #28a745;
}

.value.error {
  color: #dc3545;
}

.audio-player {
  display: flex;
  flex-direction: column;
  gap: 15px;
  padding: 15px;
  background-color: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #dee2e6;
}

.audio-controls {
  display: flex;
  gap: 10px;
  align-items: center;
}

.audio-player button {
  padding: 8px 16px;
  border: 1px solid #007bff;
  background-color: #007bff;
  color: white;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 14px;
}

.play-toggle-btn {
  background-color: #28a745 !important;
  border-color: #28a745 !important;
}

.play-toggle-btn:hover:not(:disabled) {
  background-color: #218838 !important;
}

.stop-btn {
  background-color: #dc3545 !important;
  border-color: #dc3545 !important;
}

.stop-btn:hover:not(:disabled) {
  background-color: #c82333 !important;
}

.audio-player button:hover:not(:disabled) {
  background-color: #0056b3;
  transform: translateY(-1px);
}

.audio-player button:disabled {
  background-color: #6c757d;
  border-color: #6c757d;
  cursor: not-allowed;
  opacity: 0.6;
}

.volume-control {
  display: flex;
  align-items: center;
  gap: 10px;
}

.volume-control label {
  font-size: 14px;
  color: #495057;
  min-width: 40px;
}

.volume-slider {
  flex: 1;
  max-width: 150px;
  height: 6px;
  background: #dee2e6;
  border-radius: 3px;
  outline: none;
  cursor: pointer;
}

.volume-slider::-webkit-slider-thumb {
  appearance: none;
  width: 16px;
  height: 16px;
  background: #007bff;
  border-radius: 50%;
  cursor: pointer;
}

.volume-slider::-moz-range-thumb {
  width: 16px;
  height: 16px;
  background: #007bff;
  border-radius: 50%;
  cursor: pointer;
  border: none;
}

.volume-value {
  font-size: 12px;
  color: #6c757d;
  min-width: 35px;
  text-align: right;
}

.audio-status {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: #6c757d;
}

.status-text {
  font-weight: 500;
}

.chunks-info {
  background-color: #e9ecef;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
}

.conversation-history {
  background-color: #ffffff;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  margin-bottom: 20px;
}

.conversation-history h3 {
  margin: 0;
  padding: 20px 20px 10px;
  color: #2c3e50;
  border-bottom: 2px solid #3498db;
}

.messages {
  max-height: 400px;
  overflow-y: auto;
  padding: 20px;
}

.message {
  margin-bottom: 15px;
  padding: 12px;
  border-radius: 8px;
  border-left: 4px solid #ddd;
}

.message.system {
  background-color: #f8f9fa;
  border-left-color: #6c757d;
}

.message.asr {
  background-color: #e8f5e8;
  border-left-color: #28a745;
}

.message.llm {
  background-color: #e3f2fd;
  border-left-color: #2196f3;
}

.message.tts_start,
.message.tts_end {
  background-color: #fff3cd;
  border-left-color: #ffc107;
}

.message.error {
  background-color: #f8d7da;
  border-left-color: #dc3545;
}

.message-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 5px;
  font-size: 12px;
  color: #6c757d;
}

.message-type {
  font-weight: 600;
}

.message-content {
  font-size: 14px;
  line-height: 1.4;
}

.debug-info {
  background-color: #f8f9fa;
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.debug-info h3 {
  margin-top: 0;
  color: #495057;
}

.debug-content p {
  margin: 8px 0;
  font-size: 14px;
  font-family: 'Courier New', monospace;
}

.settings-panel {
  display: flex;
  gap: 10px;
  justify-content: center;
}

.debug-toggle,
.clear-history {
  padding: 8px 16px;
  border: 1px solid #6c757d;
  background-color: #f8f9fa;
  color: #495057;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.3s ease;
}

.debug-toggle:hover,
.clear-history:hover {
  background-color: #e9ecef;
}

/* 文本聊天区域样式 */
.text-chat-section {
  margin: 20px 0;
  padding: 15px;
  border: 1px solid #ddd;
  border-radius: 8px;
  background-color: #f9f9f9;
}

.text-chat-section h3 {
  margin: 0 0 15px 0;
  color: #333;
  font-size: 16px;
}

.chat-input-area {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.input-group {
  display: flex;
  gap: 10px;
  align-items: center;
}

.chat-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #ccc;
  border-radius: 4px;
  font-size: 14px;
  outline: none;
}

.chat-input:focus {
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
}

.chat-input:disabled {
  background-color: #e9ecef;
  cursor: not-allowed;
}

.btn-send {
  background-color: #007bff;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  white-space: nowrap;
}

.btn-send:hover:not(:disabled) {
  background-color: #0056b3;
}

.btn-send:disabled {
  background-color: #6c757d;
  cursor: not-allowed;
}

.btn-abort {
  background-color: #dc3545;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  margin-left: 10px;
}

.btn-abort:hover:not(:disabled) {
  background-color: #c82333;
}

.btn-abort:disabled {
  background-color: #6c757d;
  cursor: not-allowed;
}

.chat-options {
  display: flex;
  align-items: center;
}

.chat-options label {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 14px;
  color: #666;
  cursor: pointer;
}

.chat-options input[type="checkbox"] {
  margin: 0;
}

/* 图片上传区域样式 */
.image-upload-section {
  margin: 20px 0;
  padding: 15px;
  border: 1px solid #ddd;
  border-radius: 8px;
  background-color: #f9f9f9;
}

.image-upload-section h3 {
  margin: 0 0 15px 0;
  color: #333;
  font-size: 16px;
}

.upload-area {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.upload-controls {
  display: flex;
  gap: 10px;
  align-items: center;
}

.btn-select-image,
.btn-send-image {
  background-color: #28a745;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  white-space: nowrap;
}

.btn-select-image:hover:not(:disabled),
.btn-send-image:hover:not(:disabled) {
  background-color: #218838;
}

.btn-select-image:disabled,
.btn-send-image:disabled {
  background-color: #6c757d;
  cursor: not-allowed;
}

.btn-send-image {
  background-color: #007bff;
}

.btn-send-image:hover:not(:disabled) {
  background-color: #0056b3;
}

.image-preview {
  display: flex;
  gap: 15px;
  align-items: flex-start;
  padding: 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  background-color: white;
}

.preview-img {
  max-width: 200px;
  max-height: 200px;
  object-fit: contain;
  border-radius: 4px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.image-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.image-info p {
  margin: 0;
  font-size: 14px;
  color: #666;
}

/* 视觉功能区域样式 */
.vision-section {
  margin: 20px 0;
  padding: 15px;
  border: 1px solid #ddd;
  border-radius: 8px;
  background-color: #f9f9f9;
}

.vision-section h3 {
  margin: 0 0 15px 0;
  color: #333;
  font-size: 16px;
}

.vision-controls {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.vision-buttons {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.btn-vision {
  background-color: #6f42c1;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  white-space: nowrap;
}

.btn-vision:hover:not(:disabled) {
  background-color: #5a32a3;
}

.btn-vision:disabled {
  background-color: #6c757d;
  cursor: not-allowed;
}

.vision-input {
  display: flex;
  gap: 10px;
  align-items: center;
}

.vision-prompt-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #ccc;
  border-radius: 4px;
  font-size: 14px;
  outline: none;
}

.vision-prompt-input:focus {
  border-color: #6f42c1;
  box-shadow: 0 0 0 2px rgba(111, 66, 193, 0.25);
}

.vision-prompt-input:disabled {
  background-color: #e9ecef;
  cursor: not-allowed;
}

.btn-send-vision {
  background-color: #6f42c1;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  white-space: nowrap;
}

.btn-send-vision:hover:not(:disabled) {
  background-color: #5a32a3;
}

.btn-send-vision:disabled {
  background-color: #6c757d;
  cursor: not-allowed;
}

/* IoT设备控制区域样式 */
.iot-section {
  margin: 20px 0;
  padding: 15px;
  border: 1px solid #ddd;
  border-radius: 8px;
  background-color: #f9f9f9;
}

.iot-section h3 {
  margin: 0 0 15px 0;
  color: #333;
  font-size: 16px;
}

.iot-controls {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.iot-input-group {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.input-row {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.input-row label {
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.iot-textarea {
  padding: 8px 12px;
  border: 1px solid #ccc;
  border-radius: 4px;
  font-size: 14px;
  font-family: 'Courier New', monospace;
  resize: vertical;
  outline: none;
}

.iot-textarea:focus {
  border-color: #fd7e14;
  box-shadow: 0 0 0 2px rgba(253, 126, 20, 0.25);
}

.iot-textarea:disabled {
  background-color: #e9ecef;
  cursor: not-allowed;
}

.iot-buttons {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.btn-send-iot,
.btn-clear-iot {
  background-color: #fd7e14;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  white-space: nowrap;
}

.btn-send-iot:hover:not(:disabled),
.btn-clear-iot:hover:not(:disabled) {
  background-color: #e8690b;
}

.btn-send-iot:disabled,
.btn-clear-iot:disabled {
  background-color: #6c757d;
  cursor: not-allowed;
}

.btn-clear-iot {
  background-color: #6c757d;
}

.btn-clear-iot:hover:not(:disabled) {
  background-color: #5a6268;
}

@media (max-width: 768px) {
  .audio-controls {
    grid-template-columns: 1fr;
  }
  
  .recording-controls {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .audio-player {
    flex-direction: column;
  }
}
</style>