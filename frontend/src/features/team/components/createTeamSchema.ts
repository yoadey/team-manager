import { z } from 'zod';
import { t } from '@/i18n';

export const createTeamSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, { message: t('team.nameRequired') })
    .max(60),
  icon: z.string().trim().optional(),
  photo: z.string().optional().nullable(),
});

export type CreateTeamFormValues = z.infer<typeof createTeamSchema>;
