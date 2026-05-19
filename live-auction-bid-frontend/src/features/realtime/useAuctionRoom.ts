import { useEffect, useRef, useState } from 'react';
import { WS_BASE } from '../../shared/config/env';
import type { AuctionEvent, Lot, Money, RankingItem } from '../../shared/types/auction';
import { getRoomSnapshot, listLots, placeBid as placeBidApi } from '../auction/api/auctionApi';

export function useAuctionRoom(roomId = 'demo') {
  const [lot, setLot] = useState<Lot | null>(null);
  const [ranking, setRanking] = useState<RankingItem[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let disposed = false;
    getRoomSnapshot(roomId)
      .then((snapshot) => { if (!disposed) { setLot(snapshot.currentLot ?? null); setRanking(snapshot.ranking ?? []); } })
      .catch(() => listLots(roomId).then((lots) => !disposed && setLot(lots.find((x) => x.status === 'LOT_STATUS_LIVE') ?? lots[0] ?? null)).catch((e) => !disposed && setError(e instanceof Error ? e.message : String(e))));

    const ws = new WebSocket(`${WS_BASE}/ws/rooms/${roomId}`);
    socketRef.current = ws;
    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setError('WebSocket 连接失败，请确认真实后端已启动。');
    ws.onmessage = (ev) => {
      const msg = JSON.parse(ev.data) as AuctionEvent;
      if (msg.snapshot) { setLot(msg.snapshot.currentLot ?? null); setRanking(msg.snapshot.ranking ?? []); return; }
      if (msg.lot) setLot(msg.lot);
      if (msg.ranking) setRanking(msg.ranking);
    };
    return () => { disposed = true; ws.close(); };
  }, [roomId]);

  const placeBid = async (amount: Money) => {
    if (!lot) return;
    const payload = { lotId: lot.id, userId: `u_${Math.floor(Math.random() * 9999)}`, nickname: `观众${Math.floor(Math.random() * 90 + 10)}`, amount, clientKnownVersion: lot.version, idempotencyKey: `bid_${Date.now()}_${Math.random()}` };
    const res = await placeBidApi(lot.id, payload);
    if (res.lot) setLot(res.lot);
  };

  return { lot, ranking, connected, error, placeBid, setLot };
}
