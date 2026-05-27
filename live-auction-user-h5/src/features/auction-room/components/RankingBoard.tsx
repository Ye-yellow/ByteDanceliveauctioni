import { maskPublicBuyerName } from '../../../entities/order/model/privacy';
import type { RankingItem } from '../../../shared/api/types';
import { formatMoney, moneyNumber } from '../../../shared/lib/money';

export function RankingBoard({ ranking, meId }: { ranking: RankingItem[]; meId: string }) {
  const top = ranking[0];
  const topAmount = moneyNumber(top?.amount);
  const me = ranking.find((item) => item.userId === meId);

  return (
    <section className="rankingCard">
      <header>
        <h2>实时排行榜</h2>
        {me?.rank === 1 ? <b>你已领先</b> : me ? <b className="warn">你被超越</b> : null}
      </header>
      {ranking.length ? (
        ranking.slice(0, 5).map((item, index) => (
          <div key={item.userId || `${item.nickname}-${item.rank}-${index}`} className={`rankRow ${item.userId === meId ? 'me' : ''}`}>
            <span>#{item.rank}</span>
            <div>
              <b>{item.userId === meId ? item.nickname || '我' : maskPublicBuyerName(item.nickname || item.userId)}</b>
              <small>
                {item.rank === 1 ? '当前第一' : (
                  <>
                    差距 <span className="scrollAmount inlineAmount" title={formatMoney(Math.max(0, topAmount - moneyNumber(item.amount)))}>
                      {formatMoney(Math.max(0, topAmount - moneyNumber(item.amount)))}
                    </span>
                  </>
                )}
              </small>
            </div>
            <strong className="scrollAmount" title={formatMoney(item.amount)}>{formatMoney(item.amount)}</strong>
          </div>
        ))
      ) : (
        <p className="emptyText">等待第一笔有效出价</p>
      )}
    </section>
  );
}
