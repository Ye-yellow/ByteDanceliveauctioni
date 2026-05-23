import { useMemo, useState } from 'react';
import { isBiddableLotStatus } from '../../../entities/auction/model/status';
import type { Lot } from '../../../shared/api/types';
import { amountFromMajor, formatMoney, moneyMajorNumber, moneyNumber } from '../../../shared/lib/money';

type BidPanelProps = {
  lot: Lot | null;
  loading: boolean;
  error: string;
  onBid: (amount: number) => void;
};

export function BidPanel({ lot, loading, error, onBid }: BidPanelProps) {
  const minAmount = useMemo(() => {
    if (!lot) return 0;
    const current = moneyNumber(lot.currentPrice) || moneyNumber(lot.rule.startPrice);
    return current + moneyNumber(lot.rule.minIncrement);
  }, [lot]);

  const [draftAmount, setDraftAmount] = useState<number | null>(null);
  const amount = Math.max(draftAmount ?? minAmount, minAmount);
  const step = moneyNumber(lot?.rule.minIncrement) || 100;
  const open = isBiddableLotStatus(lot?.status);
  const disabled = !lot || !open || loading;

  return (
    <section className="bidPanel" aria-busy={loading}>
      <div className="bidAmountRow">
        <button
          type="button"
          disabled={disabled}
          aria-label="减少出价"
          onClick={() => setDraftAmount((value) => Math.max(minAmount, (value ?? amount) - step))}
        >
          −
        </button>
        <label>
          <span>我的出价</span>
          <input
            type="number"
            min={moneyMajorNumber(minAmount)}
            step="0.01"
            value={moneyMajorNumber(amount)}
            onChange={(event) => setDraftAmount(amountFromMajor(Number(event.target.value)))}
          />
        </label>
        <button
          type="button"
          disabled={disabled}
          aria-label="增加出价"
          onClick={() => setDraftAmount((value) => (value ?? amount) + step)}
        >
          ＋
        </button>
      </div>

      <button className="bidButton" type="button" disabled={disabled} onClick={() => onBid(amount)}>
        {loading ? '出价确认中...' : `立即出价 ${formatMoney(amount)}`}
      </button>

      {error ? <p className="bidError" role="alert">{error}</p> : null}
      {!lot ? (
        <p className="bidHint">当前暂无竞拍，等待主播开拍</p>
      ) : !open ? (
        <p className="bidHint">当前拍品状态不可出价</p>
      ) : null}
    </section>
  );
}
