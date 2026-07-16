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
  // Not `.uuid()` -- opaque, server-issued ids toggled via UI chips, never
  // typed by the user; the MSW demo backend's role ids (e.g. "role_xyz")
  // aren't RFC4122 UUIDs, so a strict uuid() check here would silently
  // block every settings save.
  reasonRoles: z.array(z.string()).optional(),
});

export type TeamSettingsFormValues = z.infer<typeof teamSettingsSchema>;
