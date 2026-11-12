import { z } from 'zod';
import { t } from '@/i18n';
import type { StatsGroupThreshold } from '@/types';

// Upper bounds per mode -- mirror stats.validateThreshold on the backend
// (and role_stats_thresholds' CHECK constraints).
const THRESHOLD_MAX = { percent: 100, count: 10000 } as const;

// One role's attendance threshold as edited in the form. `value` stays a
// string while editing (so the field can be emptied and retyped) and is only
// parsed on submit -- see changedThresholds().
const thresholdRowSchema = z
  .object({
    // Not `.uuid()`, same reason as reasonRoles below.
    roleId: z.string(),
    mode: z.enum(['percent', 'count']),
    value: z.string().trim(),
  })
  .superRefine((row, ctx) => {
    // Digits only: rejects '', '-1', '1.5' and '1e3' alike, none of which
    // the integer-typed API field accepts.
    if (!/^\d+$/.test(row.value) || Number(row.value) > THRESHOLD_MAX[row.mode]) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['value'],
        message: t(row.mode === 'percent' ? 'team.thresholdPercentError' : 'team.thresholdCountError'),
      });
    }
  });

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
  // Absent until the thresholds have loaded (or when they can't be -- e.g.
  // no settings:read); an absent list is simply never saved.
  thresholds: z.array(thresholdRowSchema).optional(),
});

export type TeamSettingsFormValues = z.infer<typeof teamSettingsSchema>;
export type ThresholdFormRow = z.infer<typeof thresholdRowSchema>;

/** Server thresholds -> form rows. */
export function toThresholdFormRows(items: StatsGroupThreshold[]): ThresholdFormRow[] {
  return items.map((i) => ({ roleId: i.roleId, mode: i.mode, value: String(i.value) }));
}

/**
 * The (validated) form rows that differ from what the server currently has
 * -- only these are sent on save, so an untouched section costs no request
 * and a role the form doesn't know about is never overwritten.
 */
export function changedThresholds(
  rows: ThresholdFormRow[] | undefined,
  loaded: StatsGroupThreshold[] | undefined,
): StatsGroupThreshold[] {
  if (!rows || !loaded) return [];
  const byRole = new Map(loaded.map((l) => [l.roleId, l]));
  return rows
    .map((r) => ({ roleId: r.roleId, mode: r.mode, value: Number(r.value) }))
    .filter((r) => {
      const cur = byRole.get(r.roleId);
      return !!cur && (cur.mode !== r.mode || cur.value !== r.value);
    });
}
