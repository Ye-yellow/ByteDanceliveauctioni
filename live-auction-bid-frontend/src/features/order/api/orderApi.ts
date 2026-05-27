import { apiRequest } from '../../../shared/api/httpClient';
import { normalizeLot } from '../../../shared/api/normalizers';
import { toQueryString } from '../../../shared/api/query';
import { assertOkResult } from '../../../shared/api/result';
import { listLots } from '../../auction/api/auctionApi';
import type { Lot, ReplyResult } from '../../../shared/api/types';
import type { LotResultReply, OrderPage, OrderRecord, OrderStatus, OrderSummary, PaymentStatus } from '../model/orderTypes';

export type AdminOrdersQuery = {
  page?: number;
  pageSize?: number;
  status?: OrderStatus | '';
  paymentStatus?: PaymentStatus | '';
  lotId?: string;
  buyer?: string;
};

type ListOrdersReply = {
  result?: ReplyResult;
  orders?: unknown[];
  total?: number | string;
  page?: number | string;
  pageSize?: number | string;
};

function requiredValue<T>(value: T | undefined | null, field: string): T {
  if (value === undefined || value === null || value === '') throw new Error(`response missing ${field}`);
  return value;
}

function requireOrders(value: unknown[] | undefined): unknown[] {
  if (!Array.isArray(value)) throw new Error('response missing orders');
  return value;
}

function readField(raw: Record<string, unknown>, camel: string, snake = camel) {
  return raw[camel] ?? raw[snake];
}

function asRecord(input: unknown, field: string): Record<string, unknown> {
  if (!input || typeof input !== 'object' || Array.isArray(input)) throw new Error(`response missing ${field}`);
  return input as Record<string, unknown>;
}

function stringValue(value: unknown) {
  return value === undefined || value === null ? '' : String(value);
}

function normalizeOrderSummary(input: unknown): OrderSummary {
  const raw = asRecord(input, 'order');
  return {
    id: stringValue(requiredValue(readField(raw, 'id'), 'order.id')),
    lotId: stringValue(requiredValue(readField(raw, 'lotId', 'lot_id'), 'order.lotId')),
    roomId: stringValue(requiredValue(readField(raw, 'roomId', 'room_id'), 'order.roomId')),
    lotTitle: stringValue(readField(raw, 'lotTitle', 'lot_title')),
    lotImageUrl: stringValue(readField(raw, 'lotImageUrl', 'lot_image_url')),
    buyerUserId: stringValue(readField(raw, 'buyerUserId', 'buyer_user_id')),
    buyerNickname: stringValue(readField(raw, 'buyerNickname', 'buyer_nickname')),
    status: stringValue(requiredValue(readField(raw, 'status'), 'order.status')) as OrderStatus,
    paymentStatus: stringValue(requiredValue(readField(raw, 'paymentStatus', 'payment_status'), 'order.paymentStatus')) as PaymentStatus,
    paymentId: stringValue(readField(raw, 'paymentId', 'payment_id')) || undefined,
    amount: requiredValue(readField(raw, 'amount'), 'order.amount') as number | string,
    currency: stringValue(readField(raw, 'currency') ?? 'CNY') || 'CNY',
    createdAtUnixMs: requiredValue(readField(raw, 'createdAtUnixMs', 'created_at_unix_ms'), 'order.createdAtUnixMs') as number | string,
    updatedAtUnixMs: requiredValue(readField(raw, 'updatedAtUnixMs', 'updated_at_unix_ms'), 'order.updatedAtUnixMs') as number | string,
    expiresAtUnixMs: requiredValue(readField(raw, 'expiresAtUnixMs', 'expires_at_unix_ms'), 'order.expiresAtUnixMs') as number | string,
    paidAtUnixMs: readField(raw, 'paidAtUnixMs', 'paid_at_unix_ms') as number | string | undefined,
    version: readField(raw, 'version') as number | string | undefined,
  };
}

function normalizeLotResultReply(input: unknown): LotResultReply {
  const raw = asRecord(input, 'lotResult');
  const lot = readField(raw, 'lot');
  const order = readField(raw, 'order');
  return {
    ...(raw as LotResultReply),
    lot: lot === undefined || lot === null ? undefined : normalizeLot(lot),
    auctionState: stringValue(readField(raw, 'auctionState', 'auction_state')) as LotResultReply['auctionState'],
    order: order === undefined || order === null ? undefined : normalizeOrderSummary(order),
  };
}

function isSettlementCandidate(lot: Lot) {
  return lot.status === 'LOT_STATUS_SETTLED'
    || Number(lot.settledAtUnixMs || 0) > 0
    || Boolean(lot.winnerUserId);
}

export async function getLotResult(lotId: string) {
  return normalizeLotResultReply(assertOkResult(await apiRequest<LotResultReply>({
    path: `/api/lots/${encodeURIComponent(lotId)}/result`,
    method: 'GET',
    operation: 'lot-result',
  })));
}

export async function listAdminOrders(query: AdminOrdersQuery = {}): Promise<OrderPage> {
  const page = query.page ?? 1;
  const pageSize = query.pageSize ?? 20;
  const reply = assertOkResult(await apiRequest<ListOrdersReply>({
    path: `/api/admin/orders${toQueryString({
      page,
      pageSize,
      status: query.status,
      paymentStatus: query.paymentStatus,
      lotId: query.lotId?.trim(),
      buyer: query.buyer?.trim(),
    })}`,
    method: 'GET',
    operation: 'admin-list-orders',
  }));
  return {
    orders: requireOrders(reply.orders).map(normalizeOrderSummary),
    total: Number(requiredValue(reply.total, 'total')),
    page: Number(requiredValue(reply.page, 'page')),
    pageSize: Number(requiredValue(reply.pageSize, 'pageSize')),
  };
}

export async function listSettlementOrders(roomId: string): Promise<OrderRecord[]> {
  const lots = await listLots(roomId);
  const settledLots = lots.filter(isSettlementCandidate);
  const results = await Promise.all(settledLots.map(async (lot) => {
    const result = await getLotResult(lot.id);
    return {
      lot: result.lot ?? lot,
      auctionState: result.auctionState ?? (lot.status === 'LOT_STATUS_CANCELLED' ? 'CANCELLED' : 'SETTLED'),
      order: result.order ?? null,
    } satisfies OrderRecord;
  }));
  return results;
}
