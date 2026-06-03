import { useEffect, useState } from 'react';
import { canPayOrder, isOrderFailed, maskPublicBuyerName, ORDER_PAYMENT_WINDOW_MS, orderStatusLabel } from '../../../entities/order/model/privacy';
import type { Lot, OrderSummary } from '../../../shared/api/types';
import { formatMoney, moneyNumber } from '../../../shared/lib/money';

function paymentStatusLabel(order?: OrderSummary | null): string {
  if (!order) return '订单生成中';
  return orderStatusLabel(order);
}

function winnerButtonLabel(order: OrderSummary | null | undefined, canPay: boolean): string {
  if (!order) return '等待订单同步';
  if (canPay) return '确认地址并支付';
  return paymentStatusLabel(order);
}

function canStartPayment(order?: OrderSummary | null): boolean {
  if (!canPayOrder(order)) return false;
  const status = String(order?.status || '').toUpperCase();
  const paymentStatus = String(order?.paymentStatus || '').toUpperCase();
  return !['CANCELLED', 'EXPIRED', 'REFUNDED'].includes(status) && paymentStatus !== 'CLOSED';
}

function avatarText(label: string): string {
  return Array.from(label.replace(/\*/g, ''))[0] || '中';
}

function lotHasBid(lot: Lot): boolean {
  const start = moneyNumber(lot.rule.startPrice);
  const current = moneyNumber(lot.currentPrice);
  return Boolean(lot.leadingUserId || lot.winnerUserId || lot.stats?.bidCount || (current > 0 && current > start));
}

function roundsText(lot: Lot): string {
  const rounds = Number(lot.stats?.bidCount || 0);
  if (!Number.isFinite(rounds) || rounds <= 0) return '经过多轮激烈竞拍成功拍下';
  return `经过 ${rounds} 轮激烈竞拍成功拍下`;
}

function fallbackPaymentStart(order: OrderSummary | null | undefined, lot: Lot): number {
  const created = Number(order?.createdAtUnixMs || 0);
  if (Number.isFinite(created) && created > 0) return created;
  const settled = Number(lot.settledAtUnixMs || 0);
  if (Number.isFinite(settled) && settled > 0) return settled;
  return 0;
}

function paymentLeftMs(order: OrderSummary | null | undefined, lot: Lot, nowMs: number, fallbackStartAt: number): number {
  const expires = Number(order?.expiresAtUnixMs || 0);
  if (Number.isFinite(expires) && expires > 0) return Math.max(0, expires - nowMs);

  const start = fallbackPaymentStart(order, lot) || fallbackStartAt;
  return Math.max(0, start + ORDER_PAYMENT_WINDOW_MS - nowMs);
}

