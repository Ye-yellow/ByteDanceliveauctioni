import { apiRequest } from '../../../shared/api/httpClient';
import { toQueryString } from '../../../shared/api/query';
import { assertOkResult } from '../../../shared/api/result';
import { listLots } from '../../auction/api/auctionApi';
import type { Lot, ReplyResult } from '../../../shared/api/types';
import type { LotResultReply, OrderPage, OrderRecord, OrderStatus, OrderSummary } from '../model/orderTypes';

export type AdminOrdersQuery = {
  page?: number;
  pageSize?: number;
  status?: OrderStatus | '';
  lotId?: string;
  buyer?: string;
};

type ListOrdersReply = {
  result?: ReplyResult;
  orders?: OrderSummary[];
  total?: number | string;
  page?: number | string;
  pageSize?: number | string;
};

function isSettlementCandidate(lot: Lot) {
  return lot.status === 'LOT_STATUS_SETTLED'
    || lot.status === 'LOT_STATUS_SOLD'
    || Boolean(lot.settledAtUnixMs)
    || Boolean(lot.winnerUserId);
}

export async function getLotResult(lotId: string) {
  return assertOkResult(await apiRequest<LotResultReply>({
    path: `/api/lots/${encodeURIComponent(lotId)}/result`,
    method: 'GET',
    operation: 'lot-result',
  }));
}

export async function listAdminOrders(query: AdminOrdersQuery = {}): Promise<OrderPage> {
  const page = query.page ?? 1;
  const pageSize = query.pageSize ?? 20;
  const reply = assertOkResult(await apiRequest<ListOrdersReply>({
    path: `/api/admin/orders${toQueryString({
      page,
      pageSize,
      status: query.status,
      lotId: query.lotId?.trim(),
      buyer: query.buyer?.trim(),
    })}`,
    method: 'GET',
    operation: 'admin-list-orders',
  }));
  return {
    orders: reply.orders ?? [],
    total: Number(reply.total ?? 0),
    page: Number(reply.page ?? page),
    pageSize: Number(reply.pageSize ?? pageSize),
  };
}

export async function listSettlementOrders(roomId: string): Promise<OrderRecord[]> {
  const lots = await listLots(roomId);
  const settledLots = lots.filter(isSettlementCandidate);
  const results = await Promise.all(settledLots.map(async (lot) => {
    const result = await getLotResult(lot.id);
    return {
      lot: result.lot ?? lot,
      auctionState: result.auctionState ?? (lot.status === 'LOT_STATUS_CANCELLED' ? 'CANCELLED' : 'SOLD'),
      order: result.order ?? null,
    } satisfies OrderRecord;
  }));
  return results;
}
