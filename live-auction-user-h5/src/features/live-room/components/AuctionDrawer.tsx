import { useEffect, useMemo, useState } from 'react';
import { orderStatusLabel } from '../../../entities/order/model/privacy';
import { LOT_STATUS, type Lot, type OrderSummary } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';
import { formatEventTime, getServerNowMs } from '../../../shared/lib/time';
import type { AuctionPanelTab, LiveRoomController } from '../hooks/useLiveRoomController';
import { BidPanel } from '../../auction-bid/components/BidPanel';
import { BuyerAuthPanel } from './BuyerAuthPanel';
import { CurrentLotCard } from './CurrentLotCard';
import { LotQueueList } from './LotQueueList';
import { deriveLotDisplayState, orderForLot } from '../model/lotDisplayState';

function tabLabel(tab: AuctionPanelTab): string {
  if (tab === 'current') return '商品橱窗';
  return '我的订单';
}

const DRAWER_TABS: AuctionPanelTab[] = ['current', 'mine'];
type SheetMode = 'detail' | 'bid';

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
          <b>我的订单</b>
          <span>订单和互动记录来自当前账号</span>
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
        查看全部订单记录
      </a>
    </section>
  );
}

function MiniOrder({ order }: { order: OrderSummary }) {
  return (
    <article className="miniOrder">
      {order.lotImageUrl ? <img src={order.lotImageUrl} alt={order.lotTitle || order.id} /> : <div className="miniOrderFallback">单</div>}
      <div>
        <b>{order.lotTitle || order.lotId || '成交商品'}</b>
        <span>{formatEventTime(order.createdAtUnixMs)}</span>
      </div>
      <aside>
        <strong className="scrollAmount" title={formatMoney(order.amount)}>{formatMoney(order.amount)}</strong>
        <small>{orderStatusLabel(order)}</small>
      </aside>
    </article>
  );
}

