import type {
  AuctionEvent,
  AuthTokens,
  Bid,
  BidRule,
  DuelState,
  EventType,
  Lot,
  LotQueueStatus,
  LotStatus,
  Money,
  PlaybookStage,
  RankingItem,
  RoomPresence,
  RoomSnapshot,
  TrustCardType,
  TrustRevealCard,
  UploadedAsset,
  User,
  UserRole,
} from './types';

type JsonRecord = Record<string, unknown>;

const lotStatusByNumber: Record<number, LotStatus> = {
  0: 'LOT_STATUS_UNSPECIFIED',
  1: 'LOT_STATUS_DRAFT',
  2: 'LOT_STATUS_LIVE',
  3: 'LOT_STATUS_SETTLED',
  4: 'LOT_STATUS_CANCELLED',
  5: 'LOT_STATUS_READY',
  6: 'LOT_STATUS_QUEUED',
  7: 'LOT_STATUS_EXTENDED',
  8: 'LOT_STATUS_FAILED',
};

const lotStatusValues = new Set<LotStatus>([
  'LOT_STATUS_UNSPECIFIED',
  'LOT_STATUS_DRAFT',
  'LOT_STATUS_READY',
  'LOT_STATUS_QUEUED',
  'LOT_STATUS_SCHEDULED',
  'LOT_STATUS_LIVE',
  'LOT_STATUS_EXTENDED',
  'LOT_STATUS_SETTLED',
  'LOT_STATUS_SOLD',
  'LOT_STATUS_CANCELLED',
  'LOT_STATUS_FAILED',
]);

const lotQueueStatusByNumber: Record<number, LotQueueStatus> = {
  0: 'LOT_QUEUE_STATUS_UNSPECIFIED',
  1: 'LOT_QUEUE_STATUS_NONE',
  2: 'LOT_QUEUE_STATUS_QUEUED',
  3: 'LOT_QUEUE_STATUS_NEXT',
};

const lotQueueStatusValues = new Set<LotQueueStatus>([
  'LOT_QUEUE_STATUS_UNSPECIFIED',
  'LOT_QUEUE_STATUS_NONE',
  'LOT_QUEUE_STATUS_QUEUED',
  'LOT_QUEUE_STATUS_NEXT',
]);

const trustCardTypeByNumber: Record<number, TrustCardType> = {
  0: 'TRUST_CARD_TYPE_UNSPECIFIED',
  1: 'TRUST_CARD_TYPE_CERTIFICATE',
  2: 'TRUST_CARD_TYPE_FLAW',
  3: 'TRUST_CARD_TYPE_DETAIL',
  4: 'TRUST_CARD_TYPE_SERVICE',
  5: 'TRUST_CARD_TYPE_PRICE_REF',
};

const trustCardTypeValues = new Set<TrustCardType>([
  'TRUST_CARD_TYPE_UNSPECIFIED',
  'TRUST_CARD_TYPE_CERTIFICATE',
  'TRUST_CARD_TYPE_FLAW',
  'TRUST_CARD_TYPE_DETAIL',
  'TRUST_CARD_TYPE_SERVICE',
  'TRUST_CARD_TYPE_PRICE_REF',
]);

const playbookStageByNumber: Record<number, PlaybookStage> = {
  0: 'PLAYBOOK_STAGE_UNSPECIFIED',
  1: 'PLAYBOOK_STAGE_WARM_UP',
  2: 'PLAYBOOK_STAGE_TRUST_BLOCKED',
  3: 'PLAYBOOK_STAGE_BIDDING_ACTIVE',
  4: 'PLAYBOOK_STAGE_DUEL_READY',
  5: 'PLAYBOOK_STAGE_DUEL_MODE',
  6: 'PLAYBOOK_STAGE_SETTLE_READY',
};

