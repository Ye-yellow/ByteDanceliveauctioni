import { apiRequest } from '../../../shared/api/httpClient';
import type { ReplyResult } from '../../../shared/api/types';

export type ShopProduct = {
  id: string;
  title: string;
  subtitle?: string;
  description?: string;
  category: string;
  shopName: string;
  mainImageUrl: string;
  detailImageUrls: string[];
  tags: string[];
  badges: string[];
  priceAmount: number;
  originalPriceAmount?: number;
  currency: string;
  soldLabel: string;
  live: boolean;
  status?: string;
  skus: ShopSKU[];
};

export type ShopSKU = {
  id: string;
  productId: string;
  name: string;
  priceAmount: number;
  currency: string;
  stock: number;
};

export type ShopOrderStatus = 'pending_payment' | 'paid' | 'shipped' | 'completed' | 'cancelled' | string;
export type ShopPaymentStatus = 'init' | 'success' | string;
export type UserOrderSource = 'auction' | 'shop' | string;

export type ShopOrderItem = {
  id: string;
  orderId: string;
  productId: string;
  skuId: string;
  title: string;
  imageUrl: string;
  skuName: string;
  quantity: number;
  unitAmount: number;
  totalAmount: number;
  currency: string;
};

export type ShopOrder = {
  id: string;
  orderNo: string;
  userId: string;
  nickname?: string;
  status: ShopOrderStatus;
  paymentStatus: ShopPaymentStatus;
  paymentId?: string;
  shopName: string;
  totalAmount: number;
  currency: string;
  addressSnapshot?: string;
  shippingAddressId?: string;
  shippingAddressSnapshot?: ShopShippingAddressSnapshot | null;
  createdAtUnixMs: number;
  updatedAtUnixMs: number;
  paidAtUnixMs?: number;
  items: ShopOrderItem[];
};

export type UserOrderItem = {
  id: string;
  orderId: string;
  source: UserOrderSource;
  sourceItemId?: string;
  productId?: string;
  skuId?: string;
  lotId?: string;
  roomId?: string;
  title: string;
  imageUrl: string;
  skuName?: string;
  quantity: number;
  unitAmount: number;
  totalAmount: number;
  currency: string;
};

export type UserOrder = {
  id: string;
  source: UserOrderSource;
  sourceOrderId?: string;
  orderNo: string;
  mainAccountId?: string;
  userId: string;
  nickname?: string;
  status: ShopOrderStatus;
  paymentStatus: ShopPaymentStatus;
  paymentId?: string;
  title: string;
  shopName: string;
  totalAmount: number;
  currency: string;
  addressSnapshot?: string;
  shippingAddressId?: string;
  shippingAddressSnapshot?: ShopShippingAddressSnapshot | null;
  createdAtUnixMs: number;
  updatedAtUnixMs: number;
  paidAtUnixMs?: number;
  expiresAtUnixMs?: number;
  items: UserOrderItem[];
};

export type FrequentStore = {
  storeKey: string;
  storeName: string;
  source: UserOrderSource;
  orderCount: number;
  lastOrderAtUnixMs: number;
  imageUrl: string;
  targetUrl: string;
};

export type ShopShippingAddressSnapshot = {
  receiverName: string;
  phone: string;
  province: string;
  city: string;
  district: string;
  street: string;
  detail: string;
  fullAddress: string;
  postalCode?: string;
  tag?: string;
};

type ProductListReply = {
  result?: ReplyResult;
  products?: unknown[];
  total?: number;
  page?: number;
  pageSize?: number;
};

type ProductReply = {
  result?: ReplyResult;
  product?: unknown;
};

type OrderListReply = {
  result?: ReplyResult;
  orders?: unknown[];
  total?: number;
  page?: number;
  pageSize?: number;
};

type FrequentStoreListReply = {
  result?: ReplyResult;
  stores?: unknown[];
  total?: number;
  limit?: number;
};

