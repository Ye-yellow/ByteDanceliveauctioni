import { LOT_STATUS, type Lot, type OrderSummary } from '../../../shared/api/types';
import { formatMoney, moneyNumber } from '../../../shared/lib/money';
import { deriveLotDisplayState, lotHasBid, lotHasLockedResult, lotIsDisplayable, orderForLot, type LotDisplayState } from '../model/lotDisplayState';

type QueueLotView = {
  statusText: string;
  statusClass: string;
  priceLabel: string;
  priceValue: string;
  actionText: string;
  actionDisabled: boolean;
};

const statusTextByState: Record<LotDisplayState, string> = {
  upcoming: '即将开拍',
  live: '竞拍中',
  pendingPayment: '截拍中',
  finished: '已成交',
  failed: '竞拍未成交',
  syncing: '待同步',
};

const statusClassByState: Record<LotDisplayState, string> = {
  upcoming: 'isUpcoming',
  live: 'isLive',
  pendingPayment: 'isPendingPayment',
  finished: 'isFinished',
  failed: 'isFailed',
  syncing: 'isUpcoming',
};

function lotResultPrice(lot: Lot) {
  if (lotHasLockedResult(lot) && moneyNumber(lot.finalPrice) > 0) return lot.finalPrice;
  if (moneyNumber(lot.currentPrice) > 0) return lot.currentPrice;
  return lot.rule.startPrice;
}

function queueLotView(lot: Lot, order: OrderSummary | null, paymentKnownPaid: boolean, nowMs?: number): QueueLotView {
  const displayState = deriveLotDisplayState(lot, { order, paymentKnownPaid, nowMs });
  const hasBid = lotHasBid(lot);
  const resultPrice = lotResultPrice(lot);
  const priceLabel = displayState === 'finished'
    ? '成交价'
    : displayState === 'pendingPayment'
      ? '落锤价'
      : displayState === 'failed' && hasBid
        ? '落锤价'
        : displayState === 'live'
          ? '当前价'
          : hasBid
            ? '当前价'
        : '起拍价';
  const priceValue = displayState === 'finished' || displayState === 'pendingPayment' || hasBid
    ? formatMoney(resultPrice)
    : formatMoney(lot.rule.startPrice);
  return {
    statusText: statusTextByState[displayState],
    statusClass: statusClassByState[displayState],
    priceLabel,
    priceValue,
    actionText: displayState === 'live' ? '立即出价' : displayState === 'upcoming' || displayState === 'syncing' ? '去看看' : displayState === 'pendingPayment' ? '截拍中' : '已结束',
    actionDisabled: displayState === 'pendingPayment' || displayState === 'finished' || displayState === 'failed',
  };
}

function lotSortScore(lot: Lot, order: OrderSummary | null, paymentKnownPaid: boolean, nowMs?: number): number {
  const displayState = deriveLotDisplayState(lot, { order, paymentKnownPaid, nowMs });
  if (displayState === 'live') return 0;
  if (lot.status === LOT_STATUS.LIVE || lot.status === LOT_STATUS.EXTENDED) return 0;
  if (displayState === 'upcoming') {
    if (lot.status === LOT_STATUS.QUEUED) return 2;
    return 3;
  }
  if (displayState === 'pendingPayment') return 4;
  if (displayState === 'finished') return 5;
  if (displayState === 'failed') return 6;
  return 6;
}

export function LotQueueList({
  lots,
  currentLotId,
  selectedLotId,
  loading,
  error,
  onRefresh,
  onSelectLot,
  onPrimaryAction,
  orders = [],
  paidLotIds = {},
  nowMs,
}: {
  lots: Lot[];
  currentLotId?: string;
  selectedLotId?: string;
  loading: boolean;
  error: string;
  onRefresh: () => void;
  onSelectLot?: (lot: Lot) => void;
  onPrimaryAction?: (lot: Lot) => void;
  orders?: OrderSummary[];
  paidLotIds?: Record<string, boolean>;
  nowMs?: number;
}) {
  const visibleLots = lots.filter(lotIsDisplayable);
  const sortedLots = [...visibleLots].sort((a, b) => {
    const scoreDiff = lotSortScore(a, orderForLot(orders, a), Boolean(paidLotIds[a.id]), nowMs) - lotSortScore(b, orderForLot(orders, b), Boolean(paidLotIds[b.id]), nowMs);
    if (scoreDiff) return scoreDiff;
    return (a.queuePosition || 9999) - (b.queuePosition || 9999);
  });

  return (
    <section className="queuePanel">
      <header className="drawerSectionHeader">
        <div>
          <b>本场拍品</b>
          <span>{visibleLots.length ? `主播已上架 ${visibleLots.length} 件` : '等待主播上架'}</span>
        </div>
        <button type="button" onClick={onRefresh} disabled={loading}>
          {loading ? '刷新中' : '刷新'}
        </button>
      </header>

      {error ? <p className="bidError" role="alert">{error}</p> : null}
      {!loading && !error && sortedLots.length === 0 ? <section className="drawerEmpty">暂无拍品，等待主播从 PC 端上架</section> : null}

      <div className="queueList">
        {sortedLots.map((lot, index) => {
          const view = queueLotView(lot, orderForLot(orders, lot), Boolean(paidLotIds[lot.id]), nowMs);
          const active = lot.id === currentLotId;
          const selected = lot.id === selectedLotId;
          return (
            <article
              className={`queueLot ${view.statusClass} ${active ? 'active' : ''} ${selected ? 'selected' : ''}`}
              key={lot.id}
              role="button"
              tabIndex={0}
              onClick={() => onSelectLot?.(lot)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault();
                  onSelectLot?.(lot);
                }
              }}
            >
              <div className="queueLotMedia">
                <span>{lot.queuePosition || index + 1}</span>
                {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <div className="queueLotFallback">{index + 1}</div>}
              </div>
              <div>
                <span className="queueLotStatus">{view.statusText}</span>
                <h3>{lot.title || '未命名拍品'}</h3>
                <p><em>{view.priceLabel}</em><span className="scrollAmount" title={view.priceValue}>{view.priceValue}</span></p>
              </div>
              <aside>
                <button
                  type="button"
                  disabled={view.actionDisabled}
                  onClick={(event) => {
                    event.stopPropagation();
                    onPrimaryAction?.(lot);
                  }}
                >
                  {view.actionText}
                </button>
              </aside>
            </article>
          );
        })}
      </div>
    </section>
  );
}
