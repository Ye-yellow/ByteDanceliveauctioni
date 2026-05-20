import { useState } from 'react';
import { Radio } from 'lucide-react';
import { API_BASE, WS_BASE } from '../../shared/config/env';
import { StatusPill } from '../../shared/ui/StatusPill';
import { LotCard } from '../../features/auction/ui/LotCard';
import { RankingPanel } from '../../features/ranking/ui/RankingPanel';
import { PlaybookPanel } from '../../features/playbook/ui/PlaybookPanel';
import { AuthPanel } from '../../features/auth/ui/AuthPanel';
import { currentAuth } from '../../features/auth/api/authApi';
import { useAuctionRoom } from '../../features/realtime/useAuctionRoom';
import type { User } from '../../shared/types/auction';

export function LiveRoomPage() {
  const [user, setUser] = useState<User | null>(() => currentAuth().user);
  const { lot, ranking, connected, error, notice, placeBid } = useAuctionRoom('demo', user);
  if (!lot) {
    return (
      <main className="page">
        <section className="grid single">
          <AuthPanel mode="buyer" onUserChange={setUser} />
          <div className="card"><h2>等待真实后端拍品数据</h2><p>{error ?? '请先打开主播端创建并开拍。'}</p><a href="/host">去主播端</a></div>
        </section>
      </main>
    );
  }
  return (
    <main className="page">
      <section className="hero"><div><p className="eyebrow"><Radio size={16} /> LIVE AUCTION ROOM / DEMO</p><h1>短视频直播互动竞拍引擎</h1><p>围绕非标品信任揭示、群体共振和双人巅峰竞拍，构建直播原生交易新玩法。</p><p><a href="/host">主播控制台</a></p></div><StatusPill ok={connected} okText="WebSocket 已连接" pendingText="连接真实后端中..." /></section>
      {(notice || error) && <section className={error ? 'notice error' : 'notice'}>{error || notice}</section>}
      {lot.status === 'LOT_STATUS_CANCELLED' && <section className="notice error">竞拍已取消{lot.cancelReason ? `：${lot.cancelReason}` : '，请等待主播后续安排。'}</section>}
      <section className="grid"><LotCard lot={lot} onBid={placeBid} /><RankingPanel items={ranking} /><AuthPanel mode="buyer" onUserChange={setUser} /><PlaybookPanel lot={lot} /></section>
      <footer>Backend: {API_BASE} · WS: {WS_BASE}</footer>
    </main>
  );
}
