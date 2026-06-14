import { formatMoney } from '../lib/money';
import { RESULT_CODE, type Lot, type ReplyResult } from './types';

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
  [RESULT_CODE.ACCOUNT_DISABLED]: '当前账号已停用，请联系管理员',
  [RESULT_CODE.USER_NOT_FOUND]: '用户或资源不存在',
  [RESULT_CODE.LOT_VERSION_CONFLICT]: '竞拍状态已变化，请刷新后重试',
  [RESULT_CODE.USERNAME_TAKEN]: '用户名已存在，请直接登录或换一个用户名',
  [RESULT_CODE.ROOM_ACTIVE_LOT_EXISTS]: '当前直播间已有正在竞拍的拍品，请先成交或取消当前拍品',
  [RESULT_CODE.QUEUE_POSITION_CONFLICT]: '当前直播间队列正在更新，请刷新后重试',
  [RESULT_CODE.BID_TOO_LOW]: '出价金额太低，请按建议价重新出价',
  [RESULT_CODE.BID_NOT_LIVE]: '本件拍品还未开始，看看当前讲解商品',
  [RESULT_CODE.BID_ENDED]: '竞拍已结束，看看下一件',
  [RESULT_CODE.BID_ALREADY_LEADING]: '你当前已经领先，等其他人出价后再加价',
  [RESULT_CODE.BID_CURRENCY_MISMATCH]: '出价币种异常，请刷新后重试',
  [RESULT_CODE.BID_VERSION_STALE]: '价格已更新，请刷新后重试',
  [RESULT_CODE.LOT_CANCELLED]: '本件拍品已取消，无法继续出价',
  [RESULT_CODE.PROJECTION_PENDING]: '出价正在同步，请稍后刷新',
  [RESULT_CODE.DEPOSIT_REQUIRED]: '出价前需先支付保证金',
  [RESULT_CODE.ADDRESS_REQUIRED]: '请先选择收货地址',
  [RESULT_CODE.ADDRESS_NOT_FOUND]: '收货地址不存在或已删除',
  [RESULT_CODE.PAYMENT_PROVIDER_NOT_CONFIGURED]: '支付通道未配置，请联系平台',
  [RESULT_CODE.INTERNAL_ERROR]: '系统暂时不可用，请稍后重试',
};

export const BUSINESS_ERROR_CODE = {
  INVALID_ARGUMENT: 'INVALID_ARGUMENT',
  USERNAME_TAKEN: 'USERNAME_TAKEN',
  BID_REJECTED: 'BID_REJECTED',
  BID_TOO_LOW: 'BID_TOO_LOW',
  BID_NOT_LIVE: 'BID_NOT_LIVE',
  BID_ENDED: 'BID_ENDED',
  BID_ALREADY_LEADING: 'BID_ALREADY_LEADING',
  BID_CURRENCY_MISMATCH: 'BID_CURRENCY_MISMATCH',
  BID_VERSION_STALE: 'BID_VERSION_STALE',
  LOT_CANCELLED: 'LOT_CANCELLED',
  ROOM_ACTIVE_LOT_EXISTS: 'ROOM_ACTIVE_LOT_EXISTS',
  PROJECTION_PENDING: 'PROJECTION_PENDING',
  DEPOSIT_REQUIRED: 'DEPOSIT_REQUIRED',
  ADDRESS_REQUIRED: 'ADDRESS_REQUIRED',
  ADDRESS_NOT_FOUND: 'ADDRESS_NOT_FOUND',
  PAYMENT_PROVIDER_NOT_CONFIGURED: 'PAYMENT_PROVIDER_NOT_CONFIGURED',
} as const;

type BusinessErrorCode = (typeof BUSINESS_ERROR_CODE)[keyof typeof BUSINESS_ERROR_CODE];

const backendMessageMap: Record<string, string> = {
  'invalid argument': '参数不正确，请检查后重试',
  'login required': '请先登录后再操作',
  'permission denied': '当前账号没有执行该操作的权限',
  'account disabled': '当前账号已停用，请联系管理员',
  'username already exists': '用户名已存在，请直接登录或换一个用户名',
  'invalid username or password': '用户名或密码不正确',
  'user not found': '用户或资源不存在',
  'not found': '用户或资源不存在',
  'internal error, please try again later': '系统暂时不可用，请稍后重试',
};

