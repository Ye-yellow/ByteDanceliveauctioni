import { LivePlayer } from '../../live/components/LivePlayer';
import { DepositPayModal } from '../../payment-flow/components/DepositPayModal';
import { MockPayModal } from '../../payment-flow/components/MockPayModal';
import { ResultModal } from '../../result-modal/components/ResultModal';
import { isBiddableLotStatus } from '../../../entities/auction/model/status';
import { DEFAULT_DEMO_ROOM_PROFILE, getDemoRoomProfile } from '../../../shared/config/demoRooms';
import { formatMoney, moneyNumber } from '../../../shared/lib/money';
import type { LiveRoomController } from '../hooks/useLiveRoomController';
import { AuctionDrawer } from './AuctionDrawer';
import { AuctionNoticeLayer } from './AuctionNoticeLayer';

function LiveRoomChrome({ controller }: { controller: LiveRoomController }) {
  const { roomId, roomName, room } = controller;
  const roomProfile = getDemoRoomProfile(roomId);
  const anchorName = roomProfile?.anchorName || room.snapshot?.anchorName || DEFAULT_DEMO_ROOM_PROFILE.anchorName;
  const likes = Math.max(room.snapshot?.onlineCount || 0, 93000);
  const closeRoom = () => {
    window.location.assign('/');
  };

  return (
    <section className="liveRoomChrome" aria-label="直播间信息">
      <div className="liveAnchorLine">
        <div className="anchorAvatar">{anchorName.slice(0, 1)}</div>
        <div className="anchorCopy">
          <strong>{anchorName}</strong>
          <span>{likes.toLocaleString('zh-CN')} 本场点赞</span>
        </div>
        <button type="button" className="closeLive" aria-label="关闭直播间" onClick={closeRoom}>×</button>
      </div>

      <div className="roomTitleBlock">
        <span>00后</span>
        <b>{roomName || '今日游戏 - 植物大战僵尸'}</b>
      </div>
    </section>
  );
}

function minimumBidAmount(controller: LiveRoomController): number {
  const lot = controller.currentLot;
  if (!lot) return 0;
  const current = moneyNumber(lot.currentPrice) || moneyNumber(lot.rule.startPrice);
  return current + moneyNumber(lot.rule.minIncrement);
}

function quickBidDisabledReason(controller: LiveRoomController): string {
  const { currentLot, meId, isBidPending } = controller;
  if (!currentLot) return '当前暂无竞拍';
  if (!isBiddableLotStatus(currentLot.status)) return '当前不可出价';
  if (isBidPending) return '出价确认中';
  if (meId && currentLot.leadingUserId === meId) return '你已领先';
  return '';
}

function LiveActionRail({ controller, onOpenAuction }: { controller: LiveRoomController; onOpenAuction: () => void }) {
  const minBidAmount = minimumBidAmount(controller);
  const disabledReason = quickBidDisabledReason(controller);
  const disabled = Boolean(disabledReason);
  const handleQuickBid = () => {
    if (controller.showBuyerAuth) {
      onOpenAuction();
      return;
    }
    if (!disabled && minBidAmount > 0) controller.actions.submitBid(minBidAmount);
  };

  return (
    <aside className="liveActionRail" aria-label="直播互动">
      <button type="button" aria-label="打开竞拍" onClick={onOpenAuction}><span>拍</span><b>竞拍</b></button>
      <button
        type="button"
        className="quickBidButton"
        aria-label={minBidAmount > 0 ? `快速加价 ${formatMoney(minBidAmount)}` : '快速加价'}
        title={disabledReason || (minBidAmount > 0 ? `快速加价 ${formatMoney(minBidAmount)}` : '快速加价')}
        disabled={!controller.showBuyerAuth && disabled}
        onClick={handleQuickBid}
      >
        <span>＋</span>
        <b>{controller.isBidPending ? '出价中' : '加价'}</b>
      </button>
    </aside>
  );
}

function LiveComposer() {
  return (
    <footer className="liveComposer" aria-label="直播互动输入">
      <button type="button" className="commentInput">说点什么...</button>
      <button type="button" aria-label="表情">☺</button>
    </footer>
  );
}

export function LiveRoomView({ controller }: { controller: LiveRoomController }) {
  const {
    roomId,
    room,
    error,
    roomName,
    currentLot,
    meId,
    wsState,
    notices,
    auctionPanel,
    resultLot,
    visibleResultOrder,
    payOrder,
    depositPrompt,
    actions,
  } = controller;
  const roomProfile = getDemoRoomProfile(roomId);
  const anchorName = roomProfile?.anchorName || room.snapshot?.anchorName || DEFAULT_DEMO_ROOM_PROFILE.anchorName;

  return (
    <main className={`mobileShell douyinShell ${auctionPanel.open ? 'drawerVisible' : ''}`}>
      <LivePlayer
        poster={currentLot?.imageUrl}
        anchorName={anchorName}
        onlineCount={room.snapshot?.onlineCount}
        wsState={wsState}
        roomName={roomName}
      />

      <LiveRoomChrome controller={controller} />
      <LiveActionRail
        controller={controller}
        onOpenAuction={() => actions.openAuctionPanel('current')}
      />
      {wsState !== '已连接' ? <div className="liveConnectionWarn">实时连接中断，正在恢复</div> : null}
      {error ? <div className="liveConnectionWarn error">{error}</div> : null}
      <AuctionDrawer controller={controller} />
      <AuctionNoticeLayer notices={notices} />
      <LiveComposer />

      {depositPrompt ? (
        <DepositPayModal
          lot={depositPrompt.lot}
          onConfirm={actions.confirmDepositPayment}
          onClose={actions.closeDepositPrompt}
        />
      ) : null}

      {resultLot ? (
        <ResultModal
          lot={resultLot}
          meId={meId}
          order={visibleResultOrder}
          onClose={actions.closeResult}
          onNext={actions.nextLot}
          onPay={actions.setPayOrder}
        />
      ) : null}

      {payOrder ? (
        <MockPayModal
          order={payOrder}
          onStartPayment={actions.markPaymentStarted}
          onPaid={actions.handlePaymentPaid}
          onClose={() => actions.setPayOrder(null)}
        />
      ) : null}
    </main>
  );
}
