import { z } from 'zod';
import { t } from '@/i18n';

export const penaltyAssignFormSchema = z.object({
  // Not `.uuid()` -- opaque, server-issued ids chosen via a select, never
  // typed by the user; the MSW demo backend's ids (e.g. "u1", "pen_xyz")
  // aren't RFC4122 UUIDs, so a strict uuid() check here would silently
  // block every submission. `.min(1)` still catches "nothing selected".
  userId: z.string().min(1, { message: t('finances.assignPersonError') }),
  penaltyId: z.string().min(1, { message: t('finances.assignPenaltyError') }),
});

export type PenaltyAssignFormValues = z.infer<typeof penaltyAssignFormSchema>;
