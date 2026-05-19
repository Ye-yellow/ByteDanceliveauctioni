import { API_BASE } from '../../../shared/config/env';
import type { CreateLotRequest, Lot, PlaceBidRequest, RoomSnapshot, TrustRevealCard } from '../../../shared/api/types';

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const r = await fetch(`${API_BASE}${url}`, { ...init, headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) } });
  if (!r.ok) throw new Error(await r.text());
  return r.json() as Promise<T>;
}

export async function listLots(roomId = 'demo'): Promise<Lot[]> {
  return request<Lot[]>(`/api/lots?roomId=${encodeURIComponent(roomId)}`);
}
export async function createLot(payload: CreateLotRequest) {
  const r = await request<{ lot: Lot }>('/api/lots', { method: 'POST', body: JSON.stringify(payload) });
  return r.lot;
}
export async function startLot(lotId: string) { return (await request<{ lot: Lot }>(`/api/lots/${lotId}/start`, { method: 'POST' })).lot; }
export async function placeBid(lotId: string, payload: PlaceBidRequest) { return request<{ accepted: boolean; lot: Lot; rejectReason?: string }>(`/api/lots/${lotId}/bid`, { method: 'POST', body: JSON.stringify(payload) }); }
export async function revealTrustCard(lotId: string, cardId: string) { return request<{ lot: Lot; trustCard: TrustRevealCard }>(`/api/lots/${lotId}/trust-cards/${cardId}/reveal`, { method: 'POST' }); }
export async function startDuel(lotId: string) { return (await request<{ lot: Lot }>(`/api/lots/${lotId}/duel`, { method: 'POST' })).lot; }
export async function settleLot(lotId: string) { return (await request<{ lot: Lot }>(`/api/lots/${lotId}/settle`, { method: 'POST' })).lot; }
export async function getRoomSnapshot(roomId = 'demo') { return (await request<{ snapshot: RoomSnapshot }>(`/api/rooms/${roomId}/snapshot`)).snapshot; }
