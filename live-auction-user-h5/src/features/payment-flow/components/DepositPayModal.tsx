import type { Lot } from '../../../shared/api/types';
import { formatLotDeposit } from '../../../entities/auction/model/deposit';

export function DepositPayModal({
  lot,
  onConfirm,
  onClose,
}: {
  lot: Lot;
  onConfirm: () => void;
  onClose: () => void;
}) {
  return (
    <div className="modalMask paySheetMask">
      <section className="payModal depositModal" aria-modal="true" role="dialog">
        <button className="modalClose" onClick={onClose} aria-label="关闭保证金弹窗">
          ×
        </button>
        <h2>支付保证金</h2>
        <p className="depositLotName">{lot.title}</p>
        <section className="depositAmount" aria-label="保证金金额">
          <span>保证金</span>
          <b className="scrollAmount" title={formatLotDeposit(lot)}>{formatLotDeposit(lot)}</b>
        </section>
        <p className="depositNote">首次参与该拍品竞价前需确认保证金，拍品付款后退回。</p>
        <button className="resultPrimaryButton" type="button" onClick={onConfirm}>
          确认支付保证金
        </button>
      </section>
    </div>
  );
}
