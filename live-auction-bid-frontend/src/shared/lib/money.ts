import type { Money } from '../types/auction';

export const cny = (amount: number): Money => ({ amount, currency: 'CNY' });
export const moneyAmount = (v?: Money | null) => Number(v?.amount ?? 0);
export const formatMoney = (v?: Money | null) => `¥${(moneyAmount(v) / 100).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
