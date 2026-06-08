import type { AuthTokens, User } from '../api/types';
import { clearAuthState, loadAuthState, saveAuthState, saveExpiredMessage, type AuthState } from './authStorage';

type RefreshExecutor = (refreshToken: string) => Promise<AuthTokens>;

type AuthExpiredDetail = {
  message: string;
};

const refreshLeadMs = 60_000;
let refreshExecutor: RefreshExecutor | null = null;
let refreshPromise: Promise<AuthTokens | null> | null = null;
let expiring = false;

function emitAuthExpired(message: string) {
  window.dispatchEvent(new CustomEvent<AuthExpiredDetail>('auth:expired', { detail: { message } }));
}

function tokenExpiresSoon(tokens: AuthTokens | null, leadMs = refreshLeadMs) {
  const expiresAt = Number(tokens?.accessExpiresAtUnixMs ?? 0);
  return Boolean(tokens?.refreshToken) && expiresAt > 0 && expiresAt - Date.now() < leadMs;
}

function tokenExpired(tokens: AuthTokens | null) {
  const expiresAt = Number(tokens?.accessExpiresAtUnixMs ?? 0);
  return Boolean(tokens?.refreshToken) && expiresAt > 0 && expiresAt <= Date.now();
}

export const authSession = {
  configureRefresh(executor: RefreshExecutor) {
    refreshExecutor = executor;
  },

  current(): AuthState {
    return loadAuthState();
  },

  currentUser(): User | null {
    return loadAuthState().user;
  },

  currentTokens(): AuthTokens | null {
    return loadAuthState().tokens;
  },

  accessToken(): string | null {
    return loadAuthState().tokens?.accessToken ?? null;
  },

  setAuthenticated(user: User, tokens: AuthTokens) {
    expiring = false;
    saveAuthState({ user, tokens });
  },

  clear() {
    expiring = false;
    clearAuthState();
  },

  isAccessTokenExpiringSoon(leadMs = refreshLeadMs) {
    return tokenExpiresSoon(loadAuthState().tokens, leadMs);
  },

  async getValidAccessToken(): Promise<string | null> {
    const tokens = loadAuthState().tokens;
    if (!tokens?.accessToken) {
      const refreshed = await this.refreshOnce();
      return refreshed?.accessToken ?? null;
    }
    if (tokenExpiresSoon(tokens) || tokenExpired(tokens)) {
      const refreshed = await this.refreshOnce();
      return refreshed?.accessToken ?? null;
    }
    return tokens.accessToken;
  },

  async refreshOnce(): Promise<AuthTokens | null> {
    const state = loadAuthState();
    const refreshToken = state.tokens?.refreshToken;
    if (!refreshToken || !refreshExecutor) return null;
    if (!refreshPromise) {
      refreshPromise = refreshExecutor(refreshToken)
        .then((tokens) => {
          const latest = loadAuthState();
          saveAuthState({ user: latest.user ?? state.user, tokens });
          return tokens;
        })
        .catch((error) => {
          console.error('[auth] refresh failed', error);
          this.expire('登录已过期，请重新登录');
          return null;
        })
        .finally(() => {
          refreshPromise = null;
        });
    }
    return refreshPromise;
  },

  expire(message = '登录已过期，请重新登录') {
    saveExpiredMessage(message);
    clearAuthState();
    if (!expiring) {
      expiring = true;
      emitAuthExpired(message);
    }
  },
};
