import { defineStore } from 'pinia'
import { videoAPI, chatAPI } from '../services/api'

export const useVideoStore = defineStore('video', {
  state: () => ({
    videos: [],
    currentVideo: null,
    chatHistory: [],
    loading: false,
    uploading: false
  }),
  
  getters: {
    getVideoById: (state) => (id) => {
      return state.videos.find(video => video.id === id)
    }
  },
  
  actions: {
    // 上传视频
    async uploadVideo(file) {
      this.uploading = true
      try {
        const response = await videoAPI.uploadVideo(file)
        if (response.code === '0000') {
          this.videos.push(response.data)
          return response.data
        } else {
          throw new Error(response.message)
        }
      } catch (error) {
        console.error('上传视频失败:', error)
        throw error
      } finally {
        this.uploading = false
      }
    },
    
    // 获取所有视频
    async fetchVideos() {
      this.loading = true
      try {
        const response = await videoAPI.getAllVideos()
        if (response.code === '0000') {
          this.videos = response.data || []
        }
      } catch (error) {
        console.error('获取视频列表失败:', error)
      } finally {
        this.loading = false
      }
    },
    
    // 获取视频信息
    async fetchVideo(videoId) {
      try {
        const response = await videoAPI.getVideo(videoId)
        if (response.code === '0000') {
          this.currentVideo = response.data
          return response.data
        }
      } catch (error) {
        console.error('获取视频信息失败:', error)
        throw error
      }
    },
    
    // 发送聊天消息（非流式）
    async sendChatMessage(videoId, message) {
      try {
        const response = await chatAPI.sendMessage(videoId, message)
        if (response.code === '0000') {
          // 更新聊天历史
          await this.fetchChatHistory(videoId)
          return response.data
        } else {
          throw new Error(response.message)
        }
      } catch (error) {
        console.error('发送消息失败:', error)
        throw error
      }
    },
    
    // 发送流式聊天消息（支持翻译）
    async sendStreamChatMessage(videoId, message, onProgress) {
      return new Promise((resolve, reject) => {
        let translationStatus = ''
        let responseMessage = ''
        
        const onMessage = (data) => {
          if (data.type === 'translation') {
            translationStatus = data.message
            onProgress({ type: 'translation', message: data.message })
          } else if (data.type === 'response') {
            responseMessage += data.message
            onProgress({ type: 'response', message: data.message, fullMessage: responseMessage })
          }
        }
        
        const onError = (error) => {
          console.error('流式聊天失败:', error)
          reject(error)
        }
        
        const onComplete = async () => {
          try {
            // 刷新聊天历史
            await this.fetchChatHistory(videoId)
            resolve({ message: responseMessage })
          } catch (error) {
            console.error('刷新聊天历史失败:', error)
            resolve({ message: responseMessage })
          }
        }
        
        // 使用POST方式发送流式消息
        chatAPI.sendStreamMessagePost(videoId, message, onMessage, onError, onComplete)
      })
    },
    
    // 获取聊天历史
    async fetchChatHistory(videoId) {
      try {
        const response = await chatAPI.getChatHistory(videoId)
        if (response.code === '0000') {
          this.chatHistory = response.data.messages || []
        }
      } catch (error) {
        console.error('获取聊天历史失败:', error)
      }
    },
    
    // 清除聊天历史
    async clearChatHistory(videoId) {
      try {
        const response = await chatAPI.clearChatHistory(videoId)
        if (response.code === '0000') {
          this.chatHistory = []
        }
      } catch (error) {
        console.error('清除聊天历史失败:', error)
        throw error
      }
    }
  }
})