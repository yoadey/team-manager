import { t } from '@/i18n';

export interface ValidationResult<T = unknown> {
  ok: boolean;
  message?: string;
  value?: T;
}

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
const TIME_RE = /^\d{2}:\d{2}$/;
const MIN_REPEAT_WEEKS = 2;
const MAX_REPEAT_WEEKS = 26;

type MoneyOptions = {
  positive?: boolean;
  allowZero?: boolean;
  max?: number;
};

// Matches the backend's `amount` maximum (100000000 cents, openapi.yaml) on
// CreateTransactionRequest/CreatePenaltyRequest/UpdateContributionRequest.
export const MAX_MONEY_AMOUNT_EUROS = 1000000;

type EventForm = {
  title?: unknown;
  date?: unknown;
  meetT?: unknown;
  startT?: unknown;
  endT?: unknown;
  recurring?: unknown;
  repeatWeeks?: unknown;
};

type PollForm = {
  question?: unknown;
  opt0?: unknown;
  opt1?: unknown;
  opt2?: unknown;
  opt3?: unknown;
};

const text = (value: unknown) => String(value ?? '').trim();

const validDate = (value: string) => {
  if (!DATE_RE.test(value)) return false;
  const [year, month, day] = value.split('-').map(Number);
  const date = new Date(Date.UTC(year, month - 1, day));
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day;
};

const minutes = (value: string) => {
  if (!TIME_RE.test(value)) return null;
  const [h, m] = value.split(':').map(Number);
  if (h < 0 || h > 23 || m < 0 || m > 59) return null;
  return h * 60 + m;
};

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

export function validateDateRange(from: unknown, to: unknown): ValidationResult<{ from: string; to: string }> {
  const start = text(from);
  const end = text(to);
  if (!start) return { ok: false, message: t('validation.dateRangeStartMissing') };
  if (!end) return { ok: false, message: t('validation.dateRangeEndMissing') };
  if (!validDate(start)) return { ok: false, message: t('validation.dateRangeStartInvalid') };
  if (!validDate(end)) return { ok: false, message: t('validation.dateRangeEndInvalid') };
  if (end < start) return { ok: false, message: t('validation.dateRangeOrder') };
  return { ok: true, value: { from: start, to: end } };
}

export function validateEventForm(
  form: EventForm,
  mode: 'create' | 'edit' = 'create',
): ValidationResult<{ repeatWeeks: number }> {
  if (!text(form.title)) return { ok: false, message: t('validation.eventTitleMissing') };
  const date = text(form.date);
  if (!date) return { ok: false, message: t('validation.eventDateMissing') };
  if (!validDate(date)) return { ok: false, message: t('validation.eventDateInvalid') };

  const start = text(form.startT);
  const end = text(form.endT);
  const meet = text(form.meetT);
  const startMin = start ? minutes(start) : null;
  const endMin = end ? minutes(end) : null;
  const meetMin = meet ? minutes(meet) : null;

  if (start && startMin == null) return { ok: false, message: t('validation.eventStartInvalid') };
  if (end && endMin == null) return { ok: false, message: t('validation.eventEndInvalid') };
  if (meet && meetMin == null) return { ok: false, message: t('validation.eventMeetTimeInvalid') };
  if (startMin != null && endMin != null && endMin <= startMin)
    return { ok: false, message: t('validation.eventEndBeforeStart') };
  if (meetMin != null && startMin != null && meetMin > startMin)
    return { ok: false, message: t('validation.eventMeetAfterStart') };

  const repeatWeeks = Number(form.repeatWeeks);
  if (mode === 'create' && form.recurring) {
    if (!Number.isInteger(repeatWeeks)) return { ok: false, message: t('validation.eventRepeatWeeksInteger') };
    if (repeatWeeks < MIN_REPEAT_WEEKS || repeatWeeks > MAX_REPEAT_WEEKS)
      return {
        ok: false,
        message: t('validation.eventRepeatWeeksRange', { min: MIN_REPEAT_WEEKS, max: MAX_REPEAT_WEEKS }),
      };
  }
  return { ok: true, value: { repeatWeeks: Number.isFinite(repeatWeeks) ? repeatWeeks : MIN_REPEAT_WEEKS } };
}

export function validatePollForm(form: PollForm): ValidationResult<{ question: string; options: string[] }> {
  const question = text(form.question);
  if (!question) return { ok: false, message: t('validation.pollQuestionMissing') };
  const options = [form.opt0, form.opt1, form.opt2, form.opt3].map(text).filter(Boolean);
  if (options.length < 2) return { ok: false, message: t('validation.pollOptionsMissing') };
  if (new Set(options.map((o) => o.toLocaleLowerCase('de'))).size !== options.length)
    return { ok: false, message: t('validation.pollOptionsDuplicate') };
  return { ok: true, value: { question, options } };
}

export function validateRequiredText(value: unknown, message: string): ValidationResult<string> {
  const cleaned = text(value);
  return cleaned ? { ok: true, value: cleaned } : { ok: false, message };
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function validateEmail(value: unknown, message: string): ValidationResult<string> {
  const v = text(value);
  if (!v) return { ok: true, value: '' };
  return EMAIL_RE.test(v) ? { ok: true, value: v } : { ok: false, message };
}

// Accepts common DE/AT/CH formats and E.164 international numbers.
const PHONE_RE = /^\+?[\d\s\-().]{6,20}$/;

export function validatePhone(value: unknown, message: string): ValidationResult<string> {
  const v = text(value);
  if (!v) return { ok: true, value: '' };
  return PHONE_RE.test(v) ? { ok: true, value: v } : { ok: false, message };
}

// MIN_BIRTHDAY matches the backend's validate.Birthday lower bound
// (backend/internal/validate/validate.go).
const MIN_BIRTHDAY = '1900-01-01';

export function validateBirthday(value: unknown, message: string): ValidationResult<string> {
  const v = text(value);
  if (!v) return { ok: true, value: '' };
  if (!DATE_RE.test(v)) return { ok: false, message };
  const d = new Date(v + 'T00:00:00');
  if (isNaN(d.getTime())) return { ok: false, message };
  if (v < MIN_BIRTHDAY) return { ok: false, message };
  return d > new Date() ? { ok: false, message } : { ok: true, value: v };
}
