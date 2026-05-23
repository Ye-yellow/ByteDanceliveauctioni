import type { AuctionEvent, EventType, ReplyResult } from './types';
import {
  RESULT_CODE_INTERNAL_ERROR,
  RESULT_CODE_INVALID_CREDENTIALS,
  RESULT_CODE_LOGIN_REQUIRED,
  RESULT_CODE_LOT_VERSION_CONFLICT,
  RESULT_CODE_OK,
  RESULT_CODE_FORBIDDEN,
  RESULT_CODE_SESSION_EXPIRED,
  RESULT_CODE_TOKEN_EXPIRED,
  RESULT_CODE_TOKEN_INVALID,
} from './types';

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

export class ApiResultError extends Error {
  readonly result: ReplyResult;

  constructor(result: ReplyResult) {
    super(result.message || `request failed with result code ${result.code}`);
    this.name = 'ApiResultError';
    this.result = result;
  }
}

export function assertOkResult<T extends { result?: Partial<ReplyResult> }>(reply: T): T {
  const result = reply.result;
  // proto3 JSON 会省略默认值 code=0，所以 { message: 'ok' } 也应视为成功。
  if (result && (result.code ?? RESULT_CODE_OK) !== RESULT_CODE_OK) {
    throw new ApiResultError(result as ReplyResult);
  }
  return reply;
}

export function resultMessage(e: unknown): string {
  const result = (e as { result?: Partial<ReplyResult> } | null)?.result;
  if (result) return publicResultMessage(result);
  if (e instanceof ApiResultError) return publicResultMessage(e.result);
  if (e instanceof Error) return e.message;
  return String(e);
}

const errorMessages: Record<number, string> = {
  [RESULT_CODE_LOGIN_REQUIRED]: '请先登录后再操作',
  [RESULT_CODE_TOKEN_EXPIRED]: '登录已过期，正在刷新登录态',
  [RESULT_CODE_TOKEN_INVALID]: '登录凭证无效，请重新登录',
  [RESULT_CODE_SESSION_EXPIRED]: '登录会话已失效，请重新登录',
  [RESULT_CODE_INVALID_CREDENTIALS]: '用户名或密码不正确',
  [RESULT_CODE_FORBIDDEN]: '当前账号没有执行该操作的权限',
  [RESULT_CODE_LOT_VERSION_CONFLICT]: '竞拍状态已变化，已刷新最新数据后再操作',
  [RESULT_CODE_INTERNAL_ERROR]: '系统暂时不可用，请稍后重试',
};

export function publicResultMessage(result?: Partial<ReplyResult>, fallback = '请求失败') {
  if (!result) return fallback;
  const code = Number(result.code ?? RESULT_CODE_OK);
  return errorMessages[code] || result.message || `${fallback}（code=${code}）`;
}

function normalizeEventType(type: unknown): EventType {
  if (typeof type === 'number') return eventTypeByNumber[type] ?? 'AUCTION_EVENT_TYPE_UNSPECIFIED';
  if (typeof type === 'string') return eventTypeAliases[type] ?? (type as EventType);
  return 'AUCTION_EVENT_TYPE_UNSPECIFIED';
}

export function normalizeAuctionEvent(input: unknown): AuctionEvent {
  const raw = input as AuctionEvent & {
    type?: EventType | number | string;
    room_id?: string;
    lot_id?: string;
    occurred_at_unix_ms?: number | string;
    trust_card?: AuctionEvent['trustCard'];
    duel_state?: AuctionEvent['duelState'];
    order_id?: string;
    payment_id?: string;
  };
  return {
    ...raw,
    type: normalizeEventType(raw.type),
    roomId: raw.roomId ?? raw.room_id ?? '',
    lotId: raw.lotId ?? raw.lot_id ?? '',
    occurredAtUnixMs: raw.occurredAtUnixMs ?? raw.occurred_at_unix_ms ?? 0,
    trustCard: raw.trustCard ?? raw.trust_card,
    duelState: raw.duelState ?? raw.duel_state,
    orderId: raw.orderId ?? raw.order_id,
    paymentId: raw.paymentId ?? raw.payment_id,
  } as AuctionEvent;
}
