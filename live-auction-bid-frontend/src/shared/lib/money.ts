import type { Money } from '../types/auction';

export const cny = (amount: number): Money => ({ amount, currency: 'CNY' });
export const formatMoney = (v?: Money | null) => `¥${(((v?.amount ?? 0) / 100)).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
