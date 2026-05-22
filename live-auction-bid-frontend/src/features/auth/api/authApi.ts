import { API_BASE } from '../../../shared/config/env';
import { forceRelogin } from '../../../shared/api/authExpired';
import { assertOkResult } from '../../../shared/api/result';
import type { AuthTokens, GetMeReply, LoginReply, LogoutReply, RefreshTokenReply, User } from '../../../shared/api/types';
import { accessToken, clearAuthState, loadAuthState, saveAuthState } from './authStore';

let refreshPromise: Promise<AuthTokens | null> | null = null;

function shouldRefreshSoon(tokens: AuthTokens | null) {
  const expiresAt = Number(tokens?.accessExpiresAtUnixMs ?? 0);
  return Boolean(tokens?.refreshToken) && expiresAt > 0 && expiresAt - Date.now() < 60_000;
}

export async function refreshAccessToken(): Promise<AuthTokens | null> {
  const state = loadAuthState();
  const refreshToken = state.tokens?.refreshToken;
  if (!refreshToken) return null;
  if (!refreshPromise) {
    refreshPromise = fetch(`${API_BASE}/api/users/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    }).then(async (r) => {
      if (!r.ok) throw new Error(await r.text() || `HTTP ${r.status}`);
      const reply = assertOkResult(await r.json() as RefreshTokenReply);
      if (!reply.tokens) throw new Error('refresh response missing tokens');
      saveAuthState({ user: state.user, tokens: reply.tokens });
      return reply.tokens;
    }).catch((error) => {
      clearAuthState();
      forceRelogin('登录已过期，请重新登录');
      console.error('[auth] refresh failed', error);
      return null;
    }).finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  if (shouldRefreshSoon(loadAuthState().tokens)) await refreshAccessToken();
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
    const message = await r.text();
    if (r.status === 401) forceRelogin('登录已过期，请重新登录');
    throw new Error(message);
  }
  return r.json() as Promise<T>;
}

export function currentAuth() {
  return loadAuthState();
}

export async function login(username: string, password: string) {
  const reply = assertOkResult(await request<LoginReply>('/api/users/login', { method: 'POST', body: JSON.stringify({ username, password }) }));
  if (!reply.user || !reply.tokens) throw new Error('login response missing user or tokens');
  saveAuthState({ user: reply.user, tokens: reply.tokens });
  return reply.user;
}

export async function me(): Promise<User | null> {
  if (!accessToken()) return null;
  const reply = assertOkResult(await request<GetMeReply>('/api/users/me'));
  return reply.user ?? null;
}

export async function logout() {
  const refreshToken = loadAuthState().tokens?.refreshToken;
  if (refreshToken) {
    await request<LogoutReply>('/api/users/logout', { method: 'POST', body: JSON.stringify({ refreshToken }) }).catch(() => undefined);
  }
  clearAuthState();
}
