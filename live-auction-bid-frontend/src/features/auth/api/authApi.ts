import { API_BASE } from '../../../shared/config/env';
import { assertOkResult } from '../../../shared/api/result';
import type { GetMeReply, LoginReply, LogoutReply, RegisterReply, User } from '../../../shared/api/types';
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
  if (!r.ok) throw new Error(await r.text());
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

export async function register(username: string, password: string, nickname: string) {
  const reply = assertOkResult(await request<RegisterReply>('/api/users/register', { method: 'POST', body: JSON.stringify({ username, password, nickname }) }));
  if (!reply.user || !reply.tokens) throw new Error('register response missing user or tokens');
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
