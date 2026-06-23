import type { Permissions } from '@/types';

// Editing buffer shapes for the team/role sheets.

/** Role create sheet (name + per-module permission levels). */
export interface RoleFormValues extends Record<string, unknown> {
  name: string;
  perms: Permissions;
}

/** Team-settings sheet. */
export interface TeamSettingsFormValues extends Record<string, unknown> {
  name: string;
  description: string;
  icon: string;
  logo: string | null;
  photo: string | null;
  reasonRoles: string[];
}

/** New-team creation sheet. */
export interface CreateTeamFormValues extends Record<string, unknown> {
  name: string;
  icon: string;
  photo: string | null;
}
