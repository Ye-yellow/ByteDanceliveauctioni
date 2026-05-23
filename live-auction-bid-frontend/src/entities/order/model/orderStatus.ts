import type { StudioTone } from '../../../pages/host-console/components/studio-ui';

export type OrderStatus = 'CREATED' | 'PENDING_PAYMENT' | 'PAID' | 'CANCELLED' | 'EXPIRED' | 'REFUNDED' | (string & {});
export type PaymentStatus = 'INIT' | 'PROCESSING' | 'SUCCESS' | 'FAILED' | 'CLOSED' | (string & {});

export const ORDER_STATUS_FILTERS: Array<{ label: string; value: OrderStatus | '' }> = [
  { label: '全部状态', value: '' },
  { label: '已创建', value: 'CREATED' },
  { label: '待支付', value: 'PENDING_PAYMENT' },
  { label: '已支付', value: 'PAID' },
  { label: '已取消', value: 'CANCELLED' },
  { label: '已过期', value: 'EXPIRED' },
  { label: '已退款', value: 'REFUNDED' },
];

const orderStatusMeta: Record<string, { label: string; tone: StudioTone }> = {
  CREATED: { label: '已创建', tone: 'info' },
  PENDING_PAYMENT: { label: '待支付', tone: 'warning' },
  PAID: { label: '已支付', tone: 'success' },
  CANCELLED: { label: '已取消', tone: 'danger' },
  EXPIRED: { label: '已过期', tone: 'danger' },
  REFUNDED: { label: '已退款', tone: 'neutral' },
};

const paymentStatusMeta: Record<string, { label: string; tone: StudioTone }> = {
  INIT: { label: '待支付', tone: 'warning' },
  PROCESSING: { label: '处理中', tone: 'info' },
  SUCCESS: { label: '支付成功', tone: 'success' },
  FAILED: { label: '支付失败', tone: 'danger' },
  CLOSED: { label: '已关闭', tone: 'danger' },
};

export function orderStatusLabel(status?: string | null) {
  return orderStatusMeta[status || '']?.label ?? status ?? '未知订单状态';
}

export function orderStatusTone(status?: string | null): StudioTone {
  return orderStatusMeta[status || '']?.tone ?? 'neutral';
}

export function paymentStatusLabel(status?: string | null) {
  return paymentStatusMeta[status || '']?.label ?? status ?? '未知支付状态';
}

export function paymentStatusTone(status?: string | null): StudioTone {
  return paymentStatusMeta[status || '']?.tone ?? 'neutral';
}

export function isAbnormalOrder(status?: string | null, paymentStatus?: string | null) {
  return ['CANCELLED', 'EXPIRED', 'REFUNDED'].includes(String(status || ''))
    || ['FAILED', 'CLOSED'].includes(String(paymentStatus || ''));
}
