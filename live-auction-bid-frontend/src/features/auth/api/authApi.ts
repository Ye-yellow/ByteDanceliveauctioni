import { apiRequest } from '../../../shared/api/httpClient';
import { assertOkResult } from '../../../shared/api/result';
import { authSession } from '../../../shared/auth/authSession';
import { ADMIN_ACCESS_ROLES } from '../../../shared/api/types';
import type { AuthTokens, GetMeReply, LoginReply, LogoutReply, User } from '../../../shared/api/types';

const ADMIN_ACCESS_DENIED_MESSAGE = '该账号无后台访问权限';

export async function refreshAccessToken(): Promise<AuthTokens | null> {
  return authSession.refreshOnce();
}

export function currentAuth() {
  return authSession.current();
}

export async function login(username: string, password: string) {
  const reply = assertOkResult(await apiRequest<LoginReply>({
    path: '/api/users/login',
    method: 'POST',
    body: { username, password },
    auth: 'none',
    operation: 'login',
  }));
  if (!reply.user || !reply.tokens) throw new Error('login response missing user or tokens');
  if (!ADMIN_ACCESS_ROLES.includes(reply.user.role)) {
    authSession.clear();
    throw new Error(ADMIN_ACCESS_DENIED_MESSAGE);
  }
  authSession.setAuthenticated(reply.user, reply.tokens);
  return reply.user;
}

export async function me(): Promise<User | null> {
  if (!authSession.currentTokens()?.refreshToken) return null;
  const reply = assertOkResult(await apiRequest<GetMeReply>({
    path: '/api/users/me',
    method: 'GET',
    auth: 'required',
    operation: 'me',
  }));
  if (reply.user && !ADMIN_ACCESS_ROLES.includes(reply.user.role)) {
    authSession.clear();
    throw new Error(ADMIN_ACCESS_DENIED_MESSAGE);
  }
  if (reply.user) authSession.setAuthenticated(reply.user, authSession.currentTokens()!);
  return reply.user ?? null;
}

export async function logout() {
  const refreshToken = authSession.currentTokens()?.refreshToken;
  if (refreshToken) {
    await apiRequest<LogoutReply>({
      path: '/api/users/logout',
      method: 'POST',
      body: { refreshToken },
      auth: 'optional',
      operation: 'logout',
    }).catch(() => undefined);
  }
  authSession.clear();
}
