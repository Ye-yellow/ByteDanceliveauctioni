import { AUCTION_EVENT_TYPE, LOT_STATUS, type AuctionEventType, type AuctionSocketEvent, type LotStatus } from '../../../shared/api/types';

export function isBiddableLotStatus(status?: LotStatus): boolean {
  return status === LOT_STATUS.LIVE || status === LOT_STATUS.EXTENDED;
}

export function isClosedLotStatus(status?: LotStatus): boolean {
  return status === LOT_STATUS.SETTLED || status === LOT_STATUS.SOLD;
}

export function isSettlementEventType(type: AuctionEventType): boolean {
  return (
    type === AUCTION_EVENT_TYPE.AUCTION_CLOSED ||
    type === AUCTION_EVENT_TYPE.LOT_SETTLED ||
    type === AUCTION_EVENT_TYPE.ORDER_CREATED ||
    type === AUCTION_EVENT_TYPE.PAYMENT_SUCCESS
  );
}

export function isPrivateRefreshEventType(type: AuctionEventType): boolean {
  return type === AUCTION_EVENT_TYPE.ORDER_CREATED || type === AUCTION_EVENT_TYPE.PAYMENT_SUCCESS;
}

export function lotIdFromPublicEvent(event: AuctionSocketEvent, fallbackLotId = ''): string {
  return event.lot?.id || event.lotId || fallbackLotId;
}
