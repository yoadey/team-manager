import { t } from '@/i18n';

export interface ValidationResult<T = unknown> {
  ok: boolean;
  message?: string;
  value?: T;
}

type MoneyOptions = {
  positive?: boolean;
  allowZero?: boolean;
  max?: number;
};

// Matches the backend's `amount` maximum (100000000 cents, openapi.yaml) on
// CreateTransactionRequest/CreatePenaltyRequest/UpdateContributionRequest.
export const MAX_MONEY_AMOUNT_EUROS = 1000000;

const text = (value: unknown) => String(value ?? '').trim();

export function validateMoneyAmount(value: unknown, options: MoneyOptions = {}): ValidationResult<number> {
  const raw = text(value).replace(',', '.');
  if (!raw) return { ok: false, message: t('validation.moneyMissing') };
  const amount = Number(raw);
  if (!Number.isFinite(amount)) return { ok: false, message: t('validation.moneyInvalid') };
  const rounded = Math.round((amount + Number.EPSILON) * 100) / 100;
  if (options.positive && rounded <= 0) return { ok: false, message: t('validation.moneyPositive') };
  if (!options.positive && options.allowZero !== false && rounded < 0)
    return { ok: false, message: t('validation.moneyNonNegative') };
  if (!options.positive && options.allowZero === false && rounded <= 0)
    return { ok: false, message: t('validation.moneyNonZero') };
  if (options.max !== undefined && rounded > options.max)
    return { ok: false, message: t('validation.moneyTooLarge') };
  return { ok: true, value: rounded };
}
