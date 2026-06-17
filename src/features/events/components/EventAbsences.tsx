import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '../../../context/AppContext';
import { buildTokens, fmtRange, NEUTRAL } from '../../../styles/tokens';
import { todayLocalDate } from '../../../utils/date';
import { Av, Chip, EmptyState, SectionTitle, Sym, SpinnerBox } from '../../../components/ui';

export function EventAbsences() {
  const app = useApp();
  const { state } = app;
  const t = buildTokens(state.primaryColor);

  if (!state.absences) return <SpinnerBox />;

  const today = todayLocalDate();
  const list = state.absences.filter((a) => a.to >= today);

  const rows = list.map((a) => {
    const isMe = a.userId === state.user!.id;
    return (
      <Box key={a.id} sx={{ display: 'flex', alignItems: 'center', gap: '12px', background: '#fff', border: `1px solid ${NEUTRAL.line}`, borderRadius: '15px', p: '12px 14px' }}>
        <Av name={a.name} photo={a.photo} color={a.avatarColor} size={40} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: '14px', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '7px' }}>
            {a.name}
            {isMe ? <Chip label="Du" color={t.primary} bg={t.primaryContainer} /> : null}
          </Box>
          <Box sx={{ fontSize: '12px', color: '#6A6D76', mt: '2px' }}>{fmtRange(a.from, a.to) + ' · ' + a.reason}</Box>
        </Box>
        <Box component="span" sx={{ width: '10px', height: '10px', borderRadius: '50%', background: a.roleColor, flex: '0 0 auto' }} />
        {isMe ? (
          <ButtonBase onClick={() => app.openAbsenceForm(a)} sx={{ width: '34px', height: '34px', borderRadius: '50%', border: 'none', background: '#F4F4FA', color: '#44474E', flex: '0 0 auto' }}>
            <Sym name="edit" size={18} color="#44474E" />
          </ButtonBase>
        ) : null}
        {isMe ? (
          <ButtonBase onClick={() => app.removeAbsence(a.id)} sx={{ width: '34px', height: '34px', borderRadius: '50%', border: 'none', background: '#FFF4F3', color: '#BA1A1A', flex: '0 0 auto' }}>
            <Sym name="delete" size={19} color="#BA1A1A" />
          </ButtonBase>
        ) : null}
      </Box>
    );
  });

  return (
    <Box sx={{ maxWidth: '720px' }}>
      <ButtonBase
        onClick={() => app.openAbsenceForm()}
        sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '9px', width: '100%', p: '13px', borderRadius: '14px', border: '1.5px dashed #C8CAD2', background: 'transparent', color: t.primary, fontWeight: 600, fontSize: '14px', mb: '18px' }}
      >
        <Sym name="event_busy" size={20} color={t.primary} />
        Eigene Abwesenheit eintragen
      </ButtonBase>
      <SectionTitle>Geplante Abwesenheiten</SectionTitle>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '9px' }}>
        {list.length ? rows : <EmptyState icon="beach_access" text="Keine geplanten Abwesenheiten" />}
      </Box>
    </Box>
  );
}
