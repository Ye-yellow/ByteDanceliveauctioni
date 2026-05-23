import { isBiddableLotStatus, isClosedLotStatus } from '../../../entities/auction/model/status';
import type { Lot } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';
import { useServerCountdown } from '../hooks/useServerCountdown';

type CurrentLotCardProps = {
  lot: Lot;
  serverTimeUnixMs?: number | string;
  serverTimeReceivedAtUnixMs?: number;
};

export function CurrentLotCard({ lot, serverTimeUnixMs, serverTimeReceivedAtUnixMs }: CurrentLotCardProps) {
  const countdown = useServerCountdown(lot.endsAtUnixMs, serverTimeUnixMs, serverTimeReceivedAtUnixMs);
  const open = isBiddableLotStatus(lot.status);
  const closed = isClosedLotStatus(lot.status);

  return (
    <section className="lotCard">
      <div className="lotMedia">
        {lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <span>拍品图待同步</span>}
      </div>
      <div className="lotInfo">
        <span className="statusPill">{lot.status}</span>
        <h1>{lot.title}</h1>
        <p>{lot.description || '等待主播同步拍品介绍'}</p>
      </div>
      <div className="priceGrid">
        <div>
          <span>当前价</span>
          <b>{formatMoney(lot.currentPrice || lot.rule.startPrice)}</b>
        </div>
        <div>
          <span>倒计时</span>
          <b className={countdown.danger ? 'dangerText pulse' : ''}>
            {open ? countdown.text : closed ? '竞拍结束' : '等待开拍'}
          </b>
        </div>
        <div>
          <span>起拍价</span>
          <b>{formatMoney(lot.rule.startPrice)}</b>
        </div>
        <div>
          <span>加价幅度</span>
          <b>{formatMoney(lot.rule.minIncrement)}</b>
        </div>
        <div>
          <span>封顶价</span>
          <b>{lot.rule.capPrice ? formatMoney(lot.rule.capPrice) : '未设置'}</b>
        </div>
        <div>
          <span>参与 / 出价</span>
          <b>
            {lot.participantCount || 0} / {lot.bidCount || 0}
          </b>
        </div>
      </div>
      <small className="timeHint">
        {countdown.fallback ? '本地 fallback 计时，等待服务端时间校准' : '服务端时间已校准'}
      </small>
    </section>
  );
}
