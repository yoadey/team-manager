import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import type { SheetProps } from '@/sheets/types';
import type { Route } from '@/context/AppContext';
import { ROUTE_MODULE } from '@/context/urlState';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import { t } from '@/i18n';

export function TeamsSheet({ app }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const S = state;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
      {S.teams.map((tm) => {
        const active = tm.id === S.activeTeamId;
        return (
          <ButtonBase
            key={tm.id}
            onClick={() => app.selectTeam(tm.id)}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '13px',
              width: '100%',
              p: '12px 14px',
              borderRadius: '16px',
              cursor: 'pointer',
              border: '1px solid ' + (active ? 'transparent' : '#E6E7EE'),
              background: active ? tk.primaryContainer : NEUTRAL.card,
              justifyContent: 'flex-start',
              textAlign: 'left',
            }}
          >
            <Box
              component="span"
              aria-hidden="true"
              sx={{
                width: '46px',
                height: '46px',
                borderRadius: '13px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '22px',
                flex: '0 0 auto',
                overflow: 'hidden',
                ...(tm.logo
                  ? { backgroundImage: `url(${tm.logo})`, backgroundSize: 'cover', backgroundPosition: 'center' }
                  : { background: tm.iconBg, color: tm.iconFg }),
              }}
            >
              {tm.logo ? '' : tm.icon}
            </Box>
            <Box component="span" sx={{ flex: 1, minWidth: 0, textAlign: 'left' }}>
              <Box
                component="span"
                sx={{
                  display: 'block',
                  fontSize: '15px',
                  fontWeight: 600,
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                }}
              >
                {tm.name}
              </Box>
              <Box component="span" sx={{ display: 'block', fontSize: '12px', color: NEUTRAL.secondary }}>
                {[
                  tm.myRoles.map((r) => r.name).join(', '),
                  t('team.membersCount', { n: tm.memberCount, count: tm.memberCount }),
                ]
                  .filter(Boolean)
                  .join(' · ')}
              </Box>
            </Box>
            {active ? <Sym name="check_circle" size={24} color={tk.primary} /> : null}
          </ButtonBase>
        );
      })}
      <ButtonBase
        key="add"
        onClick={() => app.openCreateTeam()}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          p: '14px',
          borderRadius: '16px',
          border: `1.5px dashed ${NEUTRAL.inputBorder}`,
          background: 'transparent',
          cursor: 'pointer',
          color: tk.primary,
          fontWeight: 600,
          fontSize: '14px',
          justifyContent: 'flex-start',
          textAlign: 'left',
        }}
      >
        <Sym name="add_circle" size={24} color={tk.primary} />
        {t('team.newTeam')}
      </ButtonBase>
    </Box>
  );
}

export function MoreSheet({ app }: SheetProps) {
  // Derived from the shared ROUTE_MODULE map (same one RouteScreen's content
  // gate and AppShell's rail/bottom nav use) so a restricted role can't reach
  // a route from here that it can't actually see -- previously only
  // 'finances' checked app.can(), so a role with e.g. news:none still saw and
  // could tap a "News" entry that bounced it straight back to Home with a
  // spurious forbidden toast.
  const canSee = (route: Route) => {
    const module = ROUTE_MODULE[route];
    return !module || app.can(module, 'read');
  };
  const items: Array<[Route, string, string, boolean]> = [
    ['finances', t('nav.finances'), 'payments', canSee('finances')],
    ['stats', t('nav.stats'), 'insights', canSee('stats')],
    ['news', t('nav.news'), 'campaign', canSee('news')],
    ['polls', t('nav.polls'), 'how_to_vote', canSee('polls')],
    ['team', t('nav.team'), 'shield', canSee('team')],
    ['settings', t('nav.settings'), 'settings', canSee('settings')],
  ];
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '9px' }}>
      {items
        .filter((i) => i[3])
        .map((i) => (
          <ButtonBase
            key={i[0]}
            onClick={() => app.go(i[0])}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '13px',
              width: '100%',
              p: '14px',
              borderRadius: '14px',
              border: `1px solid ${NEUTRAL.line}`,
              background: NEUTRAL.card,
              cursor: 'pointer',
              justifyContent: 'flex-start',
              textAlign: 'left',
            }}
          >
            <Box
              component="span"
              key="i"
              sx={{
                width: '40px',
                height: '40px',
                borderRadius: '11px',
                background: NEUTRAL.line2,
                color: NEUTRAL.onSurfaceVariant,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flex: '0 0 auto',
              }}
            >
              <Sym name={i[2]} size={21} color={NEUTRAL.onSurfaceVariant} />
            </Box>
            <Box component="span" key="l" sx={{ flex: 1, textAlign: 'left', fontSize: '15px', fontWeight: 600 }}>
              {i[1]}
            </Box>
            <Sym name="chevron_right" size={22} color={NEUTRAL.faint} />
          </ButtonBase>
        ))}
    </Box>
  );
}
