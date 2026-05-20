import type { AuthTokens, User } from '../../../shared/api/types';

type AuthState = {
  user: User | null;
  tokens: AuthTokens | null;
};

const STORAGE_KEY = 'liveAuction.auth.v1';

export function loadAuthState(): AuthState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return { user: null, tokens: null };
    const parsed = JSON.parse(raw) as AuthState;
    return { user: parsed.user ?? null, tokens: parsed.tokens ?? null };
  } catch {
    return { user: null, tokens: null };
  }
}

export function saveAuthState(state: AuthState) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  window.dispatchEvent(new Event('auth-state-change'));
}

export function clearAuthState() {
  localStorage.removeItem(STORAGE_KEY);
  window.dispatchEvent(new Event('auth-state-change'));
}

export function accessToken(): string | null {
  return loadAuthState().tokens?.accessToken ?? null;
}
