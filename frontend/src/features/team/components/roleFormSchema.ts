import { z } from 'zod';
import { t } from '@/i18n';

const PermLevel = z.enum(['none', 'read', 'write']);

export const roleFormSchema = z.object({
  id: z.string().uuid().optional(),
  name: z
    .string()
    .trim()
    .min(1, { message: t('team.roleNameRequired') })
    .max(60),
  perms: z.record(z.string(), PermLevel),
});

export type RoleFormValues = z.infer<typeof roleFormSchema>;
