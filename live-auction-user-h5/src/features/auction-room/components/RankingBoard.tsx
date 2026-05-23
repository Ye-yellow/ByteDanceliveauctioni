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
        ranking.slice(0, 5).map((item) => (
          <div key={item.userId} className={`rankRow ${item.userId === meId ? 'me' : ''}`}>
            <span>#{item.rank}</span>
            <div>
              <b>{item.nickname || item.userId}</b>
              <small>
                {item.rank === 1 ? '当前第一' : `差距 ${formatMoney(Math.max(0, topAmount - moneyNumber(item.amount)))}`}
              </small>
            </div>
            <strong>{formatMoney(item.amount)}</strong>
          </div>
        ))
      ) : (
        <p className="emptyText">等待第一笔有效出价</p>
      )}
    </section>
  );
}
