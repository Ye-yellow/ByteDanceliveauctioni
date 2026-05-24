import {
  AUCTION_EVENT_TYPE,
  LOT_QUEUE_STATUS,
  LOT_STATUS,
  USER_ROLE,
  type AuctionEventType,
  type AuctionSocketEvent,
  type AuthTokens,
  type BidEvent,
  type BidRecord,
  type BidRule,
  type Lot,
  type LotQueueStatus,
  type LotResult,
  type LotStatus,
  type Money,
  type MoneyInput,
  type OrderSummary,
  type PaymentSummary,
  type PlaceBidResponse,
  type RankingItem,
  type RoomSnapshot,
  type User,
  type UserRole,
} from './types';

type RawRecord = Record<string, unknown>;

function asRecord(value: unknown): RawRecord {
  return value && typeof value === 'object' ? (value as RawRecord) : {};
}

function pick<T = unknown>(record: RawRecord, camel: string, snake?: string): T | undefined {
  return (record[camel] ?? (snake ? record[snake] : undefined)) as T | undefined;
}

function stringValue(value: unknown, fallback = ''): string {
  return typeof value === 'string' ? value : fallback;
}

function numberValue(value: unknown, fallback = 0): number {
  const next = Number(value);
  return Number.isFinite(next) ? next : fallback;
}

export function normalizeMoney(value: MoneyInput): Money {
  if (value && typeof value === 'object' && 'amount' in value) {
    const money = value as Money;
    return { amount: money.amount ?? 0, currency: money.currency || 'CNY' };
  }
  return { amount: typeof value === 'number' || typeof value === 'string' ? value : 0, currency: 'CNY' };
}

function normalizeMoneyFields(raw: RawRecord): Money {
  if (raw.amount && typeof raw.amount === 'object') return normalizeMoney(raw.amount as MoneyInput);
  return {
    amount: typeof raw.amount === 'number' || typeof raw.amount === 'string' ? raw.amount : 0,
    currency: stringValue(raw.currency, 'CNY'),
  };
}

export function normalizeAuthTokens(input: unknown): AuthTokens {
  const raw = asRecord(input);
  return {
    accessToken: stringValue(pick(raw, 'accessToken', 'access_token')),
    refreshToken: stringValue(pick(raw, 'refreshToken', 'refresh_token')),
    accessExpiresAtUnixMs: pick(raw, 'accessExpiresAtUnixMs', 'access_expires_at_unix_ms') ?? 0,
    refreshExpiresAtUnixMs: pick(raw, 'refreshExpiresAtUnixMs', 'refresh_expires_at_unix_ms') ?? 0,
  };
}

export function normalizeUser(input: unknown): User {
  const raw = asRecord(input);
  return {
    id: stringValue(raw.id),
    username: stringValue(raw.username),
    nickname: stringValue(raw.nickname),
    role: normalizeUserRole(raw.role),
    createdAtUnixMs: pick(raw, 'createdAtUnixMs', 'created_at_unix_ms'),
    updatedAtUnixMs: pick(raw, 'updatedAtUnixMs', 'updated_at_unix_ms'),
  };
}

function normalizeUserRole(value: unknown): UserRole {
  if (typeof value === 'string') {
    if (value.startsWith('USER_ROLE_')) return value as UserRole;
    const numericString = Number(value);
    if (!Number.isFinite(numericString)) return USER_ROLE.UNSPECIFIED;
    value = numericString;
  }

  const numeric: Record<number, UserRole> = {
    0: USER_ROLE.UNSPECIFIED,
    1: USER_ROLE.BUYER,
    2: USER_ROLE.ANCHOR,
    3: USER_ROLE.OPERATOR,
    4: USER_ROLE.ADMIN,
  };

  return numeric[numberValue(value, -1)] ?? USER_ROLE.UNSPECIFIED;
}

