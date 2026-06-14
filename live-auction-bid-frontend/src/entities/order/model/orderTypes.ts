import type { Lot, ReplyResult } from '../../../shared/api/types';
import type { OrderStatus, PaymentStatus } from './orderStatus';

export type AuctionState = 'DRAFT' | 'QUEUED' | 'LIVE' | 'EXTENDED' | 'SETTLED' | 'CANCELLED' | 'FAILED' | (string & {});

export type DeliveryAddressSnapshot = {
  addressId?: string;
  receiverName?: string;
  receiver?: string;
  phone?: string;
  province?: string;
  city?: string;
  district?: string;
  street?: string;
  detail?: string;
  fullAddress?: string;
};

export type OrderSummary = {
  id: string;
  lotId: string;
  roomId: string;
  lotTitle: string;
  lotImageUrl: string;
  buyerUserId: string;
  buyerNickname?: string;
  status: OrderStatus;
  paymentStatus: PaymentStatus;
  paymentId?: string;
  shippingAddressId?: string;
  shippingAddressSnapshot?: DeliveryAddressSnapshot | null;
  addressSnapshot?: string;
  amount: number | string;
  currency: string;
  createdAtUnixMs: number | string;
  updatedAtUnixMs: number | string;
  expiresAtUnixMs: number | string;
  paidAtUnixMs?: number | string;
  version?: number | string;
};

export type LotResultReply = {
  result?: ReplyResult;
  lot?: Lot;
  auctionState?: AuctionState;
  order?: OrderSummary;
};

export type OrderRecord = {
  lot: Lot;
  auctionState: AuctionState;
  order: OrderSummary | null;
};

export type OrderPage = {
  orders: OrderSummary[];
  total: number;
  page: number;
  pageSize: number;
};
