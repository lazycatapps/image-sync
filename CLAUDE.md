# Image Sync 项目说明

## 项目概述

Image Sync 是一个基于 Skopeo 的容器镜像同步工具，提供简洁的 Web UI 界面，用于在不同容器镜像仓库之间拷贝镜像。

## 技术架构

### 后端
- **语言**: Go 1.25+
- **Web 框架**: Gin v1.11.0+
- **核心依赖**:
  - `github.com/google/uuid` - 任务 ID 生成
  - `github.com/gin-gonic/gin` - HTTP 服务框架
  - `github.com/coreos/go-oidc/v3` - OIDC 认证库
  - `golang.org/x/oauth2` - OAuth2 客户端库
  - `github.com/spf13/cobra` - 命令行参数解析
  - `github.com/spf13/viper` - 配置管理
- **核心工具**: Skopeo (通过命令行调用)
- **配置管理**: 使用标准库 `os` 包读取环境变量
- **日志系统**: 自定义 Logger 接口，基于标准库 `log` 包
  - 支持 INFO/ERROR/DEBUG 三个日志级别
  - 统一的日志格式：`[LEVEL] timestamp message`
  - 输出到 stdout (INFO/DEBUG) 和 stderr (ERROR)
- **中间件**:
  - **CORS 中间件**
    - 默认允许所有来源 (`Access-Control-Allow-Origin: *`)
    - 可以通过环境变量配置为特定域名
    - 支持的方法: GET, POST, PUT, DELETE, OPTIONS
    - 支持的头: Content-Type, Authorization
  - **OIDC 认证中间件**
    - 支持基于 OpenID Connect (OIDC) 的统一认证
    - 自动验证会话 cookie
    - 支持公共端点白名单（如健康检查、认证回调）
    - API 请求认证失败返回 401，浏览器请求自动跳转登录页
- **会话管理**:
  - 内存会话存储（SessionService）
  - 会话 TTL: 7 天
  - 自动清理过期会话（每 10 分钟）
  - 支持会话刷新和注销

### 前端
- **框架**: React 19.1+
- **UI 库**: Ant Design 5.27+
- **构建工具**: Create React App (react-scripts 5.0+)
- **语言**: JavaScript (非 TypeScript)
- **状态管理**: React Hooks (useState, useEffect, useRef)
  - 不使用额外的状态管理库 (Redux/MobX)
  - 组件级状态管理，适合小型应用
- **测试框架**:
  - `@testing-library/react` - React 组件测试
  - `@testing-library/jest-dom` - Jest DOM 断言
  - `@testing-library/user-event` - 用户交互模拟
- **HTTP 通信**:
  - 使用浏览器原生 `fetch` API
  - 支持 EventSource (SSE) 接收实时日志流
- **开发环境**:
  - 后端 API 地址配置：通过环境变量 `BACKEND_API_URL` 注入到 `window._env_.BACKEND_API_URL`
    - 开发环境默认：`http://localhost:8080`
    - 生产环境示例：`http://host.lzcapp`、`https://api.example.com`
  - 无需额外的代理配置 (依赖后端 CORS 支持)
- **浏览器兼容性**:
  - 生产环境: >0.2% 浏览器市场份额，排除已停止维护的浏览器
  - 开发环境: 最新版本的 Chrome/Firefox/Safari
  - 支持移动端浏览器
- **调试功能**:
  - 全局错误捕获（错误和未处理的 Promise rejection）
  - 浮动调试按钮（右下角虫子图标）
  - 调试日志面板（记录所有关键操作和错误）
  - 支持清空日志和自动滚动
  - VSCode 风格深色主题显示
- **版本信息**:
  - 显示应用版本号
  - 显示 Git commit 短哈希（生产环境）
  - 显示 Git 分支名（非 main 分支时）
  - 显示构建时间（ISO 8601 格式）
  - 鼠标悬停显示完整 commit 哈希

## 核心功能

### 1. 镜像同步
- **基本功能**
  - 支持从源镜像仓库拷贝到目标镜像仓库
  - 支持公开仓库和私有仓库
  - 支持用户名/密码认证
  - 支持 TLS 证书验证开关
- **架构支持**
  - 支持同步所有架构（使用 skopeo `--all` 参数）
  - 支持选择特定单一架构同步（使用 `--override-os/--override-arch/--override-variant` 参数）
  - 架构格式：`os/arch` 或 `os/arch/variant`（如 `linux/amd64`、`linux/arm/v7`）
- **异步执行**
  - 任务创建后立即返回任务 ID
  - 后台异步执行同步操作
  - 支持并发多任务同步
- **日志管理**
  - 实时捕获 Skopeo 的 stdout 和 stderr 输出
  - 自动脱敏：日志中的凭据信息自动替换为 `***:***`
  - 日志缓冲区：1000 条消息

### 2. 架构查询
- **查询机制**
  - 使用 `skopeo inspect --raw` 获取镜像清单（manifest）
  - 支持 Multi-arch manifest list（多架构镜像）
  - 支持 Single-arch manifest（单架构镜像）
  - 自动解析 OS、架构和 variant 信息
- **查询流程**
  - 用户填写源镜像地址后，点击"查询可用架构"按钮
  - 默认不弹框，直接在页面上显示查询结果
  - 查询成功：以蓝色 Tag 标签形式显示所有架构
  - 查询失败：通过消息提示告知用户，不阻塞后续操作
