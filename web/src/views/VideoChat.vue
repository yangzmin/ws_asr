<template>
  <div class="video-chat">
    <a-row :gutter="24">
      <!-- 视频信息侧边栏 -->
      <a-col :xs="24" :lg="8">
        <a-card title="📹 视频信息" class="video-info-card">
          <template #extra>
            <a-button @click="$router.push('/')">
              <template #icon>
                <arrow-left-outlined />
              </template>
              返回首页
            </a-button>
          </template>
          
          <div v-if="currentVideo">
            <div class="video-preview">
              <video 
                v-if="currentVideo.filePath"
                :src="getVideoUrl(currentVideo.id)"
                controls
                preload="metadata"
                class="video-player"
                @error="handleVideoError"
              >
                您的浏览器不支持视频播放。
              </video>
              <div v-else class="video-placeholder">
                <video-camera-outlined style="font-size: 64px; color: #1890ff" />
                <p>视频文件不可用</p>
              </div>
            </div>
            
            <a-descriptions :column="1" bordered size="small">
              <a-descriptions-item label="视频名称">
                {{ currentVideo.name }}
              </a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-tag :color="getStatusColor(currentVideo.status)">
                  {{ getStatusText(currentVideo.status) }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="上传时间">
                {{ formatDate(currentVideo.uploadAt) }}
              </a-descriptions-item>
              <a-descriptions-item label="更新时间">
                {{ formatDate(currentVideo.updatedAt) }}
              </a-descriptions-item>
            </a-descriptions>
            
            <div class="action-buttons">
              <a-button 
                type="primary" 
                danger 
                block 
                @click="clearHistory"
                :loading="clearing"
              >
                <template #icon>
                  <delete-outlined />
                </template>
                清除聊天记录
              </a-button>
            </div>
          </div>
          
          <a-skeleton v-else active />
        </a-card>
      </a-col>
      
      <!-- 聊天区域 -->
      <a-col :xs="24" :lg="16">
        <a-card title="💬 视频聊天" class="chat-card">
          <div class="chat-container">
            <!-- 聊天消息列表 -->
            <div class="chat-messages" ref="messagesContainer">
              <div v-if="chatHistory.length === 0" class="empty-chat">
                <a-empty description="暂无聊天记录">
                  <template #image>
                    <message-outlined style="font-size: 48px; color: #d9d9d9" />
                  </template>
                  <p>开始与视频对话吧！</p>
                </a-empty>
              </div>
              
              <div v-else>
                <div 
                  v-for="message in chatHistory" 
                  :key="message.id" 
                  class="message-item"
                >
                  <!-- 用户消息 -->
                  <div class="message user-message">
                    <div class="message-avatar">
                      <a-avatar style="background-color: #1890ff">
                        <template #icon>
                          <user-outlined />
                        </template>
                      </a-avatar>
                    </div>
                    <div class="message-content">
                      <div class="message-bubble user-bubble">
                        {{ message.userMsg }}
                      </div>
                      <div class="message-time">
                        {{ formatTime(message.timestamp) }}
                      </div>
                    </div>
                  </div>
                  
                  <!-- AI回复 -->
                  <div class="message bot-message">
                    <div class="message-avatar">
                      <a-avatar style="background-color: #52c41a">
                        <template #icon>
                          <robot-outlined />
                        </template>
                      </a-avatar>
                    </div>
                    <div class="message-content">
                      <div class="message-bubble bot-bubble" v-html="renderMarkdown(message.botMsg)">
                      </div>
                      <div class="message-time">
                        {{ formatTime(message.timestamp) }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              
              <!-- 正在输入指示器 -->
              <div v-if="sending" class="typing-indicator">
                <div class="message bot-message">
                  <div class="message-avatar">
                    <a-avatar style="background-color: #52c41a">
                      <template #icon>
                        <robot-outlined />
                      </template>
                    </a-avatar>
                  </div>
                  <div class="message-content">
                    <div class="message-bubble bot-bubble typing">
                      <a-spin size="small" /> 
                      <span v-if="translationStatus">{{ translationStatus }}</span>
                      <span v-else-if="streamingResponse">AI正在回复中...</span>
                      <span v-else>AI正在思考中...</span>
                    </div>
                  </div>
                </div>
              </div>
              
              <!-- 流式响应显示 -->
              <div v-if="currentStreamResponse" class="streaming-response">
                <div class="message bot-message">
                  <div class="message-avatar">
                    <a-avatar style="background-color: #52c41a">
                      <template #icon>
                        <robot-outlined />
                      </template>
                    </a-avatar>
                  </div>
                  <div class="message-content">
                    <div class="message-bubble bot-bubble">
                      {{ currentStreamResponse }}
                      <span class="typing-cursor">|</span>

                      <!-- <div class="message-bubble bot-bubble" v-html="renderMarkdown(currentStreamResponse) + '<span class=\'typing-cursor\'>|</span>'"> -->
                    </div>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 输入区域 -->
            <div class="chat-input">
              <!-- 模式切换 -->
              <div class="mode-switch">
                <a-switch 
                  v-model:checked="useStreamMode" 
                  checked-children="🌍 翻译模式" 
                  un-checked-children="📝 直接模式"
                  :disabled="sending"
                />
                <a-tooltip title="翻译模式：自动将任何语言翻译为英文后与视频对话">
                  <question-circle-outlined style="margin-left: 8px; color: #999;" />
                </a-tooltip>
              </div>
              
              <a-input-search
                v-model:value="inputMessage"
                :placeholder="useStreamMode ? '输入任何语言的问题，系统会自动翻译为英文...' : '输入您想了解的关于视频的问题...'"
                enter-button="发送"
                size="large"
                :loading="sending"
                :disabled="!canChat"
                @search="sendMessage"
              >
                <template #enterButton>
                  <a-button type="primary" :loading="sending">
                    <template #icon>
                      <send-outlined />
                    </template>
                    发送
                  </a-button>
                </template>
              </a-input-search>
              
              <div v-if="!canChat" class="chat-disabled-hint">
                <a-alert
                  message="视频还在处理中，请稍后再试"
                  type="warning"
                  show-icon
                  banner
                />
              </div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script>
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { useVideoStore } from '../stores/video'
import MarkdownIt from 'markdown-it'
import {
  ArrowLeftOutlined,
  VideoCameraOutlined,
  MessageOutlined,
  UserOutlined,
  RobotOutlined,
  SendOutlined,
  DeleteOutlined,
  QuestionCircleOutlined
} from '@ant-design/icons-vue'

export default {
  name: 'VideoChat',
  components: {
    ArrowLeftOutlined,
    VideoCameraOutlined,
    MessageOutlined,
    UserOutlined,
    RobotOutlined,
    SendOutlined,
    DeleteOutlined,
    QuestionCircleOutlined
  },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const videoStore = useVideoStore()
    
    // 创建markdown-it实例
    const md = new MarkdownIt()
    
    const videoId = route.params.videoId
    const inputMessage = ref('')
    const sending = ref(false)
    const clearing = ref(false)
    const messagesContainer = ref(null)
    const translationStatus = ref('')
    const streamingResponse = ref(false)
    const currentStreamResponse = ref('')
    const useStreamMode = ref(true) // 默认使用流式模式
    
    // 渲染Markdown内容的方法
    const renderMarkdown = (content) => {
      if (!content) return ''
      return md.render(content)
    }
    
    const currentVideo = computed(() => videoStore.currentVideo)
    const chatHistory = computed(() => videoStore.chatHistory)
    console.log("chatHistory",chatHistory)
    
    const canChat = computed(() => {
      return currentVideo.value && currentVideo.value.status === 'PARSE'
    })
    
    const sendMessage = async () => {
      if (inputMessage.value.trim() === "" || inputMessage.value.trim() === " " || inputMessage.value.trim() === null || inputMessage.value.trim() === undefined) {
        message.warning('请输入消息内容')
        return
      }
      
      if (!canChat.value) {
        message.warning('视频还在处理中，请稍后再试')
        return
      }
      
      const messageText = inputMessage.value.trim()
      inputMessage.value = ''
      sending.value = true
      translationStatus.value = ''
      streamingResponse.value = false
      currentStreamResponse.value = ''
      
      try {
        if (useStreamMode.value) {
          await sendStreamMessage(messageText)
        } else {
          await videoStore.sendChatMessage(videoId, messageText)
          await scrollToBottom()
        }
      } catch (error) {
        message.error('发送消息失败: ' + (error.message || '未知错误'))
        inputMessage.value = messageText // 恢复输入内容
      } finally {
        sending.value = false
        translationStatus.value = ''
        streamingResponse.value = false
        currentStreamResponse.value = ''
      }
    }
    
    const sendStreamMessage = async (messageText) => {
      const onProgress = async (data) => {
        if (data.type === 'translation') {
          translationStatus.value = data.message
        } else if (data.type === 'response') {
          streamingResponse.value = true
          translationStatus.value = ''
          currentStreamResponse.value = data.fullMessage || data.message
          await scrollToBottom()
        }
      }
      
      try {
        await videoStore.sendStreamChatMessage(videoId, messageText, onProgress)
        await scrollToBottom()
      } catch (error) {
        throw error
      }
    }
    
    const clearHistory = async () => {
      clearing.value = true
      try {
        await videoStore.clearChatHistory(videoId)
        message.success('聊天记录已清除')
      } catch (error) {
        message.error('清除聊天记录失败: ' + (error.message || '未知错误'))
      } finally {
        clearing.value = false
      }
    }
    
    const scrollToBottom = async () => {
      await nextTick()
      if (messagesContainer.value) {
        messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
      }
    }
    
    const getStatusColor = (status) => {
      switch (status) {
        case 'PARSE': return 'green'
        case 'UNPARSE': return 'orange'
        case 'PARSE_ERROR': return 'red'
        default: return 'default'
      }
    }
    
    const getStatusText = (status) => {
      switch (status) {
        case 'PARSE': return '解析完成'
        case 'UNPARSE': return '处理中'
        case 'PARSE_ERROR': return '解析失败'
        default: return '未知状态'
      }
    }
    
    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleString('zh-CN')
    }
    
    const formatTime = (dateString) => {
      return new Date(dateString).toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit'
      })
    }
    
    // 获取视频URL
    const getVideoUrl = (videoId) => {
      return `/api/video/file/${videoId}`
    }
    
    // 处理视频加载错误
    const handleVideoError = (event) => {
      console.error('视频加载失败:', event)
      message.error('视频加载失败，请稍后重试')
    }
    
    // 监听聊天历史变化，自动滚动到底部
    watch(chatHistory, () => {
      scrollToBottom()
    }, { deep: true })
    
    onMounted(async () => {
      try {
        await videoStore.fetchVideo(videoId)
        await videoStore.fetchChatHistory(videoId)
        await scrollToBottom()
      } catch (error) {
        message.error('加载视频信息失败')
        router.push('/')
      }
    })
    
    return {
      videoId,
      currentVideo,
      chatHistory,
      inputMessage,
      sending,
      clearing,
      canChat,
      messagesContainer,
      translationStatus,
      streamingResponse,
      currentStreamResponse,
      useStreamMode,
      renderMarkdown,
      sendMessage,
      clearHistory,
      getStatusColor,
      getStatusText,
      formatDate,
      formatTime,
      getVideoUrl,
      handleVideoError
    }
  }
}
</script>

