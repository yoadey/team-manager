import { makeApi, Zodios, type ZodiosOptions } from '@zodios/core';
import { z } from 'zod';

const Provider = z
  .object({
    id: z.string(),
    name: z.string(),
    sub: z.string(),
    glyph: z.string(),
    bg: z.string(),
    fg: z.string(),
    border: z.string().optional(),
  })
  .passthrough();
const LoginRequest = z.object({ email: z.string().email(), password: z.string().min(8).max(128) }).passthrough();
const User = z
  .object({
    id: z.string().uuid(),
    name: z.string(),
    email: z.string().email(),
    phone: z.string().optional(),
    avatarColor: z.string(),
    birthday: z.string().optional(),
    address: z.string().optional(),
    hasPhoto: z.boolean().optional(),
  })
  .passthrough();
const LoginResponse = z.object({ token: z.string(), user: User }).passthrough();
const DeleteAccountRequest = z.object({ confirmEmail: z.string().email() }).passthrough();
const Team = z
  .object({
    id: z.string().uuid(),
    name: z.string(),
    short: z.string().optional(),
    icon: z.string().optional(),
    iconBg: z.string().optional(),
    iconFg: z.string().optional(),
    description: z.string().optional(),
    hasPhoto: z.boolean().optional(),
    hasLogo: z.boolean().optional(),
    reasonVisibilityRoleIds: z.array(z.string().uuid()).optional(),
  })
  .passthrough();
const PermLevel = z.enum(['none', 'read', 'write']);
const Permissions = z
  .object({
    events: PermLevel,
    members: PermLevel,
    finances: PermLevel,
    news: PermLevel,
    polls: PermLevel,
    settings: PermLevel,
  })
  .passthrough();
const Role = z
  .object({
    id: z.string().uuid(),
    teamId: z.string().uuid(),
    name: z.string(),
    system: z.boolean(),
    color: z.string().optional(),
    permissions: Permissions,
  })
  .passthrough();
const TeamForUser = Team.and(
  z
    .object({
      myRoles: z.array(Role),
      myPerms: Permissions,
      membershipId: z.string().uuid(),
      memberCount: z.number().int(),
    })
    .passthrough(),
);
const CreateTeamRequest = z
  .object({
    name: z.string().min(1).max(255),
    icon: z.string().optional(),
    iconBg: z.string().optional(),
    iconFg: z.string().optional(),
  })
  .passthrough();
const UpdateTeamRequest = z
  .object({
    name: z.string().min(1).max(255),
    short: z.string().max(50),
    icon: z.string(),
    iconBg: z.string(),
    iconFg: z.string(),
    description: z.string().max(10000),
    reasonVisibilityRoleIds: z.array(z.string().uuid()),
  })
  .partial()
  .passthrough();
const Invite = z
  .object({
    id: z.string().uuid(),
    teamId: z.string().uuid(),
    code: z.string(),
    link: z.string(),
    createdAt: z.string().datetime({ offset: true }),
    expiresAt: z.string().datetime({ offset: true }),
  })
  .passthrough();
const AcceptInviteResponse = TeamForUser.and(z.object({ alreadyMember: z.boolean() }).passthrough());
const Member = z
  .object({
    membershipId: z.string().uuid(),
    userId: z.string().uuid(),
    name: z.string(),
    email: z.string().email(),
    phone: z.string().optional(),
    birthday: z.string().optional(),
    address: z.string().optional(),
    avatarColor: z.string(),
    hasPhoto: z.boolean().optional(),
    group: z.string().optional(),
    roles: z.array(Role),
    primaryRole: Role.optional(),
    perms: Permissions.optional(),
    joinedAt: z.string().datetime({ offset: true }),
  })
  .passthrough();
const UpdateMemberRequest = z
  .object({
    name: z.string(),
    email: z.string().email(),
    phone: z.string(),
    birthday: z.string(),
    address: z.string(),
    roleIds: z.array(z.string().uuid()),
    group: z.string(),
  })
  .partial()
  .passthrough();
const SetRolesRequest = z.object({ roleIds: z.array(z.string().uuid()) }).passthrough();
const CreateRoleRequest = z
  .object({
    name: z.string(),
    color: z.string().max(32).optional(),
    permissions: Permissions,
  })
  .passthrough();
