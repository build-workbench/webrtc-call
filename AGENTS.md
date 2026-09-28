# AGENTS.md — WebRTC Call

基于浏览器原生 WebRTC API（1v1 及小规模 Mesh 多人）的音视频通话客户端（单文件原生 JS），配套 Go + gorilla/websocket 实现的轻量级 WebSocket 信令服务。

## 常用命令

```bash
make run     # 本地启动服务（即 go run ./cmd/server），访问 http://localhost:8080
make build   # 编译二进制（go build ./...）
make test    # 单元测试，带 -race 竞态检测（go test -race -count=1 ./...）
make vet     # 静态检查（go vet ./...）
make fmt     # 格式化 Go 代码（go fmt ./...）
make clean   # 清理 coverage.out / coverage.html
```

- CI（.github/workflows/ci.yml）在 push main / PR 时执行 `go build ./...`、`go test -race -count=1 ./...`、`go vet ./...`，提交前建议本地跑齐。
- Docker 部署：`docker compose -f deploy/docker/docker-compose.yml up -d --build`。

## 代码结构

- `cmd/server/main.go` — 服务入口：HTTP 路由、静态文件服务、优雅停机
- `internal/signal/hub.go` — 内存 Hub：房间生命周期、全局会话调度与广播（含并发测试 hub_test.go）
- `internal/signal/client.go` — 客户端连接抽象、写通道泵、令牌桶限流
- `internal/signal/handler.go` — HTTP 升级 WebSocket 处理器（Origin 白名单、安全响应头）
- `internal/signal/join.go` — 进房逻辑与用户 ID / 房间名规范化校验
- `internal/signal/forward.go` — 点对点信令消息路由转发（`To` 字段定向 / 房间广播）
- `internal/signal/message.go` — 信令消息结构体与类型常量
- `internal/signal/errors.go` — 协议级错误码定义
- `web/index.html` — 整个前端：音视频 UI、WebRTC 逻辑、DataChannel 聊天、屏幕共享，单文件零依赖
- `deploy/docker/` — 多阶段 Alpine 镜像 Dockerfile 与 docker-compose.yml
- `docs/screenshots/` — README 截图
- `Makefile` / `CHANGELOG.md` / `README.md` — 构建脚本、变更记录、含信令协议规范的完整文档

## 关键约束

- 前端刻意保持零框架、零打包：所有改动都落在 `web/index.html`，禁止引入 npm/Vite/前端框架（见 README「常见问题」3）。
- Go >= 1.22（go.mod），第三方依赖仅 `gorilla/websocket`；CI 无 golangci-lint，质量门禁为 build + test(-race) + vet。
- 后端代码注释统一使用中文。
- 信令协议（WebSocket `/ws`，JSON，`type` 分发）在 README「信令协议规范」有完整定义；修改 `message.go`/`errors.go` 的消息类型或错误码时必须同步 README。
- 服务配置仅两个环境变量：`ADDR`（默认 `:8080`）、`WS_ALLOWED_ORIGINS`（Origin 白名单，逗号分隔，`*` 放行全部）。
- Full Mesh P2P 拓扑，媒体不经服务器转发，单房间建议 2~4 人；不要往服务端加媒体转发能力。
- 姊妹项目 **webrtc-signaling**（github.com/build-workbench/webrtc-signaling）是后端工程化分布式信令服务，本项目定位为「前端通话客户端 + 极简信令」，二者形成对照；定位为学习实验性质，非生产级方案。

## 文档约定

- CHANGELOG.md：面向用户的变更在合入时写入 [Unreleased]（Keep a Changelog zh-CN 格式）
- 文档全中文
