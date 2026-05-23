import type { ReplyResult } from './types';
import {
  RESULT_CODE_INTERNAL_ERROR,
  RESULT_CODE_INVALID_CREDENTIALS,
  RESULT_CODE_LOGIN_REQUIRED,
  RESULT_CODE_LOT_VERSION_CONFLICT,
  RESULT_CODE_OK,
  RESULT_CODE_FORBIDDEN,
  RESULT_CODE_SESSION_EXPIRED,
  RESULT_CODE_TOKEN_EXPIRED,
  RESULT_CODE_TOKEN_INVALID,
} from './types';

export class ApiResultError extends Error {
  readonly result: ReplyResult;

  constructor(result: ReplyResult) {
    super(result.message || `request failed with result code ${result.code}`);
    this.name = 'ApiResultError';
    this.result = result;
  }
}

export function assertOkResult<T extends { result?: Partial<ReplyResult> }>(reply: T): T {
  const result = reply.result;
  // proto3 JSON 会省略默认值 code=0，所以 { message: 'ok' } 也应视为成功。
  if (result && (result.code ?? RESULT_CODE_OK) !== RESULT_CODE_OK) {
    throw new ApiResultError(result as ReplyResult);
  }
  return reply;
}

export function resultMessage(e: unknown): string {
  const result = (e as { result?: Partial<ReplyResult> } | null)?.result;
  if (result) return publicResultMessage(result);
  if (e instanceof ApiResultError) return publicResultMessage(e.result);
  if (e instanceof Error) return e.message;
  return String(e);
}

const errorMessages: Record<number, string> = {
  [RESULT_CODE_LOGIN_REQUIRED]: '请先登录后再操作',
  [RESULT_CODE_TOKEN_EXPIRED]: '登录已过期，正在刷新登录态',
  [RESULT_CODE_TOKEN_INVALID]: '登录凭证无效，请重新登录',
  [RESULT_CODE_SESSION_EXPIRED]: '登录会话已失效，请重新登录',
  [RESULT_CODE_INVALID_CREDENTIALS]: '用户名或密码不正确',
  [RESULT_CODE_FORBIDDEN]: '当前账号没有执行该操作的权限',
  [RESULT_CODE_LOT_VERSION_CONFLICT]: '竞拍状态已变化，已刷新最新数据后再操作',
  [RESULT_CODE_INTERNAL_ERROR]: '系统暂时不可用，请稍后重试',
};

export function publicResultMessage(result?: Partial<ReplyResult>, fallback = '请求失败') {
  if (!result) return fallback;
  const code = Number(result.code ?? RESULT_CODE_OK);
  return errorMessages[code] || result.message || `${fallback}（code=${code}）`;
}

export { normalizeAuctionEvent } from './normalizers';
