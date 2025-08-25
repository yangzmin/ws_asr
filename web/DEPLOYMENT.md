# 前端项目部署指南

## 问题描述

在本地开发环境中，前端项目可以通过 Vite 的代理功能正常访问后端 API，但构建后部署到 nginx 服务器时无法请求到后端接口。

## 问题原因

1. **开发环境 vs 生产环境**：
   - 开发环境：Vite 代理将 `/api` 请求转发到 `http://localhost:8080`
   - 生产环境：静态文件部署到 nginx，没有代理配置

2. **API 基础 URL 配置**：
   - 原代码中 `baseURL: '/api'` 是硬编码的相对路径
   - 生产环境需要完整的后端服务器地址

## 解决方案

### 1. 环境变量配置

已创建以下配置文件：

- `.env.development`：开发环境配置，使用相对路径 `/api`
- `.env.production`：生产环境配置，推荐使用相对路径通过nginx代理

### 2. 修改 API 配置

在 `src/services/api.js` 中，baseURL 现在支持环境变量：

```javascript
baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
withCredentials: false, // 跨域请求是否发送cookies
```

### 3. Vite 代理配置优化

在 `vite.config.js` 中增强了代理配置：

```javascript
proxy: {
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true,
    secure: false,
    rewrite: (path) => path.replace(/^\/api/, '/api')
  }
}
```

### 3. 部署步骤

#### 步骤 1：配置生产环境变量

编辑 `.env.production` 文件，设置正确的后端服务器地址：

```env
VITE_API_BASE_URL=http://your-backend-server.com/api
```

#### 步骤 2：构建生产版本

```bash
npm run build:prod
```

#### 步骤 3：配置 Nginx

参考 `nginx.conf.example` 文件配置 nginx：

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    # 前端静态文件
    location / {
        root /path/to/your/dist;
        index index.html;
        try_files $uri $uri/ /index.html;
    }
    
    # 代理后端API请求（可选）
    location /api/ {
        proxy_pass http://localhost:8080/;
        # ... 其他代理配置
    }
}
```

### 4. 两种部署方案

#### 方案 A：直接请求后端服务器

- 在 `.env.production` 中设置完整的后端地址
- 前端直接请求后端服务器
- 需要后端服务器配置 CORS

#### 方案 B：通过 Nginx 代理

- 在 `.env.production` 中保持 `VITE_API_BASE_URL=/api`
- 配置 nginx 代理 `/api` 请求到后端服务器
- 推荐方案，更安全且易于管理

### 5. 验证部署

1. 检查构建后的文件是否包含正确的环境变量
2. 在浏览器开发者工具中查看网络请求
3. 确认 API 请求的完整 URL 是否正确

## 常见问题

### Q: 为什么本地开发正常，部署后就不行？
A: 本地开发使用 Vite 代理，生产环境是静态文件，需要不同的配置。

### Q: 如何检查环境变量是否生效？
A: 在浏览器控制台输入 `console.log(import.meta.env)` 查看所有环境变量。

### Q: CORS 错误怎么解决？
A: 在后端服务器配置 CORS，或使用 nginx 代理避免跨域问题。