const UpdateRoleRequest = z
  .object({
    name: z.string(),
    color: z.string().max(32),
    permissions: Permissions,
  })
  .partial()
  .passthrough();
const EventType = z.enum(['training', 'auftritt', 'event']);
const ResponseMode = z.enum(['opt_in', 'opt_out']);
const EventStatus = z.enum(['active', 'cancelled']);
const EventSummary = z
  .object({
    yes: z.number().int(),
    no: z.number().int(),
    maybe: z.number().int(),
    pending: z.number().int(),
    notNominated: z.number().int(),
    nominated: z.number().int(),
    total: z.number().int(),
  })
  .passthrough();
const AttendanceStatus = z.enum(['yes', 'no', 'maybe', 'pending', 'not_nominated']);
const TeamEvent = z
  .object({
    id: z.string().uuid(),
    teamId: z.string().uuid(),
    seriesId: z.string().uuid().optional(),
    type: EventType,
    title: z.string(),
    date: z.string(),
    location: z.string().optional(),
    note: z.string().optional(),
    result: z.string().optional(),
    meetTime: z.string().optional(),
    startTime: z.string().optional(),
    endTime: z.string().optional(),
    meetTimeMandatory: z.boolean().optional(),
    responseMode: ResponseMode.optional(),
    nominatedRoleIds: z.array(z.string().uuid()).optional(),
    recurring: z.boolean(),
    status: EventStatus,
    summary: EventSummary,
    myStatus: AttendanceStatus.optional(),
    myAuto: z.boolean().optional(),
    myReason: z.string().optional(),
  })
  .passthrough();
const CreateEventRequest = z
  .object({
    type: EventType,
    title: z.string().min(1).max(255),
    date: z.string(),
    location: z.string().max(255).optional(),
    note: z.string().max(10000).optional(),
    meetTime: z.string().optional(),
    startTime: z.string().optional(),
    endTime: z.string().optional(),
    meetTimeMandatory: z.boolean().optional(),
    responseMode: ResponseMode.optional(),
    nominatedRoleIds: z.array(z.string().uuid()).optional(),
    recurring: z.boolean().optional(),
    repeatWeeks: z.number().int().gte(1).lte(104).optional(),
  })
  .passthrough();
const UpdateEventRequest = z
  .object({
    type: EventType,
    title: z.string().min(1).max(255),
    date: z.string(),
    location: z.string().max(255),
    note: z.string().max(10000),
    meetTime: z.string(),
    startTime: z.string(),
    endTime: z.string(),
    meetTimeMandatory: z.boolean(),
    responseMode: ResponseMode,
    nominatedRoleIds: z.array(z.string().uuid()),
  })
  .partial()
  .passthrough();
const SetEventStatusRequest = z.object({ status: EventStatus }).passthrough();
const EventComment = z
  .object({
    id: z.string().uuid(),
    eventId: z.string().uuid(),
    userId: z.string().uuid(),
    text: z.string(),
    createdAt: z.string().datetime({ offset: true }),
    authorName: z.string().optional(),
    authorColor: z.string().optional(),
    hasAuthorPhoto: z.boolean().optional(),
  })
  .passthrough();
const AddCommentRequest = z.object({ text: z.string().min(1) }).passthrough();
const AttendanceRow = z
  .object({
    userId: z.string().uuid(),
    name: z.string(),
    avatarColor: z.string(),
    hasPhoto: z.boolean().optional(),
    group: z.string().optional(),
    primaryRole: Role.optional(),
    status: AttendanceStatus,
    reason: z.string().optional(),
    reasonId: z.string().optional(),
    reasonVisibility: z.enum(['trainers', 'team']).optional(),
    auto: z.boolean().optional(),
    absent: z.boolean().optional(),
  })
  .passthrough();
const SetAttendanceRequest = z
  .object({
    userId: z.string().uuid(),
    status: AttendanceStatus,
    reason: z.string().max(500).optional(),
    reasonId: z.string().optional(),
    reasonVisibility: z.enum(['trainers', 'team']).optional(),
  })
  .passthrough();
const AttendanceRecord = z
  .object({
    id: z.string().uuid(),
    eventId: z.string().uuid(),
    userId: z.string().uuid(),
    status: AttendanceStatus,
    reason: z.string().optional(),
    reasonId: z.string().optional(),
    reasonVisibility: z.enum(['trainers', 'team']).optional(),
    at: z.string().datetime({ offset: true }).optional(),
  })
  .passthrough();
