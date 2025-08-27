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

      <!-- 会话信息 -->
      <div class="session-info" v-if="sessionId">
        <p><strong>会话ID:</strong> {{ sessionId }}</p>
        <p><strong>设备ID:</strong> {{ deviceId }}</p>
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
          <div class="audio-player" v-if="hasAudio">
            <button @click="playAudio" :disabled="!hasAudio || isAudioPlaying">
              {{ isAudioPlaying ? '播放中...' : '播放音频' }}
            </button>
            <button @click="pauseAudio" :disabled="!isAudioPlaying || ttsStatus === 'paused'">
              暂停播放
            </button>
            <button @click="resumeAudio" :disabled="ttsStatus !== 'paused'">
              恢复播放
            </button>
            <button @click="stopAudio" :disabled="!isAudioPlaying && ttsStatus !== 'paused'">
              停止播放
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

      <!-- 图片上传区域 -->
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
      </div>

      <!-- 视觉功能区域 -->
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
      </div>

      <!-- IoT设备控制区域 -->
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
      </div>

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

// WebSocket相关
const wsRef = ref(null)
const wsConnected = ref(false)
const sessionId = ref('')
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 5
const reconnectDelay = ref(3000) // 初始重连延迟3秒
const deviceId = ref('web-client-' + Date.now())
const clientId = ref('web-' + Math.random().toString(36).substr(2, 9))

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
const sampleRate = ref(16000)
const channels = ref(1)
const frameDuration = ref(60)

// ASR 相关
const asrText = ref('')

// TTS 相关
const ttsStatus = ref('idle') // idle, loading, playing, error, paused
const currentTtsText = ref('')
const hasAudio = ref(false)
const isAudioPlaying = ref(false)
const audioChunks = ref([])
const currentAudio = ref(null)
const audioChunksCount = ref(0)

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
  
  // 根据配置文件，WebSocket服务器运行在8000端口
  const wsUrl = `ws://localhost:8000/`
  
  const ws = new WebSocket(wsUrl)
  
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
  const sampleRate = 16000 // 16kHz采样率
  const numChannels = 1 // 单声道
  const bitsPerSample = 16 // 16位深度
  const byteRate = sampleRate * numChannels * bitsPerSample / 8
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
  view.setUint32(24, sampleRate, true) // Sample rate
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
 * 处理音频数据
 */
