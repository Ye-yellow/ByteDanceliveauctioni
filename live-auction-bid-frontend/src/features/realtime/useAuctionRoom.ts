import { useEffect, useRef, useState } from 'react';
import { WS_BASE } from '../../shared/config/env';
import { normalizeAuctionEvent, resultMessage } from '../../shared/api/result';
import type { AuctionEvent, Lot, Money, RankingItem, User } from '../../shared/types/auction';
import { getRoomSnapshot, listLots, placeBid as placeBidApi } from '../auction/api/auctionApi';

export function useAuctionRoom(roomId = 'demo', user: User | null = null) {
  const [lot, setLot] = useState<Lot | null>(null);
  const [ranking, setRanking] = useState<RankingItem[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let disposed = false;
    getRoomSnapshot(roomId)
      .then((snapshot) => { if (!disposed) { setLot(snapshot.currentLot ?? null); setRanking(snapshot.ranking ?? []); } })
      .catch(() => listLots(roomId).then((lots) => !disposed && setLot(lots.find((x) => x.status === 'LOT_STATUS_LIVE') ?? lots[0] ?? null)).catch((e) => !disposed && setError(resultMessage(e))));

    const ws = new WebSocket(`${WS_BASE}/ws/rooms/${roomId}`);
    socketRef.current = ws;
    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setError('WebSocket 连接失败，请确认真实后端已启动。');
    ws.onmessage = (ev) => {
      const msg = normalizeAuctionEvent(JSON.parse(ev.data)) as AuctionEvent;
      if (msg.snapshot) { setLot(msg.snapshot.currentLot ?? null); setRanking(msg.snapshot.ranking ?? []); return; }
      if (msg.lot) {
        setLot(msg.lot);
        if (msg.lot.status === 'LOT_STATUS_CANCELLED') {
          setNotice(msg.reason || msg.lot.cancelReason || '竞拍已被主播异常取消，请等待后续安排。');
        }
      }
      if (msg.ranking) setRanking(msg.ranking);
      if (msg.type === 'AUCTION_EVENT_TYPE_BID_REJECTED' && msg.reason) setNotice(msg.reason);
      if (msg.type === 'AUCTION_EVENT_TYPE_LOT_CANCELLED') setNotice(msg.reason || '竞拍已被主播异常取消，请等待后续安排。');
    };
    return () => { disposed = true; ws.close(); };
  }, [roomId]);

  const placeBid = async (amount: Money) => {
    setNotice(null);
    setError(null);
    if (!lot) return;
    if (!user) {
      setError('请先注册或登录买家账号再出价。');
      return;
    }
    const payload = { lotId: lot.id, amount, clientKnownVersion: lot.version, idempotencyKey: `bid_${user.id}_${Date.now()}_${Math.random()}` };
    try {
      const res = await placeBidApi(lot.id, payload);
      if (res.lot) {
        setLot(res.lot);
        if (res.lot.status === 'LOT_STATUS_CANCELLED') setNotice(res.lot.cancelReason || '竞拍已取消，不能继续出价。');
      }
      if (res.ranking) setRanking(res.ranking);
      if (!res.accepted) setNotice(res.rejectReason || res.result?.message || '出价未被接受');
    } catch (e) {
      setError(resultMessage(e));
    }
  };

  return { lot, ranking, connected, error, notice, placeBid, setLot };
}
