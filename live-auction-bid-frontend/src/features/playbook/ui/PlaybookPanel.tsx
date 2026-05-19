import { Sparkles } from 'lucide-react';
import type { Lot } from '../../../shared/types/auction';

export function PlaybookPanel({ lot }: { lot: Lot }) {
  return (
    <aside className="card ai">
      <h3><Sparkles size={18} /> 玩法引擎助手</h3>
      <p>当前阶段：{lot.playbookStage.replace('PLAYBOOK_STAGE_', '')}</p>
      <small>V1 先由主播手动揭示信任卡片/启动 Duel，AI 作为后续旁路助手。</small>
      <div className="trustCards">
        {(lot.trustCards ?? []).map((card) => (
          <div className={card.revealed ? 'trustCard revealed' : 'trustCard'} key={card.id}>
            <strong>{card.revealed ? '已揭示' : '未揭示'} · {card.title}</strong>
            <span>{card.revealed ? card.content : '等待主播揭示'}</span>
          </div>
        ))}
      </div>
    </aside>
  );
}