export function normalizeLotStatus(value: unknown): LotStatus {
  if (typeof value === 'string') {
    if (value.startsWith('LOT_STATUS_')) return value as LotStatus;
    if (value === 'LIVE') return LOT_STATUS.LIVE;
    if (value === 'SETTLED') return LOT_STATUS.SETTLED;
    if (value === 'CANCELLED') return LOT_STATUS.CANCELLED;
    if (value === 'READY') return LOT_STATUS.READY;
    if (value === 'QUEUED') return LOT_STATUS.QUEUED;
    if (value === 'SCHEDULED') return LOT_STATUS.SCHEDULED;
    if (value === 'EXTENDED') return LOT_STATUS.EXTENDED;
    if (value === 'SOLD') return LOT_STATUS.SOLD;
    if (value === 'FAILED') return LOT_STATUS.FAILED;
    if (value === 'DRAFT') return LOT_STATUS.DRAFT;
  }

  const numeric: Record<number, LotStatus> = {
    1: LOT_STATUS.DRAFT,
    2: LOT_STATUS.LIVE,
    3: LOT_STATUS.SETTLED,
    4: LOT_STATUS.CANCELLED,
    5: LOT_STATUS.READY,
    6: LOT_STATUS.QUEUED,
    7: LOT_STATUS.EXTENDED,
    8: LOT_STATUS.FAILED,
  };

  return numeric[numberValue(value)] ?? LOT_STATUS.UNSPECIFIED;
}

function normalizeLotQueueStatus(value: unknown): LotQueueStatus {
  if (typeof value === 'string') {
    if (value.startsWith('LOT_QUEUE_STATUS_')) return value as LotQueueStatus;
    if (value === 'NONE') return LOT_QUEUE_STATUS.NONE;
    if (value === 'QUEUED') return LOT_QUEUE_STATUS.QUEUED;
    if (value === 'NEXT') return LOT_QUEUE_STATUS.NEXT;
  }

  const numeric: Record<number, LotQueueStatus> = {
    0: LOT_QUEUE_STATUS.UNSPECIFIED,
    1: LOT_QUEUE_STATUS.NONE,
    2: LOT_QUEUE_STATUS.QUEUED,
    3: LOT_QUEUE_STATUS.NEXT,
  };

  return numeric[numberValue(value)] ?? LOT_QUEUE_STATUS.UNSPECIFIED;
}

function normalizeBidRule(input: unknown): BidRule {
  const raw = asRecord(input);
  return {
    startPrice: normalizeMoney(pick(raw, 'startPrice', 'start_price')),
    minIncrement: normalizeMoney(pick(raw, 'minIncrement', 'min_increment')),
    capPrice: pick(raw, 'capPrice', 'cap_price') ? normalizeMoney(pick(raw, 'capPrice', 'cap_price')) : undefined,
    durationSeconds: numberValue(pick(raw, 'durationSeconds', 'duration_seconds')),
    antiSnipeWindowSeconds: numberValue(pick(raw, 'antiSnipeWindowSeconds', 'anti_snipe_window_seconds')),
    antiSnipeExtendSeconds: numberValue(pick(raw, 'antiSnipeExtendSeconds', 'anti_snipe_extend_seconds')),
    maxExtendCount: numberValue(pick(raw, 'maxExtendCount', 'max_extend_count')),
  };
}

