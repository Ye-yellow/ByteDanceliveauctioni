import { LOT_QUEUE_STATUS, LOT_STATUS, type Lot } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';

function lotStatusText(lot: Lot): string {
  if (lot.status === LOT_STATUS.LIVE || lot.status === LOT_STATUS.EXTENDED) return '正在竞拍';
  if (lot.queueStatus === LOT_QUEUE_STATUS.NEXT) return '下一件';
  if (lot.queueStatus === LOT_QUEUE_STATUS.QUEUED || lot.status === LOT_STATUS.QUEUED) return '队列中';
  if (lot.status === LOT_STATUS.READY || lot.status === LOT_STATUS.DRAFT) return '已上架';
  if (lot.status === LOT_STATUS.SETTLED || lot.status === LOT_STATUS.SOLD) return '已成交';
  if (lot.status === LOT_STATUS.CANCELLED) return '已取消';
  if (lot.status === LOT_STATUS.FAILED) return '已流拍';
  return '待同步';
}

function lotSortScore(lot: Lot): number {
  if (lot.status === LOT_STATUS.LIVE || lot.status === LOT_STATUS.EXTENDED) return 0;
  if (lot.queueStatus === LOT_QUEUE_STATUS.NEXT) return 1;
  if (lot.queueStatus === LOT_QUEUE_STATUS.QUEUED || lot.status === LOT_STATUS.QUEUED) return 2;
  if (lot.status === LOT_STATUS.READY || lot.status === LOT_STATUS.DRAFT) return 3;
  if (lot.status === LOT_STATUS.SETTLED || lot.status === LOT_STATUS.SOLD) return 4;
  if (lot.status === LOT_STATUS.CANCELLED || lot.status === LOT_STATUS.FAILED) return 5;
  return 6;
}

export function LotQueueList({
  lots,
  currentLotId,
  loading,
  error,
  onRefresh,
}: {
  lots: Lot[];
  currentLotId?: string;
  loading: boolean;
  error: string;
  onRefresh: () => void;
}) {
  const sortedLots = [...lots].sort((a, b) => {
    const scoreDiff = lotSortScore(a) - lotSortScore(b);
    if (scoreDiff) return scoreDiff;
    return (a.queuePosition || 9999) - (b.queuePosition || 9999);
  });

  return (
    <section className="queuePanel">
      <header className="drawerSectionHeader">
        <div>
          <b>本场拍品</b>
          <span>{lots.length ? `主播已上架 ${lots.length} 件` : '等待主播上架'}</span>
        </div>
        <button type="button" onClick={onRefresh} disabled={loading}>
          {loading ? '刷新中' : '刷新'}
        </button>
      </header>

      {error ? <p className="bidError" role="alert">{error}</p> : null}
      {!loading && !error && sortedLots.length === 0 ? <section className="drawerEmpty">暂无拍品，等待主播从 PC 端上架</section> : null}

      <div className="queueList">
        {sortedLots.map((lot, index) => (
          <article className={`queueLot ${lot.id === currentLotId ? 'active' : ''}`} key={lot.id}>
            {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <div className="queueLotFallback">{index + 1}</div>}
            <div>
              <span>{lot.queuePosition ? `#${lot.queuePosition}` : `#${index + 1}`}</span>
              <h3>{lot.title || '未命名拍品'}</h3>
              <p>{lot.description || '主播暂未填写介绍'}</p>
            </div>
            <aside>
              <b>{formatMoney(lot.currentPrice || lot.rule.startPrice)}</b>
              <small>{lotStatusText(lot)}</small>
            </aside>
          </article>
        ))}
      </div>
    </section>
  );
}
