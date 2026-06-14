import { apiRequest } from '../api/httpClient';

export type DeliveryAddress = {
  id: string;
  province: string;
  city: string;
  district: string;
  street: string;
  detail: string;
  receiver: string;
  phone: string;
  postalCode?: string;
  tag?: string;
  isDefault: boolean;
};

export type DeliveryAddressInput = Omit<DeliveryAddress, 'id'>;

type AddressListReply = {
  addresses?: unknown[];
};

type AddressReply = {
  address?: unknown;
};

function normalizeAddress(raw: unknown): DeliveryAddress | null {
  if (!raw || typeof raw !== 'object') return null;
  const item = raw as Record<string, unknown>;
  const id = String(item.id ?? '');
  const receiver = String(item.receiver ?? item.receiverName ?? item.receiver_name ?? '');
  const phone = String(item.phone ?? '');
  if (!id || !receiver || !phone) return null;
  return {
    id,
    province: String(item.province ?? ''),
    city: String(item.city ?? ''),
    district: String(item.district ?? ''),
    street: String(item.street ?? ''),
    detail: String(item.detail ?? ''),
    receiver,
    phone,
    postalCode: String(item.postalCode ?? item.postal_code ?? ''),
    tag: String(item.tag ?? ''),
    isDefault: Boolean(item.isDefault ?? item.is_default),
  };
}

function normalizeAddresses(items: unknown[] | undefined): DeliveryAddress[] {
  return Array.isArray(items) ? items.map(normalizeAddress).filter((item): item is DeliveryAddress => Boolean(item)) : [];
}

function payloadFromInput(input: DeliveryAddressInput) {
  return {
    receiverName: input.receiver.trim(),
    phone: input.phone.trim(),
    province: input.province.trim(),
    city: input.city.trim(),
    district: input.district.trim(),
    street: input.street.trim(),
    detail: input.detail.trim(),
    postalCode: input.postalCode?.trim() ?? '',
    tag: input.tag?.trim() ?? '',
    isDefault: Boolean(input.isDefault),
  };
}

export async function listDeliveryAddresses(): Promise<DeliveryAddress[]> {
  const reply = await apiRequest<AddressListReply>({
    path: '/api/shop/addresses',
    auth: 'required',
    operation: 'listDeliveryAddresses',
  });
  return normalizeAddresses(reply.addresses);
}

export function getDefaultDeliveryAddress(addresses: DeliveryAddress[] = []): DeliveryAddress | null {
  return addresses.find((address) => address.isDefault) ?? addresses[0] ?? null;
}

export async function createDeliveryAddress(input: DeliveryAddressInput): Promise<DeliveryAddress> {
  const reply = await apiRequest<AddressReply>({
    path: '/api/shop/addresses',
    method: 'POST',
    auth: 'required',
    operation: 'createDeliveryAddress',
    body: payloadFromInput(input),
  });
  const address = normalizeAddress(reply.address);
  if (!address) throw new Error('地址保存失败');
  return address;
}

export async function updateDeliveryAddress(id: string, input: DeliveryAddressInput): Promise<DeliveryAddress> {
  const reply = await apiRequest<AddressReply>({
    path: `/api/shop/addresses/${encodeURIComponent(id)}`,
    method: 'PUT',
    auth: 'required',
    operation: 'updateDeliveryAddress',
    body: payloadFromInput(input),
  });
  const address = normalizeAddress(reply.address);
  if (!address) throw new Error('地址保存失败');
  return address;
}

export async function setDefaultDeliveryAddress(id: string): Promise<DeliveryAddress[]> {
  const reply = await apiRequest<AddressListReply>({
    path: `/api/shop/addresses/${encodeURIComponent(id)}/default`,
    method: 'POST',
    auth: 'required',
    operation: 'setDefaultDeliveryAddress',
  });
  return normalizeAddresses(reply.addresses);
}

export async function deleteDeliveryAddress(id: string): Promise<void> {
  await apiRequest<AddressReply>({
    path: `/api/shop/addresses/${encodeURIComponent(id)}`,
    method: 'DELETE',
    auth: 'required',
    operation: 'deleteDeliveryAddress',
  });
}

export function formatAddressRegion(address: DeliveryAddress): string {
  return [address.province, address.city, address.district, address.street].filter(Boolean).join('');
}

export function formatAddressLine(address: DeliveryAddress): string {
  return `${formatAddressRegion(address)}${address.detail}`;
}

export function formatMaskedPhone(phone: string): string {
  const clean = phone.replace(/\s+/g, '');
  if (clean.length < 8) return clean;
  return `${clean.slice(0, 3)}****${clean.slice(-3)}`;
}

export function formatAddressSummary(address: DeliveryAddress): string {
  return `${address.receiver} ${formatMaskedPhone(address.phone)}`;
}

export function formatAddressClipboardText(address: DeliveryAddress): string {
  return `${formatAddressLine(address)} ${address.receiver} ${address.phone}`;
}
