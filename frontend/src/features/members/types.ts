import type { RoleDto, Permissions, Role } from '@/types';

/** Raw member payload composed from user + membership + role DTOs by members.* endpoints. */
export interface MemberDto {
  membershipId: string;
  userId: string;
  name: string;
  email: string;
  phone: string;
  birthday: string;
  address: string;
  avatarColor: string;
  photo: string | null;
  group: string;
  roles: RoleDto[];
  joinedAt: string;
}

/** UI ViewModel consumed by member screens. */
export interface Member extends MemberDto {
  primaryRole: Role | null;
  perms: Permissions;
}

/** Editing buffer shape for the member edit sheet. */
export interface MemberFormValues extends Record<string, unknown> {
  membershipId: string;
  name: string;
  email: string;
  phone: string;
  birthday: string;
  address: string;
  roleIds: string[];
  group: string;
  photo: string | null;
}