function CurrentAuctionPanel({
  controller,
  onBidSheetOpenChange,
  onCloseBidSheet,
}: {
  controller: LiveRoomController;
  onBidSheetOpenChange: (open: boolean) => void;
  onCloseBidSheet: () => void;
}) {
  const {
    room,
    auctionPanel,
    loading,
    error,
    currentLot,
    accountRoleMessage,
    bidAuthPanelOpen,
    buyerAuth,
    actions,
  } = controller;
  const [sheetLotId, setSheetLotId] = useState('');
  const [sheetMode, setSheetMode] = useState<SheetMode>('detail');
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
  const sheetCanBid = Boolean(sheetLot && sheetIsCurrent && sheetDisplayState === 'live');
  const lotDisplayState = (lot: Lot) => deriveLotDisplayState(lot, {
    order: orderForLot(room.orders, lot),
    paymentKnownPaid: Boolean(room.paidLotIds[lot.id]),
    nowMs: displayNowMs,
  });
  const canBidLot = (lot: Lot) => lotDisplayState(lot) === 'live' && currentLot?.id === lot.id;
  const openLotSheet = (lot: Lot) => {
    const nextMode: SheetMode = canBidLot(lot) ? 'bid' : 'detail';
    setSheetLotId(lot.id);
    setSheetMode(nextMode);
    onBidSheetOpenChange(nextMode === 'bid');
  };
  const handleQueuePrimaryAction = (lot: Lot) => {
    const displayState = lotDisplayState(lot);

    openLotSheet(lot);
    if (displayState === 'live' && currentLot?.id !== lot.id) actions.showNotice('请等待主播切到该拍品后再出价');
  };
  const sheetInBidMode = sheetMode === 'bid' && sheetCanBid;
  const closeSheet = () => {
    if (sheetInBidMode) {
      onCloseBidSheet();
      return;
    }
    setSheetLotId('');
    setSheetMode('detail');
    onBidSheetOpenChange(false);
  };

  useEffect(() => {
    if (currentLot?.status !== LOT_STATUS.CANCELLED) return undefined;
    const timer = window.setTimeout(() => {
      setSheetLotId('');
      setSheetMode('detail');
      onBidSheetOpenChange(false);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [currentLot?.id, currentLot?.status, onBidSheetOpenChange]);

  useEffect(() => {
    if (sheetMode === 'bid' && !sheetCanBid) onBidSheetOpenChange(false);
  }, [sheetCanBid, sheetMode, onBidSheetOpenChange]);

  return (
    <section className="currentAuctionPanel">
      {accountRoleMessage ? <section className="emptyState error">{accountRoleMessage}</section> : null}
      {loading ? <section className="drawerEmpty">正在进入直播间...</section> : null}
      {error ? <section className="emptyState error">{error}</section> : null}
      {!sheetInBidMode ? (
        <LotQueueList
          lots={lots}
          currentLotId={currentLot?.id}
          selectedLotId={sheetLot?.id || currentLot?.id}
          loading={auctionPanel.loading}
          error={auctionPanel.error}
          onRefresh={() => void actions.refreshRoomLots().catch(() => undefined)}
          onSelectLot={openLotSheet}
          onPrimaryAction={handleQueuePrimaryAction}
          orders={room.orders}
          paidLotIds={room.paidLotIds}
          nowMs={displayNowMs}
        />
      ) : null}

      {bidAuthPanelOpen ? (
        <section className={`liveBidSheet liveProductSheet liveAuthSheet${sheetInBidMode ? ' liveProductSheetStandalone' : ''}`} aria-label="登录后出价">
          <button type="button" className="bidSheetClose" onClick={actions.closeBuyerAuthPanel} aria-label="关闭登录窗口">×</button>
          <BuyerAuthPanel auth={buyerAuth} />
        </section>
      ) : sheetLot ? (
        <section className={`liveBidSheet liveProductSheet${sheetInBidMode ? ' liveProductSheetStandalone' : ''}`} aria-label={sheetInBidMode ? '出价面板' : '商品详情'}>
          <button
            type="button"
            className="bidSheetClose"
            onClick={closeSheet}
            aria-label={sheetInBidMode ? '关闭出价面板' : '收起商品详情'}
          >
            ×
          </button>
          <CurrentLotCard
            lot={sheetLot}
            serverTimeUnixMs={room.serverTimeUnixMs}
            serverTimeReceivedAtUnixMs={room.serverTimeReceivedAtUnixMs}
            displayState={sheetDisplayState}
          />
          {sheetCanBid ? (
            <BidPanel
              lot={sheetLot}
              loading={controller.isBidPending}
              disabledReason={accountRoleMessage || ''}
              onBid={actions.submitBid}
              onTip={actions.showNotice}
            />
          ) : null}
        </section>
      ) : null}

      {!loading && !currentLot && !lots.length ? <section className="drawerEmpty">当前暂无商品，等待主播上架</section> : null}

      {room.localOptimistic.pendingBid ? (
        <p className="pendingHint">
          订单状态同步中，请稍候...
        </p>
      ) : null}

      {currentLot?.status === LOT_STATUS.CANCELLED ? <div className="cancelBanner">{currentLot.cancelReason ? `本件拍品已由主播取消，原因：${currentLot.cancelReason}` : '本件拍品已由主播取消'}</div> : null}
    </section>
  );
}

export function AuctionDrawer({ controller }: { controller: LiveRoomController }) {
  const { auctionPanel, currentLot, actions } = controller;
  const [bidSheetOpen, setBidSheetOpen] = useState(false);

  useEffect(() => {
    if (!auctionPanel.open) setBidSheetOpen(false);
  }, [auctionPanel.open]);

  if (!auctionPanel.open) return null;
  const activeTab = auctionPanel.tab === 'mine' ? 'mine' : 'current';

  const closeAuctionPanel = () => {
    setBidSheetOpen(false);
    actions.closeAuctionPanel();
  };

  const selectTab = (tab: AuctionPanelTab) => {
    setBidSheetOpen(false);
    actions.setAuctionPanelTab(tab);
    if (tab === 'mine') void actions.refreshOrders().catch(() => undefined);
  };

  return (
    <div className="auctionDrawerMask" onClick={closeAuctionPanel}>
      <section className={`auctionDrawer${bidSheetOpen ? ' auctionDrawerBidSheet' : ''}`} role="dialog" aria-modal="true" aria-label="直播商品" onClick={(event) => event.stopPropagation()}>
        {!bidSheetOpen ? (
          <>
            <header className="auctionDrawerHeader">
              <div>
                <span>进主播橱窗 ›</span>
                <h2>{currentLot?.title || '本场商品'}</h2>
                <div className="auctionDrawerTrust">
                  <em>带货口碑 5.0高</em>
                  <em>安心购</em>
                  <em>真实宝</em>
                </div>
              </div>
              <button type="button" className="drawerClose" onClick={closeAuctionPanel} aria-label="关闭商品面板">
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
          </>
        ) : null}

        <div className="drawerContent">
          {activeTab === 'current' ? (
            <CurrentAuctionPanel
              controller={controller}
              onBidSheetOpenChange={setBidSheetOpen}
              onCloseBidSheet={closeAuctionPanel}
            />
          ) : null}
          {activeTab === 'mine' ? <MyPanel controller={controller} /> : null}
        </div>
      </section>
    </div>
  );
}