const playbookStageValues = new Set<PlaybookStage>([
  'PLAYBOOK_STAGE_UNSPECIFIED',
  'PLAYBOOK_STAGE_WARM_UP',
  'PLAYBOOK_STAGE_TRUST_BLOCKED',
  'PLAYBOOK_STAGE_BIDDING_ACTIVE',
  'PLAYBOOK_STAGE_DUEL_READY',
  'PLAYBOOK_STAGE_DUEL_MODE',
  'PLAYBOOK_STAGE_SETTLE_READY',
]);

const eventTypeByNumber: Record<number, EventType> = {
  0: 'AUCTION_EVENT_TYPE_UNSPECIFIED',
  1: 'AUCTION_EVENT_TYPE_ROOM_SNAPSHOT',
  2: 'AUCTION_EVENT_TYPE_LOT_CREATED',
  3: 'AUCTION_EVENT_TYPE_LOT_STARTED',
  4: 'AUCTION_EVENT_TYPE_LOT_UPDATED',
  5: 'AUCTION_EVENT_TYPE_BID_ACCEPTED',
  6: 'AUCTION_EVENT_TYPE_BID_REJECTED',
  7: 'AUCTION_EVENT_TYPE_RANKING_UPDATED',
  8: 'AUCTION_EVENT_TYPE_TRUST_REVEALED',
  9: 'AUCTION_EVENT_TYPE_DUEL_STARTED',
  10: 'AUCTION_EVENT_TYPE_DUEL_ENDED',
  11: 'AUCTION_EVENT_TYPE_LOT_SETTLED',
  12: 'AUCTION_EVENT_TYPE_LOT_CANCELLED',
  13: 'AUCTION_EVENT_TYPE_LOT_QUEUED',
  14: 'AUCTION_EVENT_TYPE_BID_OUTBID',
  15: 'AUCTION_EVENT_TYPE_AUCTION_EXTENDED',
  16: 'AUCTION_EVENT_TYPE_AUCTION_CLOSED',
  17: 'AUCTION_EVENT_TYPE_ORDER_CREATED',
  18: 'AUCTION_EVENT_TYPE_PAYMENT_SUCCESS',
};

const eventTypeAliases: Record<string, EventType> = {
  BID_ACCEPTED: 'AUCTION_EVENT_TYPE_BID_ACCEPTED',
  AUCTION_EXTENDED: 'AUCTION_EVENT_TYPE_AUCTION_EXTENDED',
  AUCTION_CLOSED: 'AUCTION_EVENT_TYPE_AUCTION_CLOSED',
  ORDER_CREATED: 'AUCTION_EVENT_TYPE_ORDER_CREATED',
  PAYMENT_SUCCESS: 'AUCTION_EVENT_TYPE_PAYMENT_SUCCESS',
};

const userRoleByNumber: Record<number, UserRole> = {
  0: 'USER_ROLE_UNSPECIFIED',
  1: 'USER_ROLE_BUYER',
  2: 'USER_ROLE_ANCHOR',
  3: 'USER_ROLE_OPERATOR',
  4: 'USER_ROLE_ADMIN',
};

const userRoleValues = new Set<UserRole>([
  'USER_ROLE_UNSPECIFIED',
  'USER_ROLE_BUYER',
  'USER_ROLE_ANCHOR',
  'USER_ROLE_OPERATOR',
  'USER_ROLE_ADMIN',
]);

function asRecord(input: unknown, field: string): JsonRecord {
  if (!input || typeof input !== 'object' || Array.isArray(input)) throw new Error(`response missing ${field}`);
  return input as JsonRecord;
}

function field(raw: JsonRecord, camel: string, snake = camel) {
  return raw[camel] ?? raw[snake];
}

function requiredField<T>(value: T | undefined | null, name: string): T {
  if (value === undefined || value === null || value === '') throw new Error(`response missing ${name}`);
  return value;
}

function optionalString(value: unknown) {
  return value === undefined || value === null ? undefined : String(value);
}

function stringValue(value: unknown) {
  return value === undefined || value === null ? '' : String(value);
}

