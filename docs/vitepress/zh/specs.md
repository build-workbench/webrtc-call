---
title: OpenSpec 导航
description: LessUp WebRTC 的公开 OpenSpec 规范地图与 change 工作流说明。
---

# OpenSpec 导航

`openspec/` 是当前仓库唯一的需求与变更规划权威来源。

## 主能力规范

| 能力 | 作用 | 源文件 |
|:-----|:-----|:-------|
| Signaling | WebSocket 生命周期、房间规则、转发行为与限制 | [signaling/spec.md](https://github.com/LessUp/webrtc/blob/main/openspec/specs/signaling/spec.md) |
| Frontend | 浏览器模块边界与客户端行为 | [frontend/spec.md](https://github.com/LessUp/webrtc/blob/main/openspec/specs/frontend/spec.md) |

## Change 工作流

任何运行时或仓库级改动，都应通过 `openspec/changes/<name>/` 组织：

1. **proposal.md** — 为什么要做
2. **design.md** — 怎么做，以及权衡
3. **specs/** — 新增或修改的需求
4. **tasks.md** — 实施任务清单

## 公开工作流摘要

- 先用 OpenSpec 定义变更
- 再按 change 里的任务实现
- 完成仓库校验
- 合并前做 review
- 实现与规范同步完成后再归档 change
