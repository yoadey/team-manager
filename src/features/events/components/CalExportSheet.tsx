import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import type { SheetProps } from '@/sheets/types';
import { buildTokens } from '@/styles/tokens';
import { Sym, PrimaryButton } from '@/components/ui';

export function CalExportSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const cnt = (state.events || []).filter((e) => e.status !== 'cancelled').length;
  const url = 'https://teamverwaltung.app/cal/' + ((team && team.id) || 'team') + '.ics';

  const hint = (icon: string, title: string, text: string) => (
    <Box key={title} sx={{ display: 'flex', gap: '12px', alignItems: 'flex-start', p: '12px 0' }}>
      <Sym name={icon} size={20} color="#6A6D76" />
      <Box key="m" sx={{ flex: 1 }}>
        <Box key="t" sx={{ fontSize: '13px', fontWeight: 700, mb: '2px' }}>{title}</Box>
        <Box key="d" sx={{ fontSize: '12px', color: '#6A6D76', lineHeight: 1.5 }}>{text}</Box>
      </Box>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box key="hero" sx={{ textAlign: 'center', p: '4px 2px 6px' }}>
        <Box key="i" sx={{ width: '62px', height: '62px', borderRadius: '18px', background: t.primaryContainer, color: t.primary, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Material Symbols Outlined'", fontSize: '32px' }}>event_upcoming</Box>
        <Box key="s" sx={{ fontSize: '14px', color: '#6A6D76', mt: '12px', lineHeight: 1.5 }}>{'Exportiere alle ' + cnt + ' aktiven Termine und binde sie in deinen Kalender ein.'}</Box>
      </Box>

      <PrimaryButton label="Kalenderdatei (.ics) herunterladen" onClick={() => app.downloadIcs()} />

      <Box key="sub" sx={{ border: '1px solid #E6E7EE', borderRadius: '16px', p: '14px', background: '#fff' }}>
        <Box key="h" sx={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', fontWeight: 700, mb: '4px' }}>
          <Sym name="sync" size={18} color={t.primary} />
          Automatisch abonnieren (Abo-Link)
        </Box>
        <Box key="d" sx={{ fontSize: '12px', color: '#6A6D76', lineHeight: 1.5, mb: '10px' }}>Mit diesem Link bleibt dein Kalender automatisch aktuell – neue und geänderte Termine erscheinen ohne erneuten Export.</Box>
        <Box key="box" sx={{ display: 'flex', alignItems: 'center', gap: '8px', background: '#F4F4FA', border: '1px solid #E0E2EA', borderRadius: '12px', p: '10px 12px' }}>
          <Box key="l" component="span" sx={{ flex: 1, fontSize: '12px', fontFamily: 'monospace', color: '#1A1C20', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{url}</Box>
          <ButtonBase
            key="c"
            onClick={() => app.copyCalUrl()}
            sx={{ display: 'flex', alignItems: 'center', gap: '6px', background: t.primary, color: t.onPrimary, border: 'none', borderRadius: '9px', p: '8px 12px', fontSize: '12px', fontWeight: 600, cursor: 'pointer', flex: '0 0 auto' }}
          >
            <Sym name="content_copy" size={15} color={t.onPrimary} />
            {sheet.copied ? 'Kopiert' : 'Kopieren'}
          </ButtonBase>
        </Box>
      </Box>

      <Box key="hints" sx={{ borderTop: '1px solid #ECEDF3', pt: '6px' }}>
        {hint('calendar_month', 'Google Kalender', 'Einstellungen → „Kalender hinzufügen“ → „Per URL“ und den Abo-Link einfügen. Auf Android wird der Kalender automatisch synchronisiert.')}
        {hint('phone_iphone', 'Apple / iOS', 'Einstellungen → Kalender → Accounts → „Kalenderabo hinzufügen“ und den Abo-Link einfügen.')}
        {hint('download', 'Einmaliger Import', 'Alternativ die .ics-Datei herunterladen und in jeder Kalender-App öffnen – fügt die Termine einmalig hinzu.')}
      </Box>

      <Box key="note" sx={{ display: 'flex', gap: '10px', background: '#FFF7E6', border: '1px solid #F0DBA8', borderRadius: '13px', p: '12px 14px', fontSize: '12px', color: '#6B5413', lineHeight: 1.5 }}>
        <Sym name="info" size={18} color="#8A6100" />
        Im Prototyp ist nur der Datei-Download aktiv; der Abo-Link wird mit dem späteren Backend funktionsfähig.
      </Box>
    </Box>
  );
}
