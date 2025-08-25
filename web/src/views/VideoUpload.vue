<template>
  <div class="video-upload">
    <a-row justify="center">
      <a-col :xs="24" :sm="20" :md="16" :lg="12">
        <a-card title="📤 上传视频" class="upload-card">
          <template #extra>
            <a-button @click="$router.push('/')">
              <template #icon>
                <arrow-left-outlined />
              </template>
              返回首页
            </a-button>
          </template>
          
          <div class="upload-content">
            <a-upload-dragger
              v-model:fileList="fileList"
              name="file"
              :multiple="false"
              :before-upload="beforeUpload"
              :custom-request="handleUpload"
              accept="video/*"
              :show-upload-list="false"
            >
              <p class="ant-upload-drag-icon">
                <inbox-outlined style="font-size: 48px; color: #1890ff" />
              </p>
              <p class="ant-upload-text">点击或拖拽视频文件到此区域上传</p>
              <p class="ant-upload-hint">
                支持 MP4, AVI, MOV 等常见视频格式
              </p>
            </a-upload-dragger>
            
            <div v-if="selectedFile" class="file-info">
              <a-alert
                :message="`已选择文件: ${selectedFile.name}`"
                :description="`文件大小: ${formatFileSize(selectedFile.size)}`"
                type="info"
                show-icon
                style="margin: 16px 0"
              />
              
              <a-button 
                type="primary" 
                size="large" 
                block 
                :loading="uploading"
                @click="confirmUpload"
              >
                <template #icon>
                  <upload-outlined />
                </template>
                {{ uploading ? '上传中...' : '确认上传' }}
              </a-button>
            </div>
            
            <div v-if="uploadProgress > 0" class="upload-progress">
              <a-progress 
                :percent="uploadProgress" 
                :status="uploadStatus"
                :stroke-color="{
                  '0%': '#108ee9',
                  '100%': '#87d068',
                }"
              />
            </div>
            
            <div v-if="uploadResult" class="upload-result">
              <a-result
                :status="uploadResult.success ? 'success' : 'error'"
                :title="uploadResult.title"
                :sub-title="uploadResult.message"
              >
                <template #extra v-if="uploadResult.success">
                  <a-space>
                    <a-button type="primary" @click="goToChat">
                      开始聊天
                    </a-button>
                    <a-button @click="resetUpload">
                      继续上传
                    </a-button>
                    <a-button @click="$router.push('/')">
                      返回首页
                    </a-button>
                  </a-space>
                </template>
                <template #extra v-else>
                  <a-button type="primary" @click="resetUpload">
                    重新上传
                  </a-button>
                </template>
              </a-result>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { useVideoStore } from '../stores/video'
import { 
  InboxOutlined, 
  UploadOutlined, 
  ArrowLeftOutlined 
} from '@ant-design/icons-vue'

export default {
  name: 'VideoUpload',
  components: {
    InboxOutlined,
    UploadOutlined,
    ArrowLeftOutlined
  },
  setup() {
    const router = useRouter()
    const videoStore = useVideoStore()
    
    const fileList = ref([])
    const selectedFile = ref(null)
    const uploading = ref(false)
    const uploadProgress = ref(0)
    const uploadStatus = ref('normal')
    const uploadResult = ref(null)
    const uploadedVideo = ref(null)
    
    const beforeUpload = (file) => {
      const isVideo = file.type.startsWith('video/')
      if (!isVideo) {
        message.error('只能上传视频文件!')
        return false
      }
      
      const isLt100M = file.size / 1024 / 1024 < 100
      if (!isLt100M) {
        message.error('视频文件大小不能超过 100MB!')
        return false
      }
      
      selectedFile.value = file
      return false // 阻止自动上传
    }
    
    const handleUpload = () => {
      // 这里不做任何操作，因为我们使用自定义上传
    }
    
    const confirmUpload = async () => {
      if (!selectedFile.value) {
        message.error('请先选择视频文件')
        return
      }
      
      uploading.value = true
      uploadProgress.value = 0
      uploadStatus.value = 'active'
      uploadResult.value = null
      
      try {
        // 模拟上传进度
        const progressInterval = setInterval(() => {
          if (uploadProgress.value < 90) {
            uploadProgress.value += Math.ceil(Math.random() * 30)
          }
        }, 500)
        
        const video = await videoStore.uploadVideo(selectedFile.value)
        
        clearInterval(progressInterval)
        uploadProgress.value = 100
        uploadStatus.value = 'success'
        uploadedVideo.value = video
        
        uploadResult.value = {
          success: true,
          title: '上传成功!',
          message: `视频 "${video.name}" 已成功上传，正在处理中...`
        }
        
        message.success('视频上传成功!')
        
      } catch (error) {
        uploadProgress.value = 100
        uploadStatus.value = 'exception'
        
        uploadResult.value = {
          success: false,
          title: '上传失败',
          message: error.message || '上传过程中发生错误，请重试'
        }
        
        message.error('上传失败: ' + (error.message || '未知错误'))
      } finally {
        uploading.value = false
      }
    }
    
    const resetUpload = () => {
      fileList.value = []
      selectedFile.value = null
      uploadProgress.value = 0
      uploadStatus.value = 'normal'
      uploadResult.value = null
      uploadedVideo.value = null
    }
    
    const goToChat = () => {
      if (uploadedVideo.value) {
        router.push(`/chat/${uploadedVideo.value.id}`)
      }
    }
    
    const formatFileSize = (bytes) => {
      if (bytes === 0) return '0 Bytes'
      const k = 1024
      const sizes = ['Bytes', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    }
    
    return {
      fileList,
      selectedFile,
      uploading,
      uploadProgress,
      uploadStatus,
      uploadResult,
      beforeUpload,
      handleUpload,
      confirmUpload,
      resetUpload,
      goToChat,
      formatFileSize
    }
  }
}
</script>

<style scoped>
.video-upload {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
}

.upload-card {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.upload-content {
  padding: 24px 0;
}

.file-info {
  margin-top: 24px;
}

.upload-progress {
  margin: 24px 0;
}

.upload-result {
  margin-top: 24px;
}

.ant-upload-drag {
  border: 2px dashed #d9d9d9;
  border-radius: 8px;
  background: #fafafa;
  transition: all 0.3s;
}

.ant-upload-drag:hover {
  border-color: #1890ff;
  background: #f0f8ff;
}
</style>