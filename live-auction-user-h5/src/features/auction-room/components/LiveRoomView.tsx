import { LOT_STATUS } from '../../../shared/api/types';
import { LivePlayer } from '../../live/components/LivePlayer';
import { BidPanel } from '../../bid-panel/components/BidPanel';
import { MockPayModal } from '../../payment-flow/components/MockPayModal';
import { ResultModal } from '../../result-modal/components/ResultModal';
import type { LiveRoomController } from '../hooks/useLiveRoomController';
import { AuctionNoticeLayer } from './AuctionNoticeLayer';
import { BuyerAuthPanel } from './BuyerAuthPanel';
import { CurrentLotCard } from './CurrentLotCard';
import { RankingBoard } from './RankingBoard';
import { RecentBidFeed } from './RecentBidFeed';
import { RoomStatusBar } from './RoomStatusBar';

export function LiveRoomView({ controller }: { controller: LiveRoomController }) {
  const {
    roomId,
    room,
    loading,
    error,
    roomName,
    currentLot,
    ranking,
    meId,
    wsState,
    notices,
    bidError,
    isBidPending,
    accountRoleMessage,
    showBuyerAuth,
    buyerAuth,
    resultLot,
    visibleResultOrder,
    payOrder,
    actions,
  } = controller;

  return (
    <main className="mobileShell">
      <LivePlayer
        poster={currentLot?.imageUrl}
        anchorName={room.snapshot?.anchorName}
        onlineCount={room.snapshot?.onlineCount}
        wsState={wsState}
        roomName={roomName}
      />

      <RoomStatusBar roomId={roomId} source={room.eventState.source} />

      {wsState !== '已连接' ? <div className="connectionWarn">实时连接中断，正在重连并恢复快照</div> : null}
      {accountRoleMessage ? <section className="emptyState error">{accountRoleMessage}</section> : null}
      {showBuyerAuth ? <BuyerAuthPanel auth={buyerAuth} /> : null}

      {loading ? <section className="emptyState">正在进入直播间...</section> : null}
      {error ? <section className="emptyState error">{error}</section> : null}
      {!loading && !currentLot ? <section className="emptyState">当前暂无竞拍，等待主播开拍</section> : null}
      {currentLot ? (
        <CurrentLotCard
          lot={currentLot}
          serverTimeUnixMs={room.serverTimeUnixMs}
          serverTimeReceivedAtUnixMs={room.serverTimeReceivedAtUnixMs}
        />
      ) : null}

      <BidPanel lot={currentLot} loading={isBidPending} error={bidError} onBid={actions.submitBid} />

      {room.localOptimistic.pendingBid ? (
        <p className="pendingHint">
          出价已提交，等待后端确认，幂等键 {room.localOptimistic.pendingBid.idempotencyKey.slice(0, 18)}...
        </p>
      ) : null}

      <RankingBoard ranking={ranking} meId={meId} />
      <RecentBidFeed bids={room.recentBids} />
      <AuctionNoticeLayer notices={notices} />

      {currentLot?.status === LOT_STATUS.CANCELLED ? <div className="cancelBanner">本场竞拍已异常取消</div> : null}

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
