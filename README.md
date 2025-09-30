# Image Sync - 容器镜像同步工具

基于 Skopeo 的容器镜像同步工具，提供简洁的 Web UI 界面，支持在不同镜像仓库之间快速同步镜像。

## 功能特性

- 🚀 简单易用的 Web UI 界面
- 🔐 支持私有仓库认证（用户名/密码）
- 📦 基于 Skopeo 底层实现，稳定可靠
- 🎯 支持跨仓库镜像拷贝
- 🏗️ 支持多架构镜像同步（可查询并选择特定架构）
- 💾 用户配置保存与管理（支持多配置快速切换）
- 🔒 OIDC 统一认证支持（可选，用户配置隔离）
- ⚡ 前后端分离架构，易于部署

## 技术栈

**后端:**
- Go 1.x
- Gin Web Framework
- Skopeo（需要单独安装）

**前端:**
- React 18
- Ant Design
- JavaScript

## 前置要求

### 安装 Skopeo

**macOS:**
```bash
brew install skopeo
```

**Ubuntu/Debian:**
```bash
sudo apt-get install skopeo
```

**RHEL/CentOS:**
```bash
sudo yum install skopeo
```

## 快速开始

### 方式一：LPK 部署（推荐）

适用于 Lazycat Cloud 平台部署。

#### 1. 构建后端镜像

```bash
make build-backend
# 或手动构建
cd backend
docker build -t docker-registry-ui.liu.heiyu.space/image-sync/backend:latest .
```

#### 2. 推送后端镜像

```bash
make push
# 或手动推送
docker push docker-registry-ui.liu.heiyu.space/image-sync/backend:latest
```

#### 3. 构建 LPK 包

```bash
make build-lpk
# 或手动构建
sh build.sh
lzc build
```

#### 4. 部署 LPK

将生成的 `.lpk` 文件上传到 Lazycat Cloud 平台进行部署。

**环境变量说明：**

后端环境变量：
- `SYNC_DEFAULT_SOURCE_REGISTRY`: 源镜像地址默认值
- `SYNC_DEFAULT_DEST_REGISTRY`: 目标镜像地址默认值

LPK 部署说明：
- 前端通过 `application.routes` 配置自动代理到后端
- 修改后端地址或端口时，只需更新 `lzc-manifest.yml` 中的 `routes` 配置
- 详见 [部署指南](docs/DEPLOYMENT.md)

### 方式二：本地开发运行

#### 1. 启动后端服务

```bash
# 使用 Makefile（推荐）
make dev-backend

# 或手动启动
cd backend
go mod download
export DEFAULT_SOURCE_REGISTRY="registry.lazycat.cloud/"
export DEFAULT_DEST_REGISTRY="registry.mybox.heiyu.space/"
go run cmd/server/main.go
```

后端服务默认运行在 `http://localhost:8080`

可以通过 `-port` 参数指定端口：

```bash
go run cmd/server/main.go -port 9090
```

#### 2. 启动前端服务

```bash
# 使用 Makefile（推荐）
make dev-frontend

# 或手动启动
cd frontend
npm install
npm start
```

前端服务默认运行在 `http://localhost:3000`

**默认镜像地址：**

后端支持通过环境变量设置默认镜像地址。打开 Web UI 后，输入框会自动填充这些默认值：
- `DEFAULT_SOURCE_REGISTRY`: 源镜像地址默认值
- `DEFAULT_DEST_REGISTRY`: 目标镜像地址默认值

如果未设置环境变量，输入框将为空。

### 使用说明

1. 打开浏览器访问 `http://localhost:3000`
2. 填写源镜像地址（例如：`docker.io/library/nginx:latest`）
3. 如果是私有仓库，填写对应的用户名和密码
4. 选择镜像架构（默认为"全部架构"）
   - 可点击"查询可用架构"按钮查看镜像支持的所有架构
   - 查询结果会以蓝色标签形式显示在按钮下方
5. 填写目标镜像地址（例如：`registry.example.com/nginx:latest`）
6. 如果目标是私有仓库，填写对应的用户名和密码
7. 点击"开始同步"按钮

**配置保存（高级功能）：**

- 通过高级选项面板可以保存常用配置，方便下次使用
- 支持多配置管理，每个配置最大 4KB
- 每个用户最多保存 1000 个配置
- 启用 OIDC 认证后，配置自动隔离，每个用户独立管理
- 密码保存功能默认禁用（`--allow-password-save=false`），需要时可通过参数启用

## 项目结构

