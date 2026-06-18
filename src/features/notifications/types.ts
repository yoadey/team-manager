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
  eventDate?: string;
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
