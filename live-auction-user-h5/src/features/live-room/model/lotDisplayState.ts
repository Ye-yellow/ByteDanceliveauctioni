import { isBiddableLotStatus } from '../../../entities/auction/model/status';
import { isOrderFailed, isOrderPaid, isOrderPaying, ORDER_PAYMENT_WINDOW_MS } from '../../../entities/order/model/privacy';
import { LOT_STATUS, type Lot, type OrderSummary } from '../../../shared/api/types';
import { moneyNumber } from '../../../shared/lib/money';

export type LotDisplayState = 'upcoming' | 'live' | 'pendingPayment' | 'finished' | 'failed' | 'cancelled' | 'syncing';

export function lotIsDisplayable(lot: Lot): boolean {
  return lot.status !== LOT_STATUS.DRAFT;
}

export function lotEndsAtPassed(lot: Lot, nowMs = Date.now()): boolean {
  const endsAt = Number(lot.endsAtUnixMs || 0);
  return Number.isFinite(endsAt) && endsAt > 0 && endsAt <= nowMs;
}

export function lotHasBid(lot: Lot): boolean {
  const start = moneyNumber(lot.rule.startPrice);
  const current = moneyNumber(lot.currentPrice);
  return Boolean(lot.leadingUserId || lot.stats?.bidCount || (current > 0 && current > start));
}

export function lotHasLockedResult(lot: Lot): boolean {
  return lot.status === LOT_STATUS.SETTLED ||
    Boolean(lot.winnerUserId || moneyNumber(lot.finalPrice) > 0 || Number(lot.settledAtUnixMs || 0) > 0);
}

export function orderForLot(orders: OrderSummary[], lot: Lot): OrderSummary | null {
  return orders.find((order) => order.lotId === lot.id) || null;
}

function lotPaymentWindowPassed(lot: Lot, nowMs: number): boolean {
  const settledAt = Number(lot.settledAtUnixMs || 0);
  return Number.isFinite(settledAt) && settledAt > 0 && settledAt + ORDER_PAYMENT_WINDOW_MS <= nowMs;
}

export function deriveLotDisplayState(
  lot: Lot,
  options: { order?: OrderSummary | null; paymentKnownPaid?: boolean; nowMs?: number } = {},
): LotDisplayState {
  const nowMs = options.nowMs ?? Date.now();
  const order = options.order || null;
  const hasOrder = Boolean(order?.id);

  if (isBiddableLotStatus(lot.status)) return 'live';

  if (options.paymentKnownPaid || isOrderPaid(order)) return 'finished';
  if (isOrderFailed(order, nowMs)) return 'failed';
  if (lot.status === LOT_STATUS.CANCELLED) return 'cancelled';
  if (lot.status === LOT_STATUS.FAILED || Number(lot.cancelledAtUnixMs || 0) > 0) return 'failed';

  if (lotHasLockedResult(lot)) {
    if (!lotHasBid(lot)) return 'failed';
    if (!hasOrder) return 'finished';
    if (isOrderPaying(order, nowMs)) return 'pendingPayment';
    return lotPaymentWindowPassed(lot, nowMs) ? 'failed' : 'pendingPayment';
  }

  if (lotEndsAtPassed(lot, nowMs)) return lotHasBid(lot) ? 'pendingPayment' : 'failed';

  if (
    lot.status === LOT_STATUS.READY ||
    lot.status === LOT_STATUS.QUEUED
  ) {
    return 'upcoming';
  }

  return 'syncing';
}
