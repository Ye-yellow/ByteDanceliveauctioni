import { normalizeAuthTokens, normalizeUser } from '../api/adapters';
import { AppApiError, AuthExpiredError } from '../api/errors';
import { apiRequest, setAuthProvider } from '../api/httpClient';
import { USER_ROLE, type AuthTokens, type User } from '../api/types';

const AUTH_SESSION_STORAGE_KEY = 'live-auction-h5.auth-session.v1';
const DEMO_BUYER_STORAGE_KEY = 'live-auction-h5.demo-buyer.v1';
const REFRESH_SKEW_MS = 60_000;
const NON_BUYER_ACCOUNT_MESSAGE = '该账号不是买家账号';
const LOGIN_REQUIRED_MESSAGE = '请先登录或注册买家账号';

type StoredSession = {
  user: User;
  tokens: AuthTokens;
};

type DemoBuyerCredentials = {
  username: string;
  password: string;
  nickname: string;
};

type AuthReply = {
  user?: unknown;
  tokens?: unknown;
};

type ResetPasswordReply = {
  user?: unknown;
};

export type AuthMode = 'demo' | 'real';

export type AuthSessionSnapshot = {
  user: User | null;
  tokens: AuthTokens | null;
  status: 'anonymous' | 'authenticated' | 'refreshing' | 'expired';
  reason?: string;
};

type Listener = (snapshot: AuthSessionSnapshot) => void;

function readJson<T>(key: string): T | null {
  try {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T) : null;
  } catch {
    return null;
  }
}

function writeJson<T>(key: string, value: T) {
  localStorage.setItem(key, JSON.stringify(value));
}

function isUsableToken(tokens: AuthTokens | null, skewMs = REFRESH_SKEW_MS): boolean {
  if (!tokens?.accessToken) return false;
  return Number(tokens.accessExpiresAtUnixMs || 0) - Date.now() > skewMs;
}

function isUsableRefresh(tokens: AuthTokens | null): boolean {
  if (!tokens?.refreshToken) return false;
  return Number(tokens.refreshExpiresAtUnixMs || 0) > Date.now();
}

function normalizeAuthReply(reply: AuthReply): StoredSession {
  const user = normalizeUser(reply.user);
  const tokens = normalizeAuthTokens(reply.tokens);

  if (!user.id || !tokens.accessToken || !tokens.refreshToken) {
    throw new AppApiError('登录响应缺少用户或 token', { kind: 'result' });
  }

  return { user, tokens };
}

function randomSuffix(): string {
  return crypto.randomUUID?.().slice(0, 12) || Math.random().toString(36).slice(2, 14);
}

function readAuthMode(): AuthMode {
  return import.meta.env.VITE_AUTH_MODE === 'demo' ? 'demo' : 'real';
}

function allowDemoAutoLogin(): boolean {
  return import.meta.env.VITE_DEMO_AUTO_LOGIN === 'true';
}

class H5AuthSession {
  private readonly authMode = readAuthMode();

  private snapshot: AuthSessionSnapshot = {
    user: null,
    tokens: null,
    status: 'anonymous',
  };

  private listeners = new Set<Listener>();
  private refreshPromise: Promise<AuthTokens | null> | null = null;

  constructor() {
    const stored = readJson<StoredSession>(AUTH_SESSION_STORAGE_KEY);
    if (stored?.user && stored.tokens) {
      const session = { user: normalizeUser(stored.user), tokens: normalizeAuthTokens(stored.tokens) };
      if (session.user.role === USER_ROLE.BUYER) {
        this.snapshot = {
          user: session.user,
          tokens: session.tokens,
          status: isUsableRefresh(session.tokens) ? 'authenticated' : 'expired',
        };
      } else {
        localStorage.removeItem(AUTH_SESSION_STORAGE_KEY);
        this.snapshot = { user: null, tokens: null, status: 'anonymous', reason: NON_BUYER_ACCOUNT_MESSAGE };
      }
    }
  }

  getSnapshot(): AuthSessionSnapshot {
    return this.snapshot;
  }

  getCurrentUser(): User | null {
    return this.snapshot.user;
  }

  getAuthMode(): AuthMode {
    return this.authMode;
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    listener(this.snapshot);
    return () => this.listeners.delete(listener);
  }

