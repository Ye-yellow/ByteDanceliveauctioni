import { useEffect, useState } from 'react';
import { createLot, listLots, revealTrustCard, settleLot, startDuel, startLot } from '../../features/auction/api/auctionApi';
import { cny, formatMoney } from '../../shared/lib/money';
import type { Lot } from '../../shared/types/auction';

export function HostConsolePage() {
  const [lots, setLots] = useState<Lot[]>([]);
  const [selected, setSelected] = useState<Lot | null>(null);
  const [title, setTitle] = useState('Vintage Cartier 手镯');
  const load = async () => { const xs = await listLots('demo'); setLots(xs); setSelected(xs[xs.length - 1] ?? null); };
  useEffect(() => { load().catch(console.error); }, []);
  const create = async () => {
    const lot = await createLot({ roomId: 'demo', title, description: '二手奢侈品竞拍样品，含证书、瑕疵说明与售后承诺。', imageUrl: 'https://images.unsplash.com/photo-1611591437281-460bfbe1220a?w=900', rule: { startPrice: cny(268800), minIncrement: cny(10000), durationSeconds: 300, antiSnipeWindowSeconds: 15, antiSnipeExtendSeconds: 15, maxExtendCount: 3 }, trustCards: [{ id: 'cert', type: 'TRUST_CARD_TYPE_CERTIFICATE', title: '鉴定证书', content: '证书编号已核验，支持回放查看。' }, { id: 'flaw', type: 'TRUST_CARD_TYPE_FLAW', title: '瑕疵说明', content: '边角轻微磨损，已在细节图中标注。' }, { id: 'service', type: 'TRUST_CARD_TYPE_SERVICE', title: '售后承诺', content: '支持平台复检与保真服务。' }] });
    setSelected(lot); await load();
  };
  const act = async (fn: (id: string) => Promise<Lot>) => { if (!selected) return; const lot = await fn(selected.id); setSelected(lot); await load(); };
  return (
    <main className="page">
      <section className="hero"><div><p className="eyebrow">HOST CONSOLE / DEMO</p><h1>主播/运营控制台</h1><p>创建拍品、开拍、揭示信任卡片、触发 Duel、落锤成交。</p><p><a href="/">去观众端</a></p></div></section>
      <section className="grid">
        <article className="card"><h2>创建拍品</h2><input value={title} onChange={(e)=>setTitle(e.target.value)} /><button onClick={create}>创建草稿</button></article>
        <article className="card"><h2>当前拍品</h2>{selected ? <><p>{selected.title}</p><p className="price">{formatMoney(selected.currentPrice)}</p><p className="meta">状态 {selected.status} · v{selected.version}</p><div className="bidRow"><button onClick={()=>act(startLot)}>开拍</button><button onClick={()=>act(startDuel)}>进入 Duel</button><button onClick={()=>act(settleLot)}>落锤成交</button></div></> : <p>暂无拍品</p>}</article>
        <article className="card ai"><h2>信任揭示</h2>{selected?.trustCards?.map(card => <div className="trustCard" key={card.id}><strong>{card.title}</strong><span>{card.revealed ? card.content : '未揭示'}</span><button disabled={card.revealed} onClick={async()=>{ const r = await revealTrustCard(selected.id, card.id); setSelected(r.lot); await load(); }}>揭示</button></div>)}</article>
      </section>
    </main>
  );
}
