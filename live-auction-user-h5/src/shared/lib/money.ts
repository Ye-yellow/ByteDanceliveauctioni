import { normalizeMoney } from '../api/adapters';
import type { MoneyInput } from '../api/types';

export function moneyNumber(value?: MoneyInput): number {
  if (value === undefined || value === null || value === '') return 0;
  return Number(normalizeMoney(value).amount) || 0;
}

export function moneyMajorNumber(value?: MoneyInput): number {
  return moneyNumber(value) / 100;
}

export function amountFromMajor(value: number): number {
  return Math.max(0, Math.round(value * 100));
}

export function formatMoney(value?: MoneyInput): string {
  return `¥${moneyMajorNumber(value).toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`;
}
