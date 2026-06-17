import { useCallback } from 'react';
import type { TeamForUser } from '../../../types';
import type { AppState } from '../../../context/AppContext';
import { hhmm } from '../../../styles/tokens';
import { combineDateAndTimeLocal } from '../../../utils/date';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type CalExportDeps = {
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  toastMsg: (m: string) => void;
};

export function useCalExportActions({ S, setState, activeTeam, toastMsg }: CalExportDeps) {
  const openCalExport = useCallback(() => setState({ sheet: { type: 'calExport' } }), [setState]);

  const buildIcs = useCallback(() => {
    const team = activeTeam();
    const pad = (n: number) => String(n).padStart(2, '0');
    const fmt = (d: Date) => d.getUTCFullYear() + pad(d.getUTCMonth() + 1) + pad(d.getUTCDate()) + 'T' + pad(d.getUTCHours()) + pad(d.getUTCMinutes()) + '00Z';
    const esc = (s: string) => String(s || '').replace(/\\/g, '\\\\').replace(/\n/g, '\\n').replace(/,/g, '\\,').replace(/;/g, '\\;');
    const fold = (l: string) => (l.length <= 73 ? l : (l.match(/.{1,73}/g) || []).join('\r\n '));
    const evs = (S().events || []).filter((e) => e.status !== 'cancelled');
    const lines = ['BEGIN:VCALENDAR', 'VERSION:2.0', 'PRODID:-//Teamverwaltung//Termine//DE', 'CALSCALE:GREGORIAN', 'METHOD:PUBLISH', 'X-WR-CALNAME:' + esc(team ? team.name : 'Team'), 'X-WR-TIMEZONE:Europe/Berlin'];
    const now = new Date();
    const tMeta: Record<string, string> = { training: 'Training', auftritt: 'Auftritt / Turnier', event: 'Team-Event' };
    evs.forEach((e) => {
      const start = combineDateAndTimeLocal(e.date, hhmm(e.startTime) || hhmm(e.meetTime) || '18:00');
      const end = e.endTime ? combineDateAndTimeLocal(e.date, hhmm(e.endTime)) : new Date(start.getTime() + 2 * 3600 * 1000);
      const descParts: string[] = [];
      if (e.meetTime) descParts.push('Treffen: ' + hhmm(e.meetTime));
      if (e.note) descParts.push(e.note);
      descParts.push('Typ: ' + (tMeta[e.type] || 'Team-Event'));
      lines.push('BEGIN:VEVENT', 'UID:' + e.id + '@teamverwaltung.app', 'DTSTAMP:' + fmt(now), 'DTSTART:' + fmt(start), 'DTEND:' + fmt(end), fold('SUMMARY:' + esc(e.title)));
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
      a.href = url; a.download = ((team && team.short) ? team.short.toLowerCase() : 'team') + '-termine.ics';
      document.body.appendChild(a); a.click();
      setTimeout(() => { document.body.removeChild(a); URL.revokeObjectURL(url); }, 1500);
      toastMsg(ics.count + ' Termine als .ics exportiert');
    } catch { toastMsg('Export nicht möglich'); }
  }, [activeTeam, buildIcs, toastMsg]);

  const copyCalUrl = useCallback(() => {
    const team = activeTeam();
    const url = 'webcal://teamverwaltung.app/cal/' + ((team && team.id) || 'team') + '.ics';
    try { navigator.clipboard.writeText(url.replace('webcal://', 'https://')); } catch { /* ignore */ }
    setState((s) => (s.sheet && s.sheet.type === 'calExport') ? { sheet: { ...s.sheet, copied: true } } : {});
    toastMsg('Abo-Link kopiert');
  }, [activeTeam, setState, toastMsg]);

  return { openCalExport, downloadIcs, copyCalUrl };
}
