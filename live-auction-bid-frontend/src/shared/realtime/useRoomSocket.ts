import { useEffect, useMemo, useRef, useState } from 'react';
import type { AuctionEvent, EventType, RoomSnapshot } from '../api/types';
import { RoomSocket, roomSocketStatusLabel, type RoomSocketMeta, type RoomSocketStatus } from './roomSocket';

type UseRoomSocketOptions = {
  roomId: string;
  enabled?: boolean;
  handledEventTypes?: Iterable<EventType | string>;
  recoverSnapshot?: () => Promise<RoomSnapshot | void>;
  onStatusChange?: (status: RoomSocketStatus, attempt: number) => void;
  onEvent?: (event: AuctionEvent, meta: RoomSocketMeta) => void;
  onSnapshot?: (snapshot: RoomSnapshot, meta: RoomSocketMeta) => void;
  onError?: (error: unknown, phase?: 'ticket' | 'socket' | 'recover' | 'message') => void;
};

type RoomSocketState = {
  status: RoomSocketStatus;
  reconnectCount: number;
  lastEventAt: number | null;
  lastEventAtText: string;
  lastEventType: string;
  lastEventSeq: number;
};

const initialState: RoomSocketState = {
  status: 'connecting',
  reconnectCount: 0,
  lastEventAt: null,
  lastEventAtText: '未收到',
  lastEventType: '暂无',
  lastEventSeq: 0,
};

export function useRoomSocket(options: UseRoomSocketOptions): RoomSocketState {
  const callbacks = useRef(options);
  callbacks.current = options;
  const eventTypes = useMemo(() => Array.from(options.handledEventTypes ?? [], String), [options.handledEventTypes]);
  const eventTypesKey = eventTypes.join('|');
  const [state, setState] = useState<RoomSocketState>(initialState);

  useEffect(() => {
    if (options.enabled === false) {
      setState((current) => ({ ...current, status: 'disconnected' }));
      return;
    }

    let active = true;
    const socket = new RoomSocket({
      roomId: options.roomId,
      handledEventTypes: eventTypes,
      recoverSnapshot: () => callbacks.current.recoverSnapshot?.() ?? Promise.resolve(),
      onStatusChange: (status, attempt) => {
        if (!active) return;
        setState((current) => ({
          ...current,
          status,
          reconnectCount: status === 'reconnecting' ? current.reconnectCount + 1 : current.reconnectCount,
        }));
        callbacks.current.onStatusChange?.(status, attempt);
      },
      onEvent: (event, meta) => {
        if (!active) return;
        setState((current) => ({
          ...current,
          lastEventAt: meta.receivedAt,
          lastEventAtText: meta.receivedAtText,
          lastEventType: event.type,
          lastEventSeq: meta.seq,
        }));
        callbacks.current.onEvent?.(event, meta);
      },
      onSnapshot: (snapshot, meta) => {
        if (!active) return;
        setState((current) => ({
          ...current,
          lastEventAt: meta.receivedAt,
          lastEventAtText: meta.receivedAtText,
          lastEventType: meta.source === 'recover' ? 'ROOM_SNAPSHOT' : current.lastEventType,
          lastEventSeq: meta.source === 'recover' ? meta.seq : current.lastEventSeq,
        }));
        callbacks.current.onSnapshot?.(snapshot, meta);
      },
      onError: (error, phase) => {
        if (active) callbacks.current.onError?.(error, phase);
      },
    });

    setState(initialState);
    socket.connect();
    return () => {
      active = false;
      socket.close();
    };
  }, [options.roomId, options.enabled, eventTypesKey]);

  return state;
}

export { roomSocketStatusLabel };
export type { RoomSocketMeta, RoomSocketStatus };