const SetNominationRequest = z.object({ userId: z.string().uuid(), nominated: z.boolean() }).passthrough();
const Absence = z
  .object({
    id: z.string().uuid(),
    userId: z.string().uuid(),
    from: z.string(),
    to: z.string(),
    reason: z.string().optional(),
    createdAt: z.string().datetime({ offset: true }),
    memberName: z.string().optional(),
    memberAvatarColor: z.string().optional(),
    hasPhoto: z.boolean().optional(),
    roleColor: z.string().optional(),
    roleName: z.string().optional(),
  })
  .passthrough();
const CreateAbsenceRequest = z
  .object({
    userId: z.string().uuid(),
    from: z.string(),
    to: z.string(),
    reason: z.string().optional(),
  })
  .passthrough();
const UpdateAbsenceRequest = z.object({ from: z.string(), to: z.string(), reason: z.string() }).partial().passthrough();
const NewsItem = z
  .object({
    id: z.string().uuid(),
    teamId: z.string().uuid(),
    authorId: z.string().uuid(),
    title: z.string(),
    body: z.string(),
    pinned: z.boolean(),
    createdAt: z.string().datetime({ offset: true }),
    authorName: z.string().optional(),
    authorColor: z.string().optional(),
    hasAuthorPhoto: z.boolean().optional(),
  })
  .passthrough();
const CreateNewsRequest = z
  .object({
    title: z.string().min(1).max(255),
    body: z.string().min(1).max(10000),
    pinned: z.boolean().optional().default(false),
  })
  .passthrough();
const UpdateNewsRequest = z
  .object({
    title: z.string().min(1).max(255),
    body: z.string().min(1).max(10000),
    pinned: z.boolean(),
  })
  .partial()
  .passthrough();
const PollOption = z
  .object({
    id: z.string().uuid(),
    text: z.string(),
    count: z.number().int(),
    pct: z.number(),
    voters: z
      .array(
        z
          .object({
            name: z.string(),
            color: z.string(),
            hasPhoto: z.boolean(),
          })
          .partial()
          .passthrough(),
      )
      .optional(),
  })
  .passthrough();
const Poll = z
  .object({
    id: z.string().uuid(),
    question: z.string(),
    multiple: z.boolean(),
    anonymous: z.boolean(),
    createdAt: z.string().datetime({ offset: true }),
    totalVotes: z.number().int(),
    myVote: z.array(z.string().uuid()).optional(),
    options: z.array(PollOption),
  })
  .passthrough();
const CreatePollRequest = z
  .object({
    question: z.string(),
    options: z.array(z.string()).min(2).max(4),
    multiple: z.boolean().optional().default(false),
    anonymous: z.boolean().optional().default(false),
  })
  .passthrough();
const VotePollRequest = z.object({ optionIds: z.array(z.string().uuid()).max(4) }).passthrough();
const NotificationType = z.enum([
  'attendance',
  'event_created',
  'event_updated',
  'event_cancelled',
  'event_reactivated',
  'event_deleted',
  'news',
  'poll',
  'absence',
]);
const AppNotification = z
  .object({
    id: z.string().uuid(),
    teamId: z.string().uuid(),
    type: NotificationType,
    actorId: z.string().uuid().optional(),
    status: AttendanceStatus.optional(),
    title: z.string().optional(),
    eventId: z.string().uuid().optional(),
    eventTitle: z.string().optional(),
    eventDate: z.string().optional(),
    note: z.string().optional(),
    createdAt: z.string().datetime({ offset: true }),
    actorName: z.string().optional(),
    actorColor: z.string().optional(),
    hasActorPhoto: z.boolean().optional(),
    unread: z.boolean().optional(),
  })
  .passthrough();
const NotificationsResult = z.object({ items: z.array(AppNotification), unreadCount: z.number().int() }).passthrough();
const TransactionType = z.enum(['income', 'expense']);
const Transaction = z
  .object({
    id: z.string().uuid(),
    teamId: z.string().uuid(),
    type: TransactionType,
    title: z.string(),
    amount: z.number().int(),
    date: z.string(),
    category: z.string().optional(),
  })
  .passthrough();