- **结果展示**
  - 查询成功后动态更新架构下拉选项
  - 默认保持"全部架构"选项
  - 可以通过详情按钮查看完整的查询日志（Modal 弹窗）

### 3. 实时日志系统
- **技术实现**
  - 使用 SSE (Server-Sent Events) 推送实时日志
  - 前端使用 EventSource API 接收日志流
  - 后端使用 channel 机制广播日志到所有监听客户端
- **多客户端支持**
  - 支持多个客户端同时订阅同一任务的日志
  - 每个监听器独立的 channel，缓冲区大小 1000 条
  - 任务完成后自动关闭所有 SSE 连接
- **日志显示**
  - Modal 弹窗显示，黑底绿字终端风格
  - 自动滚动到最新日志
  - Alert 组件实时显示任务状态

### 4. 任务状态管理
- **状态流转**
  - `pending`: 任务已创建，等待执行
  - `running`: 正在执行同步操作
  - `completed`: 同步成功完成
  - `failed`: 同步失败
- **状态查询**
  - 前端通过轮询（1 秒间隔）获取任务状态
  - 任务完成（completed/failed）后停止轮询
  - 支持通过 API 查询任务详情（状态、日志、错误信息等）
- **任务信息**
  - 记录开始时间和结束时间
  - 保存完整的输出日志
  - 失败时记录错误信息

### 5. 环境变量配置
- **默认值配置**
  - 支持通过环境变量配置默认的源镜像仓库地址
  - 支持通过环境变量配置默认的目标镜像仓库地址
  - 支持通过环境变量配置其他页面参数
  - 页面加载时自动填充默认值
- **用户配置保存**
  - 通过高级选项折叠面板提供"保存配置"功能
  - 保存内容：镜像地址前缀、用户名、密码（base64 编码，可选）、TLS 验证选项
  - **密码存储安全控制**：
    - 通过 `--allow-password-save` 参数或 `SYNC_ALLOW_PASSWORD_SAVE` 环境变量控制是否允许保存密码
    - **默认值：`false`（禁止保存密码，极致安全）**
    - 当设置为 `false` 时：
      - 用户提交的密码字段会被忽略，不会写入配置文件
      - 即使旧配置文件中存在密码，返回给前端时也会被清空
      - 确保配置文件中永远不包含敏感凭据信息
    - 当设置为 `true` 时：
      - 前端使用 base64 编码密码后传输到后端
      - 后端将 base64 编码的密码存储到配置文件
      - 加载配置时前端自动解码密码
      - 所有日志输出中密码信息都会被脱敏显示为 `***`
      - 配置文件通过 OIDC 认证保护，仅限用户本人访问
  - **用户配置隔离**：
    - 启用 OIDC 认证时，每个用户的配置存储在独立目录：`/configs/users/{userID}/`
    - 未启用 OIDC 时，所有配置存储在共享目录：`/configs/`
    - 用户只能访问和管理自己的配置，完全隔离互不影响
  - **配置限制**：
    - **单个配置大小限制**：默认 4KB（4096 字节），可通过 `--max-config-size` 参数或 `SYNC_MAX_CONFIG_SIZE` 环境变量配置
    - **配置数量限制**：默认每个用户最多 1000 个配置，可通过 `--max-config-files` 参数或 `SYNC_MAX_CONFIG_FILES` 环境变量配置
    - 防止存储空间滥用和 DoS 攻击
  - 支持多配置管理：可保存多份配置并快速切换
  - 配置存储路径：基础路径可通过 `--config-dir` 参数或 `SYNC_CONFIG_DIR` 环境变量指定，默认 `./configs/`（当前工作目录下的 configs 子目录）
  - 配置文件命名格式：`config_NAME.json`
  - 记录最后使用的配置：`last_used.txt`

### 6. Web UI 设计
- **页面布局**
  - 简洁直观，功能模块清晰分离
  - 源镜像信息区域
    - 镜像地址输入框（必填）
    - 用户名/密码输入框（可选，私有仓库需要）
    - TLS 证书验证复选框（默认不验证）
  - 架构选择区域
    - 架构下拉选择框（默认"全部架构"）
    - "查询可用架构"按钮
    - 查询结果 Tag 标签显示（查询成功后显示）
    - "查看详情"按钮（查看查询日志）
  - 目标镜像信息区域
    - 镜像地址输入框（必填）
    - 用户名/密码输入框（可选，私有仓库需要）
    - TLS 证书验证复选框（默认不验证）
  - "开始同步"按钮（大按钮，醒目）
  - 高级选项折叠面板（待实现）
    - 保存当前配置
    - 清除已保存配置
    - 提示：不保存密码信息
- **隐私保护**
  - 密码输入框使用 `type="password"` 属性
  - 不通过 URL 参数传递任何敏感信息
  - 凭据仅通过 POST 请求体发送到后端
  - 后端不存储凭据，仅在执行 Skopeo 时使用
  - 日志中自动脱敏凭据信息
- **页面底部**
  - 版权信息：Image Sync · Version {version} · Copyright © {year} Lazycat Apps
  - GitHub 项目链接：https://github.com/lazycatapps/image-sync

