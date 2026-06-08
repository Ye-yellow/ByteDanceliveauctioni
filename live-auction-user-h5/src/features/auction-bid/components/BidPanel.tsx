import { useMemo, useState } from 'react';
import { isBiddableLotStatus } from '../../../entities/auction/model/status';
import type { Lot } from '../../../shared/api/types';
import { amountFromMajor, formatMoney, moneyMajorNumber, moneyNumber } from '../../../shared/lib/money';

type BidPanelProps = {
  lot: Lot | null;
  loading: boolean;
  disabledReason?: string;
  onBid: (amount: number) => void;
  onTip?: (message: string) => void;
};

function normalizeAmountInput(value: string) {
  const cleaned = value.replace(/[^\d.]/g, '');
  const [integer = '', ...decimalParts] = cleaned.split('.');
  return decimalParts.length ? `${integer}.${decimalParts.join('')}` : integer;
}

export function BidPanel({ lot, loading, disabledReason = '', onBid, onTip }: BidPanelProps) {
  const currentAmount = useMemo(() => {
    if (!lot) return 0;
    return moneyNumber(lot.currentPrice) || moneyNumber(lot.rule.startPrice);
  }, [lot]);
  const minAmount = useMemo(() => {
    if (!lot) return 0;
    return currentAmount + moneyNumber(lot.rule.minIncrement);
  }, [currentAmount, lot]);

  const lotId = lot?.id || '';
  const [draft, setDraft] = useState({ lotId: '', majorValue: '' });
  const draftMajorValue = draft.lotId === lotId ? draft.majorValue : String(moneyMajorNumber(minAmount));
  const parsedMajorValue = Number(draftMajorValue);
  const amount = Number.isFinite(parsedMajorValue) ? amountFromMajor(parsedMajorValue) : 0;
  const step = moneyNumber(lot?.rule.minIncrement) || 100;
  const open = isBiddableLotStatus(lot?.status);
  const amountTooLow = Boolean(lot && amount < minAmount);
  const controlsDisabled = !lot || !open || loading || Boolean(disabledReason);
  const submitDisabled = controlsDisabled;
  const deltaAmount = Math.max(0, amount - currentAmount);
  const deltaText = formatMoney(deltaAmount);
  const minAmountText = formatMoney(minAmount);
  const bidHintText = disabledReason || (amountTooLow ? `当前最低出价 ${minAmountText}` : '按当前最低加价出价');
  const amountDigitCount = draftMajorValue.replace(/\D/g, '').length;
  const amountSizeClass = amountDigitCount >= 10 ? 'amountTiny' : amountDigitCount >= 8 ? 'amountCompact' : '';

  const updateAmount = (nextAmount: number) => {
    setDraft({ lotId, majorValue: String(moneyMajorNumber(nextAmount)) });
  };

  const handleBid = () => {
    if (disabledReason) {
      onTip?.(disabledReason);
      return;
    }
    if (!lot) {
      onTip?.('当前暂无竞拍，等待主播开拍');
      return;
    }
    if (!open) {
      onTip?.('当前拍品状态不可出价');
      return;
    }
    if (loading) return;
    if (!draftMajorValue.trim() || !Number.isFinite(parsedMajorValue)) {
      onTip?.('请输入出价金额');
      return;
    }
    if (amount < minAmount) {
      onTip?.(`不能低于当前最低出价 ${minAmountText}`);
      return;
    }
    onBid(amount);
  };

  return (
    <section className="bidPanel" aria-busy={loading}>
      <p className={`bidDeltaPill ${disabledReason ? 'muted' : amountTooLow ? 'warning' : ''}`}>
        {!disabledReason && !amountTooLow && deltaAmount > 0 ? (
          <>
            高于当前价 <span className="scrollAmount inlineAmount" title={deltaText}>{deltaText}</span>
          </>
        ) : bidHintText}
      </p>
      <div className="bidAmountRow">
        <button
          type="button"
          disabled={controlsDisabled}
          aria-label="减少出价"
          onClick={() => updateAmount(Math.max(0, (amount || minAmount) - step))}
        >
          −
        </button>
        <label className={`bidAmountControl ${amountSizeClass}`}>
          <input
            type="text"
            inputMode="decimal"
            disabled={controlsDisabled}
            value={draftMajorValue}
            onChange={(event) => {
              setDraft({ lotId, majorValue: normalizeAmountInput(event.target.value) });
            }}
          />
          <i>元</i>
        </label>
        <button
          type="button"
          disabled={controlsDisabled}
          aria-label="增加出价"
          onClick={() => updateAmount((amount || minAmount) + step)}
        >
          ＋
        </button>
      </div>

      <button className="bidButton" type="button" disabled={submitDisabled} onClick={handleBid}>
        {loading ? '出价确认中...' : '立即出价'}
      </button>
    </section>
  );
}
