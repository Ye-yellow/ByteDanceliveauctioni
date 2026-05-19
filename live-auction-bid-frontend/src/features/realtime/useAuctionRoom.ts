import { useEffect, useRef, useState } from 'react';
import { WS_BASE } from '../../shared/config/env';
import type { Lot, Money } from '../../shared/types/auction';
import { listLots } from '../auction/api/auctionApi';

type ServerMessage = {
  type?: string;
  data?: Lot | { lot?: Lot };
};

export function useAuctionRoom(roomId = 'demo') {
  const [lot, setLot] = useState<Lot | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let disposed = false;
    listLots()
      .then((lots) => {
        if (disposed) return;
        setLot(lots.find((x) => x.roomId === roomId && x.status === 'LIVE') ?? lots[0] ?? null);
      })
      .catch((e) => {
        if (!disposed) setError(e instanceof Error ? e.message : String(e));
      });

    const ws = new WebSocket(`${WS_BASE}/ws/rooms/${roomId}`);
    socketRef.current = ws;
    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setError('WebSocket 连接失败，请确认真实后端已启动。');
    ws.onmessage = (ev) => {
      const msg = JSON.parse(ev.data) as ServerMessage;
      const next = 'lot' in (msg.data ?? {}) ? (msg.data as { lot?: Lot }).lot : (msg.data as Lot | undefined);
      if ((msg.type?.startsWith('lot.') || msg.type === 'bid.accepted') && next) setLot(next);
    };
    return () => {
      disposed = true;
      ws.close();
    };
  }, [roomId]);

  const placeBid = (amount: Money) => {
    if (!lot) return;
    const payload = {
      type: 'bid.place',
      lotId: lot.id,
      userId: `u_${Math.floor(Math.random() * 9999)}`,
      nickname: `观众${Math.floor(Math.random() * 90 + 10)}`,
      amount,
    };
    if (socketRef.current?.readyState === WebSocket.OPEN) socketRef.current.send(JSON.stringify(payload));
  };

  return { lot, connected, error, placeBid, setLot };
}
