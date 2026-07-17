import { z } from 'zod';
import { t } from '@/i18n';

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

const validDate = (value: string) => {
  if (!DATE_RE.test(value)) return false;
  const [year, month, day] = value.split('-').map(Number);
  const date = new Date(Date.UTC(year, month - 1, day));
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day;
};

export const absenceFormSchema = z
  .object({
    // Not `.uuid()` -- this is an opaque, server-issued id never typed by the
    // user (only present in edit mode, round-tripped unchanged), and the
    // MSW demo backend's ids (e.g. "abs_xyz") aren't RFC4122 UUIDs, so a
    // strict uuid() check here would silently block every edit against it.
    id: z.string().optional(),
    from: z
      .string()
      .trim()
      .min(1, { message: t('validation.dateRangeStartMissing') }),
    to: z
      .string()
      .trim()
      .min(1, { message: t('validation.dateRangeEndMissing') }),
    reason: z.string().max(500).optional().or(z.literal('')),
  })
  .superRefine((data, ctx) => {
    if (data.from && !validDate(data.from)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['from'],
        message: t('validation.dateRangeStartInvalid'),
      });
    }
    if (data.to && !validDate(data.to)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['to'],
        message: t('validation.dateRangeEndInvalid'),
      });
    }
    if (data.from && data.to && validDate(data.from) && validDate(data.to) && data.to < data.from) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['to'],
        message: t('validation.dateRangeOrder'),
      });
    }
  });

export type AbsenceFormValues = z.infer<typeof absenceFormSchema>;
