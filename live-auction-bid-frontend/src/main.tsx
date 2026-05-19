import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Gavel, Radio, Sparkles, Trophy } from 'lucide-react';
import { API_BASE, WS_BASE, Lot, createLot, formatMoney, listLots } from './lib/api';
import './styles.css';

function useAuctionRoom(roomId = 'demo') {
  const [lot, setLot] = useState<Lot | null>(null);
  const [connected, setConnected] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    listLots().then(lots => setLot(lots.find(x => x.roomId === roomId && x.status === 'LIVE') ?? lots[0] ?? null)).catch(console.error);
    const ws = new WebSocket(`${WS_BASE}/ws/rooms/${roomId}`);
    socketRef.current = ws;
    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onmessage = ev => {
      const msg = JSON.parse(ev.data);
      const next = msg.data?.lot ?? msg.data;
      if (msg.type?.startsWith('lot.') || msg.type === 'bid.accepted') setLot(next);
    };
    return () => ws.close();
  }, [roomId]);

  const placeBid = (amount: number) => {
    if (!lot) return;
    const payload = { type: 'bid.place', lotId: lot.id, userId: `u_${Math.floor(Math.random()*9999)}`, nickname: `观众${Math.floor(Math.random()*90+10)}`, amount };
    if (socketRef.current?.readyState === WebSocket.OPEN) socketRef.current.send(JSON.stringify(payload));
  };
  return { lot, connected, placeBid, setLot };
}

function App() {
  const { lot, connected, placeBid, setLot } = useAuctionRoom('demo');
  const [customBid, setCustomBid] = useState('');
  const nextBid = useMemo(() => lot ? lot.currentPrice + lot.minIncrement : 0, [lot]);
  const remaining = lot ? Math.max(0, Math.floor((new Date(lot.endsAt).getTime() - Date.now()) / 1000)) : 0;

  async function seedNewLot() {
    const created = await createLot({ roomId: 'demo', title: 'Vintage Cartier 手镯', description: '二手奢侈品竞拍样品，含鉴定报告。', imageUrl: 'https://images.unsplash.com/photo-1611591437281-460bfbe1220a?w=900', startPrice: 268800, minIncrement: 10000, durationSec: 1200 });
    setLot(created);
  }

  if (!lot) return <main className="page"><button onClick={seedNewLot}>创建演示拍品</button></main>;
  return <main className="page">
    <section className="hero">
      <div><p className="eyebrow"><Radio size={16}/> LIVE AUCTION ROOM / DEMO</p><h1>短视频直播竞拍全栈引擎</h1><p>面向珠宝、二奢、潮玩等高价值非标品：上架、规则、实时出价、动态排名、落锤成交与 AI 氛围联动。</p></div>
      <div className={connected ? 'status ok' : 'status'}>{connected ? 'WebSocket 已连接' : '连接中...'}</div>
    </section>
    <section className="grid">
      <article className="card lot"><img src={lot.imageUrl} /><div className="lotBody"><h2>{lot.title}</h2><p>{lot.description}</p><div className="price">{formatMoney(lot.currentPrice)}</div><p className="meta">下一口 ≥ {formatMoney(nextBid)} · 倒计时 {remaining}s · v{lot.version}</p><div className="bidRow"><input value={customBid} onChange={e=>setCustomBid(e.target.value)} placeholder={`${nextBid}`} /><button onClick={()=>placeBid(Number(customBid || nextBid))}><Gavel size={18}/> 出价</button><button className="ghost" onClick={()=>placeBid(nextBid)}>一键加价</button></div></div></article>
      <aside className="card"><h3><Trophy size={18}/> 实时排名</h3><ol className="ranking">{(lot.ranking ?? []).map(x => <li key={x.userId}><span>#{x.rank} {x.nickname}</span><strong>{formatMoney(x.amount)}</strong></li>)}</ol></aside>
      <aside className="card ai"><h3><Sparkles size={18}/> AI 气氛官</h3><p>{lot.atmosphereText || '等待新出价，AI 将根据价格跃迁、剩余时间与竞价人数生成直播话术。'}</p><small>后续接 Ollama/Qwen/Llama：动态估价、加价建议、主播话术、异常出价风控。</small></aside>
    </section>
    <footer>Backend: {API_BASE} · WS: {WS_BASE}</footer>
  </main>;
}

createRoot(document.getElementById('root')!).render(<App />);
