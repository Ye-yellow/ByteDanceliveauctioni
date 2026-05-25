import { useState } from 'react';
import { BuyerActivityView } from '../features/order-history/components/BuyerActivityView';
import { useBuyerActivity } from '../features/order-history/hooks/useBuyerActivity';
import { MockPayModal } from '../features/payment-flow/components/MockPayModal';
import type { OrderSummary } from '../shared/api/types';
import { useAuthSession } from '../shared/auth/useAuthSession';

export function HistoryPage() {
  const { user, authMode, ensureBuyerSession } = useAuthSession();
  const [payOrder, setPayOrder] = useState<OrderSummary | null>(null);
  const params = new URLSearchParams(location.search);
  const roomId = params.get('roomId') || 'room-jewel-01';
  const from = params.get('from');
  const roomHref = `/m/room/${encodeURIComponent(roomId)}`;
  const profileHref = `/m/profile?roomId=${encodeURIComponent(roomId)}`;
  const backLabel = '返回';
  const backHref = from === 'profile' ? profileHref : roomHref;
  const activity = useBuyerActivity(ensureBuyerSession);
  const total = activity.tab === 'orders' ? activity.ordersMeta.total : activity.bidsMeta.total;
  const handlePaid = async (order?: OrderSummary) => {
    if (order) {
      activity.updateOrder(order);
      setPayOrder(order);
    }
    await activity.refresh();
  };
  const handleBack = () => {
    if (from === 'room') {
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

  if (authMode === 'real' && !user) {
    return (
      <main className="mobileShell">
        <section className="emptyState">请先在直播间登录买家账号，再查看我的订单和竞拍记录。</section>
        {from === 'room' ? (
          <button type="button" className="bottomAction" onClick={handleBack}>{backLabel}</button>
        ) : (
          <a className="bottomAction" href={backHref}>{backLabel}</a>
        )}
      </main>
    );
  }

  return (
    <BuyerActivityView
      backHref={backHref}
      preferHistoryBack={from === 'room'}
      tab={activity.tab}
      orders={activity.orders}
      bids={activity.bids}
      total={total}
      loading={activity.loading}
      error={activity.error}
      onTabChange={activity.switchTab}
      onRefresh={() => void activity.refresh()}
      onPayOrder={setPayOrder}
      paymentModal={payOrder ? (
        <MockPayModal
          order={payOrder}
          onStartPayment={() => undefined}
          onPaid={handlePaid}
          onClose={() => setPayOrder(null)}
        />
      ) : null}
    />
  );
}
