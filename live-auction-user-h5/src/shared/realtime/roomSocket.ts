import { normalizeAuctionEvent } from '../api/adapters';
import { AUCTION_EVENT_TYPE, type AuctionSocketEvent } from '../api/types';
import { authSession } from '../auth/authSession';

export type RoomSocketState = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'failed' | 'closing';

export type RoomSocketOptions = {
  roomId: string;
  lastEventId?: string;
  onEvent: (event: AuctionSocketEvent) => void;
  onStateChange?: (state: RoomSocketState) => void;
  onSnapshotRecovery?: () => Promise<void> | void;
  maxReconnectAttempts?: number;
  heartbeatIntervalMs?: number;
  heartbeatTimeoutMs?: number;
};

const WS_BASE = import.meta.env.VITE_WS_BASE || 'ws://localhost:18080';

function roomUrl(roomId: string, lastEventId?: string): string {
  const url = new URL(`/ws/rooms/${encodeURIComponent(roomId)}`, WS_BASE);
  if (lastEventId) url.searchParams.set('last_event_id', lastEventId);
  return url.toString();
}

function reconnectDelay(attempt: number): number {
  const base = Math.min(500 * 2 ** Math.max(0, attempt - 1), 8_000);
  return base + Math.floor(Math.random() * 250);
}

export class RoomSocket {
  private socket: WebSocket | null = null;
  private state: RoomSocketState = 'idle';
  private stopped = false;
  private reconnectAttempts = 0;
  private reconnectTimer = 0;
  private heartbeatTimer = 0;
  private heartbeatTimeoutTimer = 0;
  private lastEventId: string | undefined;
  private options: RoomSocketOptions;

  constructor(options: RoomSocketOptions) {
    this.options = options;
    this.lastEventId = options.lastEventId;
  }

  getState(): RoomSocketState {
    return this.state;
  }

  connect() {
    this.stopped = false;
    void this.open();
  }

  disconnect() {
    this.stopped = true;
    this.setState('closing');
    this.clearTimers();
    this.leaveRoom();
    this.socket?.close();
    this.socket = null;
    this.setState('idle');
  }

  reconnect() {
    if (this.stopped) return;
    this.clearTimers();
    this.socket?.close();
    this.scheduleReconnect();
  }

  auth(accessToken?: string) {
    this.send({
      type: 'AUTH',
      accessToken,
    });
  }

  joinRoom(roomId = this.options.roomId) {
    this.send({
      type: 'JOIN_ROOM',
      roomId,
      lastEventId: this.lastEventId,
    });
  }

  leaveRoom(roomId = this.options.roomId) {
    this.send({
      type: 'LEAVE_ROOM',
      roomId,
    });
  }

  heartbeat() {
    this.send({
      type: AUCTION_EVENT_TYPE.CLIENT_HEARTBEAT,
      clientTimeUnixMs: Date.now(),
      roomId: this.options.roomId,
    });
  }

  private async open() {
    this.setState(this.reconnectAttempts > 0 ? 'reconnecting' : 'connecting');

    try {
      const accessToken = await authSession.getValidAccessToken();
      this.socket = new WebSocket(roomUrl(this.options.roomId, this.lastEventId));
      this.bindSocket(this.socket, accessToken);
    } catch {
      this.scheduleReconnect();
    }
  }

  private bindSocket(socket: WebSocket, accessToken: string | null) {
    socket.onopen = () => {
      this.reconnectAttempts = 0;
      this.setState('connected');
      this.auth(accessToken ?? undefined);
      this.joinRoom();
      this.startHeartbeat();
      void this.options.onSnapshotRecovery?.();
    };

    socket.onmessage = (message) => {
      this.markHeartbeat();
      try {
        const event = normalizeAuctionEvent(JSON.parse(String(message.data)));
        if (event.id) this.lastEventId = event.id;
        this.options.onEvent(event);
      } catch {
        // Realtime payloads must never crash the room page.
      }
    };

    socket.onerror = () => {
      if (this.stopped) return;
      this.setState('failed');
    };

    socket.onclose = () => {
      if (this.stopped) return;
      this.clearHeartbeat();
      this.scheduleReconnect();
    };
  }

  private send(payload: unknown) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify(payload));
  }

  private startHeartbeat() {
    this.clearHeartbeat();
    this.markHeartbeat();
    const interval = this.options.heartbeatIntervalMs ?? 20_000;
    this.heartbeatTimer = window.setInterval(() => {
      this.heartbeat();
      window.clearTimeout(this.heartbeatTimeoutTimer);
      this.heartbeatTimeoutTimer = window.setTimeout(() => {
        this.reconnect();
      }, this.options.heartbeatTimeoutMs ?? 60_000);
    }, interval);
  }

  private markHeartbeat() {
    window.clearTimeout(this.heartbeatTimeoutTimer);
  }

  private clearHeartbeat() {
    window.clearInterval(this.heartbeatTimer);
    window.clearTimeout(this.heartbeatTimeoutTimer);
    this.heartbeatTimer = 0;
    this.heartbeatTimeoutTimer = 0;
  }

  private clearTimers() {
    this.clearHeartbeat();
    window.clearTimeout(this.reconnectTimer);
    this.reconnectTimer = 0;
  }

  private scheduleReconnect() {
    this.clearTimers();
    const maxAttempts = this.options.maxReconnectAttempts ?? Infinity;
    this.reconnectAttempts += 1;

    if (this.reconnectAttempts > maxAttempts) {
      this.setState('failed');
      return;
    }

    this.setState('reconnecting');
    this.reconnectTimer = window.setTimeout(() => {
      void this.open();
    }, reconnectDelay(this.reconnectAttempts));
  }

  private setState(state: RoomSocketState) {
    this.state = state;
    this.options.onStateChange?.(state);
  }
}

export function createRoomSocket(options: RoomSocketOptions): RoomSocket {
  return new RoomSocket(options);
}
