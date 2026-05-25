import type { OrderSummary } from '../../../shared/api/types';

export function ownOrderForLot(order: OrderSummary | null | undefined, meId: string, lotId?: string): OrderSummary | null {
  if (!order || !meId || order.buyerUserId !== meId) return null;
  if (lotId && order.lotId && order.lotId !== lotId) return null;
  return order;
}

export function canPayOrder(order: OrderSummary | null | undefined): boolean {
  if (!order) return false;
  const status = String(order.status || '').toUpperCase();
  const paymentStatus = String(order.paymentStatus || '').toUpperCase();
  if (['PAID', 'CANCELLED', 'EXPIRED', 'REFUNDED'].includes(status)) return false;
  if (['SUCCESS', 'CLOSED'].includes(paymentStatus)) return false;
  return true;
}

export function orderStatusLabel(order: OrderSummary | null | undefined): string {
  if (!order) return '订单同步中';
  const status = String(order.status || '').toUpperCase();
  const paymentStatus = String(order.paymentStatus || '').toUpperCase();
  if (paymentStatus === 'SUCCESS' || status === 'PAID') return '已支付';
  if (paymentStatus === 'PROCESSING') return '支付处理中';
  if (paymentStatus === 'FAILED') return '支付失败';
  if (paymentStatus === 'CLOSED') return '支付已关闭';
  if (status === 'CANCELLED') return '订单已取消';
  if (status === 'EXPIRED') return '订单已过期';
  if (status === 'REFUNDED') return '已退款';
  return '待支付';
}

export function orderStatusTone(order: OrderSummary | null | undefined): 'danger' | 'paid' | 'pending' | 'processing' {
  if (!order) return 'pending';
  const status = String(order.status || '').toUpperCase();
  const paymentStatus = String(order.paymentStatus || '').toUpperCase();
  if (paymentStatus === 'SUCCESS' || status === 'PAID') return 'paid';
  if (paymentStatus === 'PROCESSING') return 'processing';
  if (['FAILED', 'CLOSED'].includes(paymentStatus) || ['CANCELLED', 'EXPIRED', 'REFUNDED'].includes(status)) return 'danger';
  return 'pending';
}
