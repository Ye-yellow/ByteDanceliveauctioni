import { apiRequest } from '../../../shared/api/httpClient';
import { normalizeUser } from '../../../shared/api/normalizers';
import { toQueryString } from '../../../shared/api/query';
import { assertOkResult } from '../../../shared/api/result';
import type { ReplyResult, User, UserRole, UserStatus } from '../../../shared/api/types';

export type AdminUsersQuery = {
  page?: number;
  pageSize?: number;
  role?: UserRole | '';
  keyword?: string;
};

export type AdminUsersPage = {
  users: User[];
  total: number;
  page: number;
  pageSize: number;
};

type AdminUserReply = {
  user?: unknown;
  result?: ReplyResult;
};

type AdminUsersReply = {
  users?: unknown[];
  total?: number | string;
  page?: number | string;
  pageSize?: number | string;
  result?: ReplyResult;
};

function requireUser(reply: AdminUserReply) {
  if (!reply.user) throw new Error('team user response missing user');
  return normalizeUser(reply.user);
}

function requireArray<T>(value: T[] | undefined, field: string): T[] {
  if (!Array.isArray(value)) throw new Error(`response missing ${field}`);
  return value;
}

function requiredValue<T>(value: T | undefined | null, field: string): T {
  if (value === undefined || value === null || value === '') throw new Error(`response missing ${field}`);
  return value;
}

export async function adminCreateUser(payload: { username: string; password: string; nickname: string; role: UserRole }) {
  return requireUser(assertOkResult(await apiRequest<AdminUserReply>({
    path: '/api/admin/users',
    method: 'POST',
    body: payload,
    operation: 'admin-create-user',
  })));
}

export async function listAdminUsers(query: AdminUsersQuery = {}): Promise<AdminUsersPage> {
  const page = query.page ?? 1;
  const pageSize = query.pageSize ?? 20;
  const reply = assertOkResult(await apiRequest<AdminUsersReply>({
    path: `/api/admin/users${toQueryString({
      page,
      pageSize,
      role: query.role,
      keyword: query.keyword?.trim(),
    })}`,
    method: 'GET',
    operation: 'admin-list-users',
  }));
  return {
    users: requireArray(reply.users, 'users').map(normalizeUser),
    total: Number(requiredValue(reply.total, 'total')),
    page: Number(requiredValue(reply.page, 'page')),
    pageSize: Number(requiredValue(reply.pageSize, 'pageSize')),
  };
}

export async function adminUpdateUserRole(userId: string, role: UserRole) {
  return requireUser(assertOkResult(await apiRequest<AdminUserReply>({
    path: `/api/admin/users/${encodeURIComponent(userId)}/role`,
    method: 'POST',
    body: { userId, role },
    operation: 'admin-update-user-role',
  })));
}

export async function adminUpdateUserStatus(userId: string, status: UserStatus) {
  return requireUser(assertOkResult(await apiRequest<AdminUserReply>({
    path: `/api/admin/users/${encodeURIComponent(userId)}/status`,
    method: 'POST',
    body: { userId, status },
    operation: 'admin-update-user-status',
  })));
}
