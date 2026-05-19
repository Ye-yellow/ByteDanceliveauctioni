import type { Money } from '../types/auction';

export const formatMoney = (v: Money) =>
  `¥${(v / 100).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
