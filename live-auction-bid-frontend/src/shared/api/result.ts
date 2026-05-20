import type { AuctionEvent, EventType, ReplyResult } from './types';
import { RESULT_CODE_OK } from './types';

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
};

export class ApiResultError extends Error {
  readonly result: ReplyResult;

  constructor(result: ReplyResult) {
    super(result.message || `request failed with result code ${result.code}`);
    this.name = 'ApiResultError';
    this.result = result;
  }
}

export function assertOkResult<T extends { result?: ReplyResult }>(reply: T): T {
  const result = reply.result;
  if (result && result.code !== RESULT_CODE_OK) {
    throw new ApiResultError(result);
  }
  return reply;
}

export function resultMessage(e: unknown): string {
  if (e instanceof ApiResultError) return e.result.message || `业务错误：${e.result.code}`;
  if (e instanceof Error) return e.message;
  return String(e);
}

export function normalizeAuctionEvent(input: unknown): AuctionEvent {
  const event = input as AuctionEvent & { type?: EventType | number };
  if (typeof event.type === 'number') {
    return { ...event, type: eventTypeByNumber[event.type] ?? 'AUCTION_EVENT_TYPE_UNSPECIFIED' } as AuctionEvent;
  }
  return event as AuctionEvent;
}
