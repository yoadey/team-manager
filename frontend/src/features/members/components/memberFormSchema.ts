import { z } from 'zod';
import { t } from '@/i18n';

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const PHONE_RE = /^\+?[\d\s\-().]{6,20}$/;
const MIN_BIRTHDAY = '1900-01-01';

const validDate = (value: string) => {
  if (!DATE_RE.test(value)) return false;
  const [year, month, day] = value.split('-').map(Number);
  const date = new Date(Date.UTC(year, month - 1, day));
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day;
};

export const memberFormSchema = z
  .object({
    name: z.string().trim().min(1, { message: t('members.fieldNameError') }).max(255),
    email: z.string().trim().optional().or(z.literal('')),
    phone: z.string().trim().optional().or(z.literal('')),
    birthday: z.string().trim().optional().or(z.literal('')),
    address: z.string().trim().max(500).optional().or(z.literal('')),
    photo: z.string().optional().nullable(),
    roleIds: z.array(z.string().uuid()).optional(),
  })
  .superRefine((data, ctx) => {
    if (data.email && !EMAIL_RE.test(data.email)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['email'],
        message: t('members.fieldEmailError'),
      });
    }

    if (data.phone && !PHONE_RE.test(data.phone)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['phone'],
        message: t('members.fieldPhoneError'),
      });
    }

    if (data.birthday) {
      if (!DATE_RE.test(data.birthday) || !validDate(data.birthday) || data.birthday < MIN_BIRTHDAY || new Date(data.birthday + 'T00:00:00') > new Date()) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['birthday'],
          message: t('members.fieldBirthdayError'),
        });
      }
    }
  });

export type MemberFormValues = z.infer<typeof memberFormSchema>;
