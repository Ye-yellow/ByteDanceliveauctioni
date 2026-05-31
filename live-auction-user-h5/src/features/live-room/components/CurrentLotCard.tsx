import type { Lot } from '../../../shared/api/types';
import { formatMoney, moneyNumber } from '../../../shared/lib/money';
import { useServerCountdown } from '../hooks/useServerCountdown';
import { deriveLotDisplayState, lotHasBid, type LotDisplayState } from '../model/lotDisplayState';

type CurrentLotCardProps = {
  lot: Lot;
  serverTimeUnixMs?: number | string;
  serverTimeReceivedAtUnixMs?: number;
  displayState?: LotDisplayState;
};

const cardCopyByState: Record<LotDisplayState, { status: string; label: string; value: string; className: string }> = {
  upcoming: { status: '即将开拍', label: '等待主播开拍', value: '未开始', className: 'isWaiting' },
  live: { status: '竞拍中', label: '距离拍还剩', value: '', className: 'isOpen' },
  pendingPayment: { status: '截拍中', label: '竞拍已结束', value: '截拍中', className: 'isClosed' },
  finished: { status: '竞拍结束', label: '竞拍已结束', value: '已结束', className: 'isClosed' },
  failed: { status: '竞拍结束', label: '竞拍已结束', value: '已结束', className: 'isClosed' },
  cancelled: { status: '已取消', label: '主播已取消本件', value: '已取消', className: 'isClosed' },
  syncing: { status: '待同步', label: '等待状态同步', value: '同步中', className: 'isWaiting' },
};

function currentLotCopy(lot: Lot, state: LotDisplayState) {
  if (state === 'failed' && !lotHasBid(lot)) {
    return { status: '竞拍未成交', label: '竞拍未成交', value: '已结束', className: 'isClosed' };
  }
  return cardCopyByState[state];
}

function lotPersonText(lot: Lot, state: LotDisplayState) {
  const leadingName = lot.leadingNickname || lot.leadingUserId;
  const winnerName = lot.winnerNickname || lot.winnerUserId;
  if (state === 'live') return leadingName ? `${leadingName} 当前领先` : '等待用户出价';
  if (state === 'pendingPayment') return winnerName ? `成交用户：${winnerName}` : '等待截拍确认';
  if (state === 'finished') return winnerName ? `成交用户：${winnerName}` : '成交用户待同步';
  if (state === 'failed') return lotHasBid(lot) ? '付款超时，竞拍结束' : '本轮无人出价';
  if (state === 'cancelled') return lot.cancelReason ? `取消原因：${lot.cancelReason}` : '本件拍品已由主播取消';
  if (state === 'upcoming') return '等待主播开拍';
  return '状态同步中';
}

function primaryPriceLabel(lot: Lot, state: LotDisplayState) {
  if (state === 'finished') return '落槌价';
  if (state === 'pendingPayment') return '落槌价';
  if (state === 'failed') return lotHasBid(lot) ? '落槌价' : '起拍价';
  if (state === 'cancelled') return lotHasBid(lot) ? '取消前价格' : '起拍价';
  if (state === 'live') return lotHasBid(lot) ? '当前最高价' : '起拍价';
  return '起拍价';
}

function primaryPrice(lot: Lot, state: LotDisplayState) {
  if (state === 'finished' || state === 'pendingPayment' || (state === 'failed' && lot.stats?.bidCount) || (state === 'cancelled' && lotHasBid(lot))) {
    if (moneyNumber(lot.finalPrice) > 0) return lot.finalPrice;
    if (moneyNumber(lot.currentPrice) > 0) return lot.currentPrice;
    return lot.rule.startPrice;
  }
  if (state === 'live') return moneyNumber(lot.currentPrice) > 0 ? lot.currentPrice : lot.rule.startPrice;
  return lot.rule.startPrice;
}

function secondaryPriceMetric(lot: Lot, primaryLabel: string) {
  if (primaryLabel === '起拍价') {
    const currentPrice = moneyNumber(lot.currentPrice) > 0 ? lot.currentPrice : lot.rule.startPrice;
    return { label: '参考价', value: currentPrice };
  }
  return { label: '起拍价', value: lot.rule.startPrice };
}

export function CurrentLotCard({ lot, serverTimeUnixMs, serverTimeReceivedAtUnixMs, displayState }: CurrentLotCardProps) {
  const countdown = useServerCountdown(lot.endsAtUnixMs, serverTimeUnixMs, serverTimeReceivedAtUnixMs);
  const state = displayState || deriveLotDisplayState(lot);
  const copy = currentLotCopy(lot, state);
  const countdownValue = state === 'live' ? countdown.text : copy.value;
  const participantText = lotPersonText(lot, state);
  const primaryLabel = primaryPriceLabel(lot, state);
  const primaryPriceText = formatMoney(primaryPrice(lot, state));
  const secondaryMetric = secondaryPriceMetric(lot, primaryLabel);
  const secondaryPriceText = formatMoney(secondaryMetric.value);
  const serviceText = '正品保障';
  const participantCount = lot.stats?.participantCount || 0;
  const bidCount = lot.stats?.bidCount || 0;

  return (
    <section className={`lotCard ${copy.className}`}>
      <header className="lotCountdownBar">
        <span>{copy.label}</span>
        <b className={state === 'live' && countdown.danger ? 'dangerText pulse' : ''}>
          {countdownValue}
        </b>
      </header>
      <div className="lotHeroCard">
        <div className="lotMedia">
          {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <span>商品图待同步</span>}
        </div>
        <div className="lotInfo">
          <span className="statusPill">{copy.status}</span>
          <h1>{lot.title}</h1>
          <p>{participantText}</p>
        </div>
      </div>
      <div className="priceGrid">
        <div>
          <span>{primaryLabel}</span>
          <b className="scrollAmount" title={primaryPriceText}>{primaryPriceText}</b>
        </div>
        <div>
          <span>{secondaryMetric.label}</span>
          <b className="scrollAmount" title={secondaryPriceText}>{secondaryPriceText}</b>
        </div>
        <div>
          <span>商品保障</span>
          <b>{serviceText}</b>
        </div>
        <div>
          <span>围观 / 互动</span>
          <b>
            {participantCount} / {bidCount}
          </b>
        </div>
      </div>
    </section>
  );
}
