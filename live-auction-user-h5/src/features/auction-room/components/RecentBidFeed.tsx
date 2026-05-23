import type { BidEvent } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';
import { formatEventTime } from '../../../shared/lib/time';

export function RecentBidFeed({ bids }: { bids: BidEvent[] }) {
  return (
    <section className="feedCard">
      <h2>最近出价</h2>
      {bids.length ? (
        bids.slice(0, 8).map((bid, index) => (
          <div key={bid.id || `${bid.userId}-${index}`} className={bid.accepted === false ? 'rejected' : 'accepted'}>
            <span>{bid.nickname || bid.userId}</span>
            <b>{formatMoney(bid.amount)}</b>
            <small>
              {bid.accepted === false ? bid.rejectReason || '出价失败' : '有效出价'} · {formatEventTime(bid.createdAtUnixMs)}
            </small>
          </div>
        ))
      ) : (
        <p className="emptyText">暂无出价，抢先参与竞拍</p>
      )}
    </section>
  );
}