### 7. OIDC 认证（可选）
- **认证机制**
  - 支持基于 OpenID Connect (OIDC) 的统一认证
  - 与 Lazycat Cloud (heiyu.space) 微服认证系统集成
  - 自动获取用户 ID、邮箱、权限组信息
  - 支持管理员权限组识别（`ADMIN` 组）
- **认证流程**
  - 用户访问受保护资源时自动跳转到 OIDC 登录页
  - OIDC Provider 完成认证后回调到 `/api/v1/auth/callback`
  - 验证 ID Token 并提取用户信息（sub, email, groups）
  - 创建会话并设置 HttpOnly Cookie（7 天有效期）
  - 后续请求自动携带会话 Cookie 进行认证
- **配置方式**
  - 在 `lzc-manifest.yml` 中配置 `oidc_redirect_path: /api/v1/auth/callback`
  - 在 `services.backend.environment` 中引用 Lazycat 系统注入的环境变量：
    - `SYNC_OIDC_CLIENT_ID=${LAZYCAT_AUTH_OIDC_CLIENT_ID}`: 客户端 ID
    - `SYNC_OIDC_CLIENT_SECRET=${LAZYCAT_AUTH_OIDC_CLIENT_SECRET}`: 客户端密钥
    - `SYNC_OIDC_ISSUER=${LAZYCAT_AUTH_OIDC_ISSUER}`: Issuer URL
    - `SYNC_OIDC_REDIRECT_URL=https://${LAZYCAT_APP_DOMAIN}/api/v1/auth/callback`: 回调 URL
  - 当这些环境变量都配置后，OIDC 认证自动启用
  - **排查方法**：启动日志会输出 OIDC 配置状态，如果环境变量未注入，会显示 `(empty)`
- **认证端点**
  - `GET /api/v1/auth/login`: 跳转到 OIDC 登录页
  - `GET /api/v1/auth/callback`: OIDC 认证回调处理
  - `POST /api/v1/auth/logout`: 注销当前用户会话
  - `GET /api/v1/auth/userinfo`: 获取当前用户信息
- **会话管理**
  - 会话存储：内存存储（SessionService）
  - 会话有效期：7 天
  - 自动清理：每 10 分钟清理过期会话
  - Cookie 安全：HttpOnly, Secure (生产环境)
- **访问控制**
  - 公共端点：健康检查、认证相关端点
  - 受保护端点：所有镜像同步相关 API
  - 未认证访问：API 返回 401，浏览器跳转登录
- **关闭认证**
  - 如果不配置 OIDC 环境变量，认证功能自动禁用
  - 禁用后所有 API 端点均可直接访问

## 技术细节

### 1. 日志管理

- **日志库**: 自定义 Logger 接口 (`internal/pkg/logger`)
  - 基于 Go 标准库 `log` 包实现
  - 不依赖第三方日志库,保持轻量级
- **日志级别**:
  - `INFO`: 一般信息日志 (服务启动、任务状态变更、命令执行等)
  - `ERROR`: 错误日志 (任务失败、命令执行失败等)
  - `DEBUG`: 调试日志 (详细的执行过程)
- **日志格式**: `[LEVEL] timestamp message`
  - 示例: `[INFO] 2025-10-01 10:30:45 Server starting on :8080`
- **日志输出**:
  - INFO/DEBUG 输出到 stdout
  - ERROR 输出到 stderr
  - 便于容器环境日志收集和分流
- **日志轮转**: 当前不支持文件日志和日志轮转
  - 依赖容器编排平台的日志管理 (如 Docker/K8s 日志驱动)
  - 或外部日志收集系统 (如 ELK、Loki)

### 2. 错误处理机制

- **统一错误类型**: `internal/pkg/errors/AppError`
  - 包含错误码 (Code)、错误消息 (Message)、HTTP 状态码 (StatusCode)
  - 支持错误包装 (Wrap) 保留原始错误信息
  - 实现 `error` 接口和 `Unwrap()` 方法,兼容 Go 1.13+ 错误链
- **预定义错误**:
  - `ErrTaskNotFound`: 任务不存在 (404)
  - `ErrInvalidInput`: 无效的输入参数 (400)
  - `ErrInternal`: 内部服务器错误 (500)
  - `ErrCommandFailed`: 命令执行失败 (500)
- **错误处理流程**:
  - Service 层抛出标准 Go error
  - Handler 层捕获并转换为统一的 JSON 错误响应
  - 返回格式: `{"error": "error message"}`
- **使用状态**: 已实现
  - errors 包已完成并可用
  - Handler 层使用标准错误响应格式

### 2.1. 输入验证与安全

- **输入验证库**: `internal/pkg/validator`
  - 防止命令注入、参数注入等安全漏洞
  - 所有用户输入在 Handler 层进行验证
