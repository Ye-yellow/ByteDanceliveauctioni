import { useMemo, useState } from 'react';
import { isBiddableLotStatus } from '../../../entities/auction/model/status';
import { orderStatusLabel } from '../../../entities/order/model/privacy';
import { LOT_STATUS, type OrderSummary } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';
import { formatEventTime, getServerNowMs } from '../../../shared/lib/time';
import { BidPanel } from '../../bid-panel/components/BidPanel';
import type { AuctionPanelTab, LiveRoomController } from '../hooks/useLiveRoomController';
import { BuyerAuthPanel } from './BuyerAuthPanel';
import { CurrentLotCard } from './CurrentLotCard';
import { LotQueueList } from './LotQueueList';
import { deriveLotDisplayState, orderForLot } from '../model/lotDisplayState';
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
        <strong className="scrollAmount" title={formatMoney(order.amount)}>{formatMoney(order.amount)}</strong>
        <small>{orderStatusLabel(order)}</small>
      </aside>
    </article>
  );
}

function unavailableReasonForLot(
  lot: NonNullable<LiveRoomController['currentLot']>,
  currentLot: LiveRoomController['currentLot'],
  orders: OrderSummary[],
  paidLotIds: Record<string, boolean>,
  nowMs?: number,
): string {
  const displayState = deriveLotDisplayState(lot, {
    order: orderForLot(orders, lot),
    paymentKnownPaid: Boolean(paidLotIds[lot.id]),
    nowMs,
  });

  if (displayState === 'pendingPayment') return '这件拍品已落锤，正在等待竞得者付款';
  if (displayState === 'failed') return '这件拍品未成交，已结束';
  if (displayState === 'finished') return '这件拍品已结束';
  if (!currentLot || lot.id !== currentLot.id) return '这件还未开拍，先看看本场安排';
  if (!isBiddableLotStatus(lot.status)) return '当前拍品状态不可出价';
  return '';
}

function CurrentAuctionPanel({ controller }: { controller: LiveRoomController }) {
  const {
    room,
    auctionPanel,
    loading,
    error,
    currentLot,
    ranking,
    meId,
    isBidPending,
    accountRoleMessage,
    showBuyerAuth,
    buyerAuth,
    actions,
  } = controller;
  const [sheetLotId, setSheetLotId] = useState('');
  const displayNowMs = getServerNowMs(room.serverTimeUnixMs, room.serverTimeReceivedAtUnixMs);
  const lots = useMemo(() => {
    if (auctionPanel.lots.length) {
      if (currentLot && !auctionPanel.lots.some((lot) => lot.id === currentLot.id)) return [currentLot, ...auctionPanel.lots];
      if (currentLot) return auctionPanel.lots.map((lot) => lot.id === currentLot.id ? currentLot : lot);
      return auctionPanel.lots;
    }
    return currentLot ? [currentLot] : [];
  }, [auctionPanel.lots, currentLot]);
  const sheetLot = lots.find((lot) => lot.id === sheetLotId) || null;
  const sheetIsCurrent = Boolean(sheetLot && currentLot && sheetLot.id === currentLot.id);
  const sheetDisplayState = sheetLot ? deriveLotDisplayState(sheetLot, {
    order: orderForLot(room.orders, sheetLot),
    paymentKnownPaid: Boolean(room.paidLotIds[sheetLot.id]),
    nowMs: displayNowMs,
  }) : undefined;
  const leadingBidDisabledReason = sheetIsCurrent && currentLot?.leadingUserId && currentLot.leadingUserId === meId
    ? '当前您已是最高价'
    : '';
  const unavailableReason = sheetLot && (!sheetIsCurrent || !isBiddableLotStatus(sheetLot.status))
    ? unavailableReasonForLot(sheetLot, currentLot, room.orders, room.paidLotIds, displayNowMs)
    : '';
  const bidDisabledReason = leadingBidDisabledReason || unavailableReason;

  return (
    <section className="currentAuctionPanel">
      {accountRoleMessage ? <section className="emptyState error">{accountRoleMessage}</section> : null}
      {loading ? <section className="drawerEmpty">正在进入直播间...</section> : null}
      {error ? <section className="emptyState error">{error}</section> : null}
      <LotQueueList
        lots={lots}
        currentLotId={currentLot?.id}
        selectedLotId={sheetLot?.id || currentLot?.id}
        loading={auctionPanel.loading}
        error={auctionPanel.error}
        onRefresh={() => void actions.refreshRoomLots().catch(() => undefined)}
        onSelectLot={(lot) => setSheetLotId(lot.id)}
        onPrimaryAction={(lot) => setSheetLotId(lot.id)}
        orders={room.orders}
        paidLotIds={room.paidLotIds}
        nowMs={displayNowMs}
      />

      {sheetLot ? (
        <section className="liveBidSheet" aria-label="出价面板">
          <button type="button" className="bidSheetClose" onClick={() => setSheetLotId('')} aria-label="收起出价面板">×</button>
          <CurrentLotCard
            lot={sheetLot}
            serverTimeUnixMs={room.serverTimeUnixMs}
            serverTimeReceivedAtUnixMs={room.serverTimeReceivedAtUnixMs}
            displayState={sheetDisplayState}
          />
          {showBuyerAuth ? (
            <BuyerAuthPanel auth={buyerAuth} />
          ) : (
            <BidPanel
              lot={sheetLot}
              loading={isBidPending}
              disabledReason={bidDisabledReason}
              onBid={actions.submitBid}
              onTip={actions.showNotice}
            />
          )}
        </section>
      ) : null}

      {!loading && !currentLot && !lots.length ? <section className="drawerEmpty">当前暂无竞拍，等待主播开拍</section> : null}

      {currentLot ? (
        <section className="liveAuctionTelemetry">
          <RankingBoard ranking={ranking} meId={meId} />
          <RecentBidFeed bids={room.recentBids} meId={meId} />
        </section>
      ) : null}

      {room.localOptimistic.pendingBid ? (
        <p className="pendingHint">
          出价已提交，等待后端确认，幂等键 {room.localOptimistic.pendingBid.idempotencyKey.slice(0, 18)}...
        </p>
      ) : null}

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
