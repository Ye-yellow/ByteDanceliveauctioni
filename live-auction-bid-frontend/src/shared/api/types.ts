export type Money = { amount: number | string; currency: string };
export type ReplyResult = { code: number; message: string; traceId?: string; trace_id?: string };

export const RESULT_CODE_OK = 0;
export const RESULT_CODE_LOGIN_REQUIRED = 401001;
export const RESULT_CODE_TOKEN_EXPIRED = 401002;
export const RESULT_CODE_TOKEN_INVALID = 401003;
export const RESULT_CODE_SESSION_EXPIRED = 401004;
export const RESULT_CODE_INVALID_CREDENTIALS = 401005;
export const RESULT_CODE_FORBIDDEN = 403001;
export const RESULT_CODE_ACCOUNT_DISABLED = 403002;
export const RESULT_CODE_LOT_VERSION_CONFLICT = 409001;
export const RESULT_CODE_ROOM_ACTIVE_LOT_EXISTS = 409003;
export const RESULT_CODE_BID_TOO_LOW = 409101;
export const RESULT_CODE_BID_NOT_LIVE = 409102;
export const RESULT_CODE_BID_ENDED = 409103;
export const RESULT_CODE_BID_ALREADY_LEADING = 409104;
export const RESULT_CODE_BID_CURRENCY_MISMATCH = 409105;
export const RESULT_CODE_BID_VERSION_STALE = 409106;
export const RESULT_CODE_LOT_CANCELLED = 409107;
export const RESULT_CODE_PROJECTION_PENDING = 409108;
export const RESULT_CODE_INTERNAL_ERROR = 500000;

export type LotStatus = 'LOT_STATUS_UNSPECIFIED' | 'LOT_STATUS_DRAFT' | 'LOT_STATUS_READY' | 'LOT_STATUS_QUEUED' | 'LOT_STATUS_LIVE' | 'LOT_STATUS_EXTENDED' | 'LOT_STATUS_SETTLED' | 'LOT_STATUS_CANCELLED' | 'LOT_STATUS_FAILED';
export type LotQueueStatus = 'LOT_QUEUE_STATUS_UNSPECIFIED' | 'LOT_QUEUE_STATUS_NONE' | 'LOT_QUEUE_STATUS_QUEUED' | 'LOT_QUEUE_STATUS_NEXT';
export type TrustCardType = 'TRUST_CARD_TYPE_UNSPECIFIED' | 'TRUST_CARD_TYPE_CERTIFICATE' | 'TRUST_CARD_TYPE_FLAW' | 'TRUST_CARD_TYPE_DETAIL' | 'TRUST_CARD_TYPE_SERVICE' | 'TRUST_CARD_TYPE_PRICE_REF';
export type PlaybookStage = 'PLAYBOOK_STAGE_UNSPECIFIED' | 'PLAYBOOK_STAGE_WARM_UP' | 'PLAYBOOK_STAGE_TRUST_BLOCKED' | 'PLAYBOOK_STAGE_BIDDING_ACTIVE' | 'PLAYBOOK_STAGE_DUEL_READY' | 'PLAYBOOK_STAGE_DUEL_MODE' | 'PLAYBOOK_STAGE_SETTLE_READY';
export type EventType = 'AUCTION_EVENT_TYPE_UNSPECIFIED' | 'AUCTION_EVENT_TYPE_ROOM_SNAPSHOT' | 'AUCTION_EVENT_TYPE_LOT_CREATED' | 'AUCTION_EVENT_TYPE_LOT_STARTED' | 'AUCTION_EVENT_TYPE_LOT_UPDATED' | 'AUCTION_EVENT_TYPE_BID_ACCEPTED' | 'AUCTION_EVENT_TYPE_BID_REJECTED' | 'AUCTION_EVENT_TYPE_RANKING_UPDATED' | 'AUCTION_EVENT_TYPE_TRUST_REVEALED' | 'AUCTION_EVENT_TYPE_DUEL_STARTED' | 'AUCTION_EVENT_TYPE_DUEL_ENDED' | 'AUCTION_EVENT_TYPE_LOT_SETTLED' | 'AUCTION_EVENT_TYPE_LOT_CANCELLED' | 'AUCTION_EVENT_TYPE_LOT_QUEUED' | 'AUCTION_EVENT_TYPE_BID_OUTBID' | 'AUCTION_EVENT_TYPE_AUCTION_EXTENDED' | 'AUCTION_EVENT_TYPE_AUCTION_CLOSED' | 'AUCTION_EVENT_TYPE_ORDER_CREATED' | 'AUCTION_EVENT_TYPE_PAYMENT_SUCCESS';

export const ROLE_CODE = {
  MERCHANT_OWNER: 'merchant_owner',
  ANCHOR: 'anchor',
  OPERATOR: 'operator',
  BUYER: 'buyer',
} as const;

export type RoleCode = (typeof ROLE_CODE)[keyof typeof ROLE_CODE];

