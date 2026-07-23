import { z } from 'zod';
import { t } from '@/i18n';

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const PHONE_RE = /^\+?[\d\s\-().]{6,20}$/;
const MIN_BIRTHDAY = '1900-01-01';

const validDate = (value: string) => {
  if (!DATE_RE.test(value)) return false;
  const [year, month, day] = value.split('-').map(Number);
  if (year === undefined || month === undefined || day === undefined) return false;
  const date = new Date(Date.UTC(year, month - 1, day));
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day;
};

export const memberFormSchema = z
  .object({
    // Carried through unchanged from the member being edited (never
    // rendered as an input) so saveMember can identify/patch the right
    // record -- not user input, so no validation beyond being a string.
    membershipId: z.string().optional(),
    group: z.string().optional(),
    name: z
      .string()
      .trim()
      .min(1, { message: t('members.fieldNameError') })
      .max(255),
    email: z.string().trim().optional().or(z.literal('')),
    phone: z.string().trim().optional().or(z.literal('')),
    birthday: z.string().trim().optional().or(z.literal('')),
    address: z.string().trim().max(500).optional().or(z.literal('')),
    photo: z.string().optional().nullable(),
    // Not `.uuid()` -- opaque, server-issued ids toggled via UI chips, never
    // typed by the user; the MSW demo backend's role ids (e.g. "role_xyz")
    // aren't RFC4122 UUIDs, so a strict uuid() check here would silently
    // block every profile save.
    roleIds: z.array(z.string()).optional(),
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
      if (
        !DATE_RE.test(data.birthday) ||
        !validDate(data.birthday) ||
        data.birthday < MIN_BIRTHDAY ||
        new Date(data.birthday + 'T00:00:00') > new Date()
      ) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['birthday'],
          message: t('members.fieldBirthdayError'),
        });
      }
    }
  });

export type MemberFormValues = z.infer<typeof memberFormSchema>;