type OrderReply = {
  result?: ReplyResult;
  order?: unknown;
  payment?: unknown;
  paid?: boolean;
};

export type ShopProductList = {
  products: ShopProduct[];
  total: number;
  page: number;
  pageSize: number;
};

export type ShopOrderList = {
  orders: ShopOrder[];
  total: number;
  page: number;
  pageSize: number;
};

export type UserOrderList = {
  orders: UserOrder[];
  total: number;
  page: number;
  pageSize: number;
};

export type FrequentStoreList = {
  stores: FrequentStore[];
  total: number;
  limit: number;
};

export const FALLBACK_PRODUCTS: ShopProduct[] = [
  product('imperial-green-jade-bangle', '冰阳绿翡翠手镯 正圈饱满细腻起光', '珠宝玉石', 'Ggboy珠宝严选', '/shop-assets/auction-lots/imperial-green-jade-bangle.png', 47000, 59900, '3.8万+', ['翡翠手镯', '正在直播', '包邮'], true, [
    sku('sku-jade-bangle-54', '54圈口', 47000, 88),
    sku('sku-jade-bangle-56', '56圈口', 49900, 65),
  ]),
  product('white-hetian-jade-bangle', '白月光和田玉手镯 温润通透日常款', '珠宝玉石', 'Yexieer珠宝店', '/shop-assets/auction-lots/white-hetian-jade-bangle.png', 8800, 12900, '6.4万+', ['和田玉', '低价开拍'], false, [
    sku('sku-white-jade-55', '55圈口', 8800, 120),
    sku('sku-white-jade-57', '57圈口', 9300, 80),
  ]),
  product('carved-jade-pendant', '天然翡翠平安扣项链 冰润飘花吊坠', '项链吊坠', 'Ggboy珠宝严选', '/shop-assets/auction-lots/carved-jade-pendant-necklace.png', 19900, 25900, '1.9万+', ['项链', '适合送礼'], true, [
    sku('sku-jade-pendant-gold', '金色链条', 19900, 76),
    sku('sku-jade-pendant-silver', '银色链条', 18900, 72),
  ]),
  product('freshwater-pearl-necklace', '淡水珍珠项链 近圆强光通勤气质款', '项链吊坠', '珍珠小姐旗舰店', '/shop-assets/auction-lots/freshwater-pearl-necklace.png', 36800, 42900, '2.6万+', ['珍珠', '送礼'], false, [
    sku('sku-pearl-42', '42cm', 36800, 45),
    sku('sku-pearl-45', '45cm', 38900, 38),
  ]),
  product('ruby-diamond-bracelet', '红宝石钻石手链 18K金精致叠戴款', '手链手串', '璀璨宝石馆', '/shop-assets/auction-lots/ruby-diamond-gold-bracelet.png', 69900, 89900, '9800+', ['红宝石', '18K金'], false, [
    sku('sku-ruby-bracelet-s', '15cm', 69900, 18),
    sku('sku-ruby-bracelet-m', '16.5cm', 72900, 16),
  ]),
  product('sapphire-diamond-necklace', '蓝宝石钻石项链 锁骨链高级感礼盒装', '项链吊坠', '璀璨宝石馆', '/shop-assets/auction-lots/sapphire-diamond-necklace.png', 75900, 99900, '1.2万+', ['蓝宝石', '礼盒'], true, [
    sku('sku-sapphire-necklace', '礼盒装', 75900, 22),
  ]),
];

export function formatShopMoney(amount: number | string | undefined): string {
  const cents = Number(amount ?? 0);
  const yuan = cents / 100;
  return yuan % 1 === 0 ? `${yuan.toFixed(0)}` : yuan.toFixed(2).replace(/0$/, '');
}

