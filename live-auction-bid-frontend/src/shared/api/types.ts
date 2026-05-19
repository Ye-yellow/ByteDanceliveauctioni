export type Money = { amount: number | string; currency: string };
export type LotStatus = 'LOT_STATUS_DRAFT' | 'LOT_STATUS_LIVE' | 'LOT_STATUS_SETTLED' | 'LOT_STATUS_CANCELLED';
export type TrustCardType = 'TRUST_CARD_TYPE_CERTIFICATE' | 'TRUST_CARD_TYPE_FLAW' | 'TRUST_CARD_TYPE_DETAIL' | 'TRUST_CARD_TYPE_SERVICE' | 'TRUST_CARD_TYPE_PRICE_REF';
export type PlaybookStage = 'PLAYBOOK_STAGE_WARM_UP' | 'PLAYBOOK_STAGE_TRUST_BLOCKED' | 'PLAYBOOK_STAGE_BIDDING_ACTIVE' | 'PLAYBOOK_STAGE_DUEL_READY' | 'PLAYBOOK_STAGE_DUEL_MODE' | 'PLAYBOOK_STAGE_SETTLE_READY';
export type EventType = 'AUCTION_EVENT_TYPE_ROOM_SNAPSHOT' | 'AUCTION_EVENT_TYPE_LOT_CREATED' | 'AUCTION_EVENT_TYPE_LOT_STARTED' | 'AUCTION_EVENT_TYPE_LOT_UPDATED' | 'AUCTION_EVENT_TYPE_BID_ACCEPTED' | 'AUCTION_EVENT_TYPE_BID_REJECTED' | 'AUCTION_EVENT_TYPE_RANKING_UPDATED' | 'AUCTION_EVENT_TYPE_TRUST_REVEALED' | 'AUCTION_EVENT_TYPE_DUEL_STARTED' | 'AUCTION_EVENT_TYPE_DUEL_ENDED' | 'AUCTION_EVENT_TYPE_LOT_SETTLED';

export type BidRule = {
  startPrice: Money;
  minIncrement: Money;
  durationSeconds: number;
  antiSnipeWindowSeconds: number;
  antiSnipeExtendSeconds: number;
  maxExtendCount: number;
};
export type TrustRevealCard = { id: string; lotId: string; type: TrustCardType; title: string; content: string; imageUrl?: string; revealed: boolean; revealedAtUnixMs: number };
export type Bid = { id: string; lotId: string; userId: string; nickname: string; amount: Money; createdAtUnixMs: number };
export type RankingItem = { rank: number; userId: string; nickname: string; amount: Money; bidAtUnixMs: number };
export type DuelState = { active: boolean; lotId: string; userAId: string; userANickname: string; userBId: string; userBNickname: string; startedAtUnixMs: number; endsAtUnixMs: number; extendCount: number; maxExtendCount: number };
export type Lot = { id: string; roomId: string; title: string; description: string; imageUrl: string; status: LotStatus; rule: BidRule; currentPrice: Money; leadingUserId: string; leadingNickname: string; startedAtUnixMs: number; endsAtUnixMs: number; settledAtUnixMs: number; winnerUserId: string; winnerNickname: string; finalPrice: Money; version: number; trustCards: TrustRevealCard[]; duelState: DuelState; playbookStage: PlaybookStage };
export type RoomSnapshot = { roomId: string; currentLot?: Lot; ranking: RankingItem[]; recentBids: Bid[]; playbookStage: PlaybookStage; serverTimeUnixMs: number };
export type AuctionEvent = { id: string; type: EventType; roomId: string; lotId: string; occurredAtUnixMs: number; lot?: Lot; bid?: Bid; ranking?: RankingItem[]; trustCard?: TrustRevealCard; duelState?: DuelState; snapshot?: RoomSnapshot; reason?: string };
export type CreateLotRequest = { roomId: string; title: string; description: string; imageUrl: string; rule: BidRule; trustCards: Omit<TrustRevealCard, 'lotId' | 'revealed' | 'revealedAtUnixMs'>[] };
export type PlaceBidRequest = { lotId?: string; userId: string; nickname: string; amount: Money; clientKnownVersion?: number; idempotencyKey?: string };
