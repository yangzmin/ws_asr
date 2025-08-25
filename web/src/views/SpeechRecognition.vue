<template>
  <div class="speech-recognition">
    <h1>🎤 实时语音识别 + AI聊天</h1>
    
    <div class="control-panel">
      <div class="device-selector">
        <label for="audioDevice">选择麦克风：</label>
        <select id="audioDevice" v-model="selectedDevice" @change="updateAudioDevice">
          <option v-for="device in audioDevices" :key="device.deviceId" :value="device.deviceId">
            {{ device.label }}
          </option>
        </select>
      </div>
      
      <div class="recording-controls">
        <button @click="startRecording" :disabled="isRecording" class="start-btn">
          开始录音
        </button>
        <button @click="stopRecording" :disabled="!isRecording" class="stop-btn">
          停止录音
        </button>
      </div>
      
      <div class="status-indicators">
        <div class="status-item">
          <span class="status-label">WebSocket:</span>
          <span :class="wsConnected ? 'status-connected' : 'status-disconnected'">
            {{ wsConnected ? '已连接' : '未连接' }}
          </span>
        </div>
        <div class="status-item">
          <span class="status-label">录音状态:</span>
          <span :class="isRecording ? 'status-recording' : 'status-idle'">
            {{ isRecording ? '录音中' : '空闲' }}
          </span>
        </div>
      </div>
    </div>

    <div class="results-section">
      <div class="recognition-results">
        <h3>语音识别结果:</h3>
        <div class="text-box">
          <div class="interim-text">{{ recognitionText }}</div>
          <div class="final-text">{{ fullText }}</div>
        </div>
      </div>

      <div class="chat-results">
        <h3>AI聊天回复:</h3>
        <div class="chat-box">
          <div v-if="chatLoading" class="chat-loading">正在思考中...</div>
          <div class="chat-content">{{ chatResponse }}</div>
          <div v-if="chatError" class="chat-error">{{ chatError }}</div>
        </div>
        
        <!-- TTS音频播放控制 -->
        <div class="tts-controls" v-if="chatResponse">
          <div class="tts-status">
            <span class="status-label">语音合成:</span>
            <span :class="ttsStatus === 'playing' ? 'status-playing' : ttsStatus === 'loading' ? 'status-loading' : 'status-idle'">
              {{ ttsStatusText }}
            </span>
          </div>
          <div class="audio-controls">
            <button @click="toggleAudio" :disabled="!hasAudio" class="audio-btn">
              {{ isAudioPlaying ? '暂停播放' : '播放语音' }}
            </button>
            <button @click="stopAudio" :disabled="!hasAudio" class="audio-btn stop-audio">
              停止播放
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="audio-visualization">
      <canvas ref="canvas" width="800" height="200"></canvas>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { message } from 'ant-design-vue'
import { createAudioProcessor, audioDataToWavBase64 } from '@/utils/audioUtils'

// 响应式数据
const selectedDevice = ref('')
const audioDevices = ref([])
const isRecording = ref(false)
const wsConnected = ref(false)
const recognitionText = ref('')
const fullText = ref('')
const chatResponse = ref('')
const chatLoading = ref(false)
const chatError = ref('')

// TTS相关状态
const ttsStatus = ref('idle') // idle, loading, playing, paused
const isAudioPlaying = ref(false)
const hasAudio = ref(false)
const audioChunks = ref([])
const currentAudio = ref(null)

// refs
const canvas = ref(null)
const wsRef = ref(null)
const processorRef = ref(null)
const wsCloseTimerRef = ref(null)

// 计算属性
const ttsStatusText = computed(() => {
  switch (ttsStatus.value) {
    case 'loading': return '合成中...'
    case 'playing': return '播放中'
    case 'paused': return '已暂停'
    default: return '就绪'
  }
})

onMounted(async () => {
  await getAudioDevices()
})

onUnmounted(() => {
  if (wsRef.value) {
    wsRef.value.close()
  }
  if (wsCloseTimerRef.value) {
    clearTimeout(wsCloseTimerRef.value)
  }
  stopRecording()
})