```
image-sync/
├── backend/                   # Go 后端
│   ├── cmd/
│   │   └── server/
│   │       └── main.go        # 应用入口
│   ├── internal/              # 内部包
│   │   ├── config/            # 配置管理
│   │   ├── models/            # 数据模型
│   │   ├── repository/        # 数据访问层
│   │   ├── service/           # 业务逻辑层
│   │   ├── handler/           # HTTP 处理层
│   │   ├── middleware/        # 中间件
│   │   └── pkg/               # 工具包
│   ├── go.mod
│   ├── Dockerfile
│   └── .gitignore
├── frontend/                  # React 前端
│   ├── src/
│   │   ├── App.js             # 主组件
│   │   └── App.css            # 样式
│   ├── package.json
│   └── .gitignore
├── dist/                      # 构建输出目录（由 build.sh 生成）
│   └── web/                   # 前端静态文件
├── build.sh                   # 前端构建脚本
├── lzc-build.yml              # LPK 构建配置
├── lzc-manifest.yml           # LPK 应用清单
├── image-sync-icon.png        # 应用图标
├── Makefile                   # 构建命令
├── CLAUDE.md                  # Claude 参考文档
├── docs/                      # 文档目录
└── README.md                  # 项目说明
```

## API 接口

### 创建同步任务

**POST** `/api/v1/sync`

请求体：
```json
{
  "sourceImage": "docker.io/library/nginx:latest",
  "destImage": "registry.example.com/nginx:latest",
  "architecture": "linux/amd64",
  "sourceUsername": "user1",
  "sourcePassword": "pass1",
  "destUsername": "user2",
  "destPassword": "pass2"
}
```

响应：
```json
{
  "message": "Sync started",
  "id": "sync-123"
}
```

### 查询镜像架构

**POST** `/api/v1/inspect`

请求体：
```json
{
  "image": "docker.io/library/nginx:latest",
  "username": "user1",
  "password": "pass1"
}
```

响应：
```json
{
  "architectures": [
    "linux/amd64",
    "linux/arm64",
    "linux/arm/v7"
  ]
}
```

### 查询同步状态

**GET** `/api/v1/sync/:id`

响应：
```json
{
  "id": "sync-123",
  "status": "pending"
}
```

### 获取默认配置

**GET** `/api/v1/env/defaults`

响应：
```json
{
  "sourceRegistry": "registry.lazycat.cloud/",
  "destRegistry": "registry.mybox.heiyu.space/"
}
```

## Makefile 命令

```bash
make help           # 显示所有可用命令
make build          # 构建后端镜像和前端 dist
make build-backend  # 构建后端 Docker 镜像
make build-dist     # 构建前端到 dist 目录
make build-lpk      # 构建 LPK 包（需要 lzc-cli）
make build-local    # 本地编译后端二进制
make push           # 推送后端镜像到仓库
make dev-backend    # 启动后端开发服务
make dev-frontend   # 启动前端开发服务
make test           # 运行测试
make clean          # 清理构建输出
```

## 调试功能

### 前端调试工具

应用内置了完整的调试功能，方便在移动端和生产环境排障：

1. **浮动调试按钮**
   - 位于页面右下角的虫子图标
   - 显示当前调试日志数量徽章
   - 点击打开调试日志面板

2. **调试日志记录**
   - 自动记录所有关键操作（初始化、API 请求、错误等）
   - VSCode 风格深色主题显示
   - 支持清空日志和自动滚动

3. **版本信息显示**
   - 页面底部显示应用版本、Git commit、分支名和构建时间
   - 鼠标悬停查看完整 commit 哈希
   - 方便快速确认部署版本

### 构建版本信息

`build.sh` 脚本会自动注入以下构建信息：
- Git commit 短哈希（如 `502089b`）
- Git 分支名
- 构建时间（UTC）

这些信息会编译到前端代码中，并显示在页面底部。

## 开发计划

- [x] 基础 Web UI 界面
- [x] 镜像架构查询功能
- [x] 环境变量默认配置
- [x] Skopeo 调用逻辑（支持多架构、认证、TLS）
- [x] 任务状态跟踪
- [x] 实时日志查看功能（SSE）
- [x] 后端分层架构重构
- [x] LPK 打包支持
- [x] 前端调试工具（浮动按钮 + 日志面板）
- [x] Git 版本信息显示（commit、分支、构建时间）
- [ ] 支持任务历史记录持久化
- [ ] 任务列表查询接口
- [ ] 添加进度显示

## 文档

- [部署指南](docs/DEPLOYMENT.md) - 详细的环境变量配置和部署说明
- [Claude 开发文档](CLAUDE.md) - 项目技术架构和开发指南

## 许可证

MIT

## 贡献

欢迎提交 Issue 和 Pull Request！