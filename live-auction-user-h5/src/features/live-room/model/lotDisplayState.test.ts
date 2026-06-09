import { describe, expect, it } from 'vitest';
import { LOT_STATUS, type Lot, type Money, type OrderSummary } from '../../../shared/api/types';
import { deriveLotDisplayState } from './lotDisplayState';

function money(amount: number): Money {
  return { amount, currency: 'CNY' };
}

function lot(overrides: Partial<Lot> = {}): Lot {
  return {
    id: 'lot-1',
    roomId: 'room-1',
    title: '测试拍品',
    status: LOT_STATUS.LIVE,
    currentPrice: money(100),
    stats: { participantCount: 2, bidCount: 2 },
    rule: {
      startPrice: money(0),
      minIncrement: money(50),
      durationSeconds: 300,
      antiSnipeWindowSeconds: 10,
      antiSnipeExtendSeconds: 15,
    },
    leadingUserId: 'buyer-1',
    finalPrice: money(100),
    version: 1,
    ...overrides,
  };
}

function paidOrder(overrides: Partial<OrderSummary> = {}): OrderSummary {
  return {
    id: 'order-1',
    lotId: 'lot-1',
    lotTitle: '测试拍品',
    status: 'PAID',
    paymentStatus: 'SUCCESS',
    amount: money(100),
    ...overrides,
  };
}

describe('deriveLotDisplayState', () => {
  it('treats locally paid live lots as finished while waiting for the room snapshot', () => {
    expect(deriveLotDisplayState(lot(), { paymentKnownPaid: true })).toBe('finished');
    expect(deriveLotDisplayState(lot(), { order: paidOrder() })).toBe('finished');
  });
});