- **验证规则**:
  - **镜像名称** (`ValidateImageName`):
    - 格式: `[registry/][namespace/]repository[:tag|@digest]`
    - 正则表达式验证标准镜像地址格式
    - 拒绝 shell 元字符: `;`, `&`, `|`, `` ` ``, `$`, `(`, `)`, `<`, `>`, `\n`, `\r`, `\`
    - 最大长度: 512 字符
    - 示例: `docker.io/library/nginx:latest`, `registry.example.com:5000/myapp@sha256:abc123...`
  - **架构字符串** (`ValidateArchitecture`):
    - 格式: `os/arch` 或 `os/arch/variant`，或 `all` 关键字
    - 正则表达式: `^[a-z0-9]+/[a-z0-9]+(/[a-z0-9]+)?$`
    - 最大长度: 64 字符
    - 示例: `linux/amd64`, `linux/arm/v7`
  - **用户名** (`ValidateUsername`):
    - 允许字符: 字母、数字、`-`, `_`, `.`, `@`
    - 拒绝空格和特殊字符
    - 最大长度: 256 字符
  - **密码** (`ValidatePassword`):
    - 仅拒绝控制字符: `\n`, `\r`, `\x00`
    - 允许其他特殊字符（因为使用 `exec.Command()` 参数传递，不经过 shell）
    - 最大长度: 512 字符
  - **凭据组合** (`ValidateCredentials`):
    - 用户名和密码必须同时提供或同时为空
    - 分别验证用户名和密码格式
- **验证时机**:
  - `SyncHandler.SyncImage()`: 验证源镜像、目标镜像、架构、源凭据、目标凭据
  - `ImageHandler.InspectImage()`: 验证镜像名称和凭据
- **错误响应**:
  - 验证失败返回 400 Bad Request
  - 错误消息格式: `{"error": "validation error for field 'xxx': xxx"}`
- **安全原则**:
  - **纵深防御**: 虽然使用 `exec.Command()` 已避免 shell 注入，仍进行输入验证
  - **白名单优于黑名单**: 使用正则表达式定义允许的格式
  - **最小权限**: 只允许必要的字符集
  - **DoS 防护**: 限制输入长度，防止内存耗尽攻击

### 3. 命令执行与超时控制

- **命令执行方式**:
  - 使用 Go 标准库 `os/exec` 包执行 skopeo 命令
  - 不使用 shell 包装,直接执行二进制文件,避免 shell 注入风险
  - 使用 `exec.Command(name, arg...)` 而非 `exec.Command("sh", "-c", cmd)`
  - 所有用户输入作为参数传递,不拼接到命令字符串中
- **超时控制**:
  - 默认超时时间: 10 分钟
  - 可通过环境变量 `SYNC_TIMEOUT` 指定 (单位:秒)
  - 使用 `context.WithTimeout()` 实现
  - 超时后自动终止 skopeo 进程,任务标记为失败
- **进程管理**:
  - 任务在 goroutine 中异步执行
  - 进程输出通过 pipe 实时读取
  - 任务完成、失败或超时后自动清理资源

### 4. 并发与并行

- **任务并发**:
  - 技术上支持无限并发任务
  - 每个任务独立的 goroutine 执行
  - 当前无并发限制和队列机制
- **资源管理**:
  - 每个任务占用一个 skopeo 进程
  - 高并发时可能消耗大量系统资源 (CPU/内存/网络带宽)
  - 建议: 生产环境可通过环境变量配置最大并发数

### 5. 数据存储

- **任务存储**: 内存存储 (`internal/repository/task_repository.go`)
  - 使用 `sync.RWMutex` 保证并发安全
  - 进程重启后任务历史丢失
  - 扩展性: 接口化设计,可替换为持久化存储 (数据库/文件)
- **日志缓冲**:
  - 每个任务维护独立的日志 slice
  - 每个 SSE 监听器独立的 channel,缓冲 1000 条消息
  - 任务完成后保留完整日志在内存中

### 6. 配置管理

- **命令行参数**:
  - `--host`: 指定服务监听地址，默认值 `0.0.0.0`
  - `--port`: 指定服务监听端口，默认值 8080
  - `--timeout`: 覆盖环境变量指定的超时时间
  - `--default-source-registry`: 覆盖环境变量指定的默认源镜像仓库
  - `--default-dest-registry`: 覆盖环境变量指定的默认目标
  - `--config-dir`: 配置文件存储目录，默认 `/configs`
  - `--allow-password-save`: 是否允许在配置文件中保存密码，默认 `false`（极致安全）
  - `--max-config-size`: 单个配置文件最大大小（字节），默认 `4096`
  - `--max-config-files`: 每个用户最大配置文件数量，默认 `1000`
  - 使用 `github.com/spf13/cobra` 实现命令行参数解析
  - 使用 `github.com/spf13/viper` 读取环境变量 (可选)
- **支持的环境变量**:
  - `SYNC_` 开头的变量，如果原命令行参数带 - 则替换为 _
      - 比如 `SYNC_DEFAULT_SOURCE_REGISTRY` 对应 `--default-source-registry`
      - 比如 `SYNC_ALLOW_PASSWORD_SAVE` 对应 `--allow-password-save`
      - 比如 `SYNC_MAX_CONFIG_SIZE` 对应 `--max-config-size`
      - 比如 `SYNC_MAX_CONFIG_FILES` 对应 `--max-config-files`

### 7. 安全性

- **输入验证** (详见 2.1 节):
  - 使用 `internal/pkg/validator` 包验证所有用户输入
  - 在 Handler 层进行验证,拒绝无效输入
  - 镜像名称、架构、用户名、密码均有严格验证规则
  - 防护措施: 命令注入、参数注入、DoS 攻击
- **凭据处理**:
  - 凭据仅通过 HTTPS POST 请求体传输 (生产环境应使用 HTTPS)
  - 后端默认不存储凭据，仅在执行 skopeo 时使用
  - 可通过 `--allow-password-save=true` 允许保存密码到配置文件（需谨慎使用）
  - 保存时使用 base64 编码，配合 OIDC 认证实现用户级隔离
  - 日志中自动脱敏: `***:***` 替换凭据信息
- **命令注入防护**:
  - 使用 `exec.Command(name, arg...)` 而非 shell 执行
  - 参数通过切片传递,避免 shell 解析
  - 所有用户输入均经过验证和清洗
  - 拒绝包含 shell 元字符的镜像名称
- **CORS 配置**:
  - 默认允许所有来源 `*` (开发友好)
  - 生产环境建议配置特定域名白名单

### 8. 前端技术细节

- **状态轮询**:
  - 任务启动后每 1 秒轮询一次状态 API
  - 任务完成 (completed/failed) 后停止轮询
  - 使用 `useEffect` + `setTimeout` 实现
- **实时日志**:
  - 使用浏览器原生 `EventSource` API 接收 SSE
  - 自动滚动到最新日志 (使用 `useRef` + `scrollIntoView`)
  - 任务完成后自动关闭 EventSource 连接
- **表单状态管理**:
  - 使用 React Hooks (`useState`) 管理表单状态
  - 无第三方状态管理库 (Redux/MobX)
  - 所有状态保持在单一组件 `App.js` 中

## 目录结构

```
image-sync/
├── backend/          # Go 后端服务
│   ├── cmd/
│   │   └── server/
│   │       └── main.go           # 应用入口，依赖注入
│   ├── internal/                 # 内部包（不对外暴露）
│   │   ├── config/               # 配置管理
│   │   │   └── config.go
│   │   ├── models/               # 数据模型定义
│   │   │   └── task.go
│   │   ├── repository/           # 数据访问层
│   │   │   └── task_repository.go
│   │   ├── service/              # 业务逻辑层
│   │   │   ├── sync_service.go   # 同步服务
│   │   │   └── image_service.go  # 镜像检查服务
│   │   ├── handler/              # HTTP 处理层
│   │   │   ├── sync_handler.go
│   │   │   └── image_handler.go
│   │   ├── middleware/           # 中间件
│   │   │   └── cors.go
│   │   └── pkg/                  # 内部工具包
│   │       ├── logger/           # 日志管理
│   │       │   └── logger.go
│   │       ├── errors/           # 错误处理
│   │       └── validator/        # 输入验证
│   │           ├── validator.go
│   │           └── validator_test.go
│   ├── go.mod                    # Go 模块依赖
│   ├── go.sum
│   ├── Dockerfile
│   └── main.go.old               # 旧代码备份
├── frontend/         # React 前端应用
│   ├── src/
│   │   ├── App.js    # 主应用组件
│   │   └── App.css   # 样式文件
│   ├── package.json
│   └── .gitignore
├── docs/             # 项目文档
├── Makefile          # 构建和开发命令
├── README.md         # 项目说明文档
└── CLAUDE.md         # 本文件，供 Claude 参考
```

## 架构设计

### 分层架构说明

后端采用标准的分层架构，职责清晰，易于维护和扩展：

1. **cmd/server** - 应用入口层
   - 负责程序启动和依赖注入
   - 初始化各层组件并组装依赖关系

2. **handler** - HTTP 处理层
   - 处理 HTTP 请求和响应
   - 参数验证和绑定
   - 调用 service 层执行业务逻辑

3. **service** - 业务逻辑层
   - 核心业务逻辑实现
   - 调用外部命令（skopeo）
   - 调用 repository 层进行数据操作

4. **repository** - 数据访问层
   - 封装数据存储操作
   - 提供统一的数据访问接口
   - 当前使用内存存储，可扩展为数据库

5. **models** - 数据模型层
   - 定义数据结构
   - 业务实体定义

6. **middleware** - 中间件层
   - CORS 处理
   - 可扩展：认证、日志、限流等

7. **pkg** - 工具包层
   - logger: 结构化日志
   - errors: 统一错误处理
   - validator: 输入验证和安全检查

8. **config** - 配置管理层
   - 环境变量读取
   - 配置项管理

### 设计优势

- ✅ **职责分离**: 每层职责明确，易于理解和维护
- ✅ **依赖注入**: 通过接口实现松耦合，易于测试
- ✅ **可扩展性**: 可轻松替换实现（如内存存储换成数据库）
- ✅ **可测试性**: 各层独立，便于单元测试
- ✅ **代码复用**: 公共逻辑提取到 service 和 pkg 层

## API 设计

### POST /api/v1/sync
创建镜像同步任务

**请求参数:**
```json
{
  "sourceImage": "docker.io/library/nginx:latest",
  "destImage": "registry.example.com/nginx:latest",
  "architecture": "linux/amd64",
  "sourceUsername": "user1",
  "sourcePassword": "pass1",
  "srcTLSVerify": false,
  "destUsername": "user2",
  "destPassword": "pass2",
  "destTLSVerify": false
}
```

**字段说明:**
- `sourceImage` (必填): 源镜像地址
- `destImage` (必填): 目标镜像地址
- `architecture` (可选): 指定架构,如 `linux/amd64`、`linux/arm64`、`linux/arm/v7`
  - 不传或传空字符串表示同步所有架构 (等同于 `all`)
  - 传 `all` 表示同步所有架构
- `sourceUsername` (可选): 源仓库用户名
- `sourcePassword` (可选): 源仓库密码
- `srcTLSVerify` (可选): 源仓库是否验证 TLS 证书,默认 `false`
- `destUsername` (可选): 目标仓库用户名
- `destPassword` (可选): 目标仓库密码
- `destTLSVerify` (可选): 目标仓库是否验证 TLS 证书,默认 `false`

**成功响应 (200):**
```json
{
  "message": "Sync started",
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**错误响应:**
- **400 Bad Request** - 请求参数错误
  ```json
  {
    "error": "sourceImage is required"
  }
  ```
- **500 Internal Server Error** - 服务器内部错误
  ```json
  {
    "error": "Failed to create sync task"
  }
  ```

### GET /api/v1/sync/:id
查询同步任务状态

**路径参数:**
- `id`: 任务 ID (UUID 格式)

**成功响应 (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "sourceImage": "docker.io/library/nginx:latest",
  "destImage": "registry.example.com/nginx:latest",
  "architecture": "linux/amd64",
  "status": "completed",
  "message": "Sync completed successfully",
  "startTime": "2025-10-01T10:30:00Z",
  "endTime": "2025-10-01T10:35:23Z",
  "output": "Task started...\nSync completed...",
  "errorOutput": ""
}
```

**字段说明:**
- `status`: 任务状态,可选值: `pending`、`running`、`completed`、`failed`
- `message`: 状态描述信息
- `output`: 完整的日志输出
- `errorOutput`: 错误信息 (仅 `status=failed` 时有值)

**错误响应:**
- **404 Not Found** - 任务不存在
  ```json
  {
    "error": "Task not found"
  }
  ```
- **500 Internal Server Error** - 服务器内部错误
  ```json
  {
    "error": "Failed to get task"
  }
  ```

### POST /api/v1/inspect
查询镜像支持的架构列表

**请求参数:**
```json
{
  "image": "docker.io/library/nginx:latest",
  "username": "user1",
  "password": "pass1",
  "tlsVerify": false
}
```

**字段说明:**
- `image` (必填): 镜像地址
- `username` (可选): 仓库用户名
- `password` (可选): 仓库密码
- `tlsVerify` (可选): 是否验证 TLS 证书,默认 `false`

**成功响应 (200):**
```json
{
  "architectures": [
    "linux/amd64",
    "linux/arm64",
    "linux/arm/v7"
  ]
}
```

**错误响应:**
- **400 Bad Request** - 请求参数错误
  ```json
  {
    "error": "image is required"
  }
  ```
- **500 Internal Server Error** - 镜像查询失败
  ```json
  {
    "error": "failed to inspect image: unauthorized"
  }
  ```

### GET /api/v1/env/defaults
获取环境变量配置的默认镜像地址

**成功响应 (200):**
```json
{
  "sourceRegistry": "registry.lazycat.cloud/",
  "destRegistry": "registry.example.heiyu.space/"
}
```

### GET /api/v1/sync/:id/logs
获取同步任务的实时日志流 (SSE)

**路径参数:**
- `id`: 任务 ID (UUID 格式)

**响应头:**
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**响应格式 (Server-Sent Events):**
```
data: Task started at 2025-09-30T23:14:22+08:00
data: Using source credentials
data: Copying all architectures
data: Executing: skopeo copy --src-tls-verify=false --dest-tls-verify=false --all docker://... docker://...
data: Getting image source signatures
data: Copying blob sha256:...
data: Sync completed at 2025-09-30T23:14:45+08:00
```

**错误响应:**
- **404 Not Found** - 任务不存在
  ```json
  {
    "error": "Task not found"
  }
  ```

### GET /api/v1/config/:name
获取已保存的用户配置

**路径参数:**
- `name`: 配置名称

**成功响应 (200):**
```json
{
  "sourceRegistry": "registry.example.com/",
  "destRegistry": "registry.mybox.example.com/",
  "sourceUsername": "user1",
  "destUsername": "user2",
  "sourcePassword": "cGFzczE=",
  "destPassword": "cGFzczI=",
  "srcTLSVerify": false,
  "destTLSVerify": true
}
```

**说明:**
- 返回保存的配置，如无配置则返回空对象 `{}`
- `sourcePassword` 和 `destPassword` 为 base64 编码后的密码
- 前端需要使用 `atob()` 解码密码

### POST /api/v1/config/:name
保存用户配置

**路径参数:**
- `name`: 配置名称（只能包含字母、数字、点、横线和下划线）

**请求参数:**
```json
{
  "sourceRegistry": "registry.example.com/",
  "destRegistry": "registry.mybox.example.com/",
  "sourceUsername": "user1",
  "destUsername": "user2",
  "sourcePassword": "cGFzczE=",
  "destPassword": "cGFzczI=",
  "srcTLSVerify": false,
  "destTLSVerify": true
}
```

**字段说明:**
- 所有字段均为可选
- `sourcePassword` 和 `destPassword` 应为 base64 编码后的密码
- 前端需要使用 `btoa()` 编码密码
- 配置会覆盖之前保存的同名配置

**成功响应 (200):**
```json
{
  "message": "Configuration saved successfully"
}
```

**错误响应:**
- **400 Bad Request** - 请求参数错误
  ```json
  {
    "error": "Configuration size (5000 bytes) exceeds maximum allowed size (4096 bytes)"
  }
  ```
  ```json
  {
    "error": "Maximum number of configs (1000) reached"
  }
  ```
- **500 Internal Server Error** - 服务器内部错误

### DELETE /api/v1/config/:name
删除已保存的用户配置

**路径参数:**
- `name`: 配置名称

**成功响应 (200):**
```json
{
  "message": "Configuration deleted successfully"
}
```

**说明:**
- 删除后，配置文件会被移除
- 如果配置文件不存在，仍返回成功

### GET /api/v1/configs
获取所有配置名称列表

**成功响应 (200):**
```json
{
  "configs": ["default", "prod-env", "test-config"]
}
```

**说明:**
- 返回所有已保存的配置名称列表（按字母顺序排序）
- 如果没有配置，返回空数组

### GET /api/v1/config/last-used
获取最后使用的配置名称

**成功响应 (200):**
```json
{
  "name": "default"
}
```

**说明:**
- 返回最后使用的配置名称
- 如果没有记录，返回空字符串

### GET /api/v1/sync
查询任务列表

**查询参数:**
- `page` (可选): 页码,从 1 开始,默认 1
- `pageSize` (可选): 每页数量,默认 20,最大 100
- `status` (可选): 过滤任务状态,可选值: `pending`、`running`、`completed`、`failed`
- `sortBy` (可选): 排序字段,可选值: `startTime`、`endTime`,默认 `startTime`
- `sortOrder` (可选): 排序方向,可选值: `asc`、`desc`,默认 `desc`

**成功响应 (200):**
```json
{
  "total": 100,
  "page": 1,
  "pageSize": 20,
  "tasks": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "sourceImage": "docker.io/library/nginx:latest",
      "destImage": "registry.example.com/nginx:latest",
      "architecture": "linux/amd64",
      "status": "completed",
      "message": "Sync completed successfully",
      "startTime": "2025-10-01T10:30:00Z",
      "endTime": "2025-10-01T10:35:23Z"
    }
  ]
}
```

### GET /api/v1/health
健康检查接口

**成功响应 (200):**
```json
{
  "status": "ok"
}
```

### GET /api/v1/auth/login
跳转到 OIDC Provider 进行认证登录

**说明:**
- 生成随机 state 用于 CSRF 防护
- 将 state 保存到 cookie（10 分钟有效期）
- 重定向到 OIDC Provider 的授权页面
- 仅在启用 OIDC 认证时可用

**响应:**
- **302 Found** - 重定向到 OIDC Provider 授权页面
- **503 Service Unavailable** - OIDC 认证未启用

### GET /api/v1/auth/callback
OIDC 认证回调处理

**查询参数:**
- `code` (必填): 授权码
- `state` (必填): CSRF 防护令牌

**处理流程:**
1. 验证 state 与 cookie 中的 state 是否匹配
2. 使用授权码交换访问令牌和 ID Token
3. 验证 ID Token 签名
4. 提取用户信息（sub, email, groups）
5. 创建会话并设置 session cookie
6. 重定向到首页

**响应:**
- **302 Found** - 认证成功，重定向到首页
- **400 Bad Request** - State 不匹配或缺少参数
- **500 Internal Server Error** - Token 验证失败或内部错误

### POST /api/v1/auth/logout
注销当前用户会话

**说明:**
- 删除服务器端会话
- 清除客户端 session cookie

**成功响应 (200):**
```json
{
  "message": "Logged out successfully"
}
```

### GET /api/v1/auth/userinfo
获取当前登录用户信息

**成功响应 (200) - 已认证:**
```json
{
  "authenticated": true,
  "user_id": "user-uuid",
  "email": "user@example.com",
  "groups": ["ADMIN", "USER"],
  "is_admin": true
}
```

**成功响应 (200) - 未认证:**
```json
{
  "authenticated": false
}
```

**说明:**
- 如果 OIDC 未启用，总是返回 `authenticated: false`
- `is_admin` 字段表示用户是否在 `ADMIN` 权限组中

## 开发状态

### 已完成
- ✅ 项目目录结构
- ✅ Go 后端基础框架（Gin）
- ✅ React 前端基础 UI（Ant Design）
- ✅ API 接口定义
- ✅ 前端表单组件
- ✅ 镜像架构查询功能（前端 UI + 后端 API）
  - 改进错误处理（捕获 stderr 输出）
  - 安全的类型断言
  - 支持 variant 字段（如 linux/arm/v7）
- ✅ 架构选择下拉框
- ✅ 查询结果 Tag 标签显示
- ✅ 环境变量默认配置
- ✅ TLS 证书验证选项（前端 UI + 后端 API）
- ✅ 后端 Skopeo 命令调用逻辑
  - 支持 `--all` 参数复制所有架构
  - 支持 `--override-arch/--override-os/--override-variant` 指定特定架构
  - 支持 `--src-tls-verify/--dest-tls-verify` TLS 验证控制
  - 支持 `--src-creds/--dest-creds` 凭据认证
- ✅ 任务状态管理和跟踪
  - 使用 UUID 生成唯一任务 ID
  - 支持 pending/running/completed/failed 状态
  - 异步任务执行
  - 任务状态查询 API
- ✅ 实时日志输出（SSE - Server-Sent Events）
  - 后端流式日志输出接口 `/api/v1/sync/:id/logs`
  - 捕获 stdout 和 stderr 实时输出
  - 支持多客户端同时订阅日志
  - 自动关闭连接在任务完成后
- ✅ 前端任务日志展示
  - Modal 弹窗显示实时日志
  - 黑底绿字终端风格显示
  - 任务状态 Alert 提示
  - 自动滚动到最新日志
- ✅ **后端代码重构（2025-10-01）**
  - 采用标准分层架构（handler/service/repository/model）
  - 依赖注入模式，便于测试和扩展
  - 配置管理模块化
  - 结构化日志系统
  - 接口化设计，易于替换实现
- ✅ **统一错误处理机制**（errors 包）
  - AppError 结构实现标准 error 接口
  - 预定义常用错误类型
  - Handler 层统一使用 errors 包处理错误
- ✅ **单元测试覆盖**
  - service 层测试（sync_service_test.go, image_service_test.go）
  - errors 包测试（errors_test.go）
  - repository 层测试（task_repository_test.go, task_test.go）
  - validator 包测试（validator_test.go）
- ✅ **任务列表查询接口**（GET /api/v1/sync）
  - 支持分页（page, pageSize）
  - 支持按状态过滤（status）
  - 支持排序（sortBy, sortOrder）
- ✅ **路由管理独立**（router 层）
  - router 包独立管理路由注册
  - 清晰的路由分组和组织
- ✅ **Dockerfile 更新**
  - 支持新的项目结构（cmd/server）
  - 多阶段构建优化镜像大小
- ✅ **输入验证和安全加固**（2025-10-01）
  - 新增 `internal/pkg/validator` 包
  - Handler 层对所有用户输入进行验证
  - 镜像名称正则验证，拒绝 shell 元字符
  - 架构、用户名、密码格式验证
  - 防护命令注入、参数注入、DoS 攻击
  - 完整的单元测试覆盖（65+ 测试用例）
- ✅ **用户配置保存功能**（2025-10-01）
  - 文件存储用户配置（JSON 格式）
  - 支持保存镜像地址前缀、用户名、TLS 验证选项
  - 不保存密码等敏感信息
  - 配置目录可通过 `--config-dir` 参数或环境变量指定（默认 `/configs`）
  - API 接口：GET/POST/DELETE `/api/v1/config`
  - 前端高级选项面板支持保存/清除配置
  - 页面加载时自动加载保存的配置
- ✅ **OIDC 认证功能**（2025-10-01）
  - 集成 OpenID Connect 统一认证
  - 支持 Lazycat Cloud (heiyu.space) 微服认证系统
  - 会话管理（内存存储，7 天有效期）
  - 自动跳转登录和认证回调处理
  - 用户信息和权限组管理
  - 认证中间件保护 API 端点
  - 可选功能，未配置时自动禁用

### 待实现
- [ ] 任务历史记录持久化（数据库或文件）
- [ ] 错误重试机制
- [ ] 进度条显示（需要 skopeo 支持进度输出）

## 开发指南

### 启动开发环境

1. 后端:
```bash
cd backend
# 可以通过环境变量指定默认镜像地址
export DEFAULT_SOURCE_REGISTRY="registry.lazycat.cloud/"
export DEFAULT_DEST_REGISTRY="registry.mybox.heiyu.space/"
go run cmd/server/main.go         # 默认端口 8080
go run cmd/server/main.go -port 9090  # 指定端口

