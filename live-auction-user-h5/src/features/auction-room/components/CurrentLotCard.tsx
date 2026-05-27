import type { Lot } from '../../../shared/api/types';
import { formatMoney, moneyNumber } from '../../../shared/lib/money';
import { useServerCountdown } from '../hooks/useServerCountdown';
import { deriveLotDisplayState, type LotDisplayState } from '../model/lotDisplayState';

type CurrentLotCardProps = {
  lot: Lot;
  serverTimeUnixMs?: number | string;
  serverTimeReceivedAtUnixMs?: number;
  displayState?: LotDisplayState;
};

const cardCopyByState: Record<LotDisplayState, { status: string; label: string; value: string; className: string }> = {
  upcoming: { status: '即将开拍', label: '等待主播开拍', value: '未开始', className: 'isWaiting' },
  live: { status: '竞拍中', label: '距竞拍结束仅剩', value: '', className: 'isOpen' },
  pendingPayment: { status: '截拍中', label: '竞拍已落锤', value: '截拍中', className: 'isClosed' },
  finished: { status: '已成交', label: '竞拍已成交', value: '已成交', className: 'isClosed' },
  failed: { status: '竞拍未成交', label: '竞拍未成交', value: '未成交', className: 'isClosed' },
  syncing: { status: '待同步', label: '等待状态同步', value: '同步中', className: 'isWaiting' },
};

function lotPersonText(lot: Lot, state: LotDisplayState) {
  const leadingName = lot.leadingNickname || lot.leadingUserId;
  const winnerName = lot.winnerNickname || lot.winnerUserId;
  if (state === 'live') return leadingName ? `${leadingName} 领先` : '暂无出价';
  if (state === 'pendingPayment') return winnerName ? `竞得者：${winnerName}` : '等待订单同步';
  if (state === 'finished') return winnerName ? `成交用户：${winnerName}` : '成交用户待同步';
  if (state === 'failed') return lot.stats?.bidCount ? '付款超时，竞拍未成交' : '无人出价，竞拍未成交';
  if (state === 'upcoming') return '等待主播开拍';
  return '状态同步中';
}

function primaryPriceLabel(lot: Lot, state: LotDisplayState) {
  if (state === 'finished') return '成交价';
  if (state === 'pendingPayment') return '落锤价';
  if (state === 'failed') return lot.stats?.bidCount ? '落锤价' : '起拍价';
  if (state === 'live') return '当前价';
  return '起拍价';
}

function primaryPrice(lot: Lot, state: LotDisplayState) {
  if (state === 'finished' || state === 'pendingPayment' || (state === 'failed' && lot.stats?.bidCount)) {
    if (moneyNumber(lot.finalPrice) > 0) return lot.finalPrice;
    if (moneyNumber(lot.currentPrice) > 0) return lot.currentPrice;
    return lot.rule.startPrice;
  }
  if (state === 'live') return moneyNumber(lot.currentPrice) > 0 ? lot.currentPrice : lot.rule.startPrice;
  return lot.rule.startPrice;
}

export function CurrentLotCard({ lot, serverTimeUnixMs, serverTimeReceivedAtUnixMs, displayState }: CurrentLotCardProps) {
  const countdown = useServerCountdown(lot.endsAtUnixMs, serverTimeUnixMs, serverTimeReceivedAtUnixMs);
  const state = displayState || deriveLotDisplayState(lot);
  const copy = cardCopyByState[state];
  const countdownValue = state === 'live' ? countdown.text : copy.value;
  const participantText = lotPersonText(lot, state);
  const primaryPriceText = formatMoney(primaryPrice(lot, state));
  const startPriceText = formatMoney(lot.rule.startPrice);
  const incrementText = formatMoney(lot.rule.minIncrement);
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
          {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <span>拍品图待同步</span>}
        </div>
        <div className="lotInfo">
          <span className="statusPill">{copy.status}</span>
          <h1>{lot.title}</h1>
          <p>{participantText}</p>
        </div>
      </div>
      <div className="priceGrid">
        <div>
          <span>{primaryPriceLabel(lot, state)}</span>
          <b className="scrollAmount" title={primaryPriceText}>{primaryPriceText}</b>
        </div>
        <div>
          <span>起拍价</span>
          <b className="scrollAmount" title={startPriceText}>{startPriceText}</b>
        </div>
        <div>
          <span>加价幅度</span>
          <b className="scrollAmount" title={incrementText}>{incrementText}</b>
        </div>
        <div>
          <span>参与 / 出价</span>
          <b>
            {participantCount} / {bidCount}
          </b>
        </div>
      </div>
      <small className="timeHint">{countdown.fallback ? '等待服务端时间校准' : '服务端时间已校准'}</small>
    </section>
  );
}
