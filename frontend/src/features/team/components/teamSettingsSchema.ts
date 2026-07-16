import { z } from 'zod';
import { t } from '@/i18n';

export const teamSettingsSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, { message: t('team.nameRequired') })
    .max(60),
  description: z.string().trim().max(10000).optional().or(z.literal('')),
  icon: z.string().trim().optional(),
  logo: z.string().optional().nullable(),
  reasonRoles: z.array(z.string().uuid()).optional(),
});

export type TeamSettingsFormValues = z.infer<typeof teamSettingsSchema>;
