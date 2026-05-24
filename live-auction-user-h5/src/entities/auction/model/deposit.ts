import type { Lot, MoneyInput } from '../../../shared/api/types';
import { formatMoney } from '../../../shared/lib/money';

type LotWithOptionalDeposit = Lot & {
  depositAmount?: MoneyInput;
  deposit?: MoneyInput;
  guaranteeAmount?: MoneyInput;
};

export function lotDepositAmount(lot: Lot | null | undefined): MoneyInput {
  if (!lot) return { amount: 0, currency: 'CNY' };
  const expanded = lot as LotWithOptionalDeposit;
  return expanded.depositAmount ?? expanded.deposit ?? expanded.guaranteeAmount ?? { amount: 0, currency: 'CNY' };
}

export function formatLotDeposit(lot: Lot | null | undefined): string {
  return formatMoney(lotDepositAmount(lot));
}
