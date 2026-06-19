import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Av, Chip, Sym } from '@/components/ui';
import { t } from '@/i18n';

export function MembersPage() {
  const app = useApp();
  const { state } = app;
  const tk = buildTokens(state.primaryColor);

  const list = state.members;

  const rows = list.map((m) => {
    const isMe = m.userId === state.user!.id;
    const extra = m.roles.length - 1;
    return (
      <ButtonBase
        key={m.membershipId}
        onClick={() => app.openMemberDetail(m.membershipId)}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '13px',
          width: '100%',
          textAlign: 'left',
          justifyContent: 'flex-start',
          background: '#fff',
          border: `1px solid ${NEUTRAL.line}`,
          borderRadius: '16px',
          p: '12px 14px',
        }}
      >
        <Av name={m.name} photo={m.photo} color={m.avatarColor} size={44} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: '7px' }}>
            <Box
              component="span"
              sx={{
                fontSize: '15px',
                fontWeight: 600,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {m.name}
            </Box>
            {isMe ? <Chip label={t('members.meLabel')} color={tk.primary} bg={tk.primaryContainer} /> : null}
          </Box>
          <Box
            sx={{
              fontSize: '12px',
              color: '#6A6D76',
              mt: '2px',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {m.roles.map((r) => r.name).join(' · ')}
          </Box>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
          {m.primaryRole ? (
            <Chip
              label={m.primaryRole.name}
              color={m.primaryRole.color}
              bg={m.primaryRole.color + '1A'}
              icon="circle"
              fs={11}
            />
          ) : null}
          {extra > 0 ? (
            <Box
              component="span"
              sx={{
                fontSize: '11px',
                fontWeight: 700,
                color: '#6A6D76',
                background: '#ECEDF3',
                p: '4px 8px',
                borderRadius: '999px',
              }}
            >
              {'+' + extra}
            </Box>
          ) : null}
        </Box>
        <Sym name="chevron_right" size={22} color="#C0C2CA" />
      </ButtonBase>
    );
  });

  return (
    <Box sx={{ maxWidth: '820px' }}>
      <Box sx={{ display: 'flex', gap: '8px', mb: '16px', flexWrap: 'wrap', alignItems: 'center' }}>
        <Box sx={{ fontSize: '13px', fontWeight: 600, color: '#6A6D76' }}>
          {t('members.count', { n: state.members.length })}
        </Box>
        <Box sx={{ flex: 1 }} />
        <ButtonBase
          onClick={() => app.openRoles()}
          sx={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '7px',
            p: '8px 14px',
            borderRadius: '999px',
            border: '1px solid #D0D2DA',
            background: '#fff',
            fontSize: '13px',
            fontWeight: 600,
            color: '#44474E',
          }}
        >
          <Sym name="admin_panel_settings" size={18} color="#44474E" />
          {t('members.rolesAndRights')}
        </ButtonBase>
      </Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '9px' }}>{rows}</Box>
    </Box>
  );
}
