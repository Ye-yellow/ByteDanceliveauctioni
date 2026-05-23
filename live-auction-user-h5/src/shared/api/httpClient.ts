import { AppApiError, AuthExpiredError, isAuthResultCode, isRefreshableAuthResultCode, resultCode, resultMessage, resultTraceId } from './errors';
import type { ReplyResult } from './types';

const API_BASE = import.meta.env.VITE_API_BASE_URL || 'http://localhost:18080';
const CLIENT_APP = 'buyer-h5';
const CLIENT_VERSION = import.meta.env.VITE_CLIENT_VERSION || import.meta.env.VITE_APP_VERSION || 'dev';

export type AuthMode = 'none' | 'optional' | 'required';

export type ApiRequestOptions = {
  path: string;
  method?: string;
  body?: unknown;
  headers?: HeadersInit;
  auth?: AuthMode;
  contentType?: 'json' | 'form-data' | 'none';
  keepalive?: boolean;
  idempotencyKey?: string;
  operation?: string;
  skipAuthRefresh?: boolean;
};

export type AuthProvider = {
  getValidAccessToken(): Promise<string | null>;
  refreshOnce(): Promise<string | null>;
  expire(reason?: string): void;
};

type ParsedResponse = {
  ok: boolean;
  status: number;
  requestId?: string;
  data: unknown;
  result?: ReplyResult;
};

let authProvider: AuthProvider | null = null;

export function setAuthProvider(provider: AuthProvider) {
  authProvider = provider;
}

export function createRequestId(operation = 'h5'): string {
  const random = crypto.randomUUID?.() || Math.random().toString(36).slice(2);
  return `${operation}-${Date.now()}-${random}`;
}

function buildUrl(path: string): string {
  if (/^https?:\/\//.test(path)) return path;
  return `${API_BASE}${path.startsWith('/') ? path : `/${path}`}`;
}

function resultFromPayload(data: unknown): ReplyResult | undefined {
  if (!data || typeof data !== 'object') return undefined;
  return (data as Record<string, unknown>).result as ReplyResult | undefined;
}

async function parseResponse(response: Response): Promise<ParsedResponse> {
  const text = await response.text();
  let data: unknown = null;

  if (text) {
    try {
      data = JSON.parse(text) as unknown;
    } catch {
      data = text;
    }
  }

  return {
    ok: response.ok,
    status: response.status,
    requestId: response.headers.get('X-Request-Id') || undefined,
    data,
    result: resultFromPayload(data),
  };
}

function bodyForRequest(body: unknown, contentType: ApiRequestOptions['contentType']): BodyInit | undefined {
  if (body === undefined || body === null) return undefined;
  if (body instanceof FormData || body instanceof Blob || typeof body === 'string') return body;
  if (contentType === 'none') return body as BodyInit;
  return JSON.stringify(body);
}

function headersForRequest(options: ApiRequestOptions, token: string | null, requestId: string): Headers {
  const headers = new Headers(options.headers);
  const contentType = options.contentType ?? (options.body instanceof FormData ? 'form-data' : 'json');

  if (contentType === 'json' && options.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  headers.set('X-Request-Id', requestId);
  headers.set('X-Client-App', CLIENT_APP);
  headers.set('X-Client-Version', CLIENT_VERSION);
  headers.set('X-Client-Time', String(Date.now()));

  if (token) headers.set('Authorization', `Bearer ${token}`);
  if (options.idempotencyKey) headers.set('Idempotency-Key', options.idempotencyKey);

  return headers;
}

function isAuthExpired(parsed: ParsedResponse): boolean {
  const code = resultCode(parsed.result);
  return parsed.status === 401 || isAuthResultCode(code);
}

function canRefreshAuth(parsed: ParsedResponse): boolean {
  return isRefreshableAuthResultCode(resultCode(parsed.result));
}

function throwParsedError(parsed: ParsedResponse): never {
  const code = resultCode(parsed.result);
  const message = resultMessage(parsed.result, parsed.ok ? '业务请求失败' : `HTTP ${parsed.status}`);
  const traceId = resultTraceId(parsed.result);

  if (isAuthExpired(parsed)) {
    throw new AuthExpiredError(message, { code, status: parsed.status, requestId: parsed.requestId, traceId });
  }

  if (!parsed.ok) {
    throw new AppApiError(message, { kind: 'http', code, status: parsed.status, requestId: parsed.requestId, traceId });
  }

  throw new AppApiError(message, { kind: 'result', code, status: parsed.status, requestId: parsed.requestId, traceId });
}

async function tokenFor(auth: AuthMode): Promise<string | null> {
  if (auth === 'none') return null;
  if (!authProvider) {
    if (auth === 'required') throw new AuthExpiredError('请先登录后再操作');
    return null;
  }

  const token = await authProvider.getValidAccessToken();
  if (!token && auth === 'required') throw new AuthExpiredError('请先登录后再操作');
  return token;
}

async function rawRequest(options: ApiRequestOptions, token: string | null, requestId: string): Promise<ParsedResponse> {
  const contentType = options.contentType ?? (options.body instanceof FormData ? 'form-data' : 'json');
  const response = await fetch(buildUrl(options.path), {
    method: options.method || 'GET',
    headers: headersForRequest(options, token, requestId),
    body: bodyForRequest(options.body, contentType),
    keepalive: options.keepalive,
  });

  return parseResponse(response);
}

export async function apiRequest<T>(options: ApiRequestOptions): Promise<T> {
  const auth = options.auth ?? 'optional';
  const requestId = createRequestId(options.operation);
  let token = await tokenFor(auth);

  let parsed: ParsedResponse;
  try {
    parsed = await rawRequest(options, token, requestId);
  } catch (error) {
    if (error instanceof AppApiError) throw error;
    throw new AppApiError(error instanceof Error ? error.message : '网络请求失败', {
      kind: 'network',
      requestId,
    });
  }

  if (isAuthExpired(parsed) && canRefreshAuth(parsed) && auth !== 'none' && !options.skipAuthRefresh && authProvider) {
    token = await authProvider.refreshOnce();
    if (token) parsed = await rawRequest(options, token, requestId);
  }

  const code = resultCode(parsed.result);
  if (!parsed.ok || code !== 0) {
    if (isAuthExpired(parsed) && !canRefreshAuth(parsed) && auth !== 'none' && authProvider) {
      authProvider.expire(resultMessage(parsed.result, '登录会话已失效，请重新登录'));
    }
    throwParsedError(parsed);
  }

  const payload = parsed.data && typeof parsed.data === 'object' && 'data' in parsed.data
    ? (parsed.data as { data: unknown }).data
    : parsed.data;

  return payload as T;
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  return apiRequest<T>({
    path,
    method: init?.method,
    headers: init?.headers,
    body: init?.body,
    auth: 'optional',
    contentType: init?.body instanceof FormData ? 'form-data' : 'json',
  });
}