export function normalizeLot(input: unknown): Lot {
  const raw = asRecord(input);
  return {
    id: stringValue(raw.id),
    roomId: stringValue(pick(raw, 'roomId', 'room_id')),
    title: stringValue(raw.title),
    description: stringValue(raw.description),
    imageUrl: stringValue(pick(raw, 'imageUrl', 'image_url')),
    status: normalizeLotStatus(raw.status),
    currentPrice: normalizeMoney(pick(raw, 'currentPrice', 'current_price')),
    leadingUserId: stringValue(pick(raw, 'leadingUserId', 'leading_user_id')),
    leadingNickname: stringValue(pick(raw, 'leadingNickname', 'leading_nickname')),
    winnerUserId: stringValue(pick(raw, 'winnerUserId', 'winner_user_id')),
    winnerNickname: stringValue(pick(raw, 'winnerNickname', 'winner_nickname')),
    finalPrice: pick(raw, 'finalPrice', 'final_price') ? normalizeMoney(pick(raw, 'finalPrice', 'final_price')) : undefined,
    participantCount: numberValue(pick(raw, 'participantCount', 'participant_count')),
    bidCount: numberValue(pick(raw, 'bidCount', 'bid_count')),
    startedAtUnixMs: pick(raw, 'startedAtUnixMs', 'started_at_unix_ms'),
    endsAtUnixMs: pick(raw, 'endsAtUnixMs', 'ends_at_unix_ms'),
    settledAtUnixMs: pick(raw, 'settledAtUnixMs', 'settled_at_unix_ms'),
    rule: normalizeBidRule(raw.rule),
    version: raw.version as Lot['version'],
    queueStatus: normalizeLotQueueStatus(pick(raw, 'queueStatus', 'queue_status')),
    queuePosition: numberValue(pick(raw, 'queuePosition', 'queue_position')),
    cancelReason: stringValue(pick(raw, 'cancelReason', 'cancel_reason')),
    cancelledAtUnixMs: pick(raw, 'cancelledAtUnixMs', 'cancelled_at_unix_ms'),
  };
}

export function normalizeRankingItem(input: unknown): RankingItem {
  const raw = asRecord(input);
  return {
    rank: numberValue(raw.rank),
    userId: stringValue(pick(raw, 'userId', 'user_id')),
    nickname: stringValue(raw.nickname),
    avatarUrl: stringValue(pick(raw, 'avatarUrl', 'avatar_url')),
    amount: normalizeMoney(raw.amount as MoneyInput),
    bidAtUnixMs: pick(raw, 'bidAtUnixMs', 'bid_at_unix_ms'),
  };
}

export function normalizeBid(input: unknown): BidEvent {
  const raw = asRecord(input);
  return {
    id: stringValue(raw.id),
    lotId: stringValue(pick(raw, 'lotId', 'lot_id')),
    userId: stringValue(pick(raw, 'userId', 'user_id')),
    nickname: stringValue(raw.nickname),
    amount: normalizeMoney(raw.amount as MoneyInput),
    accepted: raw.accepted as boolean | undefined,
    rejectReason: stringValue(pick(raw, 'rejectReason', 'reject_reason')),
    createdAtUnixMs: pick(raw, 'createdAtUnixMs', 'created_at_unix_ms'),
  };
}

export function normalizeRoomSnapshot(input: unknown, fallbackRoomId = ''): RoomSnapshot {
  const raw = asRecord(input);
  return {
    roomId: stringValue(pick(raw, 'roomId', 'room_id'), fallbackRoomId),
    roomName: stringValue(pick(raw, 'roomName', 'room_name')),
    anchorName: stringValue(pick(raw, 'anchorName', 'anchor_name')),
    onlineCount: numberValue(pick(raw, 'onlineCount', 'online_count')),
    serverTimeUnixMs: pick(raw, 'serverTimeUnixMs', 'server_time_unix_ms'),
    currentLot: pick(raw, 'currentLot', 'current_lot') ? normalizeLot(pick(raw, 'currentLot', 'current_lot')) : null,
    ranking: Array.isArray(raw.ranking) ? raw.ranking.map(normalizeRankingItem) : [],
    recentBids: Array.isArray(pick(raw, 'recentBids', 'recent_bids'))
      ? (pick<unknown[]>(raw, 'recentBids', 'recent_bids') ?? []).map(normalizeBid)
      : [],
    playbookStage: pick(raw, 'playbookStage', 'playbook_stage'),
  };
}

