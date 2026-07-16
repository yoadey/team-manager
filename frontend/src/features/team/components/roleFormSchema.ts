import { z } from 'zod';
import { t } from '@/i18n';

const PermLevel = z.enum(['none', 'read', 'write']);

export const roleFormSchema = z.object({
  // Not `.uuid()` -- opaque, server-issued id round-tripped unchanged in
  // edit mode; the MSW demo backend's ids (e.g. "role_xyz") aren't RFC4122
  // UUIDs, so a strict uuid() check here would silently block every edit.
  id: z.string().optional(),
  name: z
    .string()
    .trim()
    .min(1, { message: t('team.roleNameRequired') })
    .max(60),
  perms: z.record(z.string(), PermLevel),
});

export type RoleFormValues = z.infer<typeof roleFormSchema>;