const Penalty = z
  .object({
    id: z.string().uuid(),
    teamId: z.string().uuid(),
    label: z.string(),
    amount: z.number().int(),
  })
  .passthrough();
const PenaltyAssignment = z
  .object({
    id: z.string().uuid(),
    teamId: z.string().uuid(),
    userId: z.string().uuid(),
    penaltyId: z.string().uuid(),
    paid: z.boolean(),
    date: z.string(),
    memberName: z.string().optional(),
    memberAvatarColor: z.string().optional(),
    hasPhoto: z.boolean().optional(),
    label: z.string().optional(),
    amount: z.number().int().optional(),
  })
  .passthrough();
const OpenPenalty = z
  .object({
    userId: z.string().uuid(),
    name: z.string(),
    avatarColor: z.string(),
    hasPhoto: z.boolean().optional(),
    amount: z.number().int(),
  })
  .passthrough();
const ContributionStatus = z.enum(['paid', 'open']);
const Contribution = z
  .object({
    id: z.string().uuid(),
    teamId: z.string().uuid(),
    userId: z.string().uuid(),
    month: z.string(),
    label: z.string().optional(),
    amount: z.number().int(),
    status: ContributionStatus,
    memberName: z.string().optional(),
    memberAvatarColor: z.string().optional(),
    hasPhoto: z.boolean().optional(),
  })
  .passthrough();
const FinanceOverview = z
  .object({
    balance: z.number().int(),
    income: z.number().int(),
    expense: z.number().int(),
    transactions: z.array(Transaction),
    penalties: z.array(Penalty),
    assignments: z.array(PenaltyAssignment),
    openPenalties: z.array(OpenPenalty),
    openPenaltySum: z.number().int(),
    contributions: z.array(Contribution),
    contribOpen: z.number().int(),
  })
  .passthrough();
const CreateTransactionRequest = z
  .object({
    type: TransactionType,
    title: z.string().max(255),
    amount: z.number().int().gte(1).lte(100000000),
    category: z.string().max(255).optional(),
  })
  .passthrough();
const UpdateTransactionRequest = z
  .object({
    type: TransactionType,
    title: z.string().max(255),
    amount: z.number().int().gte(1).lte(100000000),
    category: z.string().max(255),
  })
  .partial()
  .passthrough();
const CreatePenaltyRequest = z
  .object({ label: z.string(), amount: z.number().int().gte(1).lte(100000000) })
  .passthrough();
const UpdatePenaltyRequest = z
  .object({ label: z.string(), amount: z.number().int().gte(1).lte(100000000) })
  .partial()
  .passthrough();
const CreatePenaltyAssignmentRequest = z
  .object({ userId: z.string().uuid(), penaltyId: z.string().uuid() })
  .passthrough();
const UpdateContributionRequest = z
  .object({ label: z.string(), amount: z.number().int().gte(1).lte(100000000) })
  .partial()
  .passthrough();
const MemberStat = z
  .object({
    userId: z.string().uuid(),
    name: z.string(),
    avatarColor: z.string(),
    hasPhoto: z.boolean().optional(),
    quote: z.number(),
    counted: z.number().int(),
    yes: z.number().int(),
  })
  .passthrough();
const EventStat = z
  .object({
    id: z.string().uuid(),
    title: z.string(),
    type: EventType,
    date: z.string(),
    yes: z.number().int(),
    nominated: z.number().int(),
    pct: z.number(),
    enough: z.boolean(),
  })
  .passthrough();
const StatsOverview = z
  .object({
    avg: z.number(),
    members: z.array(MemberStat),
    events: z.array(EventStat),
    pastCount: z.number().int(),
    from: z.string(),
    to: z.string(),
  })
  .passthrough();
const MemberAttendanceStats = z
  .object({
    quote: z.number(),
    counted: z.number().int(),
    yes: z.number().int(),
  })
  .passthrough();
const Problem = z
  .object({
    type: z.string(),
    title: z.string(),
    status: z.number().int(),
    detail: z.string(),
  })
  .partial()
  .passthrough();

