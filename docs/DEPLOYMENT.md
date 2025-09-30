# 部署指南

## 环境变量配置

### 后端环境变量

后端服务支持以下环境变量（通过 `SYNC_` 前缀）：

- `SYNC_HOST`: 服务监听地址（默认：`0.0.0.0`）
- `SYNC_PORT`: 服务监听端口（默认：`8080`）
- `SYNC_TIMEOUT`: 同步超时时间，单位秒（默认：`600`）
- `SYNC_DEFAULT_SOURCE_REGISTRY`: 默认源镜像仓库地址
- `SYNC_DEFAULT_DEST_REGISTRY`: 默认目标镜像仓库地址
- `SYNC_CORS_ALLOWED_ORIGINS`: CORS 允许的来源（默认：`*`）

### 前端环境变量

前端静态文件支持以下环境变量：

- `BACKEND_API_URL`: 后端 API 地址
  - 开发环境：配置为 `http://localhost:8080`
  - LPK 部署：使用空字符串（相对路径），由 `application.routes` 自动代理到后端
  - Docker 部署：配置完整的后端地址，如 `https://api.example.com`

## LPK 部署

### 配置说明

在 `lzc-manifest.yml` 中配置环境变量：

```yaml
application:
  routes:
    - /=file:///lzcapp/pkg/content/web
    - /api/=http://backend.cloud.lazycat.app.liu.imagesync.lzcapp:59901

services:
  backend:
    environment:
      - SYNC_HOST=host.lzcapp
      - SYNC_PORT=59901
      - SYNC_DEFAULT_SOURCE_REGISTRY=registry.lazycat.cloud/
      - SYNC_DEFAULT_DEST_REGISTRY=docker-registry-ui.${LAZYCAT_BOX_NAME}.heiyu.space/
```

### 工作原理

LPK 部署中前端使用相对路径访问后端 API：

1. 前端 JavaScript 发起请求：`/api/v1/sync`
2. `application.routes` 自动将 `/api/` 代理到后端服务
3. 实际请求：`http://backend.cloud.lazycat.app.liu.imagesync.lzcapp:59901/api/v1/sync`

**优势**：用户修改后端环境变量（如端口）后，只需更新 routes 配置，前端代码无需改动。

## Docker Compose 部署

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  backend:
    image: your-registry/image-sync/backend:latest
    environment:
      - SYNC_DEFAULT_SOURCE_REGISTRY=registry.lazycat.cloud/
      - SYNC_DEFAULT_DEST_REGISTRY=registry.example.com/
      - SYNC_CORS_ALLOWED_ORIGINS=http://localhost,https://imagesync.example.com
    ports:
      - "8080:8080"

  frontend:
    image: your-registry/image-sync/frontend:latest
    environment:
      - BACKEND_API_URL=http://localhost:8080
    ports:
      - "80:80"
    depends_on:
      - backend
```

## 开发环境

### 后端

```bash
export SYNC_DEFAULT_SOURCE_REGISTRY="registry.lazycat.cloud/"
export SYNC_DEFAULT_DEST_REGISTRY="registry.mybox.heiyu.space/"
go run cmd/server/main.go
```

### 前端

前端开发环境通过 `public/env-config.js` 配置：

```javascript
// public/env-config.js
window._env_ = {
  BACKEND_API_URL: "http://localhost:8080"
};
```

修改这个文件后刷新页面即可生效。

## 生产环境最佳实践

1. **使用 HTTPS**：生产环境建议使用 HTTPS 协议保护敏感数据传输
2. **配置 CORS**：限制 CORS 允许的来源，避免使用 `*`
   ```
   SYNC_CORS_ALLOWED_ORIGINS=https://imagesync.example.com
   ```
3. **设置超时时间**：根据镜像大小和网络情况调整超时时间
   ```
   SYNC_TIMEOUT=1200  # 20分钟
   ```
4. **配置默认镜像地址**：为用户提供便利的默认值
   ```
   SYNC_DEFAULT_SOURCE_REGISTRY=registry.lazycat.cloud/
   SYNC_DEFAULT_DEST_REGISTRY=registry.example.com/
   ```
