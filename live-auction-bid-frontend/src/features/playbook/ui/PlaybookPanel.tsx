import { Sparkles } from 'lucide-react';
import type { Lot, PlaybookStage, TrustRevealCard } from '../../../shared/types/auction';

const demoCards: TrustRevealCard[] = [
  { id: 'cert', type: 'CERTIFICATE', title: '证书信息', content: '等待后端玩法引擎推送证书揭示事件。', revealed: false },
  { id: 'flaw', type: 'FLAW', title: '瑕疵说明', content: '等待后端玩法引擎推送瑕疵揭示事件。', revealed: false },
];

export function PlaybookPanel({ lot, stage = 'WARM_UP' }: { lot: Lot; stage?: PlaybookStage }) {
  return (
    <aside className="card ai">
      <h3><Sparkles size={18} /> 玩法引擎助手</h3>
      <p>{lot.atmosphereText || '等待玩法引擎根据直播互动、出价热度和信任阻塞点推送控场建议。'}</p>
      <small>当前阶段：{stage}。后续接 Trust-Reveal / Crowd-Powered / Duel Auction 事件。</small>
      <div className="trustCards">
        {demoCards.map((card) => <div className="trustCard" key={card.id}><strong>{card.title}</strong><span>{card.content}</span></div>)}
      </div>
    </aside>
  );
}