export const schemas = {
  Provider,
  LoginRequest,
  User,
  LoginResponse,
  DeleteAccountRequest,
  Team,
  PermLevel,
  Permissions,
  Role,
  TeamForUser,
  CreateTeamRequest,
  UpdateTeamRequest,
  Invite,
  AcceptInviteResponse,
  Member,
  UpdateMemberRequest,
  SetRolesRequest,
  CreateRoleRequest,
  UpdateRoleRequest,
  EventType,
  ResponseMode,
  EventStatus,
  EventSummary,
  AttendanceStatus,
  TeamEvent,
  CreateEventRequest,
  UpdateEventRequest,
  SetEventStatusRequest,
  EventComment,
  AddCommentRequest,
  AttendanceRow,
  SetAttendanceRequest,
  AttendanceRecord,
  SetNominationRequest,
  Absence,
  CreateAbsenceRequest,
  UpdateAbsenceRequest,
  NewsItem,
  CreateNewsRequest,
  UpdateNewsRequest,
  PollOption,
  Poll,
  CreatePollRequest,
  VotePollRequest,
  NotificationType,
  AppNotification,
  NotificationsResult,
  TransactionType,
  Transaction,
  Penalty,
  PenaltyAssignment,
  OpenPenalty,
  ContributionStatus,
  Contribution,
  FinanceOverview,
  CreateTransactionRequest,
  UpdateTransactionRequest,
  CreatePenaltyRequest,
  UpdatePenaltyRequest,
  CreatePenaltyAssignmentRequest,
  UpdateContributionRequest,
  MemberStat,
  EventStat,
  StatsOverview,
  MemberAttendanceStats,
  Problem,
};