function normalizeEventType(value: unknown): AuctionEventType {
  if (typeof value === 'string') {
    const key = value.trim().toUpperCase();
    const aliases: Record<string, AuctionEventType> = {
      HEARTBEAT: AUCTION_EVENT_TYPE.SERVER_HEARTBEAT,
      SERVER_HEARTBEAT: AUCTION_EVENT_TYPE.SERVER_HEARTBEAT,
      AUCTION_EVENT_TYPE_SERVER_HEARTBEAT: AUCTION_EVENT_TYPE.SERVER_HEARTBEAT,
      ROOM_SNAPSHOT: AUCTION_EVENT_TYPE.ROOM_SNAPSHOT,
      AUCTION_EVENT_TYPE_ROOM_SNAPSHOT: AUCTION_EVENT_TYPE.ROOM_SNAPSHOT,
      LOT_CREATED: AUCTION_EVENT_TYPE.LOT_CREATED,
      AUCTION_EVENT_TYPE_LOT_CREATED: AUCTION_EVENT_TYPE.LOT_CREATED,
      LOT_STARTED: AUCTION_EVENT_TYPE.LOT_STARTED,
      AUCTION_EVENT_TYPE_LOT_STARTED: AUCTION_EVENT_TYPE.LOT_STARTED,
      LOT_UPDATED: AUCTION_EVENT_TYPE.LOT_UPDATED,
      AUCTION_EVENT_TYPE_LOT_UPDATED: AUCTION_EVENT_TYPE.LOT_UPDATED,
      BID_ACCEPTED: AUCTION_EVENT_TYPE.BID_ACCEPTED,
      AUCTION_EVENT_TYPE_BID_ACCEPTED: AUCTION_EVENT_TYPE.BID_ACCEPTED,
      BID_REJECTED: AUCTION_EVENT_TYPE.BID_REJECTED,
      AUCTION_EVENT_TYPE_BID_REJECTED: AUCTION_EVENT_TYPE.BID_REJECTED,
      RANKING_UPDATED: AUCTION_EVENT_TYPE.RANKING_UPDATED,
      AUCTION_EVENT_TYPE_RANKING_UPDATED: AUCTION_EVENT_TYPE.RANKING_UPDATED,
      TRUST_REVEALED: AUCTION_EVENT_TYPE.TRUST_REVEALED,
      AUCTION_EVENT_TYPE_TRUST_REVEALED: AUCTION_EVENT_TYPE.TRUST_REVEALED,
      DUEL_STARTED: AUCTION_EVENT_TYPE.DUEL_STARTED,
      AUCTION_EVENT_TYPE_DUEL_STARTED: AUCTION_EVENT_TYPE.DUEL_STARTED,
      DUEL_ENDED: AUCTION_EVENT_TYPE.DUEL_ENDED,
      AUCTION_EVENT_TYPE_DUEL_ENDED: AUCTION_EVENT_TYPE.DUEL_ENDED,
      LOT_SETTLED: AUCTION_EVENT_TYPE.LOT_SETTLED,
      AUCTION_EVENT_TYPE_LOT_SETTLED: AUCTION_EVENT_TYPE.LOT_SETTLED,
      LOT_CANCELLED: AUCTION_EVENT_TYPE.LOT_CANCELLED,
      AUCTION_EVENT_TYPE_LOT_CANCELLED: AUCTION_EVENT_TYPE.LOT_CANCELLED,
      LOT_QUEUED: AUCTION_EVENT_TYPE.LOT_QUEUED,
      AUCTION_EVENT_TYPE_LOT_QUEUED: AUCTION_EVENT_TYPE.LOT_QUEUED,
      BID_OUTBID: AUCTION_EVENT_TYPE.BID_OUTBID,
      AUCTION_EVENT_TYPE_BID_OUTBID: AUCTION_EVENT_TYPE.BID_OUTBID,
      AUCTION_EXTENDED: AUCTION_EVENT_TYPE.AUCTION_EXTENDED,
      AUCTION_EVENT_TYPE_AUCTION_EXTENDED: AUCTION_EVENT_TYPE.AUCTION_EXTENDED,
      AUCTION_CLOSED: AUCTION_EVENT_TYPE.AUCTION_CLOSED,
      AUCTION_EVENT_TYPE_AUCTION_CLOSED: AUCTION_EVENT_TYPE.AUCTION_CLOSED,
      ORDER_CREATED: AUCTION_EVENT_TYPE.ORDER_CREATED,
      AUCTION_EVENT_TYPE_ORDER_CREATED: AUCTION_EVENT_TYPE.ORDER_CREATED,
      PAYMENT_SUCCESS: AUCTION_EVENT_TYPE.PAYMENT_SUCCESS,
      AUCTION_EVENT_TYPE_PAYMENT_SUCCESS: AUCTION_EVENT_TYPE.PAYMENT_SUCCESS,
    };
    return aliases[key] ?? AUCTION_EVENT_TYPE.UNSPECIFIED;
  }

  const numeric: Record<number, AuctionEventType> = {
    1: AUCTION_EVENT_TYPE.ROOM_SNAPSHOT,
    2: AUCTION_EVENT_TYPE.LOT_CREATED,
    3: AUCTION_EVENT_TYPE.LOT_STARTED,
    4: AUCTION_EVENT_TYPE.LOT_UPDATED,
    5: AUCTION_EVENT_TYPE.BID_ACCEPTED,
    6: AUCTION_EVENT_TYPE.BID_REJECTED,
    7: AUCTION_EVENT_TYPE.RANKING_UPDATED,
    8: AUCTION_EVENT_TYPE.TRUST_REVEALED,
    9: AUCTION_EVENT_TYPE.DUEL_STARTED,
    10: AUCTION_EVENT_TYPE.DUEL_ENDED,
    11: AUCTION_EVENT_TYPE.LOT_SETTLED,
    12: AUCTION_EVENT_TYPE.LOT_CANCELLED,
    13: AUCTION_EVENT_TYPE.LOT_QUEUED,
    14: AUCTION_EVENT_TYPE.BID_OUTBID,
    15: AUCTION_EVENT_TYPE.AUCTION_EXTENDED,
    16: AUCTION_EVENT_TYPE.AUCTION_CLOSED,
    17: AUCTION_EVENT_TYPE.ORDER_CREATED,
    18: AUCTION_EVENT_TYPE.PAYMENT_SUCCESS,
  };

  return numeric[numberValue(value)] ?? AUCTION_EVENT_TYPE.UNSPECIFIED;
}

