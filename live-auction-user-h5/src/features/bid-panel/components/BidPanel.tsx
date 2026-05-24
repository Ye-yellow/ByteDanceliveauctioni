import { useMemo, useState } from 'react';
import { isBiddableLotStatus } from '../../../entities/auction/model/status';
import type { Lot } from '../../../shared/api/types';
import { amountFromMajor, formatMoney, moneyMajorNumber, moneyNumber } from '../../../shared/lib/money';

type BidPanelProps = {
  lot: Lot | null;
  loading: boolean;
  error: string;
  disabledReason?: string;
  onBid: (amount: number) => void;
};

export function BidPanel({ lot, loading, error, disabledReason = '', onBid }: BidPanelProps) {
  const minAmount = useMemo(() => {
    if (!lot) return 0;
    const current = moneyNumber(lot.currentPrice) || moneyNumber(lot.rule.startPrice);
    return current + moneyNumber(lot.rule.minIncrement);
  }, [lot]);

  const lotId = lot?.id || '';
  const [draft, setDraft] = useState({ lotId: '', majorValue: '' });
  const draftMajorValue = draft.lotId === lotId ? draft.majorValue : String(moneyMajorNumber(minAmount));
  const parsedMajorValue = Number(draftMajorValue);
  const amount = Number.isFinite(parsedMajorValue) ? amountFromMajor(parsedMajorValue) : 0;
  const step = moneyNumber(lot?.rule.minIncrement) || 100;
  const open = isBiddableLotStatus(lot?.status);
  const disabled = !lot || !open || loading || Boolean(disabledReason);

  const updateAmount = (nextAmount: number) => {
    setDraft({ lotId, majorValue: String(moneyMajorNumber(nextAmount)) });
  };

  return (
    <section className="bidPanel" aria-busy={loading}>
      <div className="bidAmountRow">
        <button
          type="button"
          disabled={disabled}
          aria-label="减少出价"
          onClick={() => updateAmount(Math.max(0, (amount || minAmount) - step))}
        >
          −
        </button>
        <label>
          <span>我的出价</span>
          <input
            type="number"
            disabled={disabled}
            step="0.01"
            value={draftMajorValue}
            onChange={(event) => {
              setDraft({ lotId, majorValue: event.target.value });
            }}
          />
        </label>
        <button
          type="button"
          disabled={disabled}
          aria-label="增加出价"
          onClick={() => updateAmount((amount || minAmount) + step)}
        >
          ＋
        </button>
      </div>

      <button className="bidButton" type="button" disabled={disabled} onClick={() => onBid(amount)}>
        {loading ? '出价确认中...' : disabledReason ? '等待他人出价' : `立即出价 ${formatMoney(amount)}`}
      </button>

      {error ? <p className="bidError" role="alert">{error}</p> : null}
      {disabledReason ? <p className="bidHint">{disabledReason}</p> : null}
      {!lot ? (
        <p className="bidHint">当前暂无竞拍，等待主播开拍</p>
      ) : !open ? (
        <p className="bidHint">当前拍品状态不可出价</p>
      ) : null}
    </section>
  );
}
