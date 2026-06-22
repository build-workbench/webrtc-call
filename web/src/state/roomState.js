/**
 * 房间状态模块
 * 管理房间连接的状态机，封装重连逻辑。
 */

import { createObservable } from './observable.js';

/**
 * 房间状态枚举
 * @readonly
 * @enum {string}
 */
export const RoomStatus = {
  IDLE: 'idle',
  CONNECTING: 'connecting',
  JOINED: 'joined',
  RECONNECTING: 'reconnecting'
};

/**
 * 创建房间状态管理器
 * @param {Object} options - 配置选项
 * @param {string} options.myId - 客户端 ID
 * @returns {Object} 房间状态接口
 */
export function createRoomState(options) {
  const observable = createObservable();

  // 私有状态
  let _roomId = null;
  let _status = RoomStatus.IDLE;
  let _ws = null;
  let _manualClose = false;
  let _retryJoinAfterClose = false;
  let _reconnectTimer = null;
  let _reconnectAttempts = 0;
  let _lastMembers = [];

  // === 房间 ID ===

  function getRoomId() { return _roomId; }
  function setRoomId(value) { _roomId = value; observable.notify(); }

  // === 状态 ===

  function getStatus() { return _status; }
  function setStatus(value) { _status = value; observable.notify(); }

  function isIdle() { return _status === RoomStatus.IDLE; }
  function isConnected() { return _status === RoomStatus.JOINED; }
  function isConnecting() {
    return _status === RoomStatus.CONNECTING || _status === RoomStatus.RECONNECTING;
  }

  // === WebSocket ===

  function getWs() { return _ws; }
  function setWs(value) { _ws = value; observable.notify(); }

  function isWebSocketOpen() { return _ws && _ws.readyState === WebSocket.OPEN; }
  function isWebSocketConnecting() {
    return _ws && (_ws.readyState === WebSocket.OPEN || _ws.readyState === WebSocket.CONNECTING);
  }

  // === 重连状态 ===

  function getManualClose() { return _manualClose; }
  function setManualClose(value) { _manualClose = value; observable.notify(); }

  function getRetryJoinAfterClose() { return _retryJoinAfterClose; }
  function setRetryJoinAfterClose(value) { _retryJoinAfterClose = value; observable.notify(); }

  function getReconnectTimer() { return _reconnectTimer; }
  function setReconnectTimer(value) { _reconnectTimer = value; observable.notify(); }

  function getReconnectAttempts() { return _reconnectAttempts; }
  function incrementReconnectAttempts() { _reconnectAttempts += 1; observable.notify(); }
  function resetReconnectAttempts() { _reconnectAttempts = 0; observable.notify(); }

  function clearReconnectTimer(clearFn) {
    if (_reconnectTimer) {
      const clear = clearFn || window.clearTimeout.bind(window);
      clear(_reconnectTimer);
      _reconnectTimer = null;
      observable.notify();
    }
  }

  // === 成员列表 ===

  function getLastMembers() { return _lastMembers.slice(); }
  function setLastMembers(value) {
    _lastMembers = Array.isArray(value) ? value.slice() : [];
    observable.notify();
  }

  // === 重置 ===

  function reset() {
    clearReconnectTimer();
    _ws = null;
    _roomId = null;
    _status = RoomStatus.IDLE;
    _manualClose = false;
    _retryJoinAfterClose = false;
    _reconnectAttempts = 0;
    _lastMembers = [];
    observable.notify();
  }

  return {
    // 订阅
    subscribe: observable.subscribe,

    // 房间 ID
    getRoomId: getRoomId,
    setRoomId: setRoomId,

    // 状态
    getStatus: getStatus,
    setStatus: setStatus,
    isIdle: isIdle,
    isConnected: isConnected,
    isConnecting: isConnecting,

    // WebSocket
    getWs: getWs,
    setWs: setWs,
    isWebSocketOpen: isWebSocketOpen,
    isWebSocketConnecting: isWebSocketConnecting,

    // 重连状态
    getManualClose: getManualClose,
    setManualClose: setManualClose,
    getRetryJoinAfterClose: getRetryJoinAfterClose,
    setRetryJoinAfterClose: setRetryJoinAfterClose,
    getReconnectTimer: getReconnectTimer,
    setReconnectTimer: setReconnectTimer,
    getReconnectAttempts: getReconnectAttempts,
    incrementReconnectAttempts: incrementReconnectAttempts,
    resetReconnectAttempts: resetReconnectAttempts,
    clearReconnectTimer: clearReconnectTimer,

    // 成员列表
    getLastMembers: getLastMembers,
    setLastMembers: setLastMembers,

    // 重置
    reset: reset
  };
}
