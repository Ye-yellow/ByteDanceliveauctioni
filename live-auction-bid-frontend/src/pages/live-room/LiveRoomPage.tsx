import { Radio } from 'lucide-react';
import { API_BASE, WS_BASE } from '../../shared/config/env';
import { StatusPill } from '../../shared/ui/StatusPill';
import { LotCard } from '../../features/auction/ui/LotCard';
import { RankingPanel } from '../../features/ranking/ui/RankingPanel';
import { PlaybookPanel } from '../../features/playbook/ui/PlaybookPanel';
import { useAuctionRoom } from '../../features/realtime/useAuctionRoom';

export function LiveRoomPage() {
  const { lot, connected, error, placeBid } = useAuctionRoom('demo');

  if (!lot) {
    return <main className="page"><div className="card"><h2>等待真实后端拍品数据</h2><p>{error ?? '请先启动 auction-backend。'}</p></div></main>;
  }

  return (
    <main className="page">
      <section className="hero">
        <div>
          <p className="eyebrow"><Radio size={16} /> LIVE AUCTION ROOM / DEMO</p>
          <h1>短视频直播互动竞拍引擎</h1>
          <p>围绕非标品信任揭示、群体共振和双人巅峰竞拍，构建直播原生交易新玩法。</p>
        </div>
        <StatusPill ok={connected} okText="WebSocket 已连接" pendingText="连接真实后端中..." />
      </section>
      <section className="grid">
        <LotCard lot={lot} onBid={placeBid} />
        <RankingPanel items={lot.ranking ?? []} />
        <PlaybookPanel lot={lot} />
      </section>
      <footer>Backend: {API_BASE} · WS: {WS_BASE}</footer>
    </main>
  );
}
