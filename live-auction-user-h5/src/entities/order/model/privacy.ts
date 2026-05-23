import type { OrderSummary } from '../../../shared/api/types';

export function ownOrderForLot(order: OrderSummary | null | undefined, meId: string, lotId?: string): OrderSummary | null {
  if (!order || !meId || order.buyerUserId !== meId) return null;
  if (lotId && order.lotId && order.lotId !== lotId) return null;
  return order;
}

export function canPayOrder(order: OrderSummary | null | undefined): boolean {
  return Boolean(order && order.status !== 'PAID' && order.paymentStatus !== 'SUCCESS');
}