# 或使用 Makefile
make dev-backend
make build-local  # 编译二进制文件
```

2. 前端:
```bash
cd frontend
npm start
```

### 环境变量使用示例

打开 Web UI 后，源镜像地址和目标镜像地址输入框会自动填充为环境变量指定的默认值。
如果未设置环境变量，则输入框为空。

### 注意事项
- 确保系统已安装 Skopeo
- 后端需要有执行 Skopeo 命令的权限
- CORS 已在后端配置，允许跨域请求
- 前端默认请求后端地址: http://localhost:8080
- 架构查询按钮和下拉框使用 flexbox 布局保持在同一行
- 查询结果显示在按钮下方，使用 Ant Design 的 Tag 组件

### 构建和部署

#### 构建信息注入
`build.sh` 脚本会自动注入 Git 版本信息：
- `REACT_APP_GIT_COMMIT`: Git commit 短哈希（7位）
- `REACT_APP_GIT_COMMIT_FULL`: Git commit 完整哈希
- `REACT_APP_GIT_BRANCH`: Git 分支名
- `REACT_APP_BUILD_TIME`: 构建时间（UTC，ISO 8601 格式）

这些信息会显示在页面底部，方便排障和版本追踪。

#### 构建命令
```bash
# 完整构建（清理 + 重新构建）
./build.sh

# 快速构建（如果 dist 存在则跳过）
./build.sh fast

# 使用 Makefile
make build-dist      # 构建前端到 dist 目录
make build           # 构建后端镜像和前端
make deploy          # 完整部署流程
make deploy-fast     # 快速部署（不重新构建前端/后端）
```

## 后续优化方向

1. 添加任务队列，支持批量同步
2. 支持进度条显示
3. 添加 WebSocket 实现实时状态更新
4. 支持镜像列表批量导入（YAML/JSON）
5. 添加用户认证和权限管理
6. 支持 Docker Compose 一键部署