export async function listShopProducts(query: { q?: string; category?: string; page?: number; pageSize?: number } = {}): Promise<ShopProductList> {
  const reply = await apiRequest<ProductListReply>({
    path: withQuery('/api/shop/products', query),
    auth: 'none',
    operation: 'listShopProducts',
  });
  const products = Array.isArray(reply.products) ? reply.products.map(normalizeProduct).filter(Boolean) as ShopProduct[] : [];
  return {
    products,
    total: Number(reply.total ?? products.length),
    page: Number(reply.page ?? query.page ?? 1),
    pageSize: Number(reply.pageSize ?? query.pageSize ?? products.length),
  };
}

export async function getShopProduct(productId: string): Promise<ShopProduct> {
  const reply = await apiRequest<ProductReply>({
    path: `/api/shop/products/${encodeURIComponent(productId)}`,
    auth: 'none',
    operation: 'getShopProduct',
  });
  const product = normalizeProduct(reply.product);
  if (!product) throw new Error('商品不存在');
  return product;
}

export async function createShopOrder(payload: { skuId: string; quantity: number; addressId: string; idempotencyKey?: string }): Promise<ShopOrder> {
  const reply = await apiRequest<OrderReply>({
    path: '/api/shop/orders',
    method: 'POST',
    auth: 'required',
    body: payload,
    operation: 'createShopOrder',
  });
  const order = normalizeOrder(reply.order);
  if (!order) throw new Error('订单创建失败');
  return order;
}

export async function listShopOrders(query: { status?: string; q?: string; page?: number; pageSize?: number } = {}): Promise<ShopOrderList> {
  const reply = await apiRequest<OrderListReply>({
    path: withQuery('/api/shop/orders', query),
    auth: 'required',
    operation: 'listShopOrders',
  });
  const orders = Array.isArray(reply.orders) ? reply.orders.map(normalizeOrder).filter(Boolean) as ShopOrder[] : [];
  return {
    orders,
    total: Number(reply.total ?? orders.length),
    page: Number(reply.page ?? query.page ?? 1),
    pageSize: Number(reply.pageSize ?? query.pageSize ?? orders.length),
  };
}

export async function mockPayShopOrder(orderId: string, idempotencyKey: string): Promise<{ order: ShopOrder; paid: boolean }> {
  const reply = await apiRequest<OrderReply>({
    path: `/api/shop/orders/${encodeURIComponent(orderId)}/mock-pay`,
    method: 'POST',
    auth: 'required',
    body: { idempotencyKey },
    operation: 'mockPayShopOrder',
  });
  const order = normalizeOrder(reply.order);
  if (!order) throw new Error('支付失败');
  return { order, paid: Boolean(reply.paid) };
}

export async function listUserOrders(query: { source?: string; status?: string; q?: string; page?: number; pageSize?: number } = {}): Promise<UserOrderList> {
  const reply = await apiRequest<OrderListReply>({
    path: withQuery('/api/orders/me', query),
    auth: 'required',
    operation: 'listUserOrders',
  });
  const orders = Array.isArray(reply.orders) ? reply.orders.map(normalizeUserOrder).filter(Boolean) as UserOrder[] : [];
  return {
    orders,
    total: Number(reply.total ?? orders.length),
    page: Number(reply.page ?? query.page ?? 1),
    pageSize: Number(reply.pageSize ?? query.pageSize ?? orders.length),
  };
}

export async function listMyFrequentStores(query: { limit?: number } = {}): Promise<FrequentStoreList> {
  const reply = await apiRequest<FrequentStoreListReply>({
    path: withQuery('/api/orders/me/frequent-stores', query),
    auth: 'required',
    operation: 'listMyFrequentStores',
  });
  const stores = Array.isArray(reply.stores) ? reply.stores.map(normalizeFrequentStore).filter(Boolean) as FrequentStore[] : [];
  return {
    stores,
    total: Number(reply.total ?? stores.length),
    limit: Number(reply.limit ?? query.limit ?? stores.length),
  };
}

