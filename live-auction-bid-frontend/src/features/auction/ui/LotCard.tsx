import { Gavel } from 'lucide-react';
import { useMemo, useState } from 'react';
import { cny, formatMoney, moneyAmount } from '../../../shared/lib/money';
import type { Lot, Money } from '../../../shared/types/auction';

type Props = { lot: Lot; onBid: (amount: Money) => void };

export function LotCard({ lot, onBid }: Props) {
  const [customBid, setCustomBid] = useState('');
  const nextBid = useMemo(() => moneyAmount(lot.currentPrice) + moneyAmount(lot.rule.minIncrement), [lot.currentPrice, lot.rule.minIncrement]);
  const remaining = Math.max(0, Math.floor((Number(lot.endsAtUnixMs) - Date.now()) / 1000));
  const canBid = lot.status === 'LOT_STATUS_LIVE';

  return (
    <article className="card lot">
      <img src={lot.imageUrl} alt={lot.title} />
      <div className="lotBody">
        <h2>{lot.title}</h2>
        <p>{lot.description}</p>
        <div className="price">{formatMoney(lot.currentPrice)}</div>
        <p className="meta">状态 {lot.status.replace('LOT_STATUS_', '')} · 下一口 ≥ {formatMoney(cny(nextBid))} · 倒计时 {remaining}s · v{lot.version}</p>
        {lot.status === 'LOT_STATUS_CANCELLED' && <div className="duelBanner">竞拍已取消{lot.cancelReason ? `：${lot.cancelReason}` : ''}</div>}
        {lot.duelState?.active && <div className="duelBanner">双人巅峰竞拍：{lot.duelState.userANickname} VS {lot.duelState.userBNickname}</div>}
        <div className="bidRow">
          <input disabled={!canBid} value={customBid} onChange={(e) => setCustomBid(e.target.value)} placeholder={`${nextBid}`} />
          <button disabled={!canBid} onClick={() => onBid(cny(Number(customBid || nextBid)))}><Gavel size={18} /> 出价</button>
          <button disabled={!canBid} className="ghost" onClick={() => onBid(cny(nextBid))}>一键加价</button>
        </div>
      </div>
    </article>
  );
}
