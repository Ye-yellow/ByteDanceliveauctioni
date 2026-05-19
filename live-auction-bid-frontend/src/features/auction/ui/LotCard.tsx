import { Gavel } from 'lucide-react';
import { useMemo, useState } from 'react';
import { formatMoney } from '../../../shared/lib/money';
import type { Lot, Money } from '../../../shared/types/auction';

type Props = { lot: Lot; onBid: (amount: Money) => void };

export function LotCard({ lot, onBid }: Props) {
  const [customBid, setCustomBid] = useState('');
  const nextBid = useMemo(() => lot.currentPrice + lot.minIncrement, [lot.currentPrice, lot.minIncrement]);
  const remaining = Math.max(0, Math.floor((new Date(lot.endsAt).getTime() - Date.now()) / 1000));

  return (
    <article className="card lot">
      <img src={lot.imageUrl} alt={lot.title} />
      <div className="lotBody">
        <h2>{lot.title}</h2>
        <p>{lot.description}</p>
        <div className="price">{formatMoney(lot.currentPrice)}</div>
        <p className="meta">下一口 ≥ {formatMoney(nextBid)} · 倒计时 {remaining}s · v{lot.version}</p>
        <div className="bidRow">
          <input value={customBid} onChange={(e) => setCustomBid(e.target.value)} placeholder={`${nextBid}`} />
          <button onClick={() => onBid(Number(customBid || nextBid))}><Gavel size={18} /> 出价</button>
          <button className="ghost" onClick={() => onBid(nextBid)}>一键加价</button>
        </div>
      </div>
    </article>
  );
}
