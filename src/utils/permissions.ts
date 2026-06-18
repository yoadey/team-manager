// =============================================================================
// Pure RBAC helpers. Extracted from AppContext / feature hooks so the
// permission logic — which gates every protected route and action — is unit
// testable in isolation instead of only exercised through React.
// =============================================================================

import type { AttendanceStatus, ModuleKey, Permissions, PermLevel, TeamForUser } from '@/types';

/** True if `perms` grants at least `level` for `module`. */
export function hasPermission(
  perms: Partial<Permissions> | null | undefined,
  module: ModuleKey,
  level: PermLevel = 'write',
): boolean {
  if (!perms) return false;
  const p = perms[module];
  if (level === 'read') return p === 'read' || p === 'write';
  return p === 'write';
}

/** Permission check for the user's active team. */
export function canForTeam(team: TeamForUser | null, module: ModuleKey, level: PermLevel = 'write'): boolean {
  if (!team || !team.myPerms) return false;
  return hasPermission(team.myPerms, module, level);
}

/** Staff = may write events or members (used to gate management UI). */
export function isStaffForTeam(team: TeamForUser | null): boolean {
  return canForTeam(team, 'events', 'write') || canForTeam(team, 'members', 'write');
}

/**
 * Whether the current user may see another member's absence reason for an
 * event. Own reasons are always visible; declined ("no") reasons are only
 * visible to roles whitelisted in the team's `reasonVisibilityRoles`.
 */
export function canSeeReason(opts: {
  isSelf: boolean;
  reason?: string;
  status: AttendanceStatus;
  reasonVisibilityRoles: string[];
  myRoleIds: string[];
}): boolean {
  if (opts.isSelf) return true;
  if (!opts.reason) return false;
  if (opts.status !== 'no') return true;
  return opts.myRoleIds.some((id) => opts.reasonVisibilityRoles.includes(id));
}
