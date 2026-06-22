/**
 * 应用状态模块
 * 聚合房间、媒体、Peer 三大领域状态。
 */

import { createMediaState } from './mediaState.js';
import { createPeersState } from './peersState.js';
import { createRoomState, RoomStatus } from './roomState.js';

/**
 * 创建应用状态管理器
 * @param {Object} options - 配置选项
 * @param {string} options.myId - 客户端 ID
 * @returns {Object} 应用状态接口
 */
export function createAppState(options) {
  const myId = options.myId;
  const roomState = createRoomState();
  const mediaState = createMediaState();
  const peersState = createPeersState();

  /**
   * 获取完整的客户端 ID
   * @returns {string}
   */
  function getMyId() {
    return myId;
  }

  return {
    // 常量
    RoomStatus: RoomStatus,

    // 子状态
    room: roomState,
    media: mediaState,
    peers: peersState,

    // 全局
    getMyId: getMyId
  };
}

// 导出枚举
export { RoomStatus };
