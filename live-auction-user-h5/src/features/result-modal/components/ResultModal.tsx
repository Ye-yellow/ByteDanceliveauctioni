import { useEffect, useState } from 'react';
import { canPayOrder, isOrderFailed, maskPublicBuyerName, ORDER_PAYMENT_WINDOW_MS, orderStatusLabel } from '../../../entities/order/model/privacy';
import type { Lot, OrderSummary } from '../../../shared/api/types';
import { formatMoney, moneyNumber } from '../../../shared/lib/money';

function GavelIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <path d="m14 13-7.2 7.2a1.6 1.6 0 0 1-2.3-2.3L11.7 11"></path>
      <path d="m16 16 5-5"></path>
      <path d="m20.5 10.5-7-7"></path>
      <path d="m8.5 7.5 5-5"></path>
      <path d="m8 8 8 8"></path>
    </svg>
  );
}

function CrownIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <path d="m3.8 8.2 4.4 4.1L12 5.2l3.8 7.1 4.4-4.1-1.7 10.3h-13L3.8 8.2Z"></path>
      <path d="M6.2 20h11.6"></path>
    </svg>
  );
}

function ShieldIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 3.4c2 1.6 4.2 2.5 6.7 2.7v5.5c0 4.4-2.6 7.2-6.7 8.9-4.1-1.7-6.7-4.5-6.7-8.9V6.1c2.5-.2 4.7-1.1 6.7-2.7Z"></path>
      <path d="m8.9 12 2.1 2.1 4.2-4.4"></path>
    </svg>
  );
}

function ClockIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="8.2"></circle>
      <path d="M12 7.8v4.7l3.1 1.9"></path>
    </svg>
  );
}

function ArrowIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <path d="M5 12h12.5"></path>
      <path d="m13 7.5 4.5 4.5-4.5 4.5"></path>
    </svg>
  );
}

function UserIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="8.4" r="3.5"></circle>
      <path d="M5.5 19.2c1.2-3.1 3.3-4.6 6.5-4.6s5.3 1.5 6.5 4.6"></path>
    </svg>
  );
}

function BrokenHeartIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 20.2S4.4 15.7 4.4 9.5A4.2 4.2 0 0 1 12 7a4.2 4.2 0 0 1 7.6 2.5c0 6.2-7.6 10.7-7.6 10.7Z"></path>
      <path d="m12.8 6.9-2 3.3 3.2 1.5-2.3 4"></path>
    </svg>
  );
}

function paymentStatusLabel(order?: OrderSummary | null): string {
  if (!order) return '订单生成中';
  return orderStatusLabel(order);
}

function winnerButtonLabel(order: OrderSummary | null | undefined, canPay: boolean): string {
  if (!order) return '等待订单同步';
  if (canPay) return '确认地址并完成付款';
  return paymentStatusLabel(order);
}

function canStartPayment(order?: OrderSummary | null): boolean {
  if (!canPayOrder(order)) return false;
  const status = String(order?.status || '').toUpperCase();
  const paymentStatus = String(order?.paymentStatus || '').toUpperCase();
  return !['CANCELLED', 'EXPIRED', 'REFUNDED'].includes(status) && paymentStatus !== 'CLOSED';
}

