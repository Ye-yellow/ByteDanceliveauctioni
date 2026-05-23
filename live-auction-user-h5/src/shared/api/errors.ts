import { RESULT_CODE, type ReplyResult } from './types';

export type ApiErrorKind = 'http' | 'result' | 'auth-expired' | 'network';

export class AppApiError extends Error {
  kind: ApiErrorKind;
  code?: number;
  status?: number;
  requestId?: string;
  traceId?: string;

  constructor(message: string, options: { kind: ApiErrorKind; code?: number; status?: number; requestId?: string; traceId?: string }) {
    super(message);
    this.name = 'AppApiError';
    this.kind = options.kind;
    this.code = options.code;
    this.status = options.status;
    this.requestId = options.requestId;
    this.traceId = options.traceId;
  }
}

export class AuthExpiredError extends AppApiError {
  constructor(message = '登录已过期，请重新登录', options: { code?: number; status?: number; requestId?: string; traceId?: string } = {}) {
    super(message, { ...options, kind: 'auth-expired' });
    this.name = 'AuthExpiredError';
  }
}

export function resultCode(result?: ReplyResult | null): number {
  return Number(result?.code ?? RESULT_CODE.OK);
}

const resultMessages: Record<number, string> = {
  [RESULT_CODE.LOGIN_REQUIRED]: '请先登录后再操作',
  [RESULT_CODE.TOKEN_EXPIRED]: '登录已过期，正在刷新登录态',
  [RESULT_CODE.TOKEN_INVALID]: '登录凭证无效，请重新登录',
  [RESULT_CODE.SESSION_EXPIRED]: '登录会话已失效，请重新登录',
  [RESULT_CODE.INVALID_CREDENTIALS]: '用户名或密码不正确',
  [RESULT_CODE.FORBIDDEN]: '当前账号没有执行该操作的权限',
  [RESULT_CODE.LOT_VERSION_CONFLICT]: '竞拍状态已变化，请刷新后重试',
  [RESULT_CODE.INTERNAL_ERROR]: '系统暂时不可用，请稍后重试',
};

export function resultMessage(result?: ReplyResult | null, fallback = '请求失败'): string {
  const code = resultCode(result);
  return resultMessages[code] || result?.message || fallback;
}

export function resultTraceId(result?: ReplyResult | null): string | undefined {
  return result?.traceId || result?.trace_id;
}

export function isAuthResultCode(code: number): boolean {
  return code === RESULT_CODE.LOGIN_REQUIRED || code === RESULT_CODE.TOKEN_EXPIRED || code === RESULT_CODE.TOKEN_INVALID || code === RESULT_CODE.SESSION_EXPIRED;
}

export function isRefreshableAuthResultCode(code: number): boolean {
  return code === RESULT_CODE.TOKEN_EXPIRED;
}