  async ensureBuyerSession(): Promise<StoredSession> {
    const currentTokens = this.snapshot.tokens;
    if (this.snapshot.user && this.snapshot.user.role !== USER_ROLE.BUYER) {
      return this.rejectNonBuyerAccount();
    }
    if (this.snapshot.user && currentTokens && isUsableToken(currentTokens)) {
      return { user: this.snapshot.user, tokens: currentTokens };
    }

    if (this.snapshot.user && isUsableRefresh(currentTokens)) {
      const tokens = await this.refreshOnce();
      if (tokens && this.snapshot.user) return { user: this.snapshot.user, tokens };
    }

    if (this.authMode !== 'demo' || !allowDemoAutoLogin()) {
      this.clear(LOGIN_REQUIRED_MESSAGE);
      throw new AuthExpiredError(LOGIN_REQUIRED_MESSAGE);
    }

    const session = await this.loginOrRegisterDemoBuyer();
    this.persist(session);
    return session;
  }

  async ensureReadyForBid(): Promise<StoredSession> {
    const session = await this.ensureBuyerSession();
    const tokens = await this.getValidAccessToken();
    if (!tokens) throw new AuthExpiredError('请先登录后再出价');
    return session;
  }

  async getValidAccessToken(): Promise<string | null> {
    if (this.snapshot.user && this.snapshot.user.role !== USER_ROLE.BUYER) {
      return this.rejectNonBuyerAccount();
    }
    if (isUsableToken(this.snapshot.tokens)) return this.snapshot.tokens?.accessToken ?? null;
    if (!isUsableRefresh(this.snapshot.tokens)) return null;

    const tokens = await this.refreshOnce();
    return tokens?.accessToken ?? null;
  }

  async refreshIfNeeded(): Promise<boolean> {
    if (this.snapshot.user && this.snapshot.user.role !== USER_ROLE.BUYER) {
      this.clear(NON_BUYER_ACCOUNT_MESSAGE);
      return false;
    }
    if (!this.snapshot.tokens?.refreshToken) return false;
    if (isUsableToken(this.snapshot.tokens)) return true;
    return Boolean(await this.refreshOnce());
  }

  async refreshOnce(): Promise<AuthTokens | null> {
    if (this.snapshot.user && this.snapshot.user.role !== USER_ROLE.BUYER) {
      return this.rejectNonBuyerAccount();
    }
    if (!this.snapshot.tokens?.refreshToken) return null;
    if (!isUsableRefresh(this.snapshot.tokens)) {
      this.expire('refresh token 已过期');
      return null;
    }

    if (!this.refreshPromise) {
      this.setStatus('refreshing');
      this.refreshPromise = this.refreshNow()
        .catch(() => {
          this.expire('登录已过期，请重新登录');
          return null;
        })
        .finally(() => {
          this.refreshPromise = null;
        });
    }

    return this.refreshPromise;
  }

  async logout(): Promise<void> {
    const refreshToken = this.snapshot.tokens?.refreshToken;
    localStorage.removeItem(DEMO_BUYER_STORAGE_KEY);
    this.clear('用户已退出');

    if (refreshToken) {
      try {
        await apiRequest({
          path: '/api/users/logout',
          method: 'POST',
          body: { refresh_token: refreshToken },
          auth: 'none',
          skipAuthRefresh: true,
          operation: 'logout',
        });
      } catch {
        // Local session cleanup must not depend on network availability.
      }
    }
  }

  async loginBuyer(username: string, password: string): Promise<StoredSession> {
    const session = this.requireBuyerSession(normalizeAuthReply(await apiRequest<AuthReply>({
      path: '/api/users/login',
      method: 'POST',
      body: { username: username.trim(), password },
      auth: 'none',
      skipAuthRefresh: true,
      operation: 'buyerLogin',
    })));
    this.persist(session);
    return session;
  }

  async registerBuyer(username: string, password: string, nickname: string): Promise<StoredSession> {
    const session = this.requireBuyerSession(normalizeAuthReply(await apiRequest<AuthReply>({
      path: '/api/users/register',
      method: 'POST',
      body: { username: username.trim(), password, nickname: nickname.trim() },
      auth: 'none',
      skipAuthRefresh: true,
      operation: 'buyerRegister',
    })));
    this.persist(session);
    return session;
  }

  async resetBuyerPassword(username: string, password: string): Promise<User> {
    const reply = await apiRequest<ResetPasswordReply>({
      path: '/api/users/reset-password',
      method: 'POST',
      body: { username: username.trim(), password },
      auth: 'none',
      skipAuthRefresh: true,
      operation: 'buyerResetPassword',
    });
    const user = normalizeUser(reply.user);
    if (!user.id) throw new AppApiError('重置密码响应缺少用户', { kind: 'result' });
    return user;
  }

