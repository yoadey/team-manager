import { useCallback } from 'react';
import type { TeamForUser } from '@/types';
import type { AppState } from '@/context/AppContext';
import { hhmm } from '@/styles/tokens';
import { combineDateAndTimeLocal } from '@/utils/date';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type CalExportDeps = {
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
};

export function useCalExportActions({ S, setState, activeTeam, toastMsg }: CalExportDeps) {
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
    const evs = (S().events || []).filter((e) => e.status !== 'cancelled');
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
    const tMeta: Record<string, string> = { training: 'Training', auftritt: 'Auftritt / Turnier', event: 'Team-Event' };
    evs.forEach((e) => {
      const start = combineDateAndTimeLocal(e.date, hhmm(e.startTime) || hhmm(e.meetTime) || '18:00');
      const end = e.endTime
        ? combineDateAndTimeLocal(e.date, hhmm(e.endTime))
        : new Date(start.getTime() + 2 * 3600 * 1000);
      const descParts: string[] = [];
      if (e.meetTime) descParts.push('Treffen: ' + hhmm(e.meetTime));
      if (e.note) descParts.push(e.note);
      descParts.push('Typ: ' + (tMeta[e.type] || 'Team-Event'));
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
  }, [activeTeam, S]);

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

  const copyCalUrl = useCallback(async () => {
    const teamId = S().activeTeamId;
    const team = activeTeam();
    const url = 'webcal://teamverwaltung.app/cal/' + ((team && team.id) || 'team') + '.ics';
    try {
      await navigator.clipboard.writeText(url.replace('webcal://', 'https://'));
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
      s.activeTeamId === teamId && s.sheet && s.sheet.type === 'calExport' ? { sheet: { ...s.sheet, copied: true } } : {},
    );
    toastMsg(t('events.toastCalLinkCopied'));
  }, [S, activeTeam, setState, toastMsg]);

  return { openCalExport, downloadIcs, copyCalUrl };
}
