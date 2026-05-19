import { API_BASE } from '../../../shared/config/env';
import type { Lot } from '../../../shared/types/auction';

export async function listLots(): Promise<Lot[]> {
  const r = await fetch(`${API_BASE}/api/lots`);
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

export async function createLot(payload: Partial<Lot> & { durationSec?: number }) {
  const r = await fetch(`${API_BASE}/api/lots`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!r.ok) throw new Error(await r.text());
  return r.json() as Promise<Lot>;
}