export function normalizeOrder(input: unknown): OrderSummary | undefined {
  const raw = asRecord(input);
  const id = stringValue(raw.id);
  if (!id) return undefined;
  return {
    id,
    lotId: stringValue(pick(raw, 'lotId', 'lot_id')),
    roomId: stringValue(pick(raw, 'roomId', 'room_id')),
    lotTitle: stringValue(pick(raw, 'lotTitle', 'lot_title')),
    lotImageUrl: stringValue(pick(raw, 'lotImageUrl', 'lot_image_url')),
    buyerUserId: stringValue(pick(raw, 'buyerUserId', 'buyer_user_id')),
    buyerNickname: stringValue(pick(raw, 'buyerNickname', 'buyer_nickname')),
    amount: normalizeMoneyFields(raw),
    status: stringValue(raw.status),
    paymentStatus: stringValue(pick(raw, 'paymentStatus', 'payment_status')),
    paymentId: stringValue(pick(raw, 'paymentId', 'payment_id')),
    createdAtUnixMs: pick(raw, 'createdAtUnixMs', 'created_at_unix_ms'),
    updatedAtUnixMs: pick(raw, 'updatedAtUnixMs', 'updated_at_unix_ms'),
    expiresAtUnixMs: pick(raw, 'expiresAtUnixMs', 'expires_at_unix_ms'),
    paidAtUnixMs: pick(raw, 'paidAtUnixMs', 'paid_at_unix_ms'),
  };
}

export function normalizePayment(input: unknown): PaymentSummary | undefined {
  const raw = asRecord(input);
  const id = stringValue(raw.id);
  if (!id) return undefined;
  return {
    id,
    orderId: stringValue(pick(raw, 'orderId', 'order_id')),
    status: stringValue(raw.status),
    amount: normalizeMoneyFields(raw),
    createdAtUnixMs: pick(raw, 'createdAtUnixMs', 'created_at_unix_ms'),
    succeededAtUnixMs: pick(raw, 'succeededAtUnixMs', 'succeeded_at_unix_ms'),
  };
}

