import { z } from 'zod';
import { t } from '@/i18n';

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
const TIME_RE = /^\d{2}:\d{2}$/;

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

export const eventFormSchema = z
  .object({
    type: z.enum(['training', 'auftritt', 'event']),
    title: z.string().trim().min(1, { message: t('validation.eventTitleMissing') }).max(255),
    date: z.string().trim().min(1, { message: t('validation.eventDateMissing') }),
    meetT: z.string().trim().optional().or(z.literal('')),
    startT: z.string().trim().optional().or(z.literal('')),
    endT: z.string().trim().optional().or(z.literal('')),
    meetTimeMandatory: z.boolean().optional(),
    responseMode: z.enum(['opt_in', 'opt_out']).optional(),
    nominatedRoleIds: z.array(z.string().uuid()).optional(),
    location: z.string().max(255).optional().or(z.literal('')),
    note: z.string().max(10000).optional().or(z.literal('')),
    recurring: z.boolean().optional(),
    repeatWeeks: z.coerce.number().optional(),
    seriesId: z.string().uuid().optional().nullable(),
  })
  .superRefine((data, ctx) => {
    // Validate date
    if (data.date && !validDate(data.date)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['date'],
        message: t('validation.eventDateInvalid'),
      });
    }

    const startMin = data.startT ? minutes(data.startT) : null;
    const endMin = data.endT ? minutes(data.endT) : null;
    const meetMin = data.meetT ? minutes(data.meetT) : null;

    if (data.startT && startMin === null) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['startT'],
        message: t('validation.eventStartInvalid'),
      });
    }

    if (data.endT && endMin === null) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['endT'],
        message: t('validation.eventEndInvalid'),
      });
    }

    if (data.meetT && meetMin === null) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['meetT'],
        message: t('validation.eventMeetTimeInvalid'),
      });
    }

    if (startMin !== null && endMin !== null && endMin <= startMin) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['endT'],
        message: t('validation.eventEndBeforeStart'),
      });
    }

    if (meetMin !== null && startMin !== null && meetMin > startMin) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['meetT'],
        message: t('validation.eventMeetAfterStart'),
      });
    }

    if (data.recurring) {
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
  });

export type EventFormValues = z.infer<typeof eventFormSchema>;