const businessCodeByResultCode: Record<number, BusinessErrorCode> = {
  [RESULT_CODE.ROOM_ACTIVE_LOT_EXISTS]: BUSINESS_ERROR_CODE.ROOM_ACTIVE_LOT_EXISTS,
  [RESULT_CODE.BID_TOO_LOW]: BUSINESS_ERROR_CODE.BID_TOO_LOW,
  [RESULT_CODE.BID_NOT_LIVE]: BUSINESS_ERROR_CODE.BID_NOT_LIVE,
  [RESULT_CODE.BID_ENDED]: BUSINESS_ERROR_CODE.BID_ENDED,
  [RESULT_CODE.BID_ALREADY_LEADING]: BUSINESS_ERROR_CODE.BID_ALREADY_LEADING,
  [RESULT_CODE.BID_CURRENCY_MISMATCH]: BUSINESS_ERROR_CODE.BID_CURRENCY_MISMATCH,
  [RESULT_CODE.BID_VERSION_STALE]: BUSINESS_ERROR_CODE.BID_VERSION_STALE,
  [RESULT_CODE.LOT_CANCELLED]: BUSINESS_ERROR_CODE.LOT_CANCELLED,
  [RESULT_CODE.PROJECTION_PENDING]: BUSINESS_ERROR_CODE.PROJECTION_PENDING,
  [RESULT_CODE.DEPOSIT_REQUIRED]: BUSINESS_ERROR_CODE.DEPOSIT_REQUIRED,
  [RESULT_CODE.ADDRESS_REQUIRED]: BUSINESS_ERROR_CODE.ADDRESS_REQUIRED,
  [RESULT_CODE.ADDRESS_NOT_FOUND]: BUSINESS_ERROR_CODE.ADDRESS_NOT_FOUND,
  [RESULT_CODE.PAYMENT_PROVIDER_NOT_CONFIGURED]: BUSINESS_ERROR_CODE.PAYMENT_PROVIDER_NOT_CONFIGURED,
};

const businessCodes = new Set<string>(Object.values(BUSINESS_ERROR_CODE));

function suggestedBidMessage(lot?: Pick<Lot, 'currentPrice' | 'rule'>): string {
  const currentAmount = Number(lot?.currentPrice?.amount ?? 0) || 0;
  const incrementAmount = Number(lot?.rule?.minIncrement?.amount ?? 0) || 0;
  if (currentAmount > 0 && incrementAmount > 0) {
    const nextAmount = currentAmount + incrementAmount;
    return `已被抢先，当前价 ${formatMoney(lot?.currentPrice)}，建议出价 ${formatMoney({
      amount: nextAmount,
      currency: lot?.currentPrice?.currency || lot?.rule?.minIncrement?.currency || 'CNY',
    })}`;
  }
  return '已被抢先，请按建议价重新出价';
}

export function businessErrorMessage(code: string | number | undefined, options: { lot?: Pick<Lot, 'currentPrice' | 'rule'> } = {}): string | undefined {
  const normalized = typeof code === 'number' ? businessCodeByResultCode[code] : code?.trim();
  switch (normalized) {
    case BUSINESS_ERROR_CODE.BID_TOO_LOW:
      return suggestedBidMessage(options.lot);
    case BUSINESS_ERROR_CODE.BID_ALREADY_LEADING:
      return '你当前已经领先，等其他人出价后再加价';
    case BUSINESS_ERROR_CODE.LOT_CANCELLED:
      return '本件拍品已取消，无法继续出价';
    case BUSINESS_ERROR_CODE.BID_ENDED:
      return '竞拍已结束，看看下一件';
    case BUSINESS_ERROR_CODE.BID_NOT_LIVE:
      return '本件拍品还未开始，看看当前讲解商品';
    case BUSINESS_ERROR_CODE.BID_CURRENCY_MISMATCH:
      return '出价币种异常，请刷新后重试';
    case BUSINESS_ERROR_CODE.BID_VERSION_STALE:
      return '价格已更新，请刷新后重试';
    case BUSINESS_ERROR_CODE.ROOM_ACTIVE_LOT_EXISTS:
      return '当前直播间已有正在竞拍的拍品，请先成交或取消当前拍品';
    case BUSINESS_ERROR_CODE.PROJECTION_PENDING:
      return '出价正在同步，请稍后刷新';
    case BUSINESS_ERROR_CODE.DEPOSIT_REQUIRED:
      return '出价前需先支付保证金';
    case BUSINESS_ERROR_CODE.ADDRESS_REQUIRED:
      return '请先选择收货地址';
    case BUSINESS_ERROR_CODE.ADDRESS_NOT_FOUND:
      return '收货地址不存在或已删除';
    case BUSINESS_ERROR_CODE.PAYMENT_PROVIDER_NOT_CONFIGURED:
      return '支付通道未配置，请联系平台';
    case BUSINESS_ERROR_CODE.INVALID_ARGUMENT:
      return '参数不正确，请检查后重试';
    case BUSINESS_ERROR_CODE.USERNAME_TAKEN:
      return '用户名已存在，请直接登录或换一个用户名';
    case BUSINESS_ERROR_CODE.BID_REJECTED:
      return '出价失败，请调整金额后重试';
    default:
      return undefined;
  }
}

