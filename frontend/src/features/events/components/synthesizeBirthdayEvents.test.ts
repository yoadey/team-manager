import { describe, it, expect } from 'vitest';
import { synthesizeBirthdayEvents, groupBirthdaysByDate, type BirthdayEntry } from './synthesizeBirthdayEvents';
import type { Member } from '@/features/members';

function makeMember(overrides: Partial<Pick<Member, 'membershipId' | 'name' | 'birthday'>> = {}) {
  return {
    membershipId: 'ms1',
    name: 'Alice Example',
    birthday: '1990-05-15',
    ...overrides,
  } as Pick<Member, 'membershipId' | 'name' | 'birthday'>;
}

describe('synthesizeBirthdayEvents', () => {
  it('synthesizes an occurrence on the correct day for a range covering it', () => {
    const members = [makeMember({ birthday: '1990-05-15' })];
    const entries = synthesizeBirthdayEvents(members, new Date(2026, 4, 1), new Date(2026, 4, 31));
    expect(entries).toEqual([{ membershipId: 'ms1', name: 'Alice Example', date: '2026-05-15' }]);
  });

  it('recurs every year: the same birthday synthesizes for two different years', () => {
    const members = [makeMember({ birthday: '1990-05-15' })];
    const entries2026 = synthesizeBirthdayEvents(members, new Date(2026, 4, 1), new Date(2026, 4, 31));
    const entries2030 = synthesizeBirthdayEvents(members, new Date(2030, 4, 1), new Date(2030, 4, 31));
    expect(entries2026).toEqual([{ membershipId: 'ms1', name: 'Alice Example', date: '2026-05-15' }]);
    expect(entries2030).toEqual([{ membershipId: 'ms1', name: 'Alice Example', date: '2030-05-15' }]);
  });

  it('produces one occurrence per calendar year spanned by the range', () => {
    const members = [makeMember({ birthday: '1990-01-15' })];
    // Range spans two Decembers/Januaries -- Jan 15 occurs once in each year.
    const entries = synthesizeBirthdayEvents(members, new Date(2026, 0, 1), new Date(2027, 0, 31));
    expect(entries).toEqual([
      { membershipId: 'ms1', name: 'Alice Example', date: '2026-01-15' },
      { membershipId: 'ms1', name: 'Alice Example', date: '2027-01-15' },
    ]);
  });

  it('returns nothing for a range that does not cover the birthday', () => {
    const members = [makeMember({ birthday: '1990-05-15' })];
    const entries = synthesizeBirthdayEvents(members, new Date(2026, 5, 1), new Date(2026, 5, 30));
    expect(entries).toEqual([]);
  });

  it('skips members without a birthday', () => {
    const members = [makeMember({ birthday: '' })];
    const entries = synthesizeBirthdayEvents(members, new Date(2026, 4, 1), new Date(2026, 4, 31));
    expect(entries).toEqual([]);
  });

  it('returns nothing for an undefined or empty member list', () => {
    expect(synthesizeBirthdayEvents(undefined, new Date(2026, 4, 1), new Date(2026, 4, 31))).toEqual([]);
    expect(synthesizeBirthdayEvents([], new Date(2026, 4, 1), new Date(2026, 4, 31))).toEqual([]);
  });

  it('synthesizes for multiple members independently', () => {
    const members = [
      makeMember({ membershipId: 'ms1', name: 'Alice', birthday: '1990-05-15' }),
      makeMember({ membershipId: 'ms2', name: 'Bob', birthday: '1985-05-20' }),
    ];
    const entries = synthesizeBirthdayEvents(members, new Date(2026, 4, 1), new Date(2026, 4, 31));
    expect(entries).toEqual([
      { membershipId: 'ms1', name: 'Alice', date: '2026-05-15' },
      { membershipId: 'ms2', name: 'Bob', date: '2026-05-20' },
    ]);
  });
});

describe('groupBirthdaysByDate', () => {
  it('groups entries by their occurrence date', () => {
    const entries: BirthdayEntry[] = [
      { membershipId: 'ms1', name: 'Alice', date: '2026-05-15' },
      { membershipId: 'ms2', name: 'Bob', date: '2026-05-15' },
      { membershipId: 'ms3', name: 'Carla', date: '2026-05-20' },
    ];
    const grouped = groupBirthdaysByDate(entries);
    expect(grouped['2026-05-15']).toHaveLength(2);
    expect(grouped['2026-05-20']).toHaveLength(1);
    expect(grouped['2026-05-21']).toBeUndefined();
  });

  it('returns an empty object for no entries', () => {
    expect(groupBirthdaysByDate([])).toEqual({});
  });
});
