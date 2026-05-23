import type { BidRecord, OrderSummary } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';
import { formatEventTime } from '../../../shared/lib/time';
import type { BuyerActivityTab } from '../hooks/useBuyerActivity';

type BuyerActivityViewProps = {
  roomId: string;
  tab: BuyerActivityTab;
  orders: OrderSummary[];
  bids: BidRecord[];
  total: number;
  loading: boolean;
  error: string;
  onTabChange: (tab: BuyerActivityTab) => void;
  onRefresh: () => void;
};

export function BuyerActivityView({
  roomId,
  tab,
  orders,
  bids,
  total,
  loading,
  error,
  onTabChange,
  onRefresh,
}: BuyerActivityViewProps) {
  return (
    <main className="mobileShell orderPage">
      <header className="orderPageHeader">
        <a href={`/m/room/${encodeURIComponent(roomId)}`}>返回直播间</a>
        <h1>我的记录</h1>
        <button type="button" onClick={onRefresh} disabled={loading}>
          {loading ? '刷新中' : '刷新'}
        </button>
      </header>

      <div className="activityTabs">
        <button type="button" className={tab === 'orders' ? 'active' : ''} onClick={() => onTabChange('orders')}>
          我的订单
        </button>
        <button type="button" className={tab === 'bids' ? 'active' : ''} onClick={() => onTabChange('bids')}>
          竞拍记录
        </button>
      </div>

      <p className="activityMeta">共 {total} 条，当前展示最新 20 条</p>
      {error ? <section className="emptyState error">{error}</section> : null}

      {tab === 'orders' ? <OrderList orders={orders} loading={loading} error={error} /> : <BidRecordList bids={bids} loading={loading} error={error} />}
    </main>
  );
}

function OrderList({ orders, loading, error }: { orders: OrderSummary[]; loading: boolean; error: string }) {
  if (!loading && !error && orders.length === 0) return <section className="emptyState">暂无成交订单</section>;

  return (
    <section className="orderList">
      {orders.map((order) => (
        <article className="orderCard" key={order.id}>
          {order.lotImageUrl ? (
            <img src={order.lotImageUrl} alt={order.lotTitle || order.id} />
          ) : (
            <div className="orderImageFallback">订单</div>
          )}
          <div>
            <h2>{order.lotTitle || order.lotId || '成交拍品'}</h2>
            <p>订单号 {order.id}</p>
            <p>{formatEventTime(order.createdAtUnixMs)}</p>
          </div>
          <aside>
            <b>{formatMoney(order.amount)}</b>
            <span>{order.paymentStatus || order.status}</span>
          </aside>
        </article>
      ))}
    </section>
  );
}

function BidRecordList({ bids, loading, error }: { bids: BidRecord[]; loading: boolean; error: string }) {
  if (!loading && !error && bids.length === 0) return <section className="emptyState">暂无竞拍记录</section>;

  return (
    <section className="orderList">
      {bids.map((bid) => (
        <article className="orderCard bidRecordCard" key={bid.id}>
          {bid.lotImageUrl ? (
            <img src={bid.lotImageUrl} alt={bid.lotTitle || bid.id} />
          ) : (
            <div className="orderImageFallback">出价</div>
          )}
          <div>
            <h2>{bid.lotTitle || bid.lotId || '竞拍拍品'}</h2>
            <p>{formatEventTime(bid.createdAtUnixMs)}</p>
            <p>{bid.auctionState || bid.lotStatus || '状态同步中'}</p>
          </div>
          <aside>
            <b>{formatMoney(bid.amount)}</b>
            <span className={bid.won ? 'success' : ''}>{bid.won ? '已中标' : '已出价'}</span>
          </aside>
        </article>
      ))}
    </section>
  );
}
