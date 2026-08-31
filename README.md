# WebRTC Call

浏览器端原生 WebRTC 音视频通话（1v1 / Mesh 多人）项目，配套 Go WebSocket 信令，个人练手业余作品。与后端工程化信令服务 [webrtc-signaling](https://github.com/build-workbench/webrtc-signaling) 形成「前端通话客户端 ↔ 后端信令服务」的对照。

## 关于本项目

这是我的业余练手项目，用来学习 WebRTC 和 Go：

- 后端用 Go + gorilla/websocket 写一个最小的信令服务（房间管理、消息路由）
- 前端用原生 JavaScript 写单页应用，浏览器间直接建立 P2P 连接

**这不是一个面向生产的产品**，也没打算做成教程。代码按自己看得懂、跑得起来的原则组织，可能有疏漏和简化。当前已知局限：

- 未接入 TURN 服务器，仅适合本机或局域网环境
- 前端为单文件实现，无构建工具、无框架
- 支持 1v1 及小规模 Mesh 多人通话，规模上限取决于浏览器连接数

## 特性

- WebSocket 信令服务器（房间管理、消息路由、速率限制）
- 1v1 及小规模 Mesh 音视频通话
- DataChannel 聊天
- 屏幕共享
- 静音 / 摄像头开关
- 断线指数退避重连

## 快速开始

```bash
go run ./cmd/server
```

浏览器打开 `http://localhost:8080`，输入房间名和昵称，点击加入。
新开窗口重复操作即可 1v1 通话。

## 信令协议

WebSocket 连接 `GET /ws`，消息为 JSON，结构取决于 `type` 字段。

**客户端 → 服务端：**

| type | 字段 | 说明 |
|------|------|------|
| `join` | `room`, `from` | 加入房间 |
| `leave` | - | 离开房间 |
| `offer` | `to`, `sdp` | SDP Offer |
| `answer` | `to`, `sdp` | SDP Answer |
| `candidate` | `to`, `candidate` | ICE 候选 |
| `hangup` | `to` | 挂断 |
| `ping` | - | 心跳 |

**服务端 → 客户端：**

| type | 字段 | 说明 |
|------|------|------|
| `joined` | `room`, `from` | 加入成功 |
| `room_members` | `room`, `members` | 成员列表变更 |
| `pong` | - | 心跳回复 |
| `error` | `code`, `error` | 错误 |

## 项目结构

```
cmd/server/main.go     Go 入口
internal/signal/       信令核心（Hub、消息路由、房间管理）
web/index.html         单文件前端
deploy/docker/         Docker 部署
```

## 配置

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `ADDR` | `:8080` | 监听地址 |
| `WS_ALLOWED_ORIGINS` | - | WebSocket Origin 白名单（逗号分隔，`*` 允许全部） |

## Docker

```bash
docker build -f deploy/docker/Dockerfile -t webrtc .
docker run --rm -p 8080:8080 webrtc
```

## 许可证

[MIT](LICENSE)