  expire(reason = '登录已过期，请重新登录') {
    this.clear(reason, 'expired');
  }

  private async refreshNow(): Promise<AuthTokens> {
    const refreshToken = this.snapshot.tokens?.refreshToken;
    if (!refreshToken) throw new AuthExpiredError('缺少 refresh token');

    const reply = await apiRequest<AuthReply>({
      path: '/api/users/refresh',
      method: 'POST',
      body: { refresh_token: refreshToken },
      auth: 'none',
      skipAuthRefresh: true,
      operation: 'refreshToken',
    });

    const tokens = normalizeAuthTokens(reply.tokens);
    if (!tokens.accessToken || !tokens.refreshToken) throw new AuthExpiredError('刷新登录态失败');

    const user = this.snapshot.user;
    if (!user) throw new AuthExpiredError('缺少当前用户');

    this.persist({ user, tokens });
    return tokens;
  }

  private async loginOrRegisterDemoBuyer(): Promise<StoredSession> {
    const credentials = this.getOrCreateDemoCredentials();
    const login = () => apiRequest<AuthReply>({
      path: '/api/users/login',
      method: 'POST',
      body: { username: credentials.username, password: credentials.password },
      auth: 'none',
      skipAuthRefresh: true,
      operation: 'demoBuyerLogin',
    });

    try {
      return this.requireBuyerSession(normalizeAuthReply(await login()));
    } catch (error) {
      if (error instanceof AppApiError && error.message === NON_BUYER_ACCOUNT_MESSAGE) throw error;
      const registered = await apiRequest<AuthReply>({
        path: '/api/users/register',
        method: 'POST',
        body: credentials,
        auth: 'none',
        skipAuthRefresh: true,
        operation: 'demoBuyerRegister',
      });
      return this.requireBuyerSession(normalizeAuthReply(registered));
    }
  }

  private getOrCreateDemoCredentials(): DemoBuyerCredentials {
    const envUsername = import.meta.env.VITE_DEMO_BUYER_USERNAME as string | undefined;
    const envPassword = import.meta.env.VITE_DEMO_BUYER_PASSWORD as string | undefined;
    const envNickname = import.meta.env.VITE_DEMO_BUYER_NICKNAME as string | undefined;

    if (envUsername && envPassword) {
      return {
        username: envUsername,
        password: envPassword,
        nickname: envNickname || 'H5 买家',
      };
    }

    const stored = readJson<DemoBuyerCredentials>(DEMO_BUYER_STORAGE_KEY);
    if (stored?.username && stored.password) return stored;

    const suffix = randomSuffix();
    const credentials = {
      username: `h5-buyer-${suffix}`,
      password: `h5-demo-${suffix}`,
      nickname: `H5买家${suffix.slice(0, 4)}`,
    };

    writeJson(DEMO_BUYER_STORAGE_KEY, credentials);
    return credentials;
  }

  private persist(session: StoredSession) {
    this.requireBuyerSession(session);
    writeJson(AUTH_SESSION_STORAGE_KEY, session);
    this.snapshot = {
      user: session.user,
      tokens: session.tokens,
      status: 'authenticated',
    };
    this.emit();
  }

  private requireBuyerSession(session: StoredSession): StoredSession {
    if (session.user.role === USER_ROLE.BUYER) return session;
    return this.rejectNonBuyerAccount();
  }

  private rejectNonBuyerAccount(): never {
    this.clear(NON_BUYER_ACCOUNT_MESSAGE);
    throw new AppApiError(NON_BUYER_ACCOUNT_MESSAGE, { kind: 'result' });
  }

  private clear(reason: string, status: AuthSessionSnapshot['status'] = 'anonymous') {
    localStorage.removeItem(AUTH_SESSION_STORAGE_KEY);
    this.snapshot = {
      user: null,
      tokens: null,
      status,
      reason,
    };
    this.emit();
  }

  private setStatus(status: AuthSessionSnapshot['status']) {
    this.snapshot = { ...this.snapshot, status };
    this.emit();
  }

  private emit() {
    for (const listener of this.listeners) listener(this.snapshot);
  }
}

export const authSession = new H5AuthSession();

setAuthProvider({
  getValidAccessToken: () => authSession.getValidAccessToken(),
  refreshOnce: async () => (await authSession.refreshOnce())?.accessToken ?? null,
  expire: (reason?: string) => authSession.expire(reason),
});
