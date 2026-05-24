import { LOT_STATUS, type OrderSummary } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';
import { formatEventTime } from '../../../shared/lib/time';
import { BidPanel } from '../../bid-panel/components/BidPanel';
import type { AuctionPanelTab, LiveRoomController } from '../hooks/useLiveRoomController';
import { BuyerAuthPanel } from './BuyerAuthPanel';
import { CurrentLotCard } from './CurrentLotCard';
import { RankingBoard } from './RankingBoard';
import { RecentBidFeed } from './RecentBidFeed';

function tabLabel(tab: AuctionPanelTab): string {
  if (tab === 'current') return '正在竞拍';
  return '我的记录';
}

const DRAWER_TABS: AuctionPanelTab[] = ['current', 'mine'];

function DrawerTabIcon({ tab }: { tab: AuctionPanelTab }) {
  if (tab === 'mine') {
    return (
      <svg className="drawerTabIcon" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M7 4.8h10a2 2 0 0 1 2 2v12.4l-3-1.5-3 1.5-3-1.5-3 1.5V6.8a2 2 0 0 1 2-2Z" />
        <path d="M10 9h6" />
        <path d="M10 13h5" />
      </svg>
    );
  }

  return (
    <svg className="drawerTabIcon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M4.5 11.4 11.4 4.5H19v7.6L12.1 19 4.5 11.4Z" />
      <path d="M15.8 8.2h.01" />
      <path d="M9.8 11.2 12.8 14.2" />
    </svg>
  );
}

function MyPanel({
  controller,
}: {
  controller: LiveRoomController;
}) {
  const { roomId, room, showBuyerAuth, buyerAuth, actions } = controller;
  const orders = room.orders.slice(0, 3);

  if (showBuyerAuth) {
    return (
      <section className="drawerAuthBlock">
        <BuyerAuthPanel auth={buyerAuth} />
      </section>
    );
  }

  return (
    <section className="minePanel">
      <header className="drawerSectionHeader">
        <div>
          <b>我的竞拍</b>
          <span>订单和出价记录来自当前账号</span>
        </div>
        <button type="button" onClick={() => void actions.refreshOrders().catch(() => undefined)}>
          刷新
        </button>
      </header>
      {orders.length === 0 ? <section className="drawerEmpty">暂无订单，成交后会同步到这里</section> : null}
      <div className="miniOrderList">
        {orders.map((order) => <MiniOrder order={order} key={order.id} />)}
      </div>
      <a className="historyLink" href={`/m/history?roomId=${encodeURIComponent(roomId)}&from=room`}>
        查看全部订单和竞拍记录
      </a>
    </section>
  );
}

function MiniOrder({ order }: { order: OrderSummary }) {
  return (
    <article className="miniOrder">
      {order.lotImageUrl ? <img src={order.lotImageUrl} alt={order.lotTitle || order.id} /> : <div className="miniOrderFallback">单</div>}
      <div>
        <b>{order.lotTitle || order.lotId || '成交拍品'}</b>
        <span>{formatEventTime(order.createdAtUnixMs)}</span>
      </div>
      <aside>
        <strong>{formatMoney(order.amount)}</strong>
        <small>{order.paymentStatus || order.status || '待同步'}</small>
      </aside>
    </article>
  );
}

function CurrentAuctionPanel({ controller }: { controller: LiveRoomController }) {
  const {
    room,
    loading,
    error,
    currentLot,
    ranking,
    meId,
    bidError,
    isBidPending,
    accountRoleMessage,
    showBuyerAuth,
    buyerAuth,
    actions,
  } = controller;
  const leadingBidDisabledReason = currentLot?.leadingUserId && currentLot.leadingUserId === meId
    ? '你已领先，等待其他买家出价后再加价'
    : '';

  return (
    <section className="currentAuctionPanel">
      {accountRoleMessage ? <section className="emptyState error">{accountRoleMessage}</section> : null}
      {showBuyerAuth ? <BuyerAuthPanel auth={buyerAuth} /> : null}
      {loading ? <section className="drawerEmpty">正在进入直播间...</section> : null}
      {error ? <section className="emptyState error">{error}</section> : null}
      {!loading && !currentLot ? <section className="drawerEmpty">当前暂无竞拍，等待主播开拍</section> : null}
      {currentLot ? (
        <CurrentLotCard
          lot={currentLot}
          serverTimeUnixMs={room.serverTimeUnixMs}
          serverTimeReceivedAtUnixMs={room.serverTimeReceivedAtUnixMs}
        />
      ) : null}

      {currentLot ? (
        <BidPanel
          lot={currentLot}
          loading={isBidPending}
          error={bidError}
          disabledReason={leadingBidDisabledReason}
          onBid={actions.submitBid}
        />
      ) : null}

      {room.localOptimistic.pendingBid ? (
        <p className="pendingHint">
          出价已提交，等待后端确认，幂等键 {room.localOptimistic.pendingBid.idempotencyKey.slice(0, 18)}...
        </p>
      ) : null}

      <RankingBoard ranking={ranking} meId={meId} />
      <RecentBidFeed bids={room.recentBids} meId={meId} />
      {currentLot?.status === LOT_STATUS.CANCELLED ? <div className="cancelBanner">本场竞拍已异常取消</div> : null}
    </section>
  );
}

export function AuctionDrawer({ controller }: { controller: LiveRoomController }) {
  const { auctionPanel, currentLot, room, actions } = controller;
  if (!auctionPanel.open) return null;
  const activeTab = auctionPanel.tab === 'mine' ? 'mine' : 'current';

  const selectTab = (tab: AuctionPanelTab) => {
    actions.setAuctionPanelTab(tab);
    if (tab === 'mine') void actions.refreshOrders().catch(() => undefined);
  };

  return (
    <div className="auctionDrawerMask" onClick={actions.closeAuctionPanel}>
      <section className="auctionDrawer" role="dialog" aria-modal="true" aria-label="直播竞拍" onClick={(event) => event.stopPropagation()}>
        <header className="auctionDrawerHeader">
          <div>
            <span>直播竞拍</span>
            <h2>{currentLot?.title || '本场拍品'}</h2>
          </div>
          <button type="button" className="drawerClose" onClick={actions.closeAuctionPanel} aria-label="关闭竞拍面板">
            ×
          </button>
        </header>

        <nav className="drawerTabs">
          {DRAWER_TABS.map((tab) => (
            <button type="button" className={activeTab === tab ? 'active' : ''} onClick={() => selectTab(tab)} key={tab}>
              <DrawerTabIcon tab={tab} />
              <b>{tabLabel(tab)}</b>
            </button>
          ))}
        </nav>

        <div className="drawerContent">
          {activeTab === 'current' ? <CurrentAuctionPanel controller={controller} /> : null}
          {activeTab === 'mine' ? <MyPanel controller={controller} /> : null}
        </div>

        <footer className="drawerSyncState">
          {room.eventState.source === 'websocket' ? '实时事件同步中' : room.eventState.source === 'local' ? '本地操作待确认' : '快照同步中'}
        </footer>
      </section>
    </div>
  );
}
