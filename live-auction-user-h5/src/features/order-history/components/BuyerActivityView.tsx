import { useRef, useState, type TouchEvent } from 'react';
import type { BidRecord, OrderSummary } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';
import { formatEventTime } from '../../../shared/lib/time';
import type { BuyerActivityTab } from '../hooks/useBuyerActivity';

type BuyerActivityViewProps = {
  backHref: string;
  preferHistoryBack?: boolean;
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
  backHref,
  preferHistoryBack,
  tab,
  orders,
  bids,
  total,
  loading,
  error,
  onTabChange,
  onRefresh,
}: BuyerActivityViewProps) {
  const touchStartY = useRef<number | null>(null);
  const [pullDistance, setPullDistance] = useState(0);
  const [pulling, setPulling] = useState(false);
  const canReleaseToRefresh = pullDistance >= 56;

  const handleBack = () => {
    if (preferHistoryBack) {
      try {
        const referrer = document.referrer ? new URL(document.referrer) : null;
        if (referrer?.origin === window.location.origin && referrer.pathname.startsWith('/m/room/')) {
          window.history.back();
          return;
        }
      } catch {
        // Fall through to explicit navigation.
      }
    }
    window.location.assign(backHref);
  };

  const handleTouchStart = (event: TouchEvent<HTMLElement>) => {
    const shell = event.currentTarget.closest('.mobileShell') as HTMLElement | null;
    if (loading || (shell && shell.scrollTop > 0)) {
      touchStartY.current = null;
      return;
    }
    touchStartY.current = event.touches[0]?.clientY ?? null;
  };

  const handleTouchMove = (event: TouchEvent<HTMLElement>) => {
    if (touchStartY.current === null) return;
    const deltaY = (event.touches[0]?.clientY ?? touchStartY.current) - touchStartY.current;
    if (deltaY <= 0) {
      setPulling(false);
      setPullDistance(0);
      return;
    }
    if (deltaY > 12) event.preventDefault();
    setPulling(true);
    setPullDistance(Math.min(80, deltaY * 0.45));
  };

  const handleTouchEnd = () => {
    if (pullDistance >= 56) void onRefresh();
    touchStartY.current = null;
    setPulling(false);
    setPullDistance(0);
  };

  return (
    <main className="mobileShell orderPage">
      <header className="orderPageHeader">
        <button type="button" onClick={handleBack}>返回</button>
        <h1>我的记录</h1>
        <span className="orderHeaderSpacer" aria-hidden="true" />
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

      <section
        className={`activityPullArea${pulling ? ' pulling' : ''}`}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
        onTouchCancel={handleTouchEnd}
      >
        <div className="pullRefreshHint" style={{ height: pulling ? Math.max(32, pullDistance) : 0 }}>
          {loading ? '刷新中' : canReleaseToRefresh ? '松开刷新' : '下拉刷新'}
        </div>
        {tab === 'orders' ? <OrderList orders={orders} loading={loading} error={error} /> : <BidRecordList bids={bids} loading={loading} error={error} />}
      </section>
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
