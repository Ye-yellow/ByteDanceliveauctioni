export type Money = { amount: number | string; currency: string };
export type ReplyResult = { code: number; message: string; traceId?: string; trace_id?: string };

export const RESULT_CODE_OK = 0;
export const RESULT_CODE_UNAUTHENTICATED = 401001;
export const RESULT_CODE_PERMISSION_DENIED = 403001;
export const RESULT_CODE_LOT_VERSION_CONFLICT = 409001;

export type LotStatus = 'LOT_STATUS_UNSPECIFIED' | 'LOT_STATUS_DRAFT' | 'LOT_STATUS_LIVE' | 'LOT_STATUS_SETTLED' | 'LOT_STATUS_CANCELLED';
export type TrustCardType = 'TRUST_CARD_TYPE_UNSPECIFIED' | 'TRUST_CARD_TYPE_CERTIFICATE' | 'TRUST_CARD_TYPE_FLAW' | 'TRUST_CARD_TYPE_DETAIL' | 'TRUST_CARD_TYPE_SERVICE' | 'TRUST_CARD_TYPE_PRICE_REF';
export type PlaybookStage = 'PLAYBOOK_STAGE_UNSPECIFIED' | 'PLAYBOOK_STAGE_WARM_UP' | 'PLAYBOOK_STAGE_TRUST_BLOCKED' | 'PLAYBOOK_STAGE_BIDDING_ACTIVE' | 'PLAYBOOK_STAGE_DUEL_READY' | 'PLAYBOOK_STAGE_DUEL_MODE' | 'PLAYBOOK_STAGE_SETTLE_READY';
export type EventType = 'AUCTION_EVENT_TYPE_UNSPECIFIED' | 'AUCTION_EVENT_TYPE_ROOM_SNAPSHOT' | 'AUCTION_EVENT_TYPE_LOT_CREATED' | 'AUCTION_EVENT_TYPE_LOT_STARTED' | 'AUCTION_EVENT_TYPE_LOT_UPDATED' | 'AUCTION_EVENT_TYPE_BID_ACCEPTED' | 'AUCTION_EVENT_TYPE_BID_REJECTED' | 'AUCTION_EVENT_TYPE_RANKING_UPDATED' | 'AUCTION_EVENT_TYPE_TRUST_REVEALED' | 'AUCTION_EVENT_TYPE_DUEL_STARTED' | 'AUCTION_EVENT_TYPE_DUEL_ENDED' | 'AUCTION_EVENT_TYPE_LOT_SETTLED' | 'AUCTION_EVENT_TYPE_LOT_CANCELLED';

export type UserRole = 'USER_ROLE_UNSPECIFIED' | 'USER_ROLE_ANCHOR' | 'USER_ROLE_OPERATOR' | 'USER_ROLE_ADMIN' | (string & {});
export type User = { id: string; username: string; nickname: string; role: UserRole; createdAtUnixMs: number | string; updatedAtUnixMs: number | string };
export type AuthTokens = { accessToken: string; refreshToken: string; accessExpiresAtUnixMs: number | string; refreshExpiresAtUnixMs: number | string };

export type BidRule = {
  startPrice: Money;
  minIncrement: Money;
  capPrice?: Money;
  durationSeconds: number;
  antiSnipeWindowSeconds: number;
  antiSnipeExtendSeconds: number;
  maxExtendCount: number;
};
export type TrustRevealCard = { id: string; lotId: string; type: TrustCardType; title: string; content: string; imageUrl?: string; revealed: boolean; revealedAtUnixMs: number | string };
export type Bid = { id: string; lotId: string; userId: string; nickname: string; amount: Money; createdAtUnixMs: number | string };
export type RankingItem = { rank: number; userId: string; nickname: string; amount: Money; bidAtUnixMs: number | string };
export type DuelState = { active: boolean; lotId: string; userAId: string; userANickname: string; userBId: string; userBNickname: string; startedAtUnixMs: number | string; endsAtUnixMs: number | string; extendCount: number; maxExtendCount: number };
export type Lot = { id: string; roomId: string; title: string; description: string; imageUrl: string; status: LotStatus; rule: BidRule; currentPrice: Money; leadingUserId: string; leadingNickname: string; startedAtUnixMs: number | string; endsAtUnixMs: number | string; settledAtUnixMs: number | string; cancelledAtUnixMs?: number | string; winnerUserId: string; winnerNickname: string; finalPrice: Money; version: number | string; trustCards: TrustRevealCard[]; duelState: DuelState; playbookStage: PlaybookStage; cancelReason?: string };
export type RoomSnapshot = { roomId: string; currentLot?: Lot; ranking: RankingItem[]; recentBids: Bid[]; playbookStage: PlaybookStage; serverTimeUnixMs: number | string };
export type AuctionEvent = { id: string; type: EventType; roomId: string; lotId: string; occurredAtUnixMs: number | string; lot?: Lot; bid?: Bid; ranking?: RankingItem[]; trustCard?: TrustRevealCard; duelState?: DuelState; snapshot?: RoomSnapshot; reason?: string };
export type CreateLotRequest = { roomId: string; title: string; description: string; imageUrl: string; rule: BidRule; trustCards: Omit<TrustRevealCard, 'lotId' | 'revealed' | 'revealedAtUnixMs'>[] };
export type PlaceBidRequest = { lotId?: string; amount: Money; clientKnownVersion?: number | string; idempotencyKey?: string };
export type CancelLotRequest = { lotId?: string; reason: string };

export type CreateLotReply = { lot?: Lot; result?: ReplyResult };
export type GetLotReply = { lot?: Lot; result?: ReplyResult };
export type ListLotsReply = { lots?: Lot[]; nextPageToken?: string; result?: ReplyResult };
export type StartLotReply = { lot?: Lot; event?: AuctionEvent; result?: ReplyResult };
export type PlaceBidReply = { accepted: boolean; lot?: Lot; bid?: Bid; ranking?: RankingItem[]; event?: AuctionEvent; rejectReason?: string; result?: ReplyResult };
export type RevealTrustCardReply = { lot?: Lot; trustCard?: TrustRevealCard; event?: AuctionEvent; result?: ReplyResult };
export type StartDuelReply = { lot?: Lot; duelState?: DuelState; event?: AuctionEvent; result?: ReplyResult };
export type SettleLotReply = { lot?: Lot; event?: AuctionEvent; result?: ReplyResult };
export type CancelLotReply = { lot?: Lot; event?: AuctionEvent; result?: ReplyResult };
export type GetRoomSnapshotReply = { snapshot?: RoomSnapshot; result?: ReplyResult };

export type UploadedAsset = { id: string; imageUrl: string; bucket: string; objectKey: string; mimeType: string; sizeBytes: number | string; status?: string; expiresAtUnixMs?: number | string };
export type UploadImageReply = {
  code?: number;
  message?: string;
  requestId?: string;
  serverTimeUnixMs?: number | string;
  data?: { asset?: UploadedAsset };
  asset?: UploadedAsset;
  result?: ReplyResult;
};

export type LoginReply = { user?: User; tokens?: AuthTokens; result?: ReplyResult };
export type RefreshTokenReply = { tokens?: AuthTokens; result?: ReplyResult };
export type LogoutReply = { result?: ReplyResult };
export type GetMeReply = { user?: User; result?: ReplyResult };
