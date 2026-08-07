# CHANGELOG

## 2.0.0 (2026-08-07)

重构为轻量单文件架构：

- 移除 OpenSpec 规范流程、VitePress 文档站、e2e 测试、前端工程化代码
- 前端从 28 文件压缩为单文件 `web/index.html`
- 后端移除 `/config.js` 端点，简化为纯信令 + 静态文件服务
- 新增 DataChannel 聊天、屏幕共享
- 新增断线指数退避重连
- CI 精简为 Go build + test + vet
- Makefile 去掉 golangci-lint 依赖

## 1.0.0

- WebSocket 信令服务（Go + gorilla/websocket）
- 原生 JS 前端（控制器分层、状态管理、单元测试）
- 音视频通话、DataChannel 聊天、屏幕共享
- Docker 部署、VitePress 文档站、OpenSpec 规范管理