export const PERMISSION_CODE = {
  TEAM_USER_CREATE: 'team.user.create',
  TEAM_USER_LIST: 'team.user.list',
  TEAM_USER_UPDATE_ROLE: 'team.user.update_role',
  TEAM_USER_UPDATE_STATUS: 'team.user.update_status',
  TEAM_USER_RESET_PASSWORD: 'team.user.reset_password',
  LOT_CREATE: 'lot.create',
  LOT_UPDATE: 'lot.update',
  LOT_QUEUE: 'lot.queue',
  LOT_VIEW_ADMIN: 'lot.view_admin',
  AUCTION_CONTROL: 'auction.control',
  ORDER_MANAGE: 'order.manage',
  REALTIME_VIEW: 'realtime.view',
  UPLOAD_IMAGE: 'upload.image',
  BID_PLACE: 'bid.place',
  ORDER_PAY: 'order.pay',
  ORDER_VIEW_OWN: 'order.view_own',
} as const;

export type PermissionCode = (typeof PERMISSION_CODE)[keyof typeof PERMISSION_CODE];

export const USER_STATUS = {
  UNSPECIFIED: 'USER_STATUS_UNSPECIFIED',
  ACTIVE: 'USER_STATUS_ACTIVE',
  DISABLED: 'USER_STATUS_DISABLED',
} as const;

export type UserStatus = (typeof USER_STATUS)[keyof typeof USER_STATUS];
export const BACKOFFICE_ACCESS_PERMISSIONS: PermissionCode[] = [PERMISSION_CODE.LOT_VIEW_ADMIN, PERMISSION_CODE.AUCTION_CONTROL, PERMISSION_CODE.REALTIME_VIEW, PERMISSION_CODE.ORDER_MANAGE];

export function hasRoleCode(user: Pick<User, 'roleCodes'> | undefined | null, roleCode: RoleCode | string) {
  return Boolean(user?.roleCodes?.some((item) => item === roleCode));
}

export function hasPermission(user: Pick<User, 'permissionCodes'> | undefined | null, permissionCode: PermissionCode | string) {
  return Boolean(user?.permissionCodes?.some((item) => item === permissionCode));
}

export function canAccessBackoffice(user?: Pick<User, 'permissionCodes' | 'status'> | null) {
  return Boolean(user && user.status === USER_STATUS.ACTIVE && BACKOFFICE_ACCESS_PERMISSIONS.some((permission) => hasPermission(user, permission)));
}

export function isMerchantOwner(user?: Pick<User, 'roleCodes'> | null) {
  return hasRoleCode(user, ROLE_CODE.MERCHANT_OWNER);
}

export function isManagedTeamRole(roleCode?: RoleCode | string | null) {
  return roleCode === ROLE_CODE.ANCHOR || roleCode === ROLE_CODE.OPERATOR;
}

export type User = {
  id: string;
  username: string;
  nickname: string;
  roleCodes: RoleCode[];
  permissionCodes: PermissionCode[];
  mainAccountId: string;
  createdByUserId: string;
  status: UserStatus;
  createdAtUnixMs: number | string;
  updatedAtUnixMs: number | string;
};
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
export type Room = { id: string; mainAccountId: string; name: string; platform: string; platformRoomId?: string; status: 'ACTIVE' | 'DISABLED' | string; createdByUserId?: string; createdAtUnixMs: number | string; updatedAtUnixMs: number | string };
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
export type ListRoomsReply = { rooms?: Room[]; result?: ReplyResult };

export type AIRecommendedActionType = 'reveal_trust_card' | 'start_duel' | 'navigate' | 'copy_text' | string;
export type AIRecommendedAction = { type: AIRecommendedActionType; label: string; reason: string; enabled: boolean; targetId?: string };
export type AIChecklistItem = { label: string; status: string; reason: string };
export type AITrustCardSuggestion = { type: TrustCardType | string; title: string; content: string };
export type AIDraftSuggestions = { titleSuggestion?: string; descriptionSuggestion?: string; tags?: string[]; afterSaleNote?: string; trustCards?: AITrustCardSuggestion[] };
export type AISituationMetric = { label: string; value: string; tone?: string };
export type AIMerchantSituation = { summary: string; metrics: AISituationMetric[] };
export type AIMerchantAssistantRequest = { page: string; roomId?: string; lotId?: string; draft?: Record<string, unknown>; question?: string };
export type AIMerchantAssistantReply = { result?: ReplyResult; answer: string; situation?: AIMerchantSituation; talkTracks?: string[]; evidence?: string[]; checklist: AIChecklistItem[]; nextSteps: string[]; recommendedActions: AIRecommendedAction[]; draftSuggestions: AIDraftSuggestions; warnings: string[]; fallbackUsed: boolean };

export type UploadedAsset = { id: string; imageUrl: string; bucket: string; objectKey: string; mimeType: string; sizeBytes: number | string; status?: string; expiresAtUnixMs?: number | string };
export type UploadImageReply = {
  code?: number;
  message?: string;
  requestId?: string;
  serverTimeUnixMs?: number | string;
  data?: { asset?: UploadedAsset };
  result?: ReplyResult;
};

export type LoginReply = { user?: User; tokens?: AuthTokens; result?: ReplyResult };
export type RegisterMerchantReply = { user?: User; tokens?: AuthTokens; result?: ReplyResult };
export type ResetPasswordReply = { user?: User; result?: ReplyResult };
export type RefreshTokenReply = { tokens?: AuthTokens; result?: ReplyResult };
export type LogoutReply = { result?: ReplyResult };
export type GetMeReply = { user?: User; result?: ReplyResult };
