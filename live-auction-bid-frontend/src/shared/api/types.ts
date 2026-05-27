export type Money = { amount: number | string; currency: string };
export type ReplyResult = { code: number; message: string; traceId?: string; trace_id?: string };

export const RESULT_CODE_OK = 0;
export const RESULT_CODE_LOGIN_REQUIRED = 401001;
export const RESULT_CODE_TOKEN_EXPIRED = 401002;
export const RESULT_CODE_TOKEN_INVALID = 401003;
export const RESULT_CODE_SESSION_EXPIRED = 401004;
export const RESULT_CODE_INVALID_CREDENTIALS = 401005;
export const RESULT_CODE_FORBIDDEN = 403001;
export const RESULT_CODE_LOT_VERSION_CONFLICT = 409001;
export const RESULT_CODE_INTERNAL_ERROR = 500000;

export type LotStatus = 'LOT_STATUS_UNSPECIFIED' | 'LOT_STATUS_DRAFT' | 'LOT_STATUS_READY' | 'LOT_STATUS_QUEUED' | 'LOT_STATUS_LIVE' | 'LOT_STATUS_EXTENDED' | 'LOT_STATUS_SETTLED' | 'LOT_STATUS_CANCELLED' | 'LOT_STATUS_FAILED';
export type LotQueueStatus = 'LOT_QUEUE_STATUS_UNSPECIFIED' | 'LOT_QUEUE_STATUS_NONE' | 'LOT_QUEUE_STATUS_QUEUED' | 'LOT_QUEUE_STATUS_NEXT';
export type TrustCardType = 'TRUST_CARD_TYPE_UNSPECIFIED' | 'TRUST_CARD_TYPE_CERTIFICATE' | 'TRUST_CARD_TYPE_FLAW' | 'TRUST_CARD_TYPE_DETAIL' | 'TRUST_CARD_TYPE_SERVICE' | 'TRUST_CARD_TYPE_PRICE_REF';
export type PlaybookStage = 'PLAYBOOK_STAGE_UNSPECIFIED' | 'PLAYBOOK_STAGE_WARM_UP' | 'PLAYBOOK_STAGE_TRUST_BLOCKED' | 'PLAYBOOK_STAGE_BIDDING_ACTIVE' | 'PLAYBOOK_STAGE_DUEL_READY' | 'PLAYBOOK_STAGE_DUEL_MODE' | 'PLAYBOOK_STAGE_SETTLE_READY';
export type EventType = 'AUCTION_EVENT_TYPE_UNSPECIFIED' | 'AUCTION_EVENT_TYPE_ROOM_SNAPSHOT' | 'AUCTION_EVENT_TYPE_LOT_CREATED' | 'AUCTION_EVENT_TYPE_LOT_STARTED' | 'AUCTION_EVENT_TYPE_LOT_UPDATED' | 'AUCTION_EVENT_TYPE_BID_ACCEPTED' | 'AUCTION_EVENT_TYPE_BID_REJECTED' | 'AUCTION_EVENT_TYPE_RANKING_UPDATED' | 'AUCTION_EVENT_TYPE_TRUST_REVEALED' | 'AUCTION_EVENT_TYPE_DUEL_STARTED' | 'AUCTION_EVENT_TYPE_DUEL_ENDED' | 'AUCTION_EVENT_TYPE_LOT_SETTLED' | 'AUCTION_EVENT_TYPE_LOT_CANCELLED' | 'AUCTION_EVENT_TYPE_LOT_QUEUED' | 'AUCTION_EVENT_TYPE_BID_OUTBID' | 'AUCTION_EVENT_TYPE_AUCTION_EXTENDED' | 'AUCTION_EVENT_TYPE_AUCTION_CLOSED' | 'AUCTION_EVENT_TYPE_ORDER_CREATED' | 'AUCTION_EVENT_TYPE_PAYMENT_SUCCESS';

export const USER_ROLE = {
  UNSPECIFIED: 'USER_ROLE_UNSPECIFIED',
  BUYER: 'USER_ROLE_BUYER',
  ANCHOR: 'USER_ROLE_ANCHOR',
  OPERATOR: 'USER_ROLE_OPERATOR',
  ADMIN: 'USER_ROLE_ADMIN',
} as const;