export async function mockPayUserOrder(orderId: string, idempotencyKey: string, amount?: number, currency?: string): Promise<{ order: UserOrder; paid: boolean }> {
  const reply = await apiRequest<OrderReply>({
    path: `/api/orders/${encodeURIComponent(orderId)}/mock-pay`,
    method: 'POST',
    auth: 'required',
    body: { idempotencyKey, amount, currency },
    operation: 'mockPayUserOrder',
  });
  const order = normalizeUserOrder(reply.order);
  if (!order) throw new Error('支付失败');
  return { order, paid: Boolean(reply.paid) };
}

function product(id: string, title: string, category: string, shopName: string, image: string, priceAmount: number, originalPriceAmount: number, soldLabel: string, tags: string[], live: boolean, skus: ShopSKU[]): ShopProduct {
  return {
    id,
    title,
    subtitle: tags.join(' · '),
    description: `${title}，支持直播间看货和基础售后服务。`,
    category,
    shopName,
    mainImageUrl: image,
    detailImageUrls: [image],
    tags,
    badges: live ? ['直播中'] : tags.slice(0, 1),
    priceAmount,
    originalPriceAmount,
    currency: 'CNY',
    soldLabel,
    live,
    status: 'active',
    skus: skus.map((item) => ({ ...item, productId: id })),
  };
}

function sku(id: string, name: string, priceAmount: number, stock: number): ShopSKU {
  return { id, productId: '', name, priceAmount, currency: 'CNY', stock };
}

function normalizeProduct(raw: unknown): ShopProduct | null {
  if (!raw || typeof raw !== 'object') return null;
  const item = raw as Record<string, unknown>;
  const id = String(item.id ?? '');
  const title = String(item.title ?? '');
  if (!id || !title) return null;
  const skus = Array.isArray(item.skus) ? item.skus.map(normalizeSKU).filter(Boolean) as ShopSKU[] : [];
  return {
    id,
    title,
    subtitle: String(item.subtitle ?? ''),
    description: String(item.description ?? ''),
    category: String(item.category ?? '推荐'),
    shopName: String(item.shopName ?? item.shop_name ?? '抖音商城'),
    mainImageUrl: String(item.mainImageUrl ?? item.main_image_url ?? ''),
    detailImageUrls: stringArray(item.detailImageUrls ?? item.detail_image_urls),
    tags: stringArray(item.tags),
    badges: stringArray(item.badges),
    priceAmount: Number(item.priceAmount ?? item.price_amount ?? 0),
    originalPriceAmount: Number(item.originalPriceAmount ?? item.original_price_amount ?? 0),
    currency: String(item.currency ?? 'CNY'),
    soldLabel: String(item.soldLabel ?? item.sold_label ?? ''),
    live: Boolean(item.live),
    status: String(item.status ?? 'active'),
    skus,
  };
}

function normalizeSKU(raw: unknown): ShopSKU | null {
  if (!raw || typeof raw !== 'object') return null;
  const item = raw as Record<string, unknown>;
  const id = String(item.id ?? '');
  if (!id) return null;
  return {
    id,
    productId: String(item.productId ?? item.product_id ?? ''),
    name: String(item.name ?? ''),
    priceAmount: Number(item.priceAmount ?? item.price_amount ?? 0),
    currency: String(item.currency ?? 'CNY'),
    stock: Number(item.stock ?? 0),
  };
}

