import { normalizeBidRecord, normalizeLot, normalizeLotResult, normalizeMoney, normalizeOrder, normalizePayment, normalizePlaceBidResponse, normalizeRoom, normalizeRoomSnapshot } from '../../../shared/api/adapters';
import { apiRequest } from '../../../shared/api/httpClient';
import type {
  BidRecord,
  BidRecordList,
  Lot,
  LotResult,
  Money,
  MyBidsQuery,
  MyOrdersQuery,
  OrderList,
  OrderSummary,
  PaymentSummary,
  PlaceBidRequest,
  PlaceBidResponse,
  Room,
  RoomSnapshot,
} from '../../../shared/api/types';
import { authSession } from '../../../shared/auth/authSession';

type SnapshotReply = { snapshot?: unknown };
type LotResultReply = LotResult | { lot?: unknown; order?: unknown; orderId?: string; order_id?: string };
type ListRoomsReply = { rooms?: unknown[] };
type ListLotsReply = { lots?: unknown[]; total?: number; nextPageToken?: string; next_page_token?: string };
type ListOrdersReply = { orders?: unknown[]; total?: number; page?: number; pageSize?: number; page_size?: number };
type ListBidsReply = { bids?: unknown[]; total?: number; page?: number; pageSize?: number; page_size?: number };
type MockPayReply = { paid: boolean; message?: string; order?: unknown; payment?: unknown };

function withQuery(path: string, query?: object): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query || {})) {
    if (value !== undefined && value !== '') params.set(key, String(value));
  }
  const suffix = params.toString();
  return suffix ? `${path}?${suffix}` : path;
}

export async function getRoomSnapshot(roomId: string): Promise<RoomSnapshot> {
  const reply = await apiRequest<SnapshotReply>({
    path: `/api/rooms/${encodeURIComponent(roomId)}/snapshot`,
    auth: 'optional',
    operation: 'getRoomSnapshot',
  });

  return normalizeRoomSnapshot(reply.snapshot ?? reply, roomId);
}

export async function listPublicRooms(): Promise<Room[]> {
  const reply = await apiRequest<ListRoomsReply>({
    path: '/api/rooms',
    auth: 'optional',
    operation: 'listPublicRooms',
  });

  return Array.isArray(reply.rooms) ? reply.rooms.map(normalizeRoom).filter((room) => Boolean(room.id)) : [];
}

export async function listRoomLots(roomId: string): Promise<Lot[]> {
  const reply = await apiRequest<ListLotsReply>({
    path: withQuery('/api/lots', { room_id: roomId }),
    auth: 'optional',
    operation: 'listRoomLots',
  });

  return Array.isArray(reply.lots) ? reply.lots.map(normalizeLot).filter((lot) => Boolean(lot.id)) : [];
}

export async function placeBid(lotId: string, payload: PlaceBidRequest): Promise<PlaceBidResponse> {
  await authSession.ensureReadyForBid();

  const reply = await apiRequest<unknown>({
    path: `/api/lots/${encodeURIComponent(lotId)}/bid`,
    method: 'POST',
    auth: 'required',
    idempotencyKey: payload.idempotencyKey,
    operation: 'placeBid',
    body: {
      amount: normalizeMoney(payload.amount),
      client_known_version: payload.clientKnownVersion,
      idempotency_key: payload.idempotencyKey,
    },
  });

  return normalizePlaceBidResponse(reply);
}

export async function getLotResult(lotId: string): Promise<LotResult> {
  const reply = await apiRequest<LotResultReply>({
    path: `/api/lots/${encodeURIComponent(lotId)}/result`,
    auth: 'optional',
    operation: 'getLotResult',
  });

  return normalizeLotResult(reply);
}

export async function listMyOrders(query: MyOrdersQuery = {}): Promise<OrderList> {
  const reply = await apiRequest<ListOrdersReply>({
    path: withQuery('/api/me/orders', query),
    auth: 'required',
    operation: 'listMyOrders',
  });

  const orders = Array.isArray(reply.orders)
    ? reply.orders.map((item) => normalizeOrder(item)).filter((order): order is OrderSummary => Boolean(order))
    : [];

  return {
    orders,
    total: Number(reply.total ?? orders.length),
    page: Number(reply.page ?? query.page ?? 1),
    pageSize: Number(reply.pageSize ?? reply.page_size ?? query.pageSize ?? orders.length),
  };
}

export async function listMyBids(query: MyBidsQuery = {}): Promise<BidRecordList> {
  const reply = await apiRequest<ListBidsReply>({
    path: withQuery('/api/me/bids', query),
    auth: 'required',
    operation: 'listMyBids',
  });

  const bids = Array.isArray(reply.bids)
    ? reply.bids.map(normalizeBidRecord).filter((bid): bid is BidRecord => Boolean(bid.id))
    : [];

  return {
    bids,
    total: Number(reply.total ?? bids.length),
    page: Number(reply.page ?? query.page ?? 1),
    pageSize: Number(reply.pageSize ?? reply.page_size ?? query.pageSize ?? bids.length),
  };
}

export async function mockPay(orderId: string, payload: { idempotencyKey: string; amount: Money }): Promise<{ paid: boolean; message?: string; order?: OrderSummary; payment?: PaymentSummary }> {
  const reply = await apiRequest<MockPayReply>({
    path: `/api/orders/${encodeURIComponent(orderId)}/mock-pay`,
    method: 'POST',
    body: {
      idempotencyKey: payload.idempotencyKey,
      amount: Number(payload.amount.amount),
      currency: payload.amount.currency,
    },
    auth: 'required',
    idempotencyKey: payload.idempotencyKey,
    operation: 'mockPay',
  });

  return {
    paid: Boolean(reply.paid),
    message: reply.message,
    order: normalizeOrder(reply.order),
    payment: normalizePayment(reply.payment),
  };
}