<style scoped>
.video-chat {
  max-width: 1400px;
  margin: 0 auto;
  padding: 24px;
}

/* Markdown样式 */
.bot-bubble :deep(p) {
  margin: 0 0 10px 0;
}

.bot-bubble :deep(ul), .bot-bubble :deep(ol) {
  padding-left: 20px;
  margin: 10px 0;
}

.bot-bubble :deep(h1), .bot-bubble :deep(h2), .bot-bubble :deep(h3), .bot-bubble :deep(h4), .bot-bubble :deep(h5), .bot-bubble :deep(h6) {
  margin: 15px 0 10px 0;
  font-weight: bold;
}

.bot-bubble :deep(code) {
  background-color: rgba(0, 0, 0, 0.05);
  padding: 2px 4px;
  border-radius: 3px;
  font-family: monospace;
}

.bot-bubble :deep(pre) {
  background-color: rgba(0, 0, 0, 0.05);
  padding: 10px;
  border-radius: 5px;
  overflow-x: auto;
  margin: 10px 0;
}

.bot-bubble :deep(blockquote) {
  border-left: 4px solid #ddd;
  padding-left: 10px;
  margin: 10px 0;
  color: #666;
}

.video-info-card,
.chat-card {
  height: calc(100vh - 200px);
  min-height: 600px;
}