export function normalizeBidRecord(input: unknown): BidRecord {
  const raw = asRecord(input);
  return {
    id: stringValue(raw.id),
    lotId: stringValue(pick(raw, 'lotId', 'lot_id')),
    roomId: stringValue(pick(raw, 'roomId', 'room_id')),
    lotTitle: stringValue(pick(raw, 'lotTitle', 'lot_title')),
    lotImageUrl: stringValue(pick(raw, 'lotImageUrl', 'lot_image_url')),
    userId: stringValue(pick(raw, 'userId', 'user_id')),
    nickname: stringValue(raw.nickname),
    amount: normalizeMoneyFields(raw),
    createdAtUnixMs: pick(raw, 'createdAtUnixMs', 'created_at_unix_ms'),
    lotStatus: normalizeLotStatus(pick(raw, 'lotStatus', 'lot_status')),
    auctionState: stringValue(pick(raw, 'auctionState', 'auction_state')),
    won: Boolean(raw.won),
  };
}

export function normalizeAuctionEvent(input: unknown): AuctionSocketEvent {
  const raw = asRecord(input);
  const snapshotRaw = raw.snapshot;
  const type = normalizeEventType(raw.type);
  const privateResultSignal = type === AUCTION_EVENT_TYPE.ORDER_CREATED || type === AUCTION_EVENT_TYPE.PAYMENT_SUCCESS;
  const order = privateResultSignal ? undefined : normalizeOrder(raw.order);
  const payment = privateResultSignal ? undefined : normalizePayment(raw.payment);
  const reason = stringValue(raw.reason);

  return {
    id: stringValue(raw.id),
    type,
    roomId: stringValue(pick(raw, 'roomId', 'room_id')),
    lotId: stringValue(pick(raw, 'lotId', 'lot_id')),
    occurredAtUnixMs: pick(raw, 'occurredAtUnixMs', 'occurred_at_unix_ms'),
    snapshot: snapshotRaw ? normalizeRoomSnapshot(snapshotRaw) : undefined,
    lot: raw.lot ? normalizeLot(raw.lot) : undefined,
    ranking: Array.isArray(raw.ranking) ? raw.ranking.map(normalizeRankingItem) : undefined,
    bid: raw.bid ? normalizeBid(raw.bid) : undefined,
    recentBids: Array.isArray(pick(raw, 'recentBids', 'recent_bids'))
      ? (pick<unknown[]>(raw, 'recentBids', 'recent_bids') ?? []).map(normalizeBid)
      : undefined,
    rejectReason: stringValue(pick(raw, 'rejectReason', 'reject_reason') ?? raw.reason),
    reason,
    serverTimeUnixMs: pick(raw, 'serverTimeUnixMs', 'server_time_unix_ms'),
    order,
    payment,
  };
}

export function normalizePlaceBidResponse(input: unknown): PlaceBidResponse {
  const raw = asRecord(input);
  return {
    accepted: Boolean(raw.accepted),
    lot: raw.lot ? normalizeLot(raw.lot) : undefined,
    ranking: Array.isArray(raw.ranking) ? raw.ranking.map(normalizeRankingItem) : undefined,
    rejectReason: stringValue(pick(raw, 'rejectReason', 'reject_reason')),
    bid: raw.bid ? normalizeBid(raw.bid) : undefined,
  };
}

export function normalizeLotResult(input: unknown): LotResult {
  const raw = asRecord(input);
  const lot = normalizeLot(raw.lot);
  const order = normalizeOrder(raw.order);
  return {
    lot,
    order,
    orderId: stringValue(pick(raw, 'orderId', 'order_id') ?? order?.id),
    winnerUserId: stringValue(pick(raw, 'winnerUserId', 'winner_user_id')),
    winnerNickname: stringValue(pick(raw, 'winnerNickname', 'winner_nickname')),
    finalPrice: pick(raw, 'finalPrice', 'final_price') ? normalizeMoney(pick(raw, 'finalPrice', 'final_price')) : lot.finalPrice,
    auctionState: stringValue(pick(raw, 'auctionState', 'auction_state')),
  };
}
