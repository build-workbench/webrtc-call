import { describe, it, expect, beforeEach } from 'vitest';
import { createRoomState } from '../src/state/roomState.js';
import { createMediaState } from '../src/state/mediaState.js';
import { createPeersState } from '../src/state/peersState.js';
import { createAppState } from '../src/state/index.js';

describe('state subscribe/notify', function () {
  describe('roomState', function () {
    var room;

    beforeEach(function () {
      room = createRoomState();
    });

    it('notifies on status change', function () {
      var called = false;
      room.subscribe(function () { called = true; });
      room.setStatus('connecting');
      expect(called).toBe(true);
    });

    it('notifies on roomId change', function () {
      var called = false;
      room.subscribe(function () { called = true; });
      room.setRoomId('room1');
      expect(called).toBe(true);
    });

    it('unsubscribes correctly', function () {
      var count = 0;
      var unsub = room.subscribe(function () { count++; });
      room.setStatus('connecting');
      expect(count).toBe(1);
      unsub();
      room.setStatus('joined');
      expect(count).toBe(1);
    });

    it('notifies once on reset', function () {
      var count = 0;
      room.subscribe(function () { count++; });
      room.setStatus('joined');
      room.setRoomId('room1');
      count = 0; // reset counter
      room.reset();
      expect(count).toBe(1);
    });

    it('prevents reentrant notifications', function () {
      var count = 0;
      room.subscribe(function () {
        count++;
        // This would cause infinite recursion without anti-reentrancy
        room.setRoomId('nested');
      });
      room.setStatus('connecting');
      // Should be 2: first from status change, then from roomId inside callback
      // But the roomId notification is suppressed by anti-reentrancy
      expect(count).toBe(1);
    });

    it('computes isIdle correctly', function () {
      expect(room.isIdle()).toBe(true);
      room.setStatus('joined');
      expect(room.isIdle()).toBe(false);
    });

    it('computes isConnected correctly', function () {
      expect(room.isConnected()).toBe(false);
      room.setStatus('joined');
      expect(room.isConnected()).toBe(true);
    });

    it('computes isConnecting correctly', function () {
      expect(room.isConnecting()).toBe(false);
      room.setStatus('connecting');
      expect(room.isConnecting()).toBe(true);
      room.setStatus('reconnecting');
      expect(room.isConnecting()).toBe(true);
    });
  });

  describe('mediaState', function () {
    var media;

    beforeEach(function () {
      media = createMediaState();
    });

    it('notifies on setLocalStream', function () {
      var called = false;
      media.subscribe(function () { called = true; });
      media.setLocalStream({});
      expect(called).toBe(true);
    });

    it('notifies on toggleMuted', function () {
      var called = false;
      media.subscribe(function () { called = true; });
      media.toggleMuted();
      expect(called).toBe(true);
    });

    it('notifies on toggleCameraOff', function () {
      var called = false;
      media.subscribe(function () { called = true; });
      media.toggleCameraOff();
      expect(called).toBe(true);
    });

    it('notifies once on reset', function () {
      var count = 0;
      media.subscribe(function () { count++; });
      media.setLocalStream({ getTracks: function () { return []; } });
      media.setMuted(true);
      count = 0;
      media.reset();
      expect(count).toBe(1);
    });
  });

  describe('peersState', function () {
    var peers;

    beforeEach(function () {
      peers = createPeersState();
    });

    it('notifies on set', function () {
      var called = false;
      peers.subscribe(function () { called = true; });
      peers.set('peer1', {});
      expect(called).toBe(true);
    });

    it('notifies on remove', function () {
      peers.set('peer1', {});
      var called = false;
      peers.subscribe(function () { called = true; });
      peers.remove('peer1');
      expect(called).toBe(true);
    });

    it('does not notify on remove of nonexistent key', function () {
      var called = false;
      peers.subscribe(function () { called = true; });
      peers.remove('nonexistent');
      expect(called).toBe(false);
    });

    it('notifies on clear', function () {
      peers.set('peer1', {});
      var called = false;
      peers.subscribe(function () { called = true; });
      peers.clear();
      expect(called).toBe(true);
    });
  });

  describe('integration with updateControls', function () {
    it('state changes trigger subscriptions on appState sub-modules', function () {
      var appState = createAppState({ myId: 'test' });
      var roomNotified = false;
      var mediaNotified = false;
      var peersNotified = false;

      appState.room.subscribe(function () { roomNotified = true; });
      appState.media.subscribe(function () { mediaNotified = true; });
      appState.peers.subscribe(function () { peersNotified = true; });

      appState.room.setStatus('connecting');
      appState.media.toggleMuted();
      appState.peers.set('p1', {});

      expect(roomNotified).toBe(true);
      expect(mediaNotified).toBe(true);
      expect(peersNotified).toBe(true);
    });
  });
});