function formatCountdown(leftMs: number): string {
  const totalSeconds = Math.max(0, Math.ceil(leftMs / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${String(seconds).padStart(2, '0')}`;
}

function createPaymentTimer(key: string, order: OrderSummary | null | undefined, lot: Lot) {
  const now = Date.now();
  return {
    key,
    fallbackStartAt: fallbackPaymentStart(order, lot) || now,
    nowMs: now,
  };
}

function resultAmount(order: OrderSummary | null | undefined, lot: Lot) {
  if (moneyNumber(order?.amount) > 0) return order?.amount;
  if (moneyNumber(lot.finalPrice) > 0) return lot.finalPrice;
  if (moneyNumber(lot.currentPrice) > 0) return lot.currentPrice;
  return lot.rule.startPrice;
}

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
  const hasWinningClaim = Boolean(order) || (Boolean(meId) && lot.winnerUserId === meId);
  const finalAmount = resultAmount(order, lot);
  const timerKey = `${lot.id}:${order?.id || 'pending'}`;
  const [paymentTimer, setPaymentTimer] = useState(() => createPaymentTimer(timerKey, order, lot));
  const fallbackStartAt = paymentTimer.key === timerKey ? paymentTimer.fallbackStartAt : fallbackPaymentStart(order, lot) || paymentTimer.nowMs;
  const nowMs = paymentTimer.nowMs;
  const payLeftMs = paymentLeftMs(order, lot, nowMs, fallbackStartAt);
  const failedClaim = hasWinningClaim && (Boolean(order && isOrderFailed(order, nowMs)) || (!order && payLeftMs <= 0));
  const won = hasWinningClaim && !failedClaim;
  const canPay = won && canStartPayment(order);
  const noSuccessfulBid = !won && !failedClaim && !lotHasBid(lot);
  const maskedWinnerNickname = maskPublicBuyerName(lot.winnerNickname || order?.buyerNickname, '中标者');
  const resultProfileLabel = failedClaim || noSuccessfulBid ? '竞拍未成交' : '竞拍成功者';
  const resultProfileText = noSuccessfulBid ? '无人出价' : failedClaim ? '付款超时' : maskedWinnerNickname;
  const resultStory = noSuccessfulBid ? '本轮无人出价，拍品未成交' : failedClaim ? '已落锤但未在时限内完成付款' : roundsText(lot);
  const finalPriceLabel = noSuccessfulBid ? '起拍价' : failedClaim ? '落锤价' : '最终价';
  const loseRibbonText = noSuccessfulBid ? '本轮流拍' : failedClaim ? '成交失效' : '恭喜成交';

  useEffect(() => {
    if (!hasWinningClaim) return undefined;

    const timer = window.setInterval(() => {
      setPaymentTimer((current) => {
        const now = Date.now();
        const nextFallbackStart = current.key === timerKey ? current.fallbackStartAt : fallbackPaymentStart(order, lot) || now;
        return { key: timerKey, fallbackStartAt: nextFallbackStart, nowMs: now };
      });
    }, 1000);

    return () => window.clearInterval(timer);
  }, [hasWinningClaim, lot, order, timerKey]);

  return (
    <div className="modalMask resultModalMask">
      <section className={`resultModal ${won ? 'resultModalWin' : 'resultModalLose'}`} aria-modal="true" role="dialog">
        <button className="modalClose" type="button" onClick={onClose} aria-label="关闭结果弹窗">
          ×
        </button>

        {won ? (
          <>
            <div className="resultCeremony resultCeremonyWin" aria-label="竞拍成功">
              <span>豪气冲天</span>
              <strong>恭喜竞拍成功</strong>
            </div>
            <div className="resultWinMedia">
              {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <span>{lot.title.slice(0, 1) || '拍'}</span>}
              <span className="resultWinBadge">竞拍成功</span>
            </div>
            <h2 className="resultTitle">{lot.title}</h2>
            <p className="resultWinSummary">
              成交价 <b className="scrollAmount" title={formatMoney(finalAmount)}>{formatMoney(finalAmount)}</b>
            </p>
            <section className="resultDeposit" aria-label="保证金">
              <span>保证金</span>
              <b>拍品付款后退回</b>
            </section>
            <button
              className="resultPrimaryButton"
              type="button"
              disabled={!order || !canPay}
              onClick={order && canPay ? () => onPay(order) : undefined}
            >
              {winnerButtonLabel(order, canPay)}
            </button>
            <p className="resultPayCountdown">
              {canPay ? `支付倒计时 ${formatCountdown(payLeftMs)}` : paymentStatusLabel(order)}
            </p>
          </>
        ) : (
          <>
            <div className={`resultCeremony resultCeremonyLose ${noSuccessfulBid || failedClaim ? 'isMuted' : ''}`} aria-label="竞拍结果">
              <span>落槌定音</span>
              <strong>{loseRibbonText}</strong>
            </div>
            <div className="resultLoseProduct">
              <div className="resultLoseProductMedia">
                {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <span>{lot.title.slice(0, 1) || '拍'}</span>}
              </div>
              <div>
                <span>本场拍品</span>
                <b>{lot.title}</b>
              </div>
            </div>
            <div className="resultWinnerProfile">
              <div className="resultWinnerAvatar">{avatarText(resultProfileText)}</div>
              <div>
                <span>{resultProfileLabel}</span>
                <b>{resultProfileText}</b>
              </div>
            </div>
            <p className="resultLoserStory">
              {resultStory}
            </p>
            <section className="resultFinalPrice" aria-label="结果价格">
              <span>{finalPriceLabel}</span>
              <strong className="scrollAmount" title={formatMoney(finalAmount)}>{formatMoney(finalAmount)}</strong>
            </section>
            <button className="resultPrimaryButton" type="button" onClick={onNext}>
              继续看直播
            </button>
          </>
        )}
      </section>
    </div>
  );
}
