import { BuyerActivityView } from '../features/order-history/components/BuyerActivityView';
import { useBuyerActivity } from '../features/order-history/hooks/useBuyerActivity';
import { useAuthSession } from '../shared/auth/useAuthSession';

export function HistoryPage() {
  const { user, authMode, ensureBuyerSession } = useAuthSession();
  const params = new URLSearchParams(location.search);
  const roomId = params.get('roomId') || 'room-jewel-01';
  const activity = useBuyerActivity(ensureBuyerSession);
  const total = activity.tab === 'orders' ? activity.ordersMeta.total : activity.bidsMeta.total;

  if (authMode === 'real' && !user) {
    return (
      <main className="mobileShell">
        <section className="emptyState">请先在直播间登录买家账号，再查看我的订单和竞拍记录。</section>
        <a className="bottomAction" href={`/m/room/${encodeURIComponent(roomId)}`}>
          返回直播间
        </a>
      </main>
    );
  }

  return (
    <BuyerActivityView
      roomId={roomId}
      tab={activity.tab}
      orders={activity.orders}
      bids={activity.bids}
      total={total}
      loading={activity.loading}
      error={activity.error}
      onTabChange={activity.switchTab}
      onRefresh={() => void activity.refresh()}
    />
  );
}
