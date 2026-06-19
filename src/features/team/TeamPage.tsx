import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { SectionTitle, Sym } from '@/components/ui';
import { t } from '@/i18n';

export function TeamPage() {
  const app = useApp();
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;

  const card = (icon: string, title: string, sub: string, onClick: () => void, accent?: boolean) => (
    <ButtonBase
      key={title}
      onClick={onClick}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '14px',
        width: '100%',
        textAlign: 'left',
        justifyContent: 'flex-start',
        background: '#fff',
        border: `1px solid ${NEUTRAL.line}`,
        borderRadius: '16px',
        p: '15px 16px',
      }}
    >
      <Box
        component="span"
        sx={{
          width: '44px',
          height: '44px',
          borderRadius: '12px',
          background: accent ? tk.primaryContainer : '#ECEDF3',
          color: accent ? tk.primary : '#44474E',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flex: '0 0 auto',
        }}
      >
        <Sym name={icon} size={22} color={accent ? tk.primary : '#44474E'} />
      </Box>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontSize: '15px', fontWeight: 600 }}>{title}</Box>
        <Box sx={{ fontSize: '12px', color: '#6A6D76', mt: '2px' }}>{sub}</Box>
      </Box>
      <Sym name="chevron_right" size={22} color="#C0C2CA" />
    </ButtonBase>
  );

  const header = (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '16px',
        borderRadius: '20px',
        p: '20px',
        mb: '18px',
        color: team.photo ? '#fff' : '#1A1C20',
        ...(team.photo
          ? {
              backgroundImage: `linear-gradient(90deg, rgba(10,12,20,.72), rgba(10,12,20,.3)), url(${team.photo})`,
              backgroundSize: 'cover',
              backgroundPosition: 'center',
            }
          : { background: '#fff', border: `1px solid ${NEUTRAL.line}` }),
      }}
    >
      <Box
        component="span"
        sx={{
          width: '60px',
          height: '60px',
          borderRadius: '18px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '30px',
          flex: '0 0 auto',
          overflow: 'hidden',
          ...(team.logo
            ? { backgroundImage: `url(${team.logo})`, backgroundSize: 'cover', backgroundPosition: 'center' }
            : team.photo
              ? { background: 'rgba(255,255,255,.2)', color: '#fff' }
              : { background: team.iconBg, color: team.iconFg }),
        }}
      >
        {team.logo ? '' : team.icon}
      </Box>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontSize: '18px', fontWeight: 700, lineHeight: 1.2 }}>{team.name}</Box>
        <Box
          sx={{
            fontSize: '13px',
            color: team.photo ? 'rgba(255,255,255,.85)' : '#6A6D76',
            mt: '5px',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            flexWrap: 'wrap',
          }}
        >
          <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', gap: '5px', fontWeight: 600 }}>
            <Sym name="badge" size={16} color="inherit" />
            {app
              .myRoles()
              .map((r) => r.name)
              .join(', ')}
          </Box>
          <Box component="span">·</Box>
          {t('team.membersCount', { n: team.memberCount })}
        </Box>
      </Box>
    </Box>
  );

  const actions = (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', mb: '22px' }}>
      {app.can('settings', 'write')
        ? card('group_add', t('team.invite'), t('team.inviteDesc'), () => app.openInvite(), true)
        : null}
      {app.can('settings', 'write')
        ? card('tune', t('team.teamSettings'), t('team.teamSettingsDesc'), () => app.openTeamSettings(), false)
        : null}
      {card('admin_panel_settings', t('team.rolesAndRights'), t('team.rolesDesc'), () => app.openRoles(), false)}
      {card('groups', t('team.members'), t('team.membersDesc'), () => app.go('members'), false)}
    </Box>
  );

  const teamRows = (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', mb: '14px' }}>
      {state.teams.map((tm) => {
        const active = tm.id === state.activeTeamId;
        return (
          <ButtonBase
            key={tm.id}
            onClick={() => app.selectTeam(tm.id)}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '13px',
              width: '100%',
              textAlign: 'left',
              justifyContent: 'flex-start',
              background: active ? tk.primaryContainer : '#fff',
              border: `1px solid ${active ? 'transparent' : NEUTRAL.line}`,
              borderRadius: '16px',
              p: '13px 15px',
            }}
          >
            <Box
              component="span"
              sx={{
                width: '42px',
                height: '42px',
                borderRadius: '12px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '20px',
                flex: '0 0 auto',
                overflow: 'hidden',
                ...(tm.logo
                  ? { backgroundImage: `url(${tm.logo})`, backgroundSize: 'cover', backgroundPosition: 'center' }
                  : { background: tm.iconBg, color: tm.iconFg }),
              }}
            >
              {tm.logo ? '' : tm.icon}
            </Box>
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Box sx={{ fontSize: '14px', fontWeight: 600 }}>{tm.name}</Box>
              <Box sx={{ fontSize: '12px', color: active ? tk.onPrimaryContainer : '#6A6D76', mt: '2px' }}>
                {tm.myRoles.map((r) => r.name).join(', ') + ' · ' + t('team.membersCount', { n: tm.memberCount })}
              </Box>
            </Box>
            {active ? <Sym name="check_circle" size={22} color={tk.primary} /> : null}
          </ButtonBase>
        );
      })}
    </Box>
  );

  const createBtn = (
    <ButtonBase
      onClick={() => app.openCreateTeam()}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '10px',
        width: '100%',
        p: '14px',
        borderRadius: '16px',
        border: '1.5px dashed #C8CAD2',
        background: 'transparent',
        color: tk.primary,
        fontWeight: 600,
        fontSize: '14px',
      }}
    >
      <Sym name="add_circle" size={22} color={tk.primary} />
      {t('team.newTeam')}
    </ButtonBase>
  );

  return (
    <Box sx={{ maxWidth: '640px' }}>
      {header}
      {actions}
      <SectionTitle>{t('team.myTeams')}</SectionTitle>
      {teamRows}
      {createBtn}
    </Box>
  );
}
