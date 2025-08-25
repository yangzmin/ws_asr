import { createRouter, createWebHistory } from 'vue-router'
import Home from '../views/Home.vue'
import VideoUpload from '../views/VideoUpload.vue'
import VideoChat from '../views/VideoChat.vue'
import SpeechRecognition from '../views/SpeechRecognition.vue'
import AsrTtsDemo from '../views/AsrTtsDemo.vue'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: Home
  },
  {
    path: '/upload',
    name: 'VideoUpload',
    component: VideoUpload
  },
  {
    path: '/chat/:videoId',
    name: 'VideoChat',
    component: VideoChat,
    props: true
  },
  {
    path: '/speech-recognition',
    name: 'SpeechRecognition',
    component: SpeechRecognition
  },
  {
    path: '/asr-tts-demo',
    name: 'AsrTtsDemo',
    component: AsrTtsDemo
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router