function normalizeOrder(raw: unknown): ShopOrder | null {
  if (!raw || typeof raw !== 'object') return null;
  const item = raw as Record<string, unknown>;
  const id = String(item.id ?? '');
  if (!id) return null;
  return {
    id,
    orderNo: String(item.orderNo ?? item.order_no ?? ''),
    userId: String(item.userId ?? item.user_id ?? ''),
    nickname: String(item.nickname ?? ''),
    status: String(item.status ?? 'pending_payment'),
    paymentStatus: String(item.paymentStatus ?? item.payment_status ?? 'init'),
    paymentId: String(item.paymentId ?? item.payment_id ?? ''),
    shopName: String(item.shopName ?? item.shop_name ?? ''),
    totalAmount: Number(item.totalAmount ?? item.total_amount ?? 0),
    currency: String(item.currency ?? 'CNY'),
    addressSnapshot: String(item.addressSnapshot ?? item.address_snapshot ?? ''),
    shippingAddressId: String(item.shippingAddressId ?? item.shipping_address_id ?? ''),
    shippingAddressSnapshot: normalizeShippingAddressSnapshot(item.shippingAddressSnapshot ?? item.shipping_address_snapshot),
    createdAtUnixMs: Number(item.createdAtUnixMs ?? item.created_at_unix_ms ?? 0),
    updatedAtUnixMs: Number(item.updatedAtUnixMs ?? item.updated_at_unix_ms ?? 0),
    paidAtUnixMs: Number(item.paidAtUnixMs ?? item.paid_at_unix_ms ?? 0),
    items: Array.isArray(item.items) ? item.items.map(normalizeOrderItem).filter(Boolean) as ShopOrderItem[] : [],
  };
}

function normalizeUserOrder(raw: unknown): UserOrder | null {
  if (!raw || typeof raw !== 'object') return null;
  const item = raw as Record<string, unknown>;
  const id = String(item.id ?? '');
  if (!id) return null;
  return {
    id,
    source: String(item.source ?? 'shop'),
    sourceOrderId: String(item.sourceOrderId ?? item.source_order_id ?? ''),
    orderNo: String(item.orderNo ?? item.order_no ?? ''),
    mainAccountId: String(item.mainAccountId ?? item.main_account_id ?? ''),
    userId: String(item.userId ?? item.user_id ?? ''),
    nickname: String(item.nickname ?? ''),
    status: String(item.status ?? 'pending_payment'),
    paymentStatus: String(item.paymentStatus ?? item.payment_status ?? 'init'),
    paymentId: String(item.paymentId ?? item.payment_id ?? ''),
    title: String(item.title ?? ''),
    shopName: String(item.shopName ?? item.shop_name ?? ''),
    totalAmount: Number(item.totalAmount ?? item.total_amount ?? 0),
    currency: String(item.currency ?? 'CNY'),
    addressSnapshot: String(item.addressSnapshot ?? item.address_snapshot ?? ''),
    shippingAddressId: String(item.shippingAddressId ?? item.shipping_address_id ?? ''),
    shippingAddressSnapshot: normalizeShippingAddressSnapshot(item.shippingAddressSnapshot ?? item.shipping_address_snapshot),
    createdAtUnixMs: Number(item.createdAtUnixMs ?? item.created_at_unix_ms ?? 0),
    updatedAtUnixMs: Number(item.updatedAtUnixMs ?? item.updated_at_unix_ms ?? 0),
    paidAtUnixMs: Number(item.paidAtUnixMs ?? item.paid_at_unix_ms ?? 0),
    expiresAtUnixMs: Number(item.expiresAtUnixMs ?? item.expires_at_unix_ms ?? 0),
    items: Array.isArray(item.items) ? item.items.map(normalizeUserOrderItem).filter(Boolean) as UserOrderItem[] : [],
  };
}

function normalizeFrequentStore(raw: unknown): FrequentStore | null {
  if (!raw || typeof raw !== 'object') return null;
  const item = raw as Record<string, unknown>;
  const storeKey = String(item.storeKey ?? item.store_key ?? '');
  const storeName = String(item.storeName ?? item.store_name ?? '');
  if (!storeKey || !storeName) return null;
  return {
    storeKey,
    storeName,
    source: String(item.source ?? 'shop'),
    orderCount: Number(item.orderCount ?? item.order_count ?? 0),
    lastOrderAtUnixMs: Number(item.lastOrderAtUnixMs ?? item.last_order_at_unix_ms ?? 0),
    imageUrl: String(item.imageUrl ?? item.image_url ?? ''),
    targetUrl: String(item.targetUrl ?? item.target_url ?? '/shop') || '/shop',
  };
}