function numberValue(value: unknown) {
  const number = Number(value ?? 0);
  return Number.isFinite(number) ? number : 0;
}

function scalarValue(value: unknown, fallback: number | string = 0): number | string {
  if (value === undefined || value === null || value === '') return fallback;
  if (typeof value === 'number' || typeof value === 'string') return value;
  return String(value);
}

function boolValue(value: unknown) {
  return Boolean(value);
}

function arrayValue(value: unknown, name: string): unknown[] {
  if (value === undefined || value === null) return [];
  if (!Array.isArray(value)) throw new Error(`response ${name} must be an array`);
  return value;
}

function normalizeStringArray(value: unknown, name: string) {
  return arrayValue(value, name).map((item) => String(item));
}

function normalizeEnum<T extends string>(value: unknown, name: string, byNumber: Record<number, T>, allowed: Set<T>, fallback?: T): T {
  if (value === undefined || value === null || value === '') {
    if (fallback !== undefined) return fallback;
    throw new Error(`response missing ${name}`);
  }
  if (typeof value === 'number') {
    const mapped = byNumber[value];
    if (!mapped) throw new Error(`response ${name} has unknown enum value ${value}`);
    return mapped;
  }
  const text = String(value);
  if (/^\d+$/.test(text)) {
    const mapped = byNumber[Number(text)];
    if (!mapped) throw new Error(`response ${name} has unknown enum value ${text}`);
    return mapped;
  }
  const mappedAlias = eventTypeAliases[text] as T | undefined;
  const normalized = mappedAlias ?? text as T;
  if (!allowed.has(normalized)) throw new Error(`response ${name} has unknown enum value ${text}`);
  return normalized;
}

export function normalizeMoney(input: unknown, name: string): Money {
  if (input === undefined || input === null) return { amount: 0, currency: 'CNY' };
  const raw = asRecord(input, name);
  return {
    amount: scalarValue(field(raw, 'amount'), 0),
    currency: stringValue(field(raw, 'currency') ?? 'CNY') || 'CNY',
  };
}

function normalizeBidRule(input: unknown): BidRule {
  const raw = input === undefined || input === null ? {} : asRecord(input, 'lot.rule');
  return {
    startPrice: normalizeMoney(field(raw, 'startPrice', 'start_price'), 'lot.rule.startPrice'),
    minIncrement: normalizeMoney(field(raw, 'minIncrement', 'min_increment'), 'lot.rule.minIncrement'),
    capPrice: field(raw, 'capPrice', 'cap_price') === undefined ? undefined : normalizeMoney(field(raw, 'capPrice', 'cap_price'), 'lot.rule.capPrice'),
    durationSeconds: numberValue(field(raw, 'durationSeconds', 'duration_seconds')),
    antiSnipeWindowSeconds: numberValue(field(raw, 'antiSnipeWindowSeconds', 'anti_snipe_window_seconds')),
    antiSnipeExtendSeconds: numberValue(field(raw, 'antiSnipeExtendSeconds', 'anti_snipe_extend_seconds')),
    maxExtendCount: numberValue(field(raw, 'maxExtendCount', 'max_extend_count')),
  };
}

export function normalizeTrustRevealCard(input: unknown): TrustRevealCard {
  const raw = asRecord(input, 'trustCard');
  return {
    id: stringValue(requiredField(field(raw, 'id'), 'trustCard.id')),
    lotId: stringValue(field(raw, 'lotId', 'lot_id')),
    type: normalizeEnum(field(raw, 'type'), 'trustCard.type', trustCardTypeByNumber, trustCardTypeValues, 'TRUST_CARD_TYPE_UNSPECIFIED'),
    title: stringValue(field(raw, 'title')),
    content: stringValue(field(raw, 'content')),
    imageUrl: optionalString(field(raw, 'imageUrl', 'image_url')),
    revealed: boolValue(field(raw, 'revealed')),
    revealedAtUnixMs: scalarValue(field(raw, 'revealedAtUnixMs', 'revealed_at_unix_ms'), 0),
  };
}