const handleAudioData = async (data) => {
  try {
    console.log('收到音频数据:', {
      type: data.constructor.name,
      size: data.byteLength || data.size,
      isArrayBuffer: data instanceof ArrayBuffer,
      isBlob: data instanceof Blob
    })
    
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
    
    // 如果是ArrayBuffer，转换为Blob
    let audioBlob
    if (data instanceof ArrayBuffer) {
      // 检查数据内容
      const uint8Array = new Uint8Array(data)
      console.log('音频数据前16字节:', Array.from(uint8Array.slice(0, 16)).map(b => b.toString(16).padStart(2, '0')).join(' '))
      
      // 尝试识别音频格式
      let mimeType = 'audio/wav' // 默认WAV
      let processedData = data
      
      if (uint8Array[0] === 0x4F && uint8Array[1] === 0x67 && uint8Array[2] === 0x67 && uint8Array[3] === 0x53) {
        mimeType = 'audio/ogg' // OGG格式
      } else if (uint8Array[0] === 0xFF && (uint8Array[1] & 0xE0) === 0xE0) {
        mimeType = 'audio/mpeg' // MP3格式
      } else if (uint8Array[0] === 0x52 && uint8Array[1] === 0x49 && uint8Array[2] === 0x46 && uint8Array[3] === 0x46) {
        mimeType = 'audio/wav' // WAV格式
      } else {
        // 可能是PCM原始数据，需要添加WAV头
        console.log('检测到PCM原始数据，添加WAV头')
        processedData = addWavHeader(uint8Array)
        mimeType = 'audio/wav'
      }
      
      audioBlob = new Blob([processedData], { type: mimeType })
      console.log('检测到音频格式:', mimeType)
    } else if (data instanceof Blob) {
      audioBlob = data
      
      // 检查Blob内容
      const arrayBuffer = await data.arrayBuffer()
      const uint8Array = new Uint8Array(arrayBuffer)
      console.log('Blob音频数据前16字节:', Array.from(uint8Array.slice(0, 16)).map(b => b.toString(16).padStart(2, '0')).join(' '))
    } else {
      console.error('未知的音频数据类型:', typeof data)
      addMessage('error', `未知的音频数据类型: ${typeof data}`)
      return
    }
    
    // 验证音频数据的有效性
    if (audioBlob.size === 0) {
      console.warn('音频Blob为空')
      addMessage('warning', '音频数据为空')
      return
    }
    
    // 将处理后的音频数据添加到数组中
    audioChunks.value.push(audioBlob)
    audioChunksCount.value = audioChunks.value.length
    hasAudio.value = true
    
    console.log('音频数据已添加，当前总块数:', audioChunks.value.length)
    addMessage('audio', `接收音频数据: ${(audioBlob.size / 1024).toFixed(2)}KB`)
    
  } catch (error) {
    console.error('处理音频数据时出错:', error)
    addMessage('error', `音频处理失败: ${error.message}`)
    
    // 尝试恢复
    if (error.name === 'QuotaExceededError') {
      addMessage('warning', '音频缓存已满，清理旧数据')
      // 清理一半的旧音频数据
      const halfLength = Math.floor(audioChunks.value.length / 2)
      audioChunks.value.splice(0, halfLength)
      audioChunksCount.value = audioChunks.value.length
    }
  }
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
  
  switch (state) {
    case 'start':
      ttsStatus.value = 'loading'
      currentTtsText.value = ''
      audioChunks.value = []
      hasAudio.value = false
      addMessage('tts_start', 'TTS服务启动')
      break
      
    case 'sentence_start':
      ttsStatus.value = 'loading'
      currentTtsText.value = message.text || ''
      audioChunks.value = []
      hasAudio.value = false
      addMessage('tts_start', `开始合成: ${message.text}`)
      break
      
    case 'sentence_end':
      ttsStatus.value = 'idle'
      addMessage('tts_end', `合成完成: ${message.text}`)
      
      // 如果有音频数据，自动播放
      if (audioChunks.value.length > 0) {
        setTimeout(() => {
          createAndPlayAudio()
        }, 100)
      }
      break
      
    case 'stop':
      ttsStatus.value = 'idle'
      currentTtsText.value = ''
      addMessage('tts_stop', 'TTS服务停止')
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
    mode: 'manual'
  }
  
  wsRef.value.send(JSON.stringify(listenMessage))
  console.log('发送Listen消息:', listenMessage)
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
 * 创建并播放音频
 */
const createAndPlayAudio = async () => {
  if (audioChunks.value.length === 0) {
    console.warn('没有音频数据可播放')
    return
  }
  
  try {
    // 处理不同类型的音频数据
    const audioBlobs = []
    
    for (const chunk of audioChunks.value) {
      if (chunk instanceof ArrayBuffer) {
        audioBlobs.push(new Blob([chunk]))
      } else if (chunk instanceof Blob) {
        audioBlobs.push(chunk)
      } else {
        // 如果是其他类型，尝试转换为ArrayBuffer
        audioBlobs.push(new Blob([new Uint8Array(chunk)]))
      }
    }
    
    // 合并所有音频数据并创建音频对象
    console.log('开始创建和播放音频，数据块数量:', audioBlobs.length)
  console.log('音频数据总大小:', audioBlobs.reduce((total, blob) => total + blob.size, 0), '字节')
  
  // 详细检查每个音频块
  audioBlobs.forEach((blob, index) => {
    console.log(`音频块 ${index + 1}:`, {
      size: blob.size,
      type: blob.type || '未知类型'
    })
  })
  
  // 检查是否有有效的音频数据
  if (audioBlobs.length === 0) {
    throw new Error('没有音频数据可播放')
  }
  
  const totalSize = audioBlobs.reduce((total, blob) => total + blob.size, 0)
  if (totalSize === 0) {
    throw new Error('音频数据为空')
  }
    
    // 检测音频格式
    const detectAudioFormat = (audioData) => {
      if (!audioData || audioData.length === 0) return null
      
      const firstBytes = new Uint8Array(audioData.slice(0, 12))
      console.log('音频数据前12字节:', Array.from(firstBytes).map(b => b.toString(16).padStart(2, '0')).join(' '))
      
      // WAV格式检测 (RIFF...WAVE)
      if (firstBytes[0] === 0x52 && firstBytes[1] === 0x49 && firstBytes[2] === 0x46 && firstBytes[3] === 0x46 &&
          firstBytes[8] === 0x57 && firstBytes[9] === 0x41 && firstBytes[10] === 0x56 && firstBytes[11] === 0x45) {
        return { type: 'audio/wav', name: 'WAV (检测)' }
      }
      
      // MP3格式检测 (ID3 tag或MP3 frame header)
      if ((firstBytes[0] === 0x49 && firstBytes[1] === 0x44 && firstBytes[2] === 0x33) || // ID3v2
          (firstBytes[0] === 0xFF && (firstBytes[1] & 0xE0) === 0xE0)) { // MP3 frame header
        return { type: 'audio/mpeg', name: 'MP3 (检测)' }
      }
      
      // OGG格式检测
      if (firstBytes[0] === 0x4F && firstBytes[1] === 0x67 && firstBytes[2] === 0x67 && firstBytes[3] === 0x53) {
        return { type: 'audio/ogg', name: 'OGG (检测)' }
      }
      
      // WebM格式检测
      if (firstBytes[0] === 0x1A && firstBytes[1] === 0x45 && firstBytes[2] === 0xDF && firstBytes[3] === 0xA3) {
        return { type: 'audio/webm', name: 'WebM (检测)' }
      }
      
      return null
    }
    
    // 检查第一个音频块的格式
    let detectedFormat = null
    if (audioBlobs.length > 0) {
      const firstBlob = audioBlobs[0]
      const arrayBuffer = await firstBlob.arrayBuffer()
      detectedFormat = detectAudioFormat(new Uint8Array(arrayBuffer))
      console.log('检测到的音频格式:', detectedFormat)
    }
    
    // 尝试不同的音频格式，优先使用检测到的格式
    const audioFormats = [
      // 如果检测到格式，优先使用检测到的格式
      ...(detectedFormat ? [detectedFormat] : []),
      { type: 'audio/wav', name: 'WAV' },
      { type: 'audio/mpeg', name: 'MP3' },
      { type: 'audio/mp3', name: 'MP3-Alt' },
      { type: 'audio/ogg', name: 'OGG' },
      { type: 'audio/ogg; codecs=opus', name: 'OGG-Opus' },
      { type: 'audio/webm', name: 'WebM' },
      { type: 'audio/webm; codecs=opus', name: 'WebM-Opus' },
      { type: 'audio/x-wav', name: 'X-WAV' },
      { type: 'audio/wave', name: 'WAVE' },
      { type: '', name: '默认' } // 不指定类型，让浏览器自动检测
    ]
    
    let audioUrl = null
    let audio = null
    let successFormat = null
    
    // 如果只有一个音频块，直接使用它
    if (audioBlobs.length === 1) {
      console.log('使用单个音频块')
      const singleBlob = audioBlobs[0]
      
      for (const format of audioFormats) {
        try {
          console.log(`尝试格式: ${format.name} (${format.type})`)
          
          // 如果有检测到的格式且当前格式匹配，或者没有指定类型，直接使用原始blob
          if ((detectedFormat && format.type === detectedFormat.type) || !format.type) {
            audioUrl = URL.createObjectURL(singleBlob)
          } else {
            // 创建指定类型的新blob
            const typedBlob = new Blob([singleBlob], { type: format.type })
            audioUrl = URL.createObjectURL(typedBlob)
          }
          
          audio = new Audio(audioUrl)
          audio.preload = 'auto'
          successFormat = format.name
          console.log(`使用格式: ${format.name}`)
          break
          
        } catch (formatError) {
          console.log(`格式 ${format.name} 创建失败:`, formatError.message)
          if (audioUrl) {
            URL.revokeObjectURL(audioUrl)
            audioUrl = null
          }
          audio = null
          continue
        }
      }
    } else {
      // 多个音频块需要合并
      console.log('合并多个音频块')
      for (const format of audioFormats) {
        try {
          console.log(`尝试格式: ${format.name} (${format.type})`)
          
          const combinedBlob = new Blob(audioBlobs, format.type ? { type: format.type } : {})
          audioUrl = URL.createObjectURL(combinedBlob)
          audio = new Audio(audioUrl)
          audio.preload = 'auto'
          successFormat = format.name
          console.log(`使用格式: ${format.name}`)
          break
          
        } catch (formatError) {
          console.log(`格式 ${format.name} 创建失败:`, formatError.message)
          if (audioUrl) {
            URL.revokeObjectURL(audioUrl)
            audioUrl = null
          }
          audio = null
          continue
        }
      }
    }
    
    if (!audio || !audioUrl) {
      // 如果所有格式都失败，尝试最基本的方式
      console.log('尝试基本音频创建方式')
      const combinedBlob = new Blob(audioBlobs)
      audioUrl = URL.createObjectURL(combinedBlob)
      audio = new Audio(audioUrl)
      successFormat = '基本格式'
    }
    
    // 设置音频事件监听器
    audio.onloadstart = () => {
      console.log(`音频开始加载，使用格式: ${successFormat}`)
    }
    
    audio.onloadeddata = () => {
      console.log('音频数据加载完成')
    }
    
    audio.oncanplay = () => {
      console.log('音频可以开始播放')
    }
    
    audio.onplay = () => {
      isAudioPlaying.value = true
      ttsStatus.value = 'playing'
      console.log('音频开始播放')
      addMessage('system', `开始播放TTS音频 (${successFormat})`)
    }
    
    audio.onended = () => {
      isAudioPlaying.value = false
      ttsStatus.value = 'idle'
      URL.revokeObjectURL(audioUrl)
      console.log('音频播放完成')
      addMessage('system', 'TTS音频播放完成')
    }
    
    audio.onerror = (error) => {
      console.error('音频播放错误:', error)
      console.error('音频错误详情:', {
        error: audio.error,
        networkState: audio.networkState,
        readyState: audio.readyState,
        src: audio.src
      })
      isAudioPlaying.value = false
      ttsStatus.value = 'error'
      URL.revokeObjectURL(audioUrl)
      
      const errorMsg = audio.error ? `音频播放失败 (错误代码: ${audio.error.code})` : '音频播放失败'
      message.error(errorMsg)
      addMessage('error', errorMsg)
    }
    
    audio.onpause = () => {
      console.log('音频暂停')
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
  createAndPlayAudio()
}

/**
 * 停止音频播放
 */
const stopAudio = () => {
  if (currentAudio.value) {
    currentAudio.value.pause()
    currentAudio.value.currentTime = 0
    isAudioPlaying.value = false
    ttsStatus.value = 'idle'
    console.log('音频播放已停止')
    addMessage('system', '音频播放已停止')
  }
}

/**
 * 暂停音频播放
 */
const pauseAudio = () => {
  if (currentAudio.value && !currentAudio.value.paused) {
    currentAudio.value.pause()
    isAudioPlaying.value = false
    ttsStatus.value = 'paused'
    console.log('音频播放已暂停')
    addMessage('system', '音频播放已暂停')
  }
}

/**
 * 恢复音频播放
 */
const resumeAudio = () => {
  if (currentAudio.value && currentAudio.value.paused) {
    currentAudio.value.play()
    isAudioPlaying.value = true
    ttsStatus.value = 'playing'
    console.log('音频播放已恢复')
    addMessage('system', '音频播放已恢复')
  }
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

// 生命周期
onMounted(() => {
  connectWebSocket()
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
}

.btn-record,
.btn-stop {
  padding: 12px 24px;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
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
  gap: 10px;
}

.audio-player button {
  padding: 10px 20px;
  border: 1px solid #007bff;
  background-color: #007bff;
  color: white;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.audio-player button:hover:not(:disabled) {
  background-color: #0056b3;
}

.audio-player button:disabled {
  background-color: #6c757d;
  border-color: #6c757d;
  cursor: not-allowed;
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