import { z } from 'zod';
import { t } from '@/i18n';

export const penaltyAssignFormSchema = z.object({
  userId: z.string().uuid({ message: t('finances.assignPersonError') }),
  penaltyId: z.string().uuid({ message: t('finances.assignPenaltyError') }),
});

export type PenaltyAssignFormValues = z.infer<typeof penaltyAssignFormSchema>;
