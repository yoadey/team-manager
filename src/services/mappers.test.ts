import { describe, expect, it } from 'vitest';
import { mapAttendanceDtoToRow, mapEventDtoToTeamEvent, mapMemberDtoToMember } from './mappers';
import type {
  AttendanceDto,
  EventDto,
  EventSummary,
  MemberDto,
  Permissions,
  Role,
} from './types';

const writeAll: Permissions = {
  events: 'write',
  members: 'write',
  finances: 'write',
  news: 'write',
  polls: 'write',
  settings: 'write',
};

const adminRole: Role = {
  id: 'role_admin',
  teamId: 't_a',
  name: 'Admin',
  system: true,
  color: '#1565C0',
  permissions: writeAll,
};

describe('mapEventDtoToTeamEvent', () => {
  it('merges the DTO with summary and personal status into a ViewModel', () => {
    const dto: EventDto = {
      id: 'ev1',
      teamId: 't_a',
      type: 'training',
      title: 'Training',
      date: '2024-06-15',
      location: 'Halle',
      note: '',
      meetTime: '19:15',
      startTime: '19:30',
      endTime: '21:30',
      meetTimeMandatory: true,
      responseMode: 'opt_out',
      recurring: false,
      seriesId: null,
      status: 'active',
    };
    const summary: EventSummary = { yes: 3, no: 1, maybe: 0, pending: 2, notNominated: 0, nominated: 6, total: 6 };

    const result = mapEventDtoToTeamEvent(dto, summary, { myStatus: 'yes', myAuto: true, myReason: '' });

    expect(result.id).toBe('ev1');
    expect(result.summary).toBe(summary);
    expect(result.myStatus).toBe('yes');
    expect(result.myAuto).toBe(true);
  });
});

describe('mapMemberDtoToMember', () => {
  it('attaches the derived primary role and permissions', () => {
    const dto: MemberDto = {
      membershipId: 'mem1',
      userId: 'u1',
      name: 'Lena',
      email: 'lena@example.de',
      phone: '',
      birthday: '',
      address: '',
      avatarColor: '#1565C0',
      photo: null,
      group: 'A-Formation',
      roles: [adminRole],
      joinedAt: '2024-01-01T00:00:00.000Z',
    };

    const result = mapMemberDtoToMember(dto, adminRole, writeAll);

    expect(result.userId).toBe('u1');
    expect(result.primaryRole).toBe(adminRole);
    expect(result.perms).toBe(writeAll);
  });

  it('supports members without a primary role', () => {
    const dto: MemberDto = {
      membershipId: 'mem2',
      userId: 'u2',
      name: 'Gast',
      email: '',
      phone: '',
      birthday: '',
      address: '',
      avatarColor: '#888',
      photo: null,
      group: '',
      roles: [],
      joinedAt: '2024-01-01T00:00:00.000Z',
    };

    expect(mapMemberDtoToMember(dto, null, writeAll).primaryRole).toBeNull();
  });
});

describe('mapAttendanceDtoToRow', () => {
  it('combines attendance state with member display data', () => {
    const dto: AttendanceDto = {
      id: 'att1',
      eventId: 'ev1',
      userId: 'u1',
      status: 'no',
      reason: 'Krank',
      reasonId: 'cr1',
      reasonVisibility: 'trainers',
    };

    const result = mapAttendanceDtoToRow(dto, {
      userId: 'u1',
      name: 'Lena',
      avatarColor: '#1565C0',
      photo: null,
      group: 'A-Formation',
      primaryRole: adminRole,
      auto: false,
      absent: false,
    });

    expect(result).toMatchObject({
      userId: 'u1',
      name: 'Lena',
      status: 'no',
      reason: 'Krank',
      reasonId: 'cr1',
      reasonVisibility: 'trainers',
    });
  });
});