const getAudioDevices = async () => {
  try {
    const devices = await navigator.mediaDevices.enumerateDevices()
    const audioInputs = devices
      .filter(device => device.kind === 'audioinput')
      .map(device => ({
        deviceId: device.deviceId,
        label: device.label || `麦克风 ${device.deviceId.slice(0, 8)}`
      }))
    audioDevices.value = audioInputs
    if (audioInputs.length > 0) {
      selectedDevice.value = audioInputs[0].deviceId
    }
  } catch (error) {
    console.error('获取音频设备失败:', error)
    message.error('无法获取音频设备')
  }
}

// WebSocket连接
const connectWebSocket = () => {
  const ws = new WebSocket('ws://localhost:8080/api/ws')
  
  ws.onopen = () => {
    console.log('WebSocket连接已建立')
    wsConnected.value = true
    
    // 发送开始识别消息
    const startMessage = {
      type: 'start_recognition',
      audio_config: {
        sample_rate: 16000,
        channels: 1,
        format: 'pcm'
      }
    }
    ws.send(JSON.stringify(startMessage))
  }
  
  ws.onmessage = (event) => {
    try {
      const result = JSON.parse(event.data)
      
      // 处理语音识别结果
      if (result.type === 'recognition_result') {
        if (result.is_final) {
          fullText.value += result.text + ' '
          recognitionText.value = ''
        } else {
          recognitionText.value = result.text
        }
      }
      
      // 处理聊天流式回复
      else if (result.type === 'chat_chunk') {
        chatLoading.value = false
        chatError.value = ''
        chatResponse.value += result.content || ''
      }
      
      // 处理聊天完成
      else if (result.type === 'chat_done') {
        chatLoading.value = false
        console.log('AI聊天回复完成')
      }
      
      // 处理聊天错误
      else if (result.type === 'chat_error') {
        chatLoading.value = false
        chatError.value = result.error || '聊天出现错误'
      }
      
      // 处理TTS开始
      else if (result.type === 'tts_start') {
        console.log('TTS开始合成')
        ttsStatus.value = 'loading'
        audioChunks.value = []
        hasAudio.value = false
      }
      
      // 处理TTS音频数据
      else if (result.type === 'tts_audio') {
        if (result.audio_data) {
          audioChunks.value.push(result.audio_data)
          hasAudio.value = true
        }
      }
      
      // 处理TTS完成
      else if (result.type === 'tts_done') {
        console.log('TTS合成完成')
        ttsStatus.value = 'idle'
        if (audioChunks.value.length > 0) {
          createAudioFromChunks()
        }
      }
      
      // 处理TTS错误
      else if (result.type === 'tts_error') {
        console.error('TTS错误:', result.error)
        ttsStatus.value = 'idle'
        message.error('语音合成失败: ' + (result.error || '未知错误'))
      }
      
    } catch (error) {
      console.error('解析消息失败:', error)
    }
  }
  
  ws.onclose = () => {
    console.log('WebSocket连接已关闭')
    wsConnected.value = false
  }
  
  ws.onerror = (error) => {
    console.error('WebSocket错误:', error)
    message.error('WebSocket连接失败')
    wsConnected.value = false
  }
  
  wsRef.value = ws
}

// 开始录音
const startRecording = async () => {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        deviceId: selectedDevice.value ? { exact: selectedDevice.value } : undefined,
        sampleRate: 16000,
        channelCount: 1,
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true
      }
    })

    // 重置聊天状态
    chatResponse.value = ''
    chatError.value = ''
    chatLoading.value = false
    
    // 重置TTS状态
    ttsStatus.value = 'idle'
    isAudioPlaying.value = false
    hasAudio.value = false
    audioChunks.value = []
    if (currentAudio.value) {
      currentAudio.value.pause()
      currentAudio.value = null
    }

    // 连接WebSocket
    connectWebSocket()

    isRecording.value = true
    
    // 创建音频处理器（内部已固定采样率为16k）
    const processor = createAudioProcessor(stream, (audioData) => {
      // 发送音频数据到服务器（发送PCM，无WAV头）
      if (wsRef.value && wsRef.value.readyState === WebSocket.OPEN) {
        const messageObj = {
          type: 'audio_data',
          data: audioDataToWavBase64(audioData, 16000, 1, false),
          sequence: Date.now(),
          is_final: false
        }
        wsRef.value.send(JSON.stringify(messageObj))
        
        // 可视化音频
        visualizeAudio(audioData)
      }
    })
    
    processorRef.value = processor
    
    message.success('录音已开始')
  } catch (error) {
    console.error('开始录音失败:', error)
    message.error('无法访问麦克风')
  }
}

