import { z } from 'zod';
import { t } from '@/i18n';

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
const TIME_RE = /^\d{2}:\d{2}$/;

const validDate = (value: string) => {
  if (!DATE_RE.test(value)) return false;
  const [year, month, day] = value.split('-').map(Number);
  if (year === undefined || month === undefined || day === undefined) return false;
  const date = new Date(Date.UTC(year, month - 1, day));
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day;
};

// Days between two valid YYYY-MM-DD strings (b - a). Callers must validate
// both with validDate() first -- this does no format checking itself.
const daysBetween = (a: string, b: string) => {
  const toUtcDays = (s: string) => Date.parse(s + 'T00:00:00Z') / 86_400_000;
  return toUtcDays(b) - toUtcDays(a);
};

// Mirrors the backend's maxMultiDaySpanDays (events.Service, 1095 days /
// ~3 years) -- an inline check here catches the common "typo'd the year"
// mistake with a field-specific error instead of a raw, untranslated
// server 400 surfacing through the generic save-error toast.
const MAX_MULTI_DAY_SPAN_DAYS = 1095;

const minutes = (value: string) => {
  if (!TIME_RE.test(value)) return null;
  const [h, m] = value.split(':').map(Number);
  if (h === undefined || m === undefined || h < 0 || h > 23 || m < 0 || m > 59) return null;
  return h * 60 + m;
};

// Explicit `| undefined` on the optional fields (not just `field?:`) --
// this mirrors the schema's own `.optional()` fields as zod/react-hook-form
// produce them (a form value that's genuinely absent, not "omit this key"),
// so passing the full form-values object (which always sets every key, some
// to `undefined`) straight through from .superRefine() typechecks.
interface EventFormRefineInput {
  date: string;
  multiDayEndDate?: string | undefined;
  startT?: string | undefined;
  endT?: string | undefined;
  meetT?: string | undefined;
  recurring?: boolean | undefined;
  repeatWeeks?: number | undefined;
  repeatMode?: 'weeks' | 'until' | undefined;
  repeatEndDate?: string | undefined;
}

function validateDateField(data: EventFormRefineInput, ctx: z.RefinementCtx) {
  if (data.date && !validDate(data.date)) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['date'],
      message: t('validation.eventDateInvalid'),
    });
  }
}

// Multi-day span is mutually exclusive with recurring (see design.md's
// "Mutually exclusive with recurring" decision) -- server-side rejects the
// combination too, but validating it here gives immediate inline feedback.
function validateMultiDayEndDate(data: EventFormRefineInput, ctx: z.RefinementCtx) {
  if (!data.multiDayEndDate) return;
  if (!validDate(data.multiDayEndDate)) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['multiDayEndDate'],
      message: t('validation.eventMultiDayEndDateInvalid'),
    });
    return;
  }
  if (data.recurring) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['multiDayEndDate'],
      message: t('validation.eventMultiDayEndDateOnRecurring'),
    });
    return;
  }
  if (data.date && validDate(data.date)) {
    if (data.multiDayEndDate < data.date) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['multiDayEndDate'],
        message: t('validation.eventMultiDayEndDateBeforeStart'),
      });
    } else if (daysBetween(data.date, data.multiDayEndDate) > MAX_MULTI_DAY_SPAN_DAYS) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['multiDayEndDate'],
        message: t('validation.eventMultiDayEndDateSpanTooLong', { max: MAX_MULTI_DAY_SPAN_DAYS }),
      });
    }
  }
}

// Parses+validates start/meet/end times and their relative ordering. Returns
// the parsed minute values so callers don't need to reparse.
function validateTimeFields(data: EventFormRefineInput, ctx: z.RefinementCtx) {
  const startMin = data.startT ? minutes(data.startT) : null;
  const endMin = data.endT ? minutes(data.endT) : null;
  const meetMin = data.meetT ? minutes(data.meetT) : null;

  if (data.startT && startMin === null) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['startT'], message: t('validation.eventStartInvalid') });
  }
  if (data.endT && endMin === null) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['endT'], message: t('validation.eventEndInvalid') });
  }
  if (data.meetT && meetMin === null) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['meetT'], message: t('validation.eventMeetTimeInvalid') });
  }
  if (startMin !== null && endMin !== null && endMin <= startMin) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['endT'], message: t('validation.eventEndBeforeStart') });
  }
  if (meetMin !== null && startMin !== null && meetMin > startMin) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['meetT'], message: t('validation.eventMeetAfterStart') });
  }
}

// Validates the recurrence inputs, branching on repeatMode: 'until' checks
// repeatEndDate (must be a valid date on/after the event's own date);
// 'weeks' (the default, for forms/tests predating the toggle) checks
// repeatWeeks as before.
function validateRecurring(data: EventFormRefineInput, ctx: z.RefinementCtx) {
  if (!data.recurring) return;
  if (data.repeatMode === 'until') {
    if (!data.repeatEndDate || !validDate(data.repeatEndDate)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['repeatEndDate'],
        message: t('validation.eventRepeatEndDateInvalid'),
      });
    } else if (data.date && validDate(data.date) && data.repeatEndDate < data.date) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['repeatEndDate'],
        message: t('validation.eventRepeatEndDateBeforeStart'),
      });
    }
    return;
  }
  const rw = Number(data.repeatWeeks);
  if (isNaN(rw) || !Number.isInteger(rw)) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['repeatWeeks'],
      message: t('validation.eventRepeatWeeksInteger'),
    });
  } else if (rw < 2 || rw > 26) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['repeatWeeks'],
      message: t('validation.eventRepeatWeeksRange', { min: 2, max: 26 }),
    });
  }
}

export const eventFormSchema = z
  .object({
    type: z.enum(['training', 'auftritt', 'event']),
    title: z
      .string()
      .trim()
      .min(1, { message: t('validation.eventTitleMissing') })
      .max(255),
    date: z
      .string()
      .trim()
      .min(1, { message: t('validation.eventDateMissing') }),
    multiDayEndDate: z.string().trim().optional().or(z.literal('')),
    meetT: z.string().trim().optional().or(z.literal('')),
    startT: z.string().trim().optional().or(z.literal('')),
    endT: z.string().trim().optional().or(z.literal('')),
    meetTimeMandatory: z.boolean().optional(),
    responseMode: z.enum(['opt_in', 'opt_out']).optional(),
    // Not `.uuid()` -- these are opaque, server-issued ids chosen via UI
    // toggles, never typed by the user, and the MSW demo backend's ids
    // (e.g. "role_xyz", "series_tue_thu") aren't RFC4122 UUIDs, so a strict
    // uuid() check here would silently fail validation on every submit.
    nominatedRoleIds: z.array(z.string()).optional(),
    location: z.string().max(255).optional().or(z.literal('')),
    note: z.string().max(10000).optional().or(z.literal('')),
    recurring: z.boolean().optional(),
    repeatWeeks: z.coerce.number().optional(),
    repeatMode: z.enum(['weeks', 'until']).optional(),
    repeatEndDate: z.string().trim().optional().or(z.literal('')),
    cancelLeadHours: z.coerce.number().min(0).optional(),
    cancelLeadMinutes: z.coerce.number().min(0).max(59).optional(),
    excludeFromStats: z.boolean().optional(),
    seriesId: z.string().optional().nullable(),
  })
  .superRefine((data, ctx) => {
    validateDateField(data, ctx);
    validateMultiDayEndDate(data, ctx);
    validateTimeFields(data, ctx);
    validateRecurring(data, ctx);
  });

export type EventFormValues = z.infer<typeof eventFormSchema>;
