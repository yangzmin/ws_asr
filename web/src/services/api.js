import axios from 'axios'

// 创建axios实例
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 1800000,
  withCredentials: false, // 根据需要设置是否发送cookies
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
api.interceptors.request.use(
  config => {
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  response => {
    return response.data
  },
  error => {
    console.error('API请求错误:', error)
    return Promise.reject(error)
  }
)

// 视频相关API
export const videoAPI = {
  // 上传视频
  uploadVideo(file) {
    const formData = new FormData()
    formData.append('file', file)
    return api.post('/video/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
  },
  
  // 获取视频信息
  getVideo(videoId) {
    return api.get(`/video/${videoId}`)
  },
  
  // 获取所有视频
  getAllVideos() {
    return api.get('/video/')
  }
}

// 聊天相关API
export const chatAPI = {
  // 发送聊天消息（非流式）
  sendMessage(videoId, message, history = []) {
    return api.post('/chat/', {
      videoNos: [videoId],
      message,
      history,
      stream: false
    })
  },
  
  // 发送流式聊天消息（支持翻译）
  sendStreamMessage(videoId, message, onMessage, onError, onComplete) {
    const eventSource = new EventSource(
      `/api/chat/stream?videoId=${encodeURIComponent(videoId)}&message=${encodeURIComponent(message)}`
    )
    
    eventSource.onmessage = function(event) {
      try {
        const data = JSON.parse(event.data)
        onMessage(data)
        
        if (data.type === 'done') {
          eventSource.close()
          onComplete()
        } else if (data.type === 'error') {
          eventSource.close()
          onError(new Error(data.message))
        }
      } catch (error) {
        console.error('解析SSE数据失败:', error)
        onError(error)
      }
    }
    
    eventSource.onerror = function(error) {
      console.error('SSE连接错误:', error)
      eventSource.close()
      onError(new Error('连接中断'))
    }
    
    return eventSource
  },
  
  // 使用POST方式发送流式聊天消息
  async sendStreamMessagePost(videoId, message, onMessage, onError, onComplete) {
    try {
      const response = await fetch('/api/chat/', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          videoNos: [videoId],
          message,
          stream: true
        })
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        
        const chunk = decoder.decode(value)
        const lines = chunk.split('\n')
        
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6))
              onMessage(data)
              
              if (data.type === 'done') {
                onComplete()
                return
              } else if (data.type === 'error') {
                onError(new Error(data.message))
                return
              }
            } catch (error) {
              console.error('解析SSE数据失败:', error)
            }
          }
        }
      }
    } catch (error) {
      onError(error)
    }
  },
  
  // 获取聊天历史
  getChatHistory(videoId) {
    return api.get(`/chat/history/${videoId}`)
  },
  
  // 清除聊天历史
  clearChatHistory(videoId) {
    return api.delete(`/chat/history/${videoId}`)
  }
}

export default api