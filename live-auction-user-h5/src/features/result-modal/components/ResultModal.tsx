import { canPayOrder } from '../../../entities/order/model/privacy';
import type { Lot, OrderSummary } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';

export function ResultModal({
  lot,
  meId,
  order,
  onPay,
  onNext,
  onClose,
}: {
  lot: Lot;
  meId: string;
  order?: OrderSummary | null;
  onPay: (order: OrderSummary) => void;
  onNext: () => void;
  onClose: () => void;
}) {
  const won = lot.winnerUserId === meId || lot.leadingUserId === meId || Boolean(order);
  const canPay = won && canPayOrder(order);

  return (
    <div className="modalMask">
      <section className="resultModal" aria-modal="true" role="dialog">
        <button className="modalClose" onClick={onClose} aria-label="关闭结果弹窗">
          ×
        </button>
        {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : null}
        <p className="ceremony">竞拍结束</p>
        <h2>{lot.title}</h2>
        <div className="resultGrid">
          <span>
            成交价<b>{formatMoney(order?.amount || lot.finalPrice || lot.currentPrice)}</b>
          </span>
          <span>
            中标用户<b>{lot.winnerNickname || lot.leadingNickname || order?.buyerNickname || '待同步'}</b>
          </span>
          <span>
            订单号<b>{order?.id || (won ? '同步中' : '仅中标用户可见')}</b>
          </span>
          <span>
            支付状态<b>{order?.paymentStatus || (won ? '待生成' : '仅中标用户可见')}</b>
          </span>
        </div>
        <p>
          {won
            ? order
              ? '恭喜你中标，请完成模拟支付。'
              : '恭喜你中标，正在从后端同步订单。'
            : '很遗憾，本次未中标，可以继续关注下一件。'}
        </p>
        <button className="bidButton" type="button" disabled={won && !canPay} onClick={won && order ? () => onPay(order) : onNext}>
          {won ? (canPay ? '去模拟支付' : order?.paymentStatus === 'SUCCESS' ? '已支付' : '等待订单同步') : '查看下一件'}
        </button>
      </section>
    </div>
  );
}