// 停止录音
const stopRecording = () => {
  if (isRecording.value) {
    isRecording.value = false
    
    // 发送最后一个音频包
    if (wsRef.value && wsRef.value.readyState === WebSocket.OPEN) {
      const finalMessage = {
        type: 'audio_data',
        data: '',
        sequence: Date.now(),
        is_final: true
      }
      wsRef.value.send(JSON.stringify(finalMessage))
      
      // 发送停止识别消息
      const stopMessage = {
        type: 'stop_recognition'
      }
      wsRef.value.send(JSON.stringify(stopMessage))
      
      // 设置聊天加载状态
      chatLoading.value = true
    }
    
    // 停止音频处理
    if (processorRef.value) {
      processorRef.value.stop()
      processorRef.value = null
    }
    
    message.success('录音已停止')
  }
}

// 更新音频设备
const updateAudioDevice = () => {
  if (isRecording.value) {
    message.warning('请先停止录音再切换设备')
    return
  }
  console.log('切换音频设备:', selectedDevice.value)
}

// 音频可视化
const visualizeAudio = (audioData) => {
  if (!canvas.value) return
  
  const ctx = canvas.value.getContext('2d')
  const width = canvas.value.width
  const height = canvas.value.height
  
  ctx.clearRect(0, 0, width, height)
  ctx.fillStyle = '#1890ff'
  
  const barWidth = width / audioData.length
  for (let i = 0; i < audioData.length; i++) {
    const barHeight = (audioData[i] + 1) * height / 4
    ctx.fillRect(i * barWidth, height / 2 - barHeight / 2, barWidth - 1, barHeight)
  }
}

// TTS音频处理函数
const createAudioFromChunks = () => {
  try {
    // 将所有base64音频块合并
    const combinedBase64 = audioChunks.value.join('')
    
    // 转换为二进制数据
    const binaryString = atob(combinedBase64)
    const bytes = new Uint8Array(binaryString.length)
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i)
    }
    
    // 创建音频blob
    const audioBlob = new Blob([bytes], { type: 'audio/wav' })
    const audioUrl = URL.createObjectURL(audioBlob)
    
    // 创建音频元素
    const audio = new Audio(audioUrl)
    
    // 设置音频事件监听器
    audio.onplay = () => {
      ttsStatus.value = 'playing'
      isAudioPlaying.value = true
    }
    
    audio.onpause = () => {
      ttsStatus.value = 'paused'
      isAudioPlaying.value = false
    }
    
    audio.onended = () => {
      ttsStatus.value = 'idle'
      isAudioPlaying.value = false
      URL.revokeObjectURL(audioUrl)
    }
    
    audio.onerror = (error) => {
      console.error('音频播放错误:', error)
      ttsStatus.value = 'idle'
      isAudioPlaying.value = false
      message.error('音频播放失败')
      URL.revokeObjectURL(audioUrl)
    }
    
    currentAudio.value = audio
    hasAudio.value = true
    
    // 自动开始播放
    audio.play().catch(error => {
      console.error('自动播放失败:', error)
      message.error('音频自动播放失败')
      ttsStatus.value = 'idle'
    })
    
    console.log('音频创建成功，自动播放中')
  } catch (error) {
    console.error('创建音频失败:', error)
    message.error('音频处理失败')
  }
}

// 切换音频播放/暂停
const toggleAudio = () => {
  if (!currentAudio.value) return
  
  if (isAudioPlaying.value) {
    currentAudio.value.pause()
  } else {
    currentAudio.value.play().catch(error => {
      console.error('播放失败:', error)
      message.error('音频播放失败')
    })
  }
}

