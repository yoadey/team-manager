import { useCallback } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import type { TeamForUser } from '@/types';
import type { AppState } from '@/context/AppContext';
import { hhmm } from '@/styles/tokens';
import { zonedTimeToUtc } from '@/utils/date';
import { queryKeys } from '@/query/keys';
import { useEventsQuery } from './useEventQueries';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type CalExportDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  teamId: string | null;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
};

/**
 * The team's calendar subscription URL. Modeled as a useQuery (not a plain
 * fetch-on-open) specifically so reopening the sheet within the same session
 * reuses the cached URL instead of re-issuing a token -- issuing rotates the
 * token server-side, so a background/automatic refetch would silently break
 * a link the user already copied or added to their calendar app. Only
 * `regenerateCalUrl` (an explicit user action) is allowed to replace it.
 */
export function useCalendarFeedUrlQuery(api: typeof defaultApi, teamId: string | null) {
  return useQuery({
    queryKey: queryKeys.calendarFeedUrl(teamId ?? ''),
    queryFn: () => api.events.issueCalendarFeedToken(teamId!),
    enabled: !!teamId,
    staleTime: Infinity,
  });
}

export function useCalExportActions({ api, S, setState, activeTeam, teamId, toastMsg }: CalExportDeps) {
  const { data: events } = useEventsQuery(api, teamId);
  const qc = useQueryClient();
  const openCalExport = useCallback(() => setState({ sheet: { type: 'calExport' } }), [setState]);

  const buildIcs = useCallback(() => {
    const team = activeTeam();
    const pad = (n: number) => String(n).padStart(2, '0');
    const fmt = (d: Date) =>
      d.getUTCFullYear() +
      pad(d.getUTCMonth() + 1) +
      pad(d.getUTCDate()) +
      'T' +
      pad(d.getUTCHours()) +
      pad(d.getUTCMinutes()) +
      '00Z';
    const esc = (s: string) =>
      String(s || '')
        .replace(/\\/g, '\\\\')
        .replace(/\r\n/g, '\\n')
        .replace(/\r/g, '\\n')
        .replace(/\n/g, '\\n')
        .replace(/,/g, '\\,')
        .replace(/;/g, '\\;');
    const fold = (l: string) => (l.length <= 73 ? l : (l.match(/.{1,73}/g) || []).join('\r\n '));
    const evs = (events || []).filter((e) => e.status !== 'cancelled');
    const lines = [
      'BEGIN:VCALENDAR',
      'VERSION:2.0',
      'PRODID:-//Teamverwaltung//Termine//DE',
      'CALSCALE:GREGORIAN',
      'METHOD:PUBLISH',
      'X-WR-CALNAME:' + esc(team ? team.name : 'Team'),
      'X-WR-TIMEZONE:Europe/Berlin',
    ];
    const now = new Date();
    const typeLabel = (type: string) =>
      type === 'training'
        ? t('eventType.training')
        : type === 'auftritt'
          ? t('eventType.auftritt')
          : t('eventType.event');
    evs.forEach((e) => {
      // e.date/startTime/endTime are team-local (Europe/Berlin) wall-clock
      // strings (see EventDto's doc comment) -- must resolve to the same
      // absolute instant regardless of the exporting browser's own
      // timezone, unlike combineDateAndTimeLocal which would silently
      // reinterpret them in whatever timezone the browser happens to run in.
      const start = zonedTimeToUtc(e.date, hhmm(e.startTime) || hhmm(e.meetTime) || '18:00', 'Europe/Berlin');
      const end = e.endTime
        ? zonedTimeToUtc(e.date, hhmm(e.endTime), 'Europe/Berlin')
        : new Date(start.getTime() + 2 * 3600 * 1000);
      const descParts: string[] = [];
      if (e.meetTime) descParts.push(t('events.meetTime', { time: hhmm(e.meetTime) }));
      if (e.note) descParts.push(e.note);
      descParts.push(t('events.eventType') + ': ' + typeLabel(e.type));
      lines.push(
        'BEGIN:VEVENT',
        'UID:' + e.id + '@teamverwaltung.app',
        'DTSTAMP:' + fmt(now),
        'DTSTART:' + fmt(start),
        'DTEND:' + fmt(end),
        fold('SUMMARY:' + esc(e.title)),
      );
      if (e.location) lines.push(fold('LOCATION:' + esc(e.location)));
      lines.push(fold('DESCRIPTION:' + esc(descParts.join('\n'))), 'END:VEVENT');
    });
    lines.push('END:VCALENDAR');
    return { text: lines.join('\r\n'), count: evs.length };
  }, [activeTeam, events]);

  const downloadIcs = useCallback(() => {
    const team = activeTeam();
    const ics = buildIcs();
    try {
      const blob = new Blob([ics.text], { type: 'text/calendar;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = (team && team.short ? team.short.toLowerCase() : 'team') + '-termine.ics';
      document.body.appendChild(a);
      a.click();
      setTimeout(() => {
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
      }, 1500);
      toastMsg(t('events.toastCalExported', { n: ics.count }));
    } catch {
      toastMsg(t('events.exportFailed'), undefined, 'error');
    }
  }, [activeTeam, buildIcs, toastMsg]);

  const copyCalUrl = useCallback(
    async (url: string) => {
      const teamId = S().activeTeamId;
      try {
        await navigator.clipboard.writeText(url);
      } catch {
        toastMsg(t('error.copy'), undefined, 'error');
        return;
      }
      // Must check the team too, not just the sheet type: if the user switched
      // teams and reopened the calExport sheet (also type 'calExport') for the
      // new team before the clipboard write resolved, a type-only check would
      // show "Copied!" on the new team's sheet even though nothing was copied
      // for it.
      setState((s) =>
        s.activeTeamId === teamId && s.sheet && s.sheet.type === 'calExport'
          ? { sheet: { ...s.sheet, copied: true } }
          : {},
      );
      toastMsg(t('events.toastCalLinkCopied'));
    },
    [S, setState, toastMsg],
  );

  // Explicit "renew link" action: re-issues the token (rotating it
  // server-side -- the previous URL stops working immediately) and updates
  // the cached query result so the sheet re-renders with the new URL.
  const regenerateCalUrl = useCallback(async () => {
    if (!teamId) return;
    try {
      const url = await api.events.issueCalendarFeedToken(teamId);
      qc.setQueryData(queryKeys.calendarFeedUrl(teamId), url);
      setState((s) => (s.sheet && s.sheet.type === 'calExport' ? { sheet: { ...s.sheet, copied: false } } : {}));
      toastMsg(t('events.toastCalLinkRenewed'));
    } catch {
      toastMsg(t('events.calRenewFailed'), undefined, 'error');
    }
  }, [api, qc, setState, teamId, toastMsg]);

  return { openCalExport, downloadIcs, copyCalUrl, regenerateCalUrl };
}
