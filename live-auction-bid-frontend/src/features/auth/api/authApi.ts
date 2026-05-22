import { API_BASE } from '../../../shared/config/env';
import { forceRelogin } from '../../../shared/api/authExpired';
import { assertOkResult } from '../../../shared/api/result';
import type { GetMeReply, LoginReply, LogoutReply, User } from '../../../shared/api/types';
import { accessToken, clearAuthState, loadAuthState, saveAuthState } from './authStore';

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
