import type { OrderSummary } from '../../../shared/api/types';

export const ORDER_PAYMENT_WINDOW_MS = 30 * 60 * 1000;

export function maskPublicBuyerName(value?: string, fallback = '***'): string {
  const normalized = value?.trim().replace(/\*/g, '') || '';
  const visibleChar = Array.from(normalized)[0];
  return visibleChar ? `${visibleChar}***` : fallback;
}

export function ownOrderForLot(order: OrderSummary | null | undefined, meId: string, lotId?: string): OrderSummary | null {
  if (!order || !meId || order.buyerUserId !== meId) return null;
  if (lotId && order.lotId && order.lotId !== lotId) return null;
  return order;
}

export function orderExpiresAtPassed(order: OrderSummary | null | undefined, nowMs = Date.now()): boolean {
  const expiresAt = Number(order?.expiresAtUnixMs || 0);
  return Number.isFinite(expiresAt) && expiresAt > 0 && expiresAt <= nowMs;
}

export function isOrderPaid(order: OrderSummary | null | undefined): boolean {
  if (!order) return false;
  const status = String(order.status || '').toUpperCase();
  const paymentStatus = String(order.paymentStatus || '').toUpperCase();
  return paymentStatus === 'SUCCESS' || status === 'PAID';
}

export function isOrderFailed(order: OrderSummary | null | undefined, nowMs = Date.now()): boolean {
  if (!order) return false;
  const status = String(order.status || '').toUpperCase();
  const paymentStatus = String(order.paymentStatus || '').toUpperCase();
  return orderExpiresAtPassed(order, nowMs) ||
    ['CANCELLED', 'EXPIRED', 'REFUNDED'].includes(status) ||
    ['FAILED', 'CLOSED'].includes(paymentStatus);
}

export function isOrderPaying(order: OrderSummary | null | undefined, nowMs = Date.now()): boolean {
  if (!order) return false;
  return !isOrderPaid(order) && !isOrderFailed(order, nowMs);
}

export function canPayOrder(order: OrderSummary | null | undefined): boolean {
  if (!order) return false;
  return !isOrderPaid(order) && !isOrderFailed(order);
}

export function orderStatusLabel(order: OrderSummary | null | undefined): string {
  if (!order) return '订单同步中';
  const status = String(order.status || '').toUpperCase();
  const paymentStatus = String(order.paymentStatus || '').toUpperCase();
  if (isOrderPaid(order)) return '已支付';
  if (paymentStatus === 'FAILED') return '支付失败';
  if (paymentStatus === 'CLOSED') return '支付已关闭';
  if (orderExpiresAtPassed(order)) return '订单已过期';
  if (status === 'CANCELLED') return '订单已取消';
  if (status === 'EXPIRED') return '订单已过期';
  if (status === 'REFUNDED') return '已退款';
  if (paymentStatus === 'PROCESSING') return '支付处理中';
  return '待支付';
}

export function orderPaymentStatusLabel(order: OrderSummary | null | undefined): string {
  if (!order) return '支付同步中';
  const paymentStatus = String(order.paymentStatus || '').toUpperCase();
  if (isOrderPaid(order)) return '支付成功';
  if (paymentStatus === 'FAILED') return '支付失败';
  if (paymentStatus === 'CLOSED') return '支付已关闭';
  if (paymentStatus === 'PROCESSING') return '支付处理中';
  if (orderExpiresAtPassed(order)) return '支付已超时';
  return '待支付';
}

export function orderStatusTone(order: OrderSummary | null | undefined): 'danger' | 'paid' | 'pending' | 'processing' {
  if (!order) return 'pending';
  const paymentStatus = String(order.paymentStatus || '').toUpperCase();
  if (isOrderPaid(order)) return 'paid';
  if (isOrderFailed(order)) return 'danger';
  if (paymentStatus === 'PROCESSING') return 'processing';
  return 'pending';
}
