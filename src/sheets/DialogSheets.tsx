import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import type { SheetProps } from './types';
import { buildTokens, statusMeta, NEUTRAL } from '@/styles/tokens';
import { Sym, Chip, PrimaryButton, inputSx } from '@/components/ui';

export function ConfirmSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const t = buildTokens(state.primaryColor);
  const c = sheet.cfg || {};
  const danger = !!c.danger;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
      <Box
        key="ic"
        sx={{
          width: '56px',
          height: '56px',
          borderRadius: '16px',
          background: danger ? NEUTRAL.errorBg : t.primaryContainer,
          color: danger ? NEUTRAL.error : t.primary,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontFamily: "'Material Symbols Outlined'",
          fontSize: '30px',
        }}
      >
        {danger ? 'warning' : 'help'}
      </Box>
      <Box key="m" sx={{ fontSize: '14px', color: '#44474E', lineHeight: 1.55 }}>
        {c.message || 'Bist du sicher?'}
      </Box>
      <Box key="b" sx={{ display: 'flex', gap: '10px' }}>
        <ButtonBase
          key="x"
          onClick={() => app.cancelConfirm()}
          sx={{
            flex: 1,
            p: '13px',
            borderRadius: '13px',
            border: '1px solid #C8CAD2',
            background: '#fff',
            color: '#44474E',
            fontWeight: 600,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          Abbrechen
        </ButtonBase>
        <ButtonBase
          key="ok"
          onClick={() => app.runConfirm()}
          sx={{
            flex: 1,
            p: '13px',
            borderRadius: '13px',
            border: 'none',
            background: danger ? NEUTRAL.error : t.primary,
            color: '#fff',
            fontWeight: 700,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          {c.confirmLabel || 'Bestätigen'}
        </ButtonBase>
      </Box>
    </Box>
  );
}

export function SeriesActionSheet({ app, sheet }: SheetProps) {
  const act = sheet.action!;
  const cfg: Record<string, { d: string; ic: string; col: string }> = {
    cancel: {
      d: 'Diesen Termin oder die ganze Serie absagen? Abgesagte Termine bleiben in der Liste sichtbar, Rückmeldungen sind nicht mehr nötig.',
      ic: 'event_busy',
      col: '#8A6100',
    },
    delete: {
      d: 'Diesen Termin oder die ganze Serie löschen? Gelöschte Termine werden vollständig und unwiderruflich entfernt.',
      ic: 'delete',
      col: NEUTRAL.error,
    },
    reactivate: {
      d: 'Diesen Termin oder die ganze Serie wieder aktivieren?',
      ic: 'event_available',
      col: NEUTRAL.success,
    },
  };
  const L = cfg[act] || cfg.cancel;

  const opt = (scope: 'single' | 'series', title: string, sub: string, icon: string) => (
    <ButtonBase
      key={scope}
      onClick={() => app.runEventAction(sheet.action!, sheet.event!, scope)}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '13px',
        width: '100%',
        p: '15px',
        borderRadius: '15px',
        cursor: 'pointer',
        border: '1px solid #E0E2EA',
        background: '#fff',
        textAlign: 'left',
        justifyContent: 'flex-start',
      }}
    >
      <Box
        component="span"
        key="i"
        sx={{
          width: '40px',
          height: '40px',
          borderRadius: '11px',
          background: L.col + '1A',
          color: L.col,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontFamily: "'Material Symbols Outlined'",
          fontSize: '21px',
          flex: '0 0 auto',
        }}
      >
        {icon}
      </Box>
      <Box key="m" sx={{ flex: 1, minWidth: 0 }}>
        <Box key="t" sx={{ fontSize: '15px', fontWeight: 700 }}>
          {title}
        </Box>
        <Box key="s" sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '2px' }}>
          {sub}
        </Box>
      </Box>
      <Sym name="chevron_right" size={20} color="#C0C2CA" />
    </ButtonBase>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '11px' }}>
      <Box key="h" sx={{ display: 'flex', gap: '11px', alignItems: 'flex-start', mb: '2px' }}>
        <Sym name={L.ic} size={24} color={L.col} />
        <Box key="d" sx={{ flex: 1, fontSize: '13px', color: '#44474E', lineHeight: 1.5 }}>
          {L.d}
        </Box>
      </Box>
      {opt('single', 'Nur diesen Termin', 'Betrifft ausschließlich diesen einen Termin', 'event')}
      {opt('series', 'Ganze Serie', 'Betrifft alle Termine dieser Serie', 'repeat')}
    </Box>
  );
}

export function CommentSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const F = state.form;
  const sm = statusMeta(sheet.status!);
  const isMe = sheet.userId === state.user!.id;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
      <Box
        key="who"
        sx={{ display: 'flex', alignItems: 'center', gap: '9px', fontSize: '13px', color: NEUTRAL.secondary }}
      >
        <Chip key="c" label={sm.label} color={sm.color} bg={sm.bg} icon={sm.icon} />
        <Box component="span" key="n">
          {isMe ? 'Dein Kommentar' : 'Kommentar für ' + sheet.name}
        </Box>
      </Box>
      <textarea
        key="t"
        name="commentText"
        value={F.commentText || ''}
        onChange={(e) => app.onFormInput(e)}
        placeholder="Kurzer Kommentar (optional)…"
        style={{ ...inputSx, minHeight: '100px', resize: 'vertical' }}
      />
      {sheet.status === 'no' ? (
        <Box key="h" sx={{ display: 'flex', gap: '8px', fontSize: '12px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>
          <Sym name="visibility" size={16} color={NEUTRAL.faint} />
          Absage-Kommentare sehen nur die in den Team-Einstellungen freigegebenen Rollen.
        </Box>
      ) : null}
      <PrimaryButton label="Kommentar speichern" onClick={() => app.submitComment()} busy={state.busy === 'save'} />
    </Box>
  );
}
