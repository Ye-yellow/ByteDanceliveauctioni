import { useEffect, useState } from 'react';
import { createLot, listLots, revealTrustCard, settleLot, startDuel, startLot } from '../../features/auction/api/auctionApi';
import { cny, formatMoney } from '../../shared/lib/money';
import type { Lot } from '../../shared/types/auction';

export function HostConsolePage() {
  const [lots, setLots] = useState<Lot[]>([]);
  const [selected, setSelected] = useState<Lot | null>(null);
  const [title, setTitle] = useState('Vintage Cartier 手镯');
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');

  const showError = (e: unknown) => {
    const msg = e instanceof Error ? e.message : String(e);
    setError(msg || '操作失败，请确认后端服务是否启动。');
    setNotice('');
  };

  const load = async () => {
    const xs = await listLots('demo');
    setLots(xs);
    setSelected((prev) => xs.find((x) => x.id === prev?.id) ?? xs[xs.length - 1] ?? null);
  };

  useEffect(() => {
    load().catch(showError);
  }, []);

  const create = async () => {
    if (busy) return;
    setBusy(true);
    setError('');
    setNotice('正在创建草稿...');
    try {
      const lot = await createLot({
        roomId: 'demo',
        title,
        description: '二手奢侈品竞拍样品，含证书、瑕疵说明与售后承诺。',
        imageUrl: 'https://images.unsplash.com/photo-1611591437281-460bfbe1220a?w=900',
        rule: {
          startPrice: cny(268800),
          minIncrement: cny(10000),
          durationSeconds: 300,
          antiSnipeWindowSeconds: 15,
          antiSnipeExtendSeconds: 15,
          maxExtendCount: 3,
        },
        trustCards: [
          { id: 'cert', type: 'TRUST_CARD_TYPE_CERTIFICATE', title: '鉴定证书', content: '证书编号已核验，支持回放查看。' },
          { id: 'flaw', type: 'TRUST_CARD_TYPE_FLAW', title: '瑕疵说明', content: '边角轻微磨损，已在细节图中标注。' },
          { id: 'service', type: 'TRUST_CARD_TYPE_SERVICE', title: '售后承诺', content: '支持平台复检与保真服务。' },
        ],
      });
      setSelected(lot);
      await load();
      setNotice(`草稿创建成功：${lot.title}。下一步点击“开拍”。`);
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };

  const act = async (label: string, fn: (id: string) => Promise<Lot>) => {
    if (!selected || busy) return;
    setBusy(true);
    setError('');
    setNotice(`正在${label}...`);
    try {
      const lot = await fn(selected.id);
      setSelected(lot);
      await load();
      setNotice(`${label}成功。`);
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };

  const reveal = async (cardId: string) => {
    if (!selected || busy) return;
    setBusy(true);
    setError('');
    setNotice('正在揭示信任卡片...');
    try {
      const r = await revealTrustCard(selected.id, cardId);
      setSelected(r.lot);
      await load();
      setNotice(`已揭示：${r.trustCard.title}`);
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="page">
      <section className="hero">
        <div>
          <p className="eyebrow">HOST CONSOLE / DEMO</p>
          <h1>主播/运营控制台</h1>
          <p>创建拍品、开拍、揭示信任卡片、触发 Duel、落锤成交。</p>
          <p><a href="/">去观众端</a></p>
        </div>
      </section>

      {(notice || error) && <section className={error ? 'notice error' : 'notice'}>{error || notice}</section>}

      <section className="grid">
        <article className="card">
          <h2>创建拍品</h2>
          <input value={title} onChange={(e) => setTitle(e.target.value)} />
          <button disabled={busy || !title.trim()} onClick={create}>{busy ? '处理中...' : '创建草稿'}</button>
          <p className="meta">当前已有 {lots.length} 个拍品。创建后会出现在右侧。</p>
        </article>

        <article className="card">
          <h2>当前拍品</h2>
          {selected ? (
            <>
              <p>{selected.title}</p>
              <p className="price">{formatMoney(selected.currentPrice)}</p>
              <p className="meta">状态 {selected.status} · v{selected.version}</p>
              <div className="bidRow">
                <button disabled={busy || selected.status !== 'LOT_STATUS_DRAFT'} onClick={() => act('开拍', startLot)}>开拍</button>
                <button disabled={busy || selected.status !== 'LOT_STATUS_LIVE'} onClick={() => act('进入 Duel', startDuel)}>进入 Duel</button>
                <button disabled={busy || selected.status !== 'LOT_STATUS_LIVE'} onClick={() => act('落锤成交', settleLot)}>落锤成交</button>
              </div>
            </>
          ) : <p>暂无拍品</p>}
        </article>

        <article className="card ai">
          <h2>信任揭示</h2>
          {selected?.trustCards?.map((card) => (
            <div className="trustCard" key={card.id}>
              <strong>{card.title}</strong>
              <span>{card.revealed ? card.content : '未揭示'}</span>
              <button disabled={busy || card.revealed || selected.status === 'LOT_STATUS_DRAFT'} onClick={() => reveal(card.id)}>揭示</button>
            </div>
          ))}
          {!selected && <p className="meta">创建拍品后会显示可揭示卡片。</p>}
        </article>
      </section>
    </main>
  );
}