export function businessErrorMessageFromUnknown(reason: unknown, options: { lot?: Pick<Lot, 'currentPrice' | 'rule'> } = {}): string | undefined {
  if (reason instanceof AppApiError) {
    return businessErrorMessage(reason.code, options) || businessErrorMessage(reason.message, options);
  }
  const rawMessage = reason instanceof Error ? reason.message : typeof reason === 'string' ? reason : '';
  const message = rawMessage.replace(/^invalid argument:\s*/i, '').replace(/^操作失败：\s*/i, '').trim();
  if (businessCodes.has(message)) return businessErrorMessage(message, options);
  if (message.includes('leading bidder must wait') || message.includes('最高价')) return businessErrorMessage(BUSINESS_ERROR_CODE.BID_ALREADY_LEADING, options);
  if (message.includes('主播取消') || message.includes('already cancelled')) return businessErrorMessage(BUSINESS_ERROR_CODE.LOT_CANCELLED, options);
  if (message.includes('bid amount is lower')) return businessErrorMessage(BUSINESS_ERROR_CODE.BID_TOO_LOW, options);
  if (message.includes('lot is not live')) return businessErrorMessage(BUSINESS_ERROR_CODE.BID_NOT_LIVE, options);
  if (message.includes('auction has ended')) return businessErrorMessage(BUSINESS_ERROR_CODE.BID_ENDED, options);
  if (message.includes('currency')) return businessErrorMessage(BUSINESS_ERROR_CODE.BID_CURRENCY_MISMATCH, options);
  if (message.includes('runtime state is missing')) return businessErrorMessage(BUSINESS_ERROR_CODE.PROJECTION_PENDING, options);
  return undefined;
}

export function resultMessage(result?: ReplyResult | null, fallback = '请求失败'): string {
  const code = resultCode(result);
  const rawMessage = String(result?.message || '').trim();
  return businessErrorMessage(code) || businessErrorMessage(rawMessage) || resultMessages[code] || backendMessageMap[rawMessage] || fallback;
}

export function resultTraceId(result?: ReplyResult | null): string | undefined {
  return result?.traceId || result?.trace_id;
}

export function isAuthResultCode(code: number): boolean {
  return code === RESULT_CODE.LOGIN_REQUIRED || code === RESULT_CODE.TOKEN_EXPIRED || code === RESULT_CODE.TOKEN_INVALID || code === RESULT_CODE.SESSION_EXPIRED;
}

export function isAuthRequiredError(reason: unknown): boolean {
  if (reason instanceof AuthExpiredError) return true;
  if (reason instanceof AppApiError && typeof reason.code === 'number' && isAuthResultCode(reason.code)) return true;
  const message = reason instanceof Error ? reason.message : typeof reason === 'string' ? reason : '';
  return message.includes('请先登录') ||
    message.includes('登录已过期') ||
    message.includes('登录会话已失效') ||
    message.includes('登录凭证无效') ||
    message.includes('login required');
}

export function isRefreshableAuthResultCode(code: number): boolean {
  return code === RESULT_CODE.TOKEN_EXPIRED;
}
