export type Money = number;
export type Bid = { id: string; lotId: string; userId: string; nickname: string; amount: Money; createdAt: string };
export type RankingItem = { rank: number; userId: string; nickname: string; amount: Money; at: string };
export type Lot = {
  id: string; roomId: string; title: string; description: string; imageUrl: string;
  startPrice: Money; currentPrice: Money; minIncrement: Money; status: string; endsAt: string;
  winnerUserId?: string; version: number; bids: Bid[]; ranking?: RankingItem[]; atmosphereText?: string;
};

export const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080';
export const WS_BASE = import.meta.env.VITE_WS_BASE ?? API_BASE.replace(/^http/, 'ws');
export const formatMoney = (v: Money) => `¥${(v / 100).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;

export async function listLots(): Promise<Lot[]> {
  const r = await fetch(`${API_BASE}/api/lots`);
  return r.json();
}

export async function createLot(payload: Partial<Lot> & { durationSec?: number }) {
  const r = await fetch(`${API_BASE}/api/lots`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
  if (!r.ok) throw new Error(await r.text());
  return r.json() as Promise<Lot>;
}
