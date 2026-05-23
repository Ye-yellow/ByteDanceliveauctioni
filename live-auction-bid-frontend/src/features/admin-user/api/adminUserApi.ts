import { apiRequest } from '../../../shared/api/httpClient';
import { toQueryString } from '../../../shared/api/query';
import { assertOkResult } from '../../../shared/api/result';
import type { ReplyResult, User, UserRole } from '../../../shared/api/types';

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
  user?: User;
  result?: ReplyResult;
};

type AdminUsersReply = {
  users?: User[];
  total?: number | string;
  page?: number | string;
  pageSize?: number | string;
  result?: ReplyResult;
};

function requireUser(reply: AdminUserReply) {
  if (!reply.user) throw new Error('admin user response missing user');
  return reply.user;
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
    users: reply.users ?? [],
    total: Number(reply.total ?? 0),
    page: Number(reply.page ?? page),
    pageSize: Number(reply.pageSize ?? pageSize),
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
