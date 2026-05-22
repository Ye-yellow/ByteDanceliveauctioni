import { API_BASE } from '../../../shared/config/env';
import { assertOkResult } from '../../../shared/api/result';
import { accessToken } from '../../auth/api/authStore';
import { refreshAccessToken } from '../../auth/api/authApi';
import { forceRelogin } from '../../../shared/api/authExpired';
import { clientLog, createRequestId } from '../../../shared/lib/clientLogger';
import type { CancelLotReply, CreateLotReply, CreateLotRequest, GetRoomSnapshotReply, ListLotsReply, Lot, RevealTrustCardReply, RoomSnapshot, SettleLotReply, StartDuelReply, StartLotReply, TrustRevealCard, UploadedAsset, UploadImageReply } from '../../../shared/api/types';

function formatApiError(input: { status?: number; code?: number; message?: string; requestId?: string; result?: { code?: number; message?: string; traceId?: string; trace_id?: string }; error?: string }) {
  const code = input.code ?? input.result?.code;
  const requestId = input.requestId ?? input.result?.traceId ?? input.result?.trace_id;
  const message = input.message || input.result?.message || input.error || (input.status ? `HTTP ${input.status}` : 'request failed');
  const meta = [code !== undefined ? `code=${code}` : '', requestId ? `requestId=${requestId}` : ''].filter(Boolean).join('，');
  return meta ? `${message}（${meta}）` : message;
}

async function readErrorMessage(response: Response): Promise<string> {
  const text = await response.text();
  if (!text) return `HTTP ${response.status}`;
  try {
    const body = JSON.parse(text) as { code?: number; message?: string; requestId?: string; result?: { code?: number; message?: string; traceId?: string; trace_id?: string }; error?: string };
    return formatApiError({ ...body, status: response.status });
  } catch {
    return `${text}（HTTP ${response.status}）`;
  }
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const token = accessToken();
  const buildInit = (nextToken: string | null): RequestInit => ({
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(nextToken ? { Authorization: `Bearer ${nextToken}` } : {}),
      ...(init?.headers ?? {}),
    },
  });
  let r = await fetch(`${API_BASE}${url}`, buildInit(token));
  if (r.status === 401 && await refreshAccessToken()) {
    r = await fetch(`${API_BASE}${url}`, buildInit(accessToken()));
  }
  if (!r.ok) {
    const message = await readErrorMessage(r);
    if (r.status === 401) forceRelogin(`登录已过期，请重新登录（${message}）`);
    throw new Error(r.status === 401 ? `登录已过期，请重新登录（${message}）` : message);
  }
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

export async function getRoomSnapshot(roomId = 'demo'): Promise<RoomSnapshot> {
  const reply = assertOkResult(await request<GetRoomSnapshotReply>(`/api/${'rooms'}/${encodeURIComponent(roomId)}/snapshot`));
  if (!reply.snapshot) throw new Error('response missing snapshot');
  return reply.snapshot;
}

export async function uploadImage(file: File, input?: { roomId?: string; bizType?: string }): Promise<UploadedAsset> {
  const form = new FormData();
  form.append('file', file);
  if (input?.roomId) form.append('roomId', input.roomId);
  if (input?.bizType) form.append('bizType', input.bizType);
  const requestId = createRequestId('upload');
  const startedAt = performance.now();
  clientLog('info', 'upload_image.request', {
    requestId,
    endpoint: '/api/uploads/images',
    roomId: input?.roomId,
    bizType: input?.bizType,
    fileName: file.name,
    fileType: file.type,
    fileSize: file.size,
    hasToken: Boolean(accessToken()),
  });
  try {
    const doUpload = () => fetch(`${API_BASE}/api/uploads/images`, {
      method: 'POST',
      headers: {
        'X-Request-Id': requestId,
        ...(accessToken() ? { Authorization: `Bearer ${accessToken()}` } : {}),
      },
      body: form,
    });
    let r = await doUpload();
    if (r.status === 401 && await refreshAccessToken()) r = await doUpload();
    if (!r.ok) {
      const message = await readErrorMessage(r);
      if (r.status === 401) forceRelogin(`登录已过期，请重新登录（${message}）`);
      throw new Error(r.status === 401 ? `登录已过期，请重新登录（${message}）` : message);
    }
    const reply = await r.json() as UploadImageReply;
    const code = reply.code ?? reply.result?.code ?? 0;
    if (code !== 0) throw new Error(formatApiError(reply));
    const asset = reply.data?.asset ?? reply.asset;
    if (!asset?.imageUrl) throw new Error('response missing uploaded image url');
    clientLog('info', 'upload_image.success', {
      requestId: reply.requestId ?? requestId,
      durationMs: Math.round(performance.now() - startedAt),
      code,
      assetId: asset.id,
      imageUrl: asset.imageUrl,
      objectKey: asset.objectKey,
      mimeType: asset.mimeType,
      sizeBytes: asset.sizeBytes,
    });
    return asset;
  } catch (error) {
    clientLog('error', 'upload_image.failure', {
      requestId,
      durationMs: Math.round(performance.now() - startedAt),
      error: error instanceof Error ? error.message : String(error),
    });
    throw error;
  }
}

export async function deleteUploadedImage(assetId: string, options?: { keepalive?: boolean; silent?: boolean }): Promise<void> {
  if (!assetId) return;
  const requestId = createRequestId('delete-upload');
  const doDelete = () => fetch(`${API_BASE}/api/uploads/images/${encodeURIComponent(assetId)}`, {
    method: 'DELETE',
    keepalive: options?.keepalive,
    headers: {
      'X-Request-Id': requestId,
      ...(accessToken() ? { Authorization: `Bearer ${accessToken()}` } : {}),
    },
  });
  let r = await doDelete();
  if (r.status === 401 && !options?.keepalive && await refreshAccessToken()) r = await doDelete();
  if (!r.ok) {
    const message = await readErrorMessage(r);
    if (options?.silent) {
      clientLog('warn', 'upload_image.delete_failure', { requestId, assetId, message });
      return;
    }
    throw new Error(message);
  }
  clientLog('info', 'upload_image.deleted', { requestId, assetId });
}

export async function createLot(payload: CreateLotRequest) {
  return requireLot(assertOkResult(await request<CreateLotReply>('/api/lots', { method: 'POST', body: JSON.stringify(payload) })));
}

export async function startLot(lotId: string) {
  return requireLot(assertOkResult(await request<StartLotReply>(`/api/lots/${lotId}/start`, { method: 'POST', body: JSON.stringify({}) })));
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