.video-preview {
  text-align: center;
  padding: 24px;
  background: #f5f5f5;
  border-radius: 8px;
  margin-bottom: 16px;
}

.video-player {
  width: 100%;
  max-width: 400px;
  height: auto;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.video-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  color: #999;
}

.video-placeholder p {
  margin-top: 16px;
  font-size: 14px;
}

.action-buttons {
  margin-top: 16px;
}

.chat-container {
  height: calc(100vh - 280px);
  min-height: 500px;
  display: flex;
  flex-direction: column;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px 0;
  margin-bottom: 16px;
}

.empty-chat {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.message-item {
  margin-bottom: 24px;
}

.message {
  display: flex;
  margin-bottom: 12px;
}

.user-message {
  justify-content: flex-end;
}

.bot-message {
  justify-content: flex-start;
}

.message-avatar {
  margin: 0 8px;
}

.message-content {
  max-width: 70%;
}

.message-bubble {
  padding: 12px 16px;
  border-radius: 18px;
  word-wrap: break-word;
  line-height: 1.5;
}

.user-bubble {
  background: #1890ff;
  color: white;
  margin-left: auto;
}

.bot-bubble {
  background: #f0f0f0;
  color: #333;
}

.bot-bubble.typing {
  background: #e6f7ff;
  color: #1890ff;
}

.message-time {
  font-size: 12px;
  color: #999;
  text-align: center;
  margin-top: 4px;
}

.chat-input {
  border-top: 1px solid #f0f0f0;
  padding-top: 16px;
}

.mode-switch {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

.chat-disabled-hint {
  margin-top: 8px;
}

.typing-indicator {
  margin-bottom: 16px;
}

.streaming-response {
  margin-bottom: 16px;
}

.typing-cursor {
  animation: blink 1s infinite;
  color: #1890ff;
  font-weight: bold;
}

@keyframes blink {
  0%, 50% {
    opacity: 1;
  }
  51%, 100% {
    opacity: 0;
  }
}

/* 滚动条样式 */
.chat-messages::-webkit-scrollbar {
  width: 6px;
}

.chat-messages::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.chat-messages::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 响应式设计 */
@media (max-width: 992px) {
  .video-info-card {
    margin-bottom: 24px;
    height: auto;
  }
  
  .chat-card {
    height: 70vh;
  }
  
  .chat-container {
    height: calc(70vh - 80px);
  }
  
  .message-content {
    max-width: 85%;
  }
}
</style>