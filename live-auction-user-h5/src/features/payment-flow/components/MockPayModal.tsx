import { useEffect, useMemo, useState } from 'react';
import { mockPay } from '../../auction/api/auctionApi';
import type { OrderSummary, PaymentSummary } from '../../../shared/api/types';
import { createIdempotencyKey } from '../../../shared/lib/idempotency';
import { formatMoney } from '../../../shared/lib/money';

export function MockPayModal({
  order,
  onStartPayment,
  onPaid,
  onClose,
}: {
  order: OrderSummary;
  onStartPayment: (orderId: string, idempotencyKey: string) => void;
  onPaid: (order?: OrderSummary, payment?: PaymentSummary) => Promise<void> | void;
  onClose: () => void;
}) {
  const [method, setMethod] = useState('模拟余额');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ orderId: string; text: string } | null>(null);
  const [paidOrderId, setPaidOrderId] = useState<string | null>(order.paymentStatus === 'SUCCESS' ? order.id : null);
  const settled = paidOrderId === order.id || order.paymentStatus === 'SUCCESS';
  const idempotencyKey = useMemo(
    () => createIdempotencyKey('pay', order.id, order.amount.amount),
    [order.amount.amount, order.id],
  );

  useEffect(() => {
    if (!settled) return undefined;
    const timer = window.setTimeout(onClose, message?.orderId === order.id ? 900 : 500);
    return () => window.clearTimeout(timer);
  }, [message?.orderId, onClose, order.id, settled]);

  const pay = async () => {
    if (loading) return;
    setLoading(true);
    setMessage(null);
    onStartPayment(order.id, idempotencyKey);
    try {
      const result = await mockPay(order.id, { idempotencyKey, amount: order.amount });
      await onPaid(result.order, result.payment);
      if (result.paid || result.order?.paymentStatus === 'SUCCESS' || result.payment?.status === 'SUCCESS') setPaidOrderId(order.id);
      setMessage({ orderId: order.id, text: result.paid ? '支付成功，订单已刷新' : result.message || '支付状态已同步' });
    } catch (e) {
      setMessage({ orderId: order.id, text: e instanceof Error ? e.message : '模拟支付失败，请重试' });
      await onPaid(undefined, undefined);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modalMask paySheetMask">
      <section className="payModal" aria-modal="true" role="dialog">
        <button className="modalClose" onClick={onClose} aria-label="关闭支付弹窗">
          ×
        </button>
        <h2>模拟支付</h2>
        <p>
          订单号 <b>{order.id}</b>
        </p>
        <p>
          订单金额 <b className="scrollAmount inlineAmount" title={formatMoney(order.amount)}>{formatMoney(order.amount)}</b>
        </p>
        <div className="payMethods">
          {['模拟余额', '模拟银行卡'].map((item) => (
            <button key={item} type="button" className={method === item ? 'active' : ''} onClick={() => setMethod(item)}>
              {item}
            </button>
          ))}
        </div>
        <button className="bidButton" type="button" disabled={loading || settled} onClick={pay}>
          {loading ? '支付中...' : settled ? '已支付' : '确认模拟支付'}
        </button>
        {message?.orderId === order.id ? <p className="payResult" role="status">{message.text}</p> : null}
      </section>
    </div>
  );
}
