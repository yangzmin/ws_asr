<template>
  <div class="home">
    <a-row :gutter="24">
      <a-col :span="24">
        <a-card title="🎥 视频管理中心" class="main-card">
          <template #extra>
            <a-space>
              <a-button type="primary" @click="$router.push('/upload')">
                <template #icon>
                  <upload-outlined />
                </template>
                上传视频
              </a-button>
            </a-space>
          </template>
          
          <a-spin :spinning="loading">
            <div v-if="videos.length === 0" class="empty-state">
              <a-empty description="暂无视频">
                <a-button type="primary" @click="$router.push('/upload')">
                  立即上传
                </a-button>
              </a-empty>
            </div>
            
            <a-row :gutter="[16, 16]" v-else>
              <a-col :xs="24" :sm="12" :md="8" :lg="6" v-for="video in videos" :key="video.id">
                <a-card hoverable class="video-card">
                  <template #cover>
                    <div class="video-cover">
                      <video-camera-outlined style="font-size: 48px; color: #1890ff" />
                    </div>
                  </template>
                  
                  <a-card-meta :title="video.name">
                    <template #description>
                      <div class="video-info">
                        <p><strong>状态:</strong> 
                          <a-tag :color="getStatusColor(video.status)">
                            {{ getStatusText(video.status) }}
                          </a-tag>
                        </p>
                        <p><strong>上传时间:</strong> {{ formatDate(video.uploadAt) }}</p>
                      </div>
                    </template>
                  </a-card-meta>
                  
                  <template #actions>
                    <a-button 
                      type="primary" 
                      size="small" 
                      @click="startChat(video.id)"
                    >
                      <template #icon>
                        <message-outlined />
                      </template>
                      开始聊天
                    </a-button>
                  </template>
                </a-card>
              </a-col>
            </a-row>
          </a-spin>
        </a-card>
      </a-col>
      
      <!-- 语音识别中心卡片 -->
      <a-col :span="24">
        <a-card title="🎤 语音识别中心" class="main-card">
          <template #extra>
            <a-space>
              <a-button type="primary" @click="$router.push('/speech-recognition')">
                <template #icon>
                  <audio-outlined />
                </template>
                开始语音识别
              </a-button>
            </a-space>
          </template>
          
          <div class="feature-description">
            <p>🎙️ 实时语音识别功能</p>
            <p>📝 支持中文语音转文字</p>
            <p>🔊 高质量音频处理</p>
          </div>
        </a-card>
      </a-col>
      
      <!-- ASR-TTS 演示中心卡片 -->
      <a-col :span="24">
        <a-card title="🗣️ ASR-TTS 语音对话演示" class="main-card">
          <template #extra>
            <a-space>
              <a-button type="primary" @click="$router.push('/asr-tts-demo')">
                <template #icon>
                  <sound-outlined />
                </template>
                开始语音对话
              </a-button>
            </a-space>
          </template>
          
          <div class="feature-description">
            <p>🎙️ 实时语音识别 (ASR)</p>
            <p>🤖 智能对话处理 (LLM)</p>
            <p>🔊 语音合成播放 (TTS)</p>
            <p>💬 完整语音对话流程演示</p>
          </div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useVideoStore } from '../stores/video'
import { 
  UploadOutlined, 
  VideoCameraOutlined, 
  MessageOutlined,
  AudioOutlined,
  SoundOutlined
} from '@ant-design/icons-vue'

export default {
  name: 'Home',
  components: {
    UploadOutlined,
    VideoCameraOutlined,
    MessageOutlined,
    AudioOutlined,
    SoundOutlined
  },
  setup() {
    const router = useRouter()
    const videoStore = useVideoStore()
    const loading = ref(false)

    
    const fetchVideos = async () => {
      loading.value = true
      await videoStore.fetchVideos()
      console.log("videoStore.videos after fetch:", videoStore.videos)
      loading.value = false
    }
    
    const startChat = (videoId) => {
      router.push(`/chat/${videoId}`)
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
    
    onMounted(() => {
      console.log("onMounted - videoStore.videos before fetch:", videoStore.videos)
      fetchVideos()
    })
    const videos = computed(() => videoStore.videos)
    
    return {
      loading,
      videos,
      startChat,
      getStatusColor,
      getStatusText,
      formatDate
    }
  }
}
</script>

<style scoped>
.home {
  max-width: 1200px;
  margin: 0 auto;
}

.main-card {
  margin-bottom: 24px;
}

.video-card {
  height: 100%;
}

.video-cover {
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
}

.video-info p {
  margin: 4px 0;
  font-size: 12px;
}

.empty-state {
  padding: 40px 0;
}

.feature-description {
  color: #666;
  line-height: 1.6;
}

.feature-description p {
  margin: 8px 0;
  font-size: 14px;
}
</style>