export type UserRole = (typeof USER_ROLE)[keyof typeof USER_ROLE];
export const ADMIN_ACCESS_ROLES: UserRole[] = [USER_ROLE.ANCHOR, USER_ROLE.OPERATOR, USER_ROLE.ADMIN];
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
export type LotStats = { participantCount: number; bidCount: number };
export type Lot = { id: string; roomId: string; title: string; description: string; imageUrl: string; status: LotStatus; queueStatus?: LotQueueStatus; queuePosition?: number; rule: BidRule; currentPrice: Money; leadingUserId: string; leadingNickname: string; startedAtUnixMs: number | string; endsAtUnixMs: number | string; settledAtUnixMs: number | string; cancelledAtUnixMs?: number | string; winnerUserId: string; winnerNickname: string; finalPrice: Money; version: number | string; trustCards: TrustRevealCard[]; duelState: DuelState; playbookStage: PlaybookStage; stats: LotStats; cancelReason?: string; galleryImageUrls?: string[]; category?: string; tags?: string[]; estimatePrice?: Money; stock?: number | string; afterSaleNotes?: string; depositAmount?: Money };
export type RoomSnapshot = { roomId: string; currentLot?: Lot; ranking: RankingItem[]; recentBids: Bid[]; playbookStage: PlaybookStage; serverTimeUnixMs: number | string };
export type RoomPresence = { roomId: string; totalConnections: number | string; viewerConnections: number | string; operatorConnections: number | string; serverTimeUnixMs: number | string };
export type AuctionEvent = { id: string; type: EventType; roomId: string; lotId: string; occurredAtUnixMs: number | string; lot?: Lot; bid?: Bid; ranking?: RankingItem[]; trustCard?: TrustRevealCard; duelState?: DuelState; snapshot?: RoomSnapshot; reason?: string; orderId?: string; paymentId?: string };
export type CreateLotRequest = { roomId: string; title: string; description: string; imageUrl: string; rule: BidRule; trustCards: Omit<TrustRevealCard, 'lotId' | 'revealed' | 'revealedAtUnixMs'>[]; galleryImageUrls?: string[]; category?: string; tags?: string[]; estimatePrice?: Money; stock?: number; afterSaleNotes?: string; depositAmount?: Money };
export type PatchLotDraftRequest = Partial<CreateLotRequest> & { lotId: string };
export type PlaceBidRequest = { lotId?: string; amount: Money; clientKnownVersion?: number | string; idempotencyKey?: string };
export type CancelLotRequest = { lotId?: string; reason: string };

export type CreateLotReply = { lot?: Lot; result?: ReplyResult };
export type PatchLotDraftReply = { lot?: Lot; result?: ReplyResult };
export type QueueLotReply = { lot?: Lot; queuePosition?: number; event?: AuctionEvent; result?: ReplyResult };
export type GetLotReply = { lot?: Lot; result?: ReplyResult };
export type ListLotsReply = { lots?: Lot[]; nextPageToken?: string; result?: ReplyResult };
export type StartLotReply = { lot?: Lot; event?: AuctionEvent; result?: ReplyResult };
export type PlaceBidReply = { accepted: boolean; lot?: Lot; bid?: Bid; ranking?: RankingItem[]; event?: AuctionEvent; rejectReason?: string; result?: ReplyResult };
export type RevealTrustCardReply = { lot?: Lot; trustCard?: TrustRevealCard; event?: AuctionEvent; result?: ReplyResult };
export type StartDuelReply = { lot?: Lot; duelState?: DuelState; event?: AuctionEvent; result?: ReplyResult };
export type SettleLotReply = { lot?: Lot; event?: AuctionEvent; result?: ReplyResult };
export type CancelLotReply = { lot?: Lot; event?: AuctionEvent; result?: ReplyResult };
export type GetRoomSnapshotReply = { snapshot?: RoomSnapshot; result?: ReplyResult };
export type GetRoomPresenceReply = { presence?: RoomPresence; result?: ReplyResult };
export type ListRoomEventsReply = { events?: AuctionEvent[]; nextPageToken?: string; result?: ReplyResult };

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