function normalizeShippingAddressSnapshot(raw: unknown): ShopShippingAddressSnapshot | null {
  if (!raw || typeof raw !== 'object') return null;
  const item = raw as Record<string, unknown>;
  const receiverName = String(item.receiverName ?? item.receiver_name ?? item.receiver ?? '');
  const phone = String(item.phone ?? '');
  if (!receiverName || !phone) return null;
  return {
    receiverName,
    phone,
    province: String(item.province ?? ''),
    city: String(item.city ?? ''),
    district: String(item.district ?? ''),
    street: String(item.street ?? ''),
    detail: String(item.detail ?? ''),
    fullAddress: String(item.fullAddress ?? item.full_address ?? ''),
    postalCode: String(item.postalCode ?? item.postal_code ?? ''),
    tag: String(item.tag ?? ''),
  };
}

function normalizeOrderItem(raw: unknown): ShopOrderItem | null {
  if (!raw || typeof raw !== 'object') return null;
  const item = raw as Record<string, unknown>;
  const id = String(item.id ?? '');
  if (!id) return null;
  return {
    id,
    orderId: String(item.orderId ?? item.order_id ?? ''),
    productId: String(item.productId ?? item.product_id ?? ''),
    skuId: String(item.skuId ?? item.sku_id ?? ''),
    title: String(item.title ?? ''),
    imageUrl: String(item.imageUrl ?? item.image_url ?? ''),
    skuName: String(item.skuName ?? item.sku_name ?? ''),
    quantity: Number(item.quantity ?? 0),
    unitAmount: Number(item.unitAmount ?? item.unit_amount ?? 0),
    totalAmount: Number(item.totalAmount ?? item.total_amount ?? 0),
    currency: String(item.currency ?? 'CNY'),
  };
}

function normalizeUserOrderItem(raw: unknown): UserOrderItem | null {
  if (!raw || typeof raw !== 'object') return null;
  const item = raw as Record<string, unknown>;
  const id = String(item.id ?? '');
  if (!id) return null;
  return {
    id,
    orderId: String(item.orderId ?? item.order_id ?? ''),
    source: String(item.source ?? 'shop'),
    sourceItemId: String(item.sourceItemId ?? item.source_item_id ?? ''),
    productId: String(item.productId ?? item.product_id ?? ''),
    skuId: String(item.skuId ?? item.sku_id ?? ''),
    lotId: String(item.lotId ?? item.lot_id ?? ''),
    roomId: String(item.roomId ?? item.room_id ?? ''),
    title: String(item.title ?? ''),
    imageUrl: String(item.imageUrl ?? item.image_url ?? ''),
    skuName: String(item.skuName ?? item.sku_name ?? ''),
    quantity: Number(item.quantity ?? 0),
    unitAmount: Number(item.unitAmount ?? item.unit_amount ?? 0),
    totalAmount: Number(item.totalAmount ?? item.total_amount ?? 0),
    currency: String(item.currency ?? 'CNY'),
  };
}

function stringArray(raw: unknown): string[] {
  if (Array.isArray(raw)) return raw.map((item) => String(item)).filter(Boolean);
  if (typeof raw === 'string' && raw.trim()) {
    try {
      const parsed = JSON.parse(raw) as unknown;
      if (Array.isArray(parsed)) return parsed.map((item) => String(item)).filter(Boolean);
    } catch {
      return raw.split(',').map((item) => item.trim()).filter(Boolean);
    }
  }
  return [];
}

function withQuery(path: string, query: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  Object.entries(query).forEach(([key, value]) => {
    if (value !== undefined && value !== '') params.set(key, String(value));
  });
  const suffix = params.toString();
  return suffix ? `${path}?${suffix}` : path;
}
