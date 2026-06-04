import { WS_BASE } from '../config/env';
import { normalizeAuctionEvent } from '../api/result';
import type { AuctionEvent, EventType, RoomSnapshot } from '../api/types';
import { getWsTicket } from './wsTicket';

export type RoomSocketStatus = 'connecting' | 'connected' | 'reconnecting' | 'disconnected';

export type RoomSocketMeta = {
  seq: number;
  receivedAt: number;
  receivedAtText: string;
  source: 'socket' | 'recover';
};

type RoomSocketOptions = {
  roomId: string;
  handledEventTypes?: Iterable<EventType | string>;
  recoverSnapshot?: () => Promise<RoomSnapshot | void>;
  onStatusChange?: (status: RoomSocketStatus, attempt: number) => void;
  onEvent?: (event: AuctionEvent, meta: RoomSocketMeta) => void;
  onSnapshot?: (snapshot: RoomSnapshot, meta: RoomSocketMeta) => void;
  onError?: (error: unknown, phase?: 'ticket' | 'socket' | 'recover' | 'message') => void;
};

function nowText() {
  return new Date().toLocaleTimeString('zh-CN', { hour12: false });
}

function reconnectDelay(attempt: number) {
  const base = Math.min(30_000, 800 * 2 ** Math.max(0, attempt - 1));
  return base + Math.floor(Math.random() * 400);
}

function roomURL(roomId: string, ticket: string) {
  const url = new URL(`${WS_BASE}/ws/rooms/${encodeURIComponent(roomId)}`);
  url.searchParams.set('client_app', 'admin-web');
  url.searchParams.set('scope', 'admin');
  url.searchParams.set('ticket', ticket);
  return url.toString();
}

function realtimeConnectionError() {
  const error = new Error('实时连接短暂中断，正在自动重连');
  error.name = 'RealtimeConnectionError';
  return error;
}

export class RoomSocket {
  private readonly options: RoomSocketOptions;
  private readonly handledEventTypes?: Set<string>;
  private socket: WebSocket | null = null;
  private reconnectTimer = 0;
  private closed = true;
  private attempt = 0;
  private seq = 0;

  constructor(options: RoomSocketOptions) {
    this.options = options;
    this.handledEventTypes = options.handledEventTypes ? new Set(Array.from(options.handledEventTypes, String)) : undefined;
  }

  connect() {
    this.closed = false;
    this.open('connecting');
  }

  close() {
    this.closed = true;
    this.clearTimers();
    this.socket?.close();
    this.socket = null;
    this.emitStatus('disconnected');
  }

  private emitStatus(status: RoomSocketStatus) {
    this.options.onStatusChange?.(status, this.attempt);
  }

  private nextMeta(source: RoomSocketMeta['source']): RoomSocketMeta {
    this.seq += 1;
    return { seq: this.seq, source, receivedAt: Date.now(), receivedAtText: nowText() };
  }

  private clearTimers() {
    if (this.reconnectTimer) window.clearTimeout(this.reconnectTimer);
    this.reconnectTimer = 0;
  }

  private async open(status: RoomSocketStatus) {
    if (this.closed) return;
    this.emitStatus(status);
    try {
      const ticket = await getWsTicket({ roomId: this.options.roomId, scope: 'admin' });
      if (this.closed) return;
      this.socket = new WebSocket(roomURL(this.options.roomId, ticket));
    } catch (error) {
      this.options.onError?.(error, 'ticket');
      this.scheduleReconnect();
      return;
    }

    this.socket.onopen = () => {
      this.attempt = 0;
      this.emitStatus('connected');
      void this.recoverSnapshot();
    };
    this.socket.onmessage = (message) => {
      void this.handleMessage(message.data);
    };
    this.socket.onerror = () => {
      this.options.onError?.(realtimeConnectionError(), 'socket');
    };
    this.socket.onclose = () => {
      this.socket = null;
      if (this.closed) return;
      this.scheduleReconnect();
    };
  }

  private async recoverSnapshot() {
    if (!this.options.recoverSnapshot) return;
    try {
      const snapshot = await this.options.recoverSnapshot();
      if (snapshot) this.options.onSnapshot?.(snapshot, this.nextMeta('recover'));
    } catch (error) {
      this.options.onError?.(error, 'recover');
    }
  }

  private scheduleReconnect() {
    if (this.closed || this.reconnectTimer) return;
    this.attempt += 1;
    this.emitStatus('reconnecting');
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = 0;
      this.open('reconnecting');
    }, reconnectDelay(this.attempt));
  }

  private async handleMessage(data: unknown) {
    try {
      const text = typeof data === 'string'
        ? data
        : data instanceof Blob
          ? await data.text()
          : '';
      if (!text) return;
      const event = normalizeAuctionEvent(JSON.parse(text));
      if (event.roomId && event.roomId !== this.options.roomId) return;
      if (this.handledEventTypes && !this.handledEventTypes.has(event.type)) return;
      const meta = this.nextMeta('socket');
      if (event.snapshot) this.options.onSnapshot?.(event.snapshot, meta);
      this.options.onEvent?.(event, meta);
    } catch (error) {
      this.options.onError?.(error, 'message');
    }
  }

}

export function roomSocketStatusLabel(status: RoomSocketStatus) {
  if (status === 'connected') return '已连接';
  if (status === 'reconnecting') return '重连中';
  if (status === 'disconnected') return '已断开';
  return '连接中';
}
