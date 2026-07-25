import type { AttendanceStatus } from '@/types';

export type NotificationType =
  | 'attendance'
  | 'event_created'
  | 'event_updated'
  | 'event_cancelled'
  | 'event_reactivated'
  | 'event_deleted'
  | 'news'
  | 'poll'
  | 'absence';

export interface AppNotification {
  id: string;
  teamId: string;
  type: NotificationType;
  actorId?: string;
  status?: AttendanceStatus;
  title?: string;
  eventId?: string | null;
  eventTitle?: string;
  // Explicit `| undefined` -- eventDate is optional per the OpenAPI
  // AppNotification schema (nullable event_date DB column); an attendance
  // notification for an event with no date genuinely has `eventDate:
  // undefined`, not just an omitted field (see NotificationsSheet.test.tsx's
  // regression test for the crash this used to cause).
  eventDate?: string | undefined;
  note?: string;
  createdAt: string;
  actorName?: string;
  actorColor?: string;
  actorPhoto?: string | null;
  unread?: boolean;
}

export interface NotificationsResult {
  items: AppNotification[];
  unreadCount: number;
}
