<template>
  <div id="app">
    <a-layout style="min-height: 100vh">
      <a-layout-header style="background: #001529; padding: 0">
        <div style="color: white; text-align: center; font-size: 20px; line-height: 64px">
          🎥 视频聊天
        </div>
      </a-layout-header>
      
      <a-layout-content style="padding: 24px">
        <router-view />
      </a-layout-content>
      
      
    </a-layout>
  </div>
</template>

<script>
export default {
  name: 'App'
}
</script>

<style>
#app {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

.ant-layout-header {
  position: fixed;
  z-index: 1;
  width: 100%;
}

.ant-layout-content {
  margin-top: 64px;
  background: #fff;
}
</style>