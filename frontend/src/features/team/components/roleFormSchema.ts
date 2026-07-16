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
  // Fixed module set, matching `Permissions` (@/types) -- not an open
  // z.record(), since the backend's RoleDto always carries exactly these six
  // module keys.
  perms: z.object({
    events: PermLevel,
    members: PermLevel,
    finances: PermLevel,
    news: PermLevel,
    polls: PermLevel,
    settings: PermLevel,
  }),
});

export type RoleFormValues = z.infer<typeof roleFormSchema>;
