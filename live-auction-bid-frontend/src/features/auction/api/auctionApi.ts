import { API_BASE } from '../../../shared/config/env';
import type { CreateLotRequest, Lot, PlaceBidRequest } from '../../../shared/api/types';

export async function listLots(): Promise<Lot[]> {
  const r = await fetch(`${API_BASE}/api/lots`);
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

export async function createLot(payload: CreateLotRequest) {
  const r = await fetch(`${API_BASE}/api/lots`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!r.ok) throw new Error(await r.text());
  return r.json() as Promise<Lot>;
}

export async function placeBid(lotId: string, payload: PlaceBidRequest) {
  const r = await fetch(`${API_BASE}/api/lots/${lotId}/bid`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!r.ok) throw new Error(await r.text());
  return r.json() as Promise<Lot>;
}