function lotHasBid(lot: Lot): boolean {
  const start = moneyNumber(lot.rule.startPrice);
  const current = moneyNumber(lot.currentPrice);
  return Boolean(lot.leadingUserId || lot.winnerUserId || lot.stats?.bidCount || (current > 0 && current > start));
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
  const resultProfileLabel = failedClaim || noSuccessfulBid ? '竞拍未成交' : '最终竞得者';
  const resultProfileText = noSuccessfulBid ? '无人出价' : failedClaim ? '付款超时' : maskedWinnerNickname;
  const finalPriceLabel = noSuccessfulBid ? '起拍价' : failedClaim ? '落槌价' : '最终落槌价';
  const missedStatusText = noSuccessfulBid ? '本轮已结束' : failedClaim ? '成交已失效' : '本轮已落槌';
  const missedTitleLead = noSuccessfulBid ? '本轮' : failedClaim ? '成交' : '差点就是';
  const missedTitleAccent = noSuccessfulBid ? '流拍' : failedClaim ? '失效' : '你';
  const missedSubcopy = noSuccessfulBid
    ? '本轮无人出价，拍品暂未成交'
    : failedClaim
      ? '已落锤但未在时限内完成付款'
      : '最后时刻被反超，这件拍品与你擦肩而过';
  const missedEmotionLead = noSuccessfulBid ? '这轮还' : failedClaim ? '这次错过' : '这次只晚了';
  const missedEmotionAccent = noSuccessfulBid ? '没人举牌' : failedClaim ? '付款' : '一步';
  const missedEmotionSubcopy = noSuccessfulBid ? '下一件可以先观察再出手' : failedClaim ? '下一件别再错过' : '下一件别再犹豫';

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
            <div className="resultStatusPill resultStatusWin">
              <CrownIcon className="resultMiniIcon" />
              <span>竞拍成功</span>
            </div>
            <div className="resultHeroCopy">
              <h2 className="resultHeroTitle">
                <span>落槌</span><em>归你</em>
                <GavelIcon className="resultDecorIcon" />
              </h2>
              <p>最后一锤已定，这件拍品已为你锁定</p>
            </div>
            <div className="resultWinMedia">
              {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <span>{lot.title.slice(0, 1) || '拍'}</span>}
              <span className="resultWinBadge"><CrownIcon className="resultMiniIcon" />已竞得</span>
            </div>
            <section className="resultProductName" aria-label="竞得拍品">
              <span>你的拍品</span>
              <h3>{lot.title}</h3>
            </section>
            <section className="resultPricePanel resultPriceWin" aria-label="落槌价格">
              <span>你的落槌价</span>
              <strong className="scrollAmount" title={formatMoney(finalAmount)}>{formatMoney(finalAmount)}</strong>
            </section>
            <section className="resultDeposit" aria-label="保证金">
              <span><ShieldIcon className="resultMiniIcon" />保证金</span>
              <b>付款完成后原路退回</b>
            </section>
            <button
              className="resultPrimaryButton"
              type="button"
              disabled={!order || !canPay}
              onClick={order && canPay ? () => onPay(order) : undefined}
            >
              {winnerButtonLabel(order, canPay)}
              {canPay ? <ArrowIcon className="resultButtonIcon" /> : null}
            </button>
            <p className="resultPayCountdown">
              <ClockIcon className="resultMiniIcon" />
              {canPay ? <>支付剩余 <b>{formatCountdown(payLeftMs)}</b> ｜ 逾期视为放弃</> : paymentStatusLabel(order)}
            </p>
          </>
        ) : (
          <>
            <div className="resultStatusPill resultStatusMissed">
              <GavelIcon className="resultMiniIcon" />
              <span>{missedStatusText}</span>
            </div>
            <div className="resultHeroCopy resultMissedHero">
              <h2 className="resultHeroTitle">
                <span>{missedTitleLead}</span><em>{missedTitleAccent}</em>
                <BrokenHeartIcon className="resultDecorIcon" />
              </h2>
              <p>{missedSubcopy}</p>
            </div>
            <div className="resultMissedProductCard">
              <div className="resultMissedProductMedia">
                {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <span>{lot.title.slice(0, 1) || '拍'}</span>}
              </div>
              <div className="resultMissedProductInfo">
                <span>{noSuccessfulBid || failedClaim ? '本场拍品' : '你错过的拍品'}</span>
                <h3>{lot.title}</h3>
                <i aria-hidden="true"></i>
                <div className="resultMissedWinner">
                  <UserIcon className="resultUserIcon" />
                  <div>
                    <span>{resultProfileLabel}</span>
                    <b>{resultProfileText}</b>
                  </div>
                </div>
              </div>
            </div>
            <div className="resultMissedEmotion">
              <span><BrokenHeartIcon className="resultMiniIcon" /></span>
              <p>{missedEmotionLead}<em>{missedEmotionAccent}</em></p>
              <small>{missedEmotionSubcopy}</small>
            </div>
            <section className="resultPricePanel resultPriceMissed" aria-label="结果价格">
              <span>{finalPriceLabel}</span>
              <strong className="scrollAmount" title={formatMoney(finalAmount)}>{formatMoney(finalAmount)}</strong>
            </section>
            <button className="resultPrimaryButton" type="button" onClick={onNext}>
              {noSuccessfulBid || failedClaim ? '继续看直播' : '继续守下一件'}
            </button>
            <p className="resultPayCountdown resultMissedFooter">
              <ClockIcon className="resultMiniIcon" />
              {noSuccessfulBid ? '下一轮即将开始，留意主播上新' : '下一轮即将开始，别让喜欢的再擦肩'}
            </p>
          </>
        )}
      </section>
    </div>
  );
}
