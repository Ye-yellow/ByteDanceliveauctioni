import { AUCTION_EVENT_TYPE, type AuctionSocketEvent } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';

export function noticeForAuctionEvent(event: AuctionSocketEvent, currentUserId: string, previousLeaderId?: string): string {
  if (event.type === AUCTION_EVENT_TYPE.BID_ACCEPTED) {
    if (event.bid?.userId === currentUserId || event.lot?.leadingUserId === currentUserId) {
      return `出价已确认，当前价 ${formatMoney(event.bid?.amount || event.lot?.currentPrice || 0)}`;
    }
    return `${event.bid?.nickname || '有买家'} 出价 ${formatMoney(event.bid?.amount || event.lot?.currentPrice || 0)}`;
  }

  if (event.type === AUCTION_EVENT_TYPE.BID_OUTBID) {
    if (previousLeaderId === currentUserId && event.lot?.leadingUserId !== currentUserId) {
      return '你已被超越，可继续加价';
    }
    if (event.lot?.leadingUserId === currentUserId) return '你重新领先';
    return '排名发生变化';
  }

  if (event.type === AUCTION_EVENT_TYPE.BID_REJECTED) {
    return event.rejectReason ? `出价失败：${event.rejectReason}` : '出价失败，请调整金额后重试';
  }

  if (event.type === AUCTION_EVENT_TYPE.AUCTION_EXTENDED) return '最后时刻出价，倒计时已延长';
  if (event.type === AUCTION_EVENT_TYPE.AUCTION_CLOSED || event.type === AUCTION_EVENT_TYPE.LOT_SETTLED) {
    return '竞拍已结束，正在同步成交结果';
  }
  if (event.type === AUCTION_EVENT_TYPE.ORDER_CREATED) return '成交订单已生成，正在同步你的结果';
  if (event.type === AUCTION_EVENT_TYPE.PAYMENT_SUCCESS) return '支付成功，正在刷新订单';
  if (event.type === AUCTION_EVENT_TYPE.LOT_CANCELLED) return event.reason || '本场竞拍已取消';

  return '';
}
