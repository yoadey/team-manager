import { describe, it, expect } from 'vitest';
import { canForTeam, canSeeReason, hasPermission, isStaffForTeam } from './permissions';
import type { Permissions, TeamForUser } from '@/types';

const perms = (over: Partial<Permissions> = {}): Permissions => ({
  events: 'none',
  members: 'none',
  finances: 'none',
  news: 'none',
  polls: 'none',
  settings: 'none',
  ...over,
});

function team(myPerms: Permissions, reasonVisibilityRoles: string[] = []): TeamForUser {
  return {
    id: 't1',
    name: 'T',
    short: 'T',
    icon: '⭐',
    iconBg: '#000',
    iconFg: '#fff',
    photo: null,
    logo: null,
    description: '',
    reasonVisibilityRoles,
    myRoles: [],
    myPerms,
    membershipId: 'm1',
    memberCount: 1,
  };
}

describe('hasPermission', () => {
  it('returns false for missing perms', () => {
    expect(hasPermission(null, 'events')).toBe(false);
    expect(hasPermission(undefined, 'events', 'read')).toBe(false);
  });

  it('read is satisfied by read or write', () => {
    expect(hasPermission(perms({ events: 'read' }), 'events', 'read')).toBe(true);
    expect(hasPermission(perms({ events: 'write' }), 'events', 'read')).toBe(true);
    expect(hasPermission(perms({ events: 'none' }), 'events', 'read')).toBe(false);
  });

  it('write requires write', () => {
    expect(hasPermission(perms({ finances: 'write' }), 'finances', 'write')).toBe(true);
    expect(hasPermission(perms({ finances: 'read' }), 'finances', 'write')).toBe(false);
  });

  it('defaults to write level', () => {
    expect(hasPermission(perms({ news: 'read' }), 'news')).toBe(false);
    expect(hasPermission(perms({ news: 'write' }), 'news')).toBe(true);
  });
});

describe('canForTeam', () => {
  it('returns false without a team', () => {
    expect(canForTeam(null, 'events', 'read')).toBe(false);
  });

  it('reads permissions from the active team', () => {
    const t = team(perms({ finances: 'read' }));
    expect(canForTeam(t, 'finances', 'read')).toBe(true);
    expect(canForTeam(t, 'finances', 'write')).toBe(false);
  });
});

describe('isStaffForTeam', () => {
  it('is true when events OR members are writable', () => {
    expect(isStaffForTeam(team(perms({ events: 'write' })))).toBe(true);
    expect(isStaffForTeam(team(perms({ members: 'write' })))).toBe(true);
  });

  it('is false for read-only / no team', () => {
    expect(isStaffForTeam(team(perms({ events: 'read', members: 'read' })))).toBe(false);
    expect(isStaffForTeam(null)).toBe(false);
  });
});

describe('canSeeReason', () => {
  const base = { reason: 'Krank', status: 'no' as const, reasonVisibilityRoles: ['trainer'], myRoleIds: ['player'] };

  it('always shows own reason', () => {
    expect(canSeeReason({ ...base, isSelf: true })).toBe(true);
  });

  it('hides when there is no reason', () => {
    expect(canSeeReason({ ...base, isSelf: false, reason: '' })).toBe(false);
  });

  it('shows non-declined reasons to everyone', () => {
    expect(canSeeReason({ ...base, isSelf: false, status: 'maybe' })).toBe(true);
  });

  it('declined reasons require a whitelisted role', () => {
    expect(canSeeReason({ ...base, isSelf: false })).toBe(false);
    expect(canSeeReason({ ...base, isSelf: false, myRoleIds: ['trainer'] })).toBe(true);
  });
});
