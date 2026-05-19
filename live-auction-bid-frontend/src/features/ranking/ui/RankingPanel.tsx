import { Trophy } from 'lucide-react';
import { formatMoney } from '../../../shared/lib/money';
import type { RankingItem } from '../../../shared/types/auction';

export function RankingPanel({ items }: { items: RankingItem[] }) {
  return (
    <aside className="card">
      <h3><Trophy size={18} /> 实时排名</h3>
      <ol className="ranking">
        {items.map((x) => <li key={x.userId}><span>#{x.rank} {x.nickname}</span><strong>{formatMoney(x.amount)}</strong></li>)}
      </ol>
      {items.length === 0 && <p className="meta">暂无出价，等待第一位观众举牌。</p>}
    </aside>
  );
}
