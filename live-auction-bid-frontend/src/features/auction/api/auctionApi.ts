import { API_BASE } from '../../../shared/config/env';
import { assertOkResult } from '../../../shared/api/result';
import { accessToken } from '../../auth/api/authStore';
import type { CancelLotReply, CreateLotReply, CreateLotRequest, GetRoomSnapshotReply, ListLotsReply, Lot, PlaceBidReply, PlaceBidRequest, RevealTrustCardReply, RoomSnapshot, SettleLotReply, StartDuelReply, StartLotReply, TrustRevealCard } from '../../../shared/api/types';

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const token = accessToken();
  const r = await fetch(`${API_BASE}${url}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers ?? {}),
    },
  });
  if (!r.ok) throw new Error(await r.text());
  return r.json() as Promise<T>;
}

function requireLot(reply: { lot?: Lot }) {
  if (!reply.lot) throw new Error('response missing lot');
  return reply.lot;
}

export async function listLots(roomId = 'demo'): Promise<Lot[]> {
  const reply = assertOkResult(await request<ListLotsReply>(`/api/lots?room_id=${encodeURIComponent(roomId)}`));
  return reply.lots ?? [];
}

export async function createLot(payload: CreateLotRequest) {
  return requireLot(assertOkResult(await request<CreateLotReply>('/api/lots', { method: 'POST', body: JSON.stringify(payload) })));
}

export async function startLot(lotId: string) {
  return requireLot(assertOkResult(await request<StartLotReply>(`/api/lots/${lotId}/start`, { method: 'POST', body: JSON.stringify({}) })));
}

export async function placeBid(lotId: string, payload: PlaceBidRequest) {
  return assertOkResult(await request<PlaceBidReply>(`/api/lots/${lotId}/bid`, { method: 'POST', body: JSON.stringify(payload) }));
}

export async function revealTrustCard(lotId: string, cardId: string) {
  const reply = assertOkResult(await request<RevealTrustCardReply>(`/api/lots/${lotId}/trust-cards/${cardId}/reveal`, { method: 'POST', body: JSON.stringify({}) }));
  if (!reply.lot || !reply.trustCard) throw new Error('response missing lot or trust card');
  return reply as { lot: Lot; trustCard: TrustRevealCard };
}

export async function startDuel(lotId: string) {
  return requireLot(assertOkResult(await request<StartDuelReply>(`/api/lots/${lotId}/duel`, { method: 'POST', body: JSON.stringify({}) })));
}

export async function settleLot(lotId: string) {
  return requireLot(assertOkResult(await request<SettleLotReply>(`/api/lots/${lotId}/settle`, { method: 'POST', body: JSON.stringify({}) })));
}

export async function cancelLot(lotId: string, reason: string) {
  return requireLot(assertOkResult(await request<CancelLotReply>(`/api/lots/${lotId}/cancel`, { method: 'POST', body: JSON.stringify({ lotId, reason }) })));
}

export async function getRoomSnapshot(roomId = 'demo'): Promise<RoomSnapshot> {
  const reply = assertOkResult(await request<GetRoomSnapshotReply>(`/api/rooms/${roomId}/snapshot`));
  if (!reply.snapshot) throw new Error('response missing room snapshot');
  return reply.snapshot;
}