// 停止音频播放
const stopAudio = () => {
  if (!currentAudio.value) return
  
  currentAudio.value.pause()
  currentAudio.value.currentTime = 0
  ttsStatus.value = 'idle'
  isAudioPlaying.value = false
}
</script>

<style scoped>
.speech-recognition {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

h1 {
  text-align: center;
  color: #1890ff;
  margin-bottom: 30px;
}

.control-panel {
  background: #f5f5f5;
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.device-selector {
  margin-bottom: 20px;
}

.device-selector label {
  display: inline-block;
  width: 100px;
  font-weight: bold;
}

.device-selector select {
  padding: 8px 12px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  width: 300px;
}

.recording-controls {
  margin-bottom: 20px;
}

.start-btn, .stop-btn {
  padding: 10px 20px;
  margin-right: 10px;
  border: none;
  border-radius: 4px;
  font-size: 16px;
  cursor: pointer;
  transition: background-color 0.3s;
}

.start-btn {
  background-color: #52c41a;
  color: white;
}

.start-btn:hover:not(:disabled) {
  background-color: #73d13d;
}

.stop-btn {
  background-color: #ff4d4f;
  color: white;
}

.stop-btn:hover:not(:disabled) {
  background-color: #ff7875;
}

.start-btn:disabled, .stop-btn:disabled {
  background-color: #d9d9d9;
  cursor: not-allowed;
}

.status-indicators {
  display: flex;
  gap: 20px;
}

.status-item {
  display: flex;
  align-items: center;
}

.status-label {
  font-weight: bold;
  margin-right: 8px;
}

.status-connected {
  color: #52c41a;
}

.status-disconnected {
  color: #ff4d4f;
}

.status-recording {
  color: #ff7875;
  animation: pulse 1s infinite;
}

.status-idle {
  color: #8c8c8c;
}

@keyframes pulse {
  0% { opacity: 1; }
  50% { opacity: 0.5; }
  100% { opacity: 1; }
}

.results-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-gap: 20px;
  margin-bottom: 20px;
}

.recognition-results, .chat-results {
  background: white;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  padding: 20px;
}

.recognition-results h3, .chat-results h3 {
  margin-top: 0;
  color: #1890ff;
}

.text-box, .chat-box {
  min-height: 200px;
  max-height: 400px;
  overflow-y: auto;
  padding: 15px;
  background: #fafafa;
  border-radius: 4px;
  border: 1px solid #e8e8e8;
}

.interim-text {
  color: #8c8c8c;
  font-style: italic;
  margin-bottom: 10px;
}

.final-text {
  color: #262626;
  line-height: 1.6;
}

.chat-content {
  color: #262626;
  line-height: 1.6;
  white-space: pre-wrap;
}

.chat-loading {
  color: #1890ff;
  font-style: italic;
  animation: pulse 1s infinite;
}

.chat-error {
  color: #ff4d4f;
  font-weight: bold;
  margin-top: 10px;
}

.tts-controls {
  margin-top: 15px;
  padding-top: 15px;
  border-top: 1px solid #e8e8e8;
}

.tts-status {
  margin-bottom: 10px;
}

.status-playing {
  color: #52c41a;
  animation: pulse 1s infinite;
}

.status-loading {
  color: #1890ff;
  animation: pulse 1s infinite;
}

.audio-controls {
  display: flex;
  gap: 10px;
}

.audio-btn {
  padding: 8px 16px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: white;
  color: #262626;
  cursor: pointer;
  transition: all 0.3s;
}

.audio-btn:hover:not(:disabled) {
  border-color: #1890ff;
  color: #1890ff;
}

.audio-btn:disabled {
  background: #f5f5f5;
  color: #bfbfbf;
  cursor: not-allowed;
}

.stop-audio {
  border-color: #ff4d4f;
  color: #ff4d4f;
}

.stop-audio:hover:not(:disabled) {
  background: #ff4d4f;
  color: white;
}

.audio-visualization {
  background: white;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  padding: 20px;
  text-align: center;
}

canvas {
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  background: #fafafa;
}

@media (max-width: 768px) {
  .results-section {
    grid-template-columns: 1fr;
  }
  
  .device-selector select {
    width: 100%;
  }
  
  canvas {
    width: 100%;
    height: 150px;
  }
}
</style>