function normalizeBid(input: unknown): Bid {
  const raw = asRecord(input, 'bid');
  return {
    id: stringValue(requiredField(field(raw, 'id'), 'bid.id')),
    lotId: stringValue(field(raw, 'lotId', 'lot_id')),
    userId: stringValue(field(raw, 'userId', 'user_id')),
    nickname: stringValue(field(raw, 'nickname')),
    amount: normalizeMoney(field(raw, 'amount'), 'bid.amount'),
    createdAtUnixMs: scalarValue(field(raw, 'createdAtUnixMs', 'created_at_unix_ms'), 0),
  };
}

function normalizeRankingItem(input: unknown): RankingItem {
  const raw = asRecord(input, 'rankingItem');
  return {
    rank: numberValue(field(raw, 'rank')),
    userId: stringValue(field(raw, 'userId', 'user_id')),
    nickname: stringValue(field(raw, 'nickname')),
    amount: normalizeMoney(field(raw, 'amount'), 'rankingItem.amount'),
    bidAtUnixMs: scalarValue(field(raw, 'bidAtUnixMs', 'bid_at_unix_ms'), 0),
  };
}

function normalizeDuelState(input: unknown): DuelState {
  const raw = input === undefined || input === null ? {} : asRecord(input, 'duelState');
  return {
    active: boolValue(field(raw, 'active')),
    lotId: stringValue(field(raw, 'lotId', 'lot_id')),
    userAId: stringValue(field(raw, 'userAId', 'user_a_id')),
    userANickname: stringValue(field(raw, 'userANickname', 'user_a_nickname')),
    userBId: stringValue(field(raw, 'userBId', 'user_b_id')),
    userBNickname: stringValue(field(raw, 'userBNickname', 'user_b_nickname')),
    startedAtUnixMs: scalarValue(field(raw, 'startedAtUnixMs', 'started_at_unix_ms'), 0),
    endsAtUnixMs: scalarValue(field(raw, 'endsAtUnixMs', 'ends_at_unix_ms'), 0),
    extendCount: numberValue(field(raw, 'extendCount', 'extend_count')),
    maxExtendCount: numberValue(field(raw, 'maxExtendCount', 'max_extend_count')),
  };
}

export function normalizeLot(input: unknown): Lot {
  const raw = asRecord(input, 'lot');
  const estimatePrice = field(raw, 'estimatePrice', 'estimate_price');
  return {
    id: stringValue(requiredField(field(raw, 'id'), 'lot.id')),
    roomId: stringValue(field(raw, 'roomId', 'room_id')),
    title: stringValue(field(raw, 'title')),
    description: stringValue(field(raw, 'description')),
    imageUrl: stringValue(field(raw, 'imageUrl', 'image_url')),
    status: normalizeEnum(field(raw, 'status'), 'lot.status', lotStatusByNumber, lotStatusValues, 'LOT_STATUS_UNSPECIFIED'),
    queueStatus: field(raw, 'queueStatus', 'queue_status') === undefined ? undefined : normalizeEnum(field(raw, 'queueStatus', 'queue_status'), 'lot.queueStatus', lotQueueStatusByNumber, lotQueueStatusValues),
    queuePosition: field(raw, 'queuePosition', 'queue_position') === undefined ? undefined : numberValue(field(raw, 'queuePosition', 'queue_position')),
    rule: normalizeBidRule(field(raw, 'rule')),
    currentPrice: normalizeMoney(field(raw, 'currentPrice', 'current_price'), 'lot.currentPrice'),
    leadingUserId: stringValue(field(raw, 'leadingUserId', 'leading_user_id')),
    leadingNickname: stringValue(field(raw, 'leadingNickname', 'leading_nickname')),
    startedAtUnixMs: scalarValue(field(raw, 'startedAtUnixMs', 'started_at_unix_ms'), 0),
    endsAtUnixMs: scalarValue(field(raw, 'endsAtUnixMs', 'ends_at_unix_ms'), 0),
    settledAtUnixMs: scalarValue(field(raw, 'settledAtUnixMs', 'settled_at_unix_ms'), 0),
    cancelledAtUnixMs: field(raw, 'cancelledAtUnixMs', 'cancelled_at_unix_ms') as number | string | undefined,
    winnerUserId: stringValue(field(raw, 'winnerUserId', 'winner_user_id')),
    winnerNickname: stringValue(field(raw, 'winnerNickname', 'winner_nickname')),
    finalPrice: normalizeMoney(field(raw, 'finalPrice', 'final_price'), 'lot.finalPrice'),
    version: scalarValue(field(raw, 'version'), 0),
    trustCards: arrayValue(field(raw, 'trustCards', 'trust_cards'), 'lot.trustCards').map(normalizeTrustRevealCard),
    duelState: normalizeDuelState(field(raw, 'duelState', 'duel_state')),
    playbookStage: normalizeEnum(field(raw, 'playbookStage', 'playbook_stage'), 'lot.playbookStage', playbookStageByNumber, playbookStageValues, 'PLAYBOOK_STAGE_UNSPECIFIED'),
    cancelReason: optionalString(field(raw, 'cancelReason', 'cancel_reason')),
    galleryImageUrls: normalizeStringArray(field(raw, 'galleryImageUrls', 'gallery_image_urls'), 'lot.galleryImageUrls'),
    category: optionalString(field(raw, 'category')),
    tags: normalizeStringArray(field(raw, 'tags'), 'lot.tags'),
    estimatePrice: estimatePrice === undefined ? undefined : normalizeMoney(estimatePrice, 'lot.estimatePrice'),
    stock: field(raw, 'stock') as number | string | undefined,
    afterSaleNotes: optionalString(field(raw, 'afterSaleNotes', 'after_sale_notes')),
  };
}

export function normalizeRoomSnapshot(input: unknown): RoomSnapshot {
  const raw = asRecord(input, 'snapshot');
  const currentLot = field(raw, 'currentLot', 'current_lot');
  return {
    roomId: stringValue(requiredField(field(raw, 'roomId', 'room_id'), 'snapshot.roomId')),
    currentLot: currentLot === undefined || currentLot === null ? undefined : normalizeLot(currentLot),
    ranking: arrayValue(field(raw, 'ranking'), 'snapshot.ranking').map(normalizeRankingItem),
    recentBids: arrayValue(field(raw, 'recentBids', 'recent_bids'), 'snapshot.recentBids').map(normalizeBid),
    playbookStage: normalizeEnum(field(raw, 'playbookStage', 'playbook_stage'), 'snapshot.playbookStage', playbookStageByNumber, playbookStageValues, 'PLAYBOOK_STAGE_UNSPECIFIED'),
    serverTimeUnixMs: scalarValue(field(raw, 'serverTimeUnixMs', 'server_time_unix_ms'), 0),
  };
}

export function normalizeRoomPresence(input: unknown): RoomPresence {
  const raw = asRecord(input, 'presence');
  return {
    roomId: stringValue(requiredField(field(raw, 'roomId', 'room_id'), 'presence.roomId')),
    totalConnections: requiredField(field(raw, 'totalConnections', 'total_connections'), 'presence.totalConnections') as number | string,
    viewerConnections: requiredField(field(raw, 'viewerConnections', 'viewer_connections'), 'presence.viewerConnections') as number | string,
    operatorConnections: requiredField(field(raw, 'operatorConnections', 'operator_connections'), 'presence.operatorConnections') as number | string,
    serverTimeUnixMs: requiredField(field(raw, 'serverTimeUnixMs', 'server_time_unix_ms'), 'presence.serverTimeUnixMs') as number | string,
  };
}

export function normalizeAuctionEvent(input: unknown): AuctionEvent {
  const raw = asRecord(input, 'event');
  const lot = field(raw, 'lot');
  const bid = field(raw, 'bid');
  const ranking = field(raw, 'ranking');
  const trustCard = field(raw, 'trustCard', 'trust_card');
  const duelState = field(raw, 'duelState', 'duel_state');
  const snapshot = field(raw, 'snapshot');
  return {
    ...raw,
    id: stringValue(field(raw, 'id')),
    type: normalizeEnum(field(raw, 'type'), 'event.type', eventTypeByNumber, new Set(Object.values(eventTypeByNumber))),
    roomId: stringValue(field(raw, 'roomId', 'room_id')),
    lotId: stringValue(field(raw, 'lotId', 'lot_id')),
    occurredAtUnixMs: scalarValue(field(raw, 'occurredAtUnixMs', 'occurred_at_unix_ms'), 0),
    lot: lot === undefined || lot === null ? undefined : normalizeLot(lot),
    bid: bid === undefined || bid === null ? undefined : normalizeBid(bid),
    ranking: ranking === undefined || ranking === null ? undefined : arrayValue(ranking, 'event.ranking').map(normalizeRankingItem),
    trustCard: trustCard === undefined || trustCard === null ? undefined : normalizeTrustRevealCard(trustCard),
    duelState: duelState === undefined || duelState === null ? undefined : normalizeDuelState(duelState),
    snapshot: snapshot === undefined || snapshot === null ? undefined : normalizeRoomSnapshot(snapshot),
    reason: optionalString(field(raw, 'reason')),
    orderId: optionalString(field(raw, 'orderId', 'order_id')),
    paymentId: optionalString(field(raw, 'paymentId', 'payment_id')),
  } as AuctionEvent;
}

export function normalizeUser(input: unknown): User {
  const raw = asRecord(input, 'user');
  return {
    id: stringValue(requiredField(field(raw, 'id'), 'user.id')),
    username: stringValue(requiredField(field(raw, 'username'), 'user.username')),
    nickname: stringValue(field(raw, 'nickname')),
    role: normalizeEnum(field(raw, 'role'), 'user.role', userRoleByNumber, userRoleValues),
    createdAtUnixMs: scalarValue(field(raw, 'createdAtUnixMs', 'created_at_unix_ms'), 0),
    updatedAtUnixMs: scalarValue(field(raw, 'updatedAtUnixMs', 'updated_at_unix_ms'), 0),
  };
}

export function normalizeAuthTokens(input: unknown): AuthTokens {
  const raw = asRecord(input, 'tokens');
  return {
    accessToken: stringValue(requiredField(field(raw, 'accessToken', 'access_token'), 'tokens.accessToken')),
    refreshToken: stringValue(requiredField(field(raw, 'refreshToken', 'refresh_token'), 'tokens.refreshToken')),
    accessExpiresAtUnixMs: requiredField(field(raw, 'accessExpiresAtUnixMs', 'access_expires_at_unix_ms'), 'tokens.accessExpiresAtUnixMs') as number | string,
    refreshExpiresAtUnixMs: requiredField(field(raw, 'refreshExpiresAtUnixMs', 'refresh_expires_at_unix_ms'), 'tokens.refreshExpiresAtUnixMs') as number | string,
  };
}

export function normalizeUploadedAsset(input: unknown): UploadedAsset {
  const raw = asRecord(input, 'asset');
  return {
    id: stringValue(requiredField(field(raw, 'id'), 'asset.id')),
    imageUrl: stringValue(requiredField(field(raw, 'imageUrl', 'image_url'), 'asset.imageUrl')),
    bucket: stringValue(field(raw, 'bucket')),
    objectKey: stringValue(field(raw, 'objectKey', 'object_key')),
    mimeType: stringValue(field(raw, 'mimeType', 'mime_type')),
    sizeBytes: scalarValue(field(raw, 'sizeBytes', 'size_bytes'), 0),
    status: optionalString(field(raw, 'status')),
    expiresAtUnixMs: field(raw, 'expiresAtUnixMs', 'expires_at_unix_ms') as number | string | undefined,
  };
}