const endpoints = makeApi([
  {
    method: 'post',
    path: '/auth/login',
    alias: 'login',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: LoginRequest,
      },
    ],
    response: LoginResponse,
    errors: [
      {
        status: 401,
        description: `Unauthorized`,
        schema: z.void(),
      },
      {
        status: 429,
        description: `Too Many Requests. Every endpoint is subject to the global per-IP rate limit (RATE_LIMIT_RPS); /auth/login additionally enforces a stricter per-IP limit (LOGIN_RATE_LIMIT_PER_MIN) for brute-force protection.`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'post',
    path: '/auth/logout',
    alias: 'logout',
    requestFormat: 'json',
    response: z.void(),
  },
  {
    method: 'get',
    path: '/auth/me',
    alias: 'getCurrentUser',
    requestFormat: 'json',
    response: User,
    errors: [
      {
        status: 401,
        description: `Unauthorized`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'delete',
    path: '/auth/me',
    alias: 'deleteCurrentUser',
    description: `Anonymizes the user&#x27;s personal data (name, email, phone, birthday, address, photo) and strips free-text PII from their comments and absence reasons, then deletes all of their sessions. Membership, attendance and finance records are retained in anonymized form so that shared and legally required data (e.g. accounting) stays intact. The request is authorized by the active session; to confirm intent the caller must echo the account&#x27;s own email address (works regardless of login method, including OIDC accounts that have no password).
`,
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: z.object({ confirmEmail: z.string().email() }).passthrough(),
      },
    ],
    response: z.void(),
    errors: [
      {
        status: 401,
        description: `Unauthorized`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'get',
    path: '/auth/me/data-export',
    alias: 'getMyDataExport',
    description: `Returns a single JSON document with all personal data held about the authenticated user: profile, memberships and roles, attendance, comments, absences, authored news, created polls, votes, penalty assignments and contributions.
`,
    requestFormat: 'json',
    response: z.object({}).partial().passthrough(),
    errors: [
      {
        status: 401,
        description: `Unauthorized`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'get',
    path: '/auth/me/photo',
    alias: 'getMyPhoto',
    requestFormat: 'json',
    response: z.void(),
    errors: [
      {
        status: 404,
        description: `Not Found`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'put',
    path: '/auth/me/photo',
    alias: 'uploadMyPhoto',
    requestFormat: 'form-data',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: z.object({ photo: z.instanceof(File) }).passthrough(),
      },
    ],
    response: User,
    errors: [
      {
        status: 413,
        description: `Payload Too Large. All request bodies are capped at 4 MB by default; image upload endpoints additionally enforce their own lower limit.`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'get',
    path: '/auth/providers',
    alias: 'listProviders',
    requestFormat: 'json',
    response: z.array(Provider),
  },
  {
    method: 'post',
    path: '/invites/:code/accept',
    alias: 'acceptInvite',
    description: `Idempotent: redeeming a code for a team the caller already belongs to just returns that team rather than erroring.`,
    requestFormat: 'json',
    parameters: [
      {
        name: 'code',
        type: 'Path',
        schema: z.string(),
      },
    ],
    response: AcceptInviteResponse,
    errors: [
      {
        status: 404,
        description: `Not Found`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'get',
    path: '/teams',
    alias: 'listTeams',
    requestFormat: 'json',
    response: z.array(TeamForUser),
  },
  {
    method: 'post',
    path: '/teams',
    alias: 'createTeam',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: CreateTeamRequest,
      },
    ],
    response: TeamForUser,
  },
  {
    method: 'get',
    path: '/teams/:teamId',
    alias: 'getTeam',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Team,
    errors: [
      {
        status: 404,
        description: `Not Found`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'patch',
    path: '/teams/:teamId',
    alias: 'updateTeam',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: UpdateTeamRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Team,
  },
  {
    method: 'get',
    path: '/teams/:teamId/absences',
    alias: 'listAbsences',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'limit',
        type: 'Query',
        schema: z.number().int().gte(1).lte(500).optional().default(50),
      },
      {
        name: 'cursor',
        type: 'Query',
        schema: z.string().optional(),
      },
    ],
    response: z.object({ items: z.array(Absence), nextCursor: z.string().nullable() }).passthrough(),
  },
  {
    method: 'post',
    path: '/teams/:teamId/absences',
    alias: 'createAbsence',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: CreateAbsenceRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Absence,
  },
  {
    method: 'patch',
    path: '/teams/:teamId/absences/:absenceId',
    alias: 'updateAbsence',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: UpdateAbsenceRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'absenceId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Absence,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/absences/:absenceId',
    alias: 'deleteAbsence',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'absenceId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'get',
    path: '/teams/:teamId/absences/mine',
    alias: 'listMyAbsences',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'limit',
        type: 'Query',
        schema: z.number().int().gte(1).lte(500).optional().default(50),
      },
      {
        name: 'cursor',
        type: 'Query',
        schema: z.string().optional(),
      },
    ],
    response: z.object({ items: z.array(Absence), nextCursor: z.string().nullable() }).passthrough(),
  },
  {
    method: 'get',
    path: '/teams/:teamId/events',
    alias: 'listEvents',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'scope',
        type: 'Query',
        schema: z.enum(['upcoming', 'past', 'all']).optional().default('upcoming'),
      },
      {
        name: 'limit',
        type: 'Query',
        schema: z.number().int().gte(1).lte(500).optional().default(50),
      },
      {
        name: 'cursor',
        type: 'Query',
        schema: z.string().optional(),
      },
    ],
    response: z.object({ items: z.array(TeamEvent), nextCursor: z.string().nullable() }).passthrough(),
  },
  {
    method: 'post',
    path: '/teams/:teamId/events',
    alias: 'createEvent',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: CreateEventRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: TeamEvent,
  },
  {
    method: 'get',
    path: '/teams/:teamId/events/:eventId',
    alias: 'getEvent',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: TeamEvent,
    errors: [
      {
        status: 404,
        description: `Not Found`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'patch',
    path: '/teams/:teamId/events/:eventId',
    alias: 'updateEvent',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: UpdateEventRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'scope',
        type: 'Query',
        schema: z.enum(['single', 'series']).optional().default('single'),
      },
    ],
    response: TeamEvent,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/events/:eventId',
    alias: 'deleteEvent',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'scope',
        type: 'Query',
        schema: z.enum(['single', 'series']).optional().default('single'),
      },
    ],
    response: z.void(),
  },
  {
    method: 'get',
    path: '/teams/:teamId/events/:eventId/attendance',
    alias: 'listAttendance',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.array(AttendanceRow),
  },
  {
    method: 'post',
    path: '/teams/:teamId/events/:eventId/attendance',
    alias: 'setAttendance',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: SetAttendanceRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: AttendanceRecord,
  },
  {
    method: 'put',
    path: '/teams/:teamId/events/:eventId/attendance/nominations',
    alias: 'setNomination',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: SetNominationRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'get',
    path: '/teams/:teamId/events/:eventId/comments',
    alias: 'listEventComments',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'limit',
        type: 'Query',
        schema: z.number().int().gte(1).lte(500).optional().default(50),
      },
      {
        name: 'offset',
        type: 'Query',
        schema: z.number().int().gte(0).optional().default(0),
      },
    ],
    response: z.array(EventComment),
  },
  {
    method: 'post',
    path: '/teams/:teamId/events/:eventId/comments',
    alias: 'addEventComment',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: z.object({ text: z.string().min(1) }).passthrough(),
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: EventComment,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/events/:eventId/comments/:commentId',
    alias: 'deleteEventComment',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'commentId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'post',
    path: '/teams/:teamId/events/:eventId/status',
    alias: 'setEventStatus',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: SetEventStatusRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'eventId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'scope',
        type: 'Query',
        schema: z.enum(['single', 'series']).optional().default('single'),
      },
    ],
    response: TeamEvent,
  },
  {
    method: 'get',
    path: '/teams/:teamId/finances',
    alias: 'getFinanceOverview',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: FinanceOverview,
  },
  {
    method: 'patch',
    path: '/teams/:teamId/finances/contributions/:contributionId',
    alias: 'updateContribution',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: UpdateContributionRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'contributionId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Contribution,
  },
  {
    method: 'post',
    path: '/teams/:teamId/finances/contributions/:contributionId/toggle',
    alias: 'toggleContribution',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'contributionId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Contribution,
  },
  {
    method: 'post',
    path: '/teams/:teamId/finances/penalties',
    alias: 'createPenalty',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: CreatePenaltyRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Penalty,
  },
  {
    method: 'patch',
    path: '/teams/:teamId/finances/penalties/:penaltyId',
    alias: 'updatePenalty',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: UpdatePenaltyRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'penaltyId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Penalty,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/finances/penalties/:penaltyId',
    alias: 'deletePenalty',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'penaltyId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'post',
    path: '/teams/:teamId/finances/penalty-assignments',
    alias: 'createPenaltyAssignment',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: CreatePenaltyAssignmentRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: PenaltyAssignment,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/finances/penalty-assignments/:assignmentId',
    alias: 'deletePenaltyAssignment',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'assignmentId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'post',
    path: '/teams/:teamId/finances/penalty-assignments/:assignmentId/toggle-paid',
    alias: 'togglePenaltyPaid',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'assignmentId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: PenaltyAssignment,
  },
  {
    method: 'post',
    path: '/teams/:teamId/finances/transactions',
    alias: 'createTransaction',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: CreateTransactionRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Transaction,
  },
  {
    method: 'patch',
    path: '/teams/:teamId/finances/transactions/:transactionId',
    alias: 'updateTransaction',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: UpdateTransactionRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'transactionId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Transaction,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/finances/transactions/:transactionId',
    alias: 'deleteTransaction',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'transactionId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'post',
    path: '/teams/:teamId/invite',
    alias: 'createInvite',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Invite,
  },
  {
    method: 'get',
    path: '/teams/:teamId/logo',
    alias: 'getTeamLogo',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
    errors: [
      {
        status: 404,
        description: `Not Found`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'put',
    path: '/teams/:teamId/logo',
    alias: 'uploadTeamLogo',
    requestFormat: 'form-data',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: z.object({ logo: z.instanceof(File) }).passthrough(),
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Team,
    errors: [
      {
        status: 413,
        description: `Payload Too Large. All request bodies are capped at 4 MB by default; image upload endpoints additionally enforce their own lower limit.`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'delete',
    path: '/teams/:teamId/logo',
    alias: 'deleteTeamLogo',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'get',
    path: '/teams/:teamId/members',
    alias: 'listMembers',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'limit',
        type: 'Query',
        schema: z.number().int().gte(1).lte(500).optional().default(50),
      },
      {
        name: 'cursor',
        type: 'Query',
        schema: z.string().optional(),
      },
    ],
    response: z.object({ items: z.array(Member), nextCursor: z.string().nullable() }).passthrough(),
  },
  {
    method: 'patch',
    path: '/teams/:teamId/members/:membershipId',
    alias: 'updateMember',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: UpdateMemberRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'membershipId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Member,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/members/:membershipId',
    alias: 'removeMember',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'membershipId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'put',
    path: '/teams/:teamId/members/:membershipId/roles',
    alias: 'setMemberRoles',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: SetRolesRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'membershipId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Member,
  },
  {
    method: 'get',
    path: '/teams/:teamId/news',
    alias: 'listNews',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'limit',
        type: 'Query',
        schema: z.number().int().gte(1).lte(500).optional().default(50),
      },
      {
        name: 'cursor',
        type: 'Query',
        schema: z.string().optional(),
      },
    ],
    response: z.object({ items: z.array(NewsItem), nextCursor: z.string().nullable() }).passthrough(),
  },
  {
    method: 'post',
    path: '/teams/:teamId/news',
    alias: 'createNews',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: CreateNewsRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: NewsItem,
  },
  {
    method: 'patch',
    path: '/teams/:teamId/news/:newsId',
    alias: 'updateNews',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: UpdateNewsRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'newsId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: NewsItem,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/news/:newsId',
    alias: 'deleteNews',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'newsId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'get',
    path: '/teams/:teamId/notifications',
    alias: 'listNotifications',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: NotificationsResult,
  },
  {
    method: 'post',
    path: '/teams/:teamId/notifications/seen',
    alias: 'markNotificationsSeen',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'get',
    path: '/teams/:teamId/photo',
    alias: 'getTeamPhoto',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
    errors: [
      {
        status: 404,
        description: `Not Found`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'put',
    path: '/teams/:teamId/photo',
    alias: 'uploadTeamPhoto',
    requestFormat: 'form-data',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: z.object({ photo: z.instanceof(File) }).passthrough(),
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Team,
    errors: [
      {
        status: 413,
        description: `Payload Too Large. All request bodies are capped at 4 MB by default; image upload endpoints additionally enforce their own lower limit.`,
        schema: z.void(),
      },
    ],
  },
  {
    method: 'delete',
    path: '/teams/:teamId/photo',
    alias: 'deleteTeamPhoto',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'get',
    path: '/teams/:teamId/polls',
    alias: 'listPolls',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'limit',
        type: 'Query',
        schema: z.number().int().gte(1).lte(500).optional().default(50),
      },
      {
        name: 'cursor',
        type: 'Query',
        schema: z.string().optional(),
      },
    ],
    response: z.object({ items: z.array(Poll), nextCursor: z.string().nullable() }).passthrough(),
  },
  {
    method: 'post',
    path: '/teams/:teamId/polls',
    alias: 'createPoll',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: CreatePollRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Poll,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/polls/:pollId',
    alias: 'deletePoll',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'pollId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'post',
    path: '/teams/:teamId/polls/:pollId/vote',
    alias: 'votePoll',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: VotePollRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'pollId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Poll,
  },
  {
    method: 'get',
    path: '/teams/:teamId/roles',
    alias: 'listRoles',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.array(Role),
  },
  {
    method: 'post',
    path: '/teams/:teamId/roles',
    alias: 'createRole',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: CreateRoleRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Role,
  },
  {
    method: 'patch',
    path: '/teams/:teamId/roles/:roleId',
    alias: 'updateRole',
    requestFormat: 'json',
    parameters: [
      {
        name: 'body',
        type: 'Body',
        schema: UpdateRoleRequest,
      },
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'roleId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: Role,
  },
  {
    method: 'delete',
    path: '/teams/:teamId/roles/:roleId',
    alias: 'deleteRole',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'roleId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: z.void(),
  },
  {
    method: 'get',
    path: '/teams/:teamId/stats',
    alias: 'getStatsOverview',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'from',
        type: 'Query',
        schema: z.string().optional(),
      },
      {
        name: 'to',
        type: 'Query',
        schema: z.string().optional(),
      },
    ],
    response: StatsOverview,
  },
  {
    method: 'get',
    path: '/teams/:teamId/stats/members/:userId',
    alias: 'getMemberStats',
    requestFormat: 'json',
    parameters: [
      {
        name: 'teamId',
        type: 'Path',
        schema: z.string().uuid(),
      },
      {
        name: 'userId',
        type: 'Path',
        schema: z.string().uuid(),
      },
    ],
    response: MemberAttendanceStats,
  },
]);

export const api = new Zodios(endpoints);

export function createApiClient(baseUrl: string, options?: ZodiosOptions) {
  return new Zodios(baseUrl, endpoints, options);
}
