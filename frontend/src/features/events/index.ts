export type {
  ResponseMode,
  EventSummary,
  EventDto,
  TeamEvent,
  AttendanceDto,
  AttendanceRow,
  EventComment,
  Absence,
  EventFormValues,
  AbsenceFormValues,
  AttendanceCommentFormValues,
  EventCommentFormValues,
} from './types';

// The sheet components and `eventSheetMap` are declared FIRST — before the
// heavier re-exports below (EventsPage, hooks). The sheet dispatcher
// (src/sheets/index.tsx) spreads `eventSheetMap` at module-init time; defining
// it up here ensures it is initialised before any circular re-entry that the
// page/hook re-exports might trigger, avoiding a temporal-dead-zone error in
// Vite's unbundled dev ESM ("can't access 'eventSheetMap' before initialization").
import { EventDetailSheet } from './components/EventDetailSheet';
import { EventFormSheet } from './components/EventFormSheet';
import { AbsenceFormSheet } from './components/AbsenceFormSheet';
import { CalExportSheet } from './components/CalExportSheet';

export const eventSheetMap = {
  eventDetail: EventDetailSheet,
  eventForm: EventFormSheet,
  absenceForm: AbsenceFormSheet,
  calExport: CalExportSheet,
} as const;

export { EventDetailSheet, EventFormSheet, AbsenceFormSheet, CalExportSheet };

export { EventsPage } from './EventsPage';
export { EventCalendar } from './components/EventCalendar';
export { EventAbsences } from './components/EventAbsences';
export { useEventActionFeatures, useEventDetailActions } from './hooks/useEventActions';
export { useAbsenceActions } from './hooks/useAbsenceActions';
export { useCalExportActions } from './hooks/useCalExportActions';
export { useEventFormActions } from './hooks/useEventFormActions';
export { useEventsQuery, useEventDetailQuery } from './hooks/useEventQueries';
export type { EventDetailData } from './hooks/useEventQueries';
export { useInvalidateEvents } from './hooks/useEventMutations';
