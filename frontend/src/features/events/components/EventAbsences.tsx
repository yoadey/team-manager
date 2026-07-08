import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtRange, NEUTRAL } from '@/styles/tokens';
import { todayLocalDate } from '@/utils/date';
import { Av, Chip, EmptyState, SectionTitle, Sym, SpinnerBox } from '@/components/ui';
import { t } from '@/i18n';

export function EventAbsences() {
  const app = useApp();
  const { state } = app;
  const tk = buildTokens(state.primaryColor);

  if (!state.absences) return <SpinnerBox />;

  const today = todayLocalDate();
  const list = state.absences.filter((a) => a.to >= today);

  const rows = list.map((a) => {
    const isMe = a.userId === state.user!.id;
    return (
      <Box
        key={a.id}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          background: NEUTRAL.card,
          border: `1px solid ${NEUTRAL.line}`,
          borderRadius: '15px',
          p: '12px 14px',
        }}
      >
        <Av name={a.name} photo={a.photo} color={a.avatarColor} size={40} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: '14px', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '7px' }}>
            {a.name}
            {isMe ? <Chip label={t('events.meLabel')} color={tk.primary} bg={tk.primaryContainer} /> : null}
          </Box>
          <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '2px' }}>
            {fmtRange(a.from, a.to) + ' · ' + a.reason}
          </Box>
        </Box>
        <Box
          component="span"
          sx={{ width: '10px', height: '10px', borderRadius: '50%', background: a.roleColor, flex: '0 0 auto' }}
        />
        {isMe ? (
          <ButtonBase
            onClick={() => app.openAbsenceForm(a)}
            aria-label={t('events.editAbsenceLabel')}
            sx={{
              width: '34px',
              height: '34px',
              borderRadius: '50%',
              border: 'none',
              background: NEUTRAL.sidebar,
              color: NEUTRAL.onSurfaceVariant,
              flex: '0 0 auto',
            }}
          >
            <Sym name="edit" size={18} color={NEUTRAL.onSurfaceVariant} />
          </ButtonBase>
        ) : null}
        {isMe ? (
          <ButtonBase
            onClick={() => app.removeAbsence(a.id)}
            aria-label={t('events.deleteAbsenceLabel')}
            sx={{
              width: '34px',
              height: '34px',
              borderRadius: '50%',
              border: 'none',
              background: NEUTRAL.errorBg,
              color: NEUTRAL.error,
              flex: '0 0 auto',
            }}
          >
            <Sym name="delete" size={19} color={NEUTRAL.error} />
          </ButtonBase>
        ) : null}
      </Box>
    );
  });

  return (
    <Box sx={{ maxWidth: '720px' }}>
      <ButtonBase
        onClick={() => app.openAbsenceForm()}
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '9px',
          width: '100%',
          p: '13px',
          borderRadius: '14px',
          border: `1.5px dashed ${NEUTRAL.inputBorder}`,
          background: 'transparent',
          color: tk.primary,
          fontWeight: 600,
          fontSize: '14px',
          mb: '18px',
        }}
      >
        <Sym name="event_busy" size={20} color={tk.primary} />
        {t('events.addAbsence')}
      </ButtonBase>
      <SectionTitle>{t('events.plannedAbsences')}</SectionTitle>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '9px' }}>
        {list.length ? rows : <EmptyState icon="beach_access" text={t('events.noAbsences')} />}
      </Box>
    </Box>
  );
}
