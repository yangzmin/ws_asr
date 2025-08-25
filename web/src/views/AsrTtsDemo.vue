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

// WebSocket 相关
const wsRef = ref(null)
const wsConnected = ref(false)
const sessionId = ref('')
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

// 对话历史
const messages = ref([])
const messagesContainer = ref(null)

// 调试
const showDebug = ref(false)

/**
 * 连接WebSocket服务器
 */
const connectWebSocket = () => {
  // 根据配置文件，WebSocket服务器运行在8000端口
  const wsUrl = `ws://localhost:8000/`
  
  const ws = new WebSocket(wsUrl)
  
  // 设置请求头（通过URL参数或连接后发送）
  ws.onopen = () => {
    console.log('WebSocket连接已建立')
    wsConnected.value = true
    
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
  
  ws.onclose = () => {
    console.log('WebSocket连接已关闭')
    wsConnected.value = false
    addMessage('system', 'WebSocket连接已关闭')
    message.error('WebSocket连接已断开')
  }
  
  ws.onerror = (error) => {
    console.error('WebSocket错误:', error)
    message.error(`WebSocket连接失败: ${error}`)
    wsConnected.value = false
    addMessage('error', 'WebSocket连接失败')
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
      return
    }
    
    // 如果是ArrayBuffer，转换为Blob
    let audioBlob
    if (data instanceof ArrayBuffer) {
      // 检查数据内容
      const uint8Array = new Uint8Array(data)
      console.log('音频数据前16字节:', Array.from(uint8Array.slice(0, 16)).map(b => b.toString(16).padStart(2, '0')).join(' '))
      
      audioBlob = new Blob([data], { type: 'audio/mpeg' }) // 默认尝试MP3格式
    } else if (data instanceof Blob) {
      audioBlob = data
      
      // 检查Blob内容
      const arrayBuffer = await data.arrayBuffer()
      const uint8Array = new Uint8Array(arrayBuffer)
      console.log('Blob音频数据前16字节:', Array.from(uint8Array.slice(0, 16)).map(b => b.toString(16).padStart(2, '0')).join(' '))
    } else {
      console.error('未知的音频数据类型:', typeof data)
      return
    }
    
    // 将处理后的音频数据添加到数组中
    audioChunks.value.push(audioBlob)
    audioChunksCount.value = audioChunks.value.length
    hasAudio.value = true
    
    console.log('音频数据已添加，当前总块数:', audioChunks.value.length)
    
  } catch (error) {
    console.error('处理音频数据时出错:', error)
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
      default:
        console.log('未知消息类型:', message.type)
    }
  } catch (error) {
    console.error('解析WebSocket消息失败:', error)
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
      break
  }
}

/**
 * 处理LLM消息
 */
const handleLlmMessage = (message) => {
  const text = message.text || ''
  llmText.value = text
  addMessage('llm', text)
  console.log('收到LLM回复:', text)
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