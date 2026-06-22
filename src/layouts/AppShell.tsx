import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp, type Route } from '@/context/AppContext';
import { buildTokens, initials, NEUTRAL } from '@/styles/tokens';
import { todayLocalDate } from '@/utils/date';
import { Sym } from '@/components/ui';
import { RouteScreen } from '@/pages';
import { renderSheet } from '@/sheets';
import { useCompact, shortName } from './useCompact';
import { t as tl } from '@/i18n';
import { pageMeta } from './pageMeta';
export { COMPACT_BP, useCompact, shortName } from './useCompact';

interface NavDef {
  key: string;
  label: string;
  icon: string;
  badge?: number;
  gate?: () => boolean;
}

export function Shell() {
  const app = useApp();
  const { state } = app;
  const compact = useCompact();
  const t = buildTokens(state.primaryColor);
  const team = app.activeTeam();
  if (!team || !state.user) return null;

  const today = todayLocalDate();
  const pending = state.events.filter(
    (e) => e.date >= today && e.myStatus === 'pending' && e.status !== 'cancelled',
  ).length;

  const pageSheet = app.activePageSheet();
  const pm = pageMeta(app);

  // ---- shared chrome bits ----
  const teamIcon = (
    <Box
      component="span"
      sx={{
        width: 40,
        height: 40,
        borderRadius: '12px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: '20px',
        flex: '0 0 auto',
        overflow: 'hidden',
        ...(team.logo
          ? { backgroundImage: `url(${team.logo})`, backgroundSize: 'cover', backgroundPosition: 'center' }
          : { background: team.iconBg, color: team.iconFg }),
      }}
    >
      {team.logo ? '' : team.icon}
    </Box>
  );
  const myAvatar = (
    <Box
      component="span"
      sx={{
        width: 38,
        height: 38,
        borderRadius: '50%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: '13px',
        fontWeight: 700,
        flex: '0 0 auto',
        overflow: 'hidden',
        color: '#fff',
        ...(state.user.photo
          ? { backgroundImage: `url(${state.user.photo})`, backgroundSize: 'cover', backgroundPosition: 'center' }
          : { background: state.user.avatarColor }),
      }}
    >
      {state.user.photo ? '' : initials(state.user.name)}
    </Box>
  );
  const notifBadge = state.notifUnread > 9 ? '9+' : String(state.notifUnread);
  const hasUnread = state.notifUnread > 0;

  const content = pageSheet ? <Box sx={{ maxWidth: '860px' }}>{renderSheet(app, pageSheet)}</Box> : <RouteScreen />;

  const railDefs: NavDef[] = [
    { key: 'home', label: tl('nav.home'), icon: 'home' },
    { key: 'events', label: tl('nav.events'), icon: 'event', badge: pending },
    { key: 'members', label: tl('nav.members'), icon: 'group' },
    { key: 'finances', label: tl('nav.finances'), icon: 'payments', gate: () => app.can('finances', 'read') },
    { key: 'stats', label: tl('nav.stats'), icon: 'insights' },
    { key: 'news', label: tl('nav.news'), icon: 'campaign' },
    { key: 'polls', label: tl('nav.polls'), icon: 'how_to_vote' },
    { key: 'team', label: tl('nav.team'), icon: 'shield' },
  ];
  const bottomDefs: NavDef[] = [
    { key: 'home', label: tl('nav.home'), icon: 'home' },
    { key: 'events', label: tl('nav.events'), icon: 'event', badge: pending },
    { key: 'members', label: tl('nav.members'), icon: 'group' },
    { key: '__more', label: tl('nav.more'), icon: 'apps' },
  ];

  const skipLinkSx = {
    position: 'absolute' as const,
    left: '-9999px',
    zIndex: 9999,
    '&:focus': {
      position: 'fixed' as const,
      left: '8px',
      top: '8px',
      background: NEUTRAL.card,
      color: NEUTRAL.onSurface,
      padding: '10px 16px',
      borderRadius: '8px',
      fontSize: '14px',
      fontWeight: 600,
      border: `2px solid ${NEUTRAL.onSurface}`,
      textDecoration: 'none',
    },
  };

  // ===================== MOBILE =====================
  if (compact) {
    return (
      <Box
        sx={{ display: 'flex', flexDirection: 'column', height: '100vh', minHeight: 0, background: NEUTRAL.surface }}
      >
        <Box component="a" href="#main-content" sx={skipLinkSx}>
          {tl('shell.skipToMain')}
        </Box>
        <Box
          sx={{
            flex: '0 0 auto',
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            p: '12px 14px',
            background: t.primaryContainer,
            color: t.onPrimaryContainer,
          }}
        >
          {pageSheet ? (
            <ButtonBase
              onClick={app.closeSheet}
              aria-label={tl('shell.back')}
              sx={{
                width: 38,
                height: 38,
                borderRadius: '50%',
                background: 'rgba(255,255,255,.28)',
                color: 'inherit',
                flex: '0 0 auto',
              }}
            >
              <Sym name="arrow_back" size={22} />
            </ButtonBase>
          ) : null}
          <ButtonBase
            onClick={app.openTeamSwitcher}
            aria-label={tl('shell.openTeamSwitcher', { name: shortName(team.name) })}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '10px',
              flex: 1,
              minWidth: 0,
              textAlign: 'left',
              color: 'inherit',
              justifyContent: 'flex-start',
            }}
          >
            {teamIcon}
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Box
                sx={{
                  fontSize: '14px',
                  fontWeight: 700,
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                }}
              >
                {shortName(team.name)}
              </Box>
              <Box
                sx={{
                  fontSize: '11px',
                  opacity: 0.8,
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                }}
              >
                {pageSheet ? pm.title : pm.title}
              </Box>
            </Box>
            <Sym name="unfold_more" size={20} sx={{ opacity: 0.8 }} />
          </ButtonBase>
          <ButtonBase
            onClick={app.openNotifications}
            aria-label={
              hasUnread ? tl('shell.unreadNotifications', { n: state.notifUnread }) : tl('shell.openNotifications')
            }
            sx={{
              position: 'relative',
              width: 38,
              height: 38,
              borderRadius: '50%',
              background: 'rgba(255,255,255,.28)',
              color: 'inherit',
              flex: '0 0 auto',
            }}
          >
            <Sym name="notifications" size={21} />
            {hasUnread ? (
              <Box
                aria-hidden="true"
                sx={{
                  position: 'absolute',
                  top: -3,
                  right: -3,
                  minWidth: 17,
                  height: 17,
                  borderRadius: '9px',
                  background: t.primary,
                  color: t.onPrimary,
                  fontSize: '10px',
                  fontWeight: 700,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  px: '4px',
                  border: `2px solid ${t.primaryContainer}`,
                }}
              >
                {notifBadge}
              </Box>
            ) : null}
          </ButtonBase>
          <ButtonBase
            onClick={app.openProfile}
            aria-label={`${state.user.name} – ${tl('shell.openProfile')}`}
            sx={{ borderRadius: '50%' }}
          >
            {myAvatar}
          </ButtonBase>
        </Box>

        <Box
          component="main"
          id="main-content"
          sx={{ flex: 1, minHeight: 0, overflow: 'auto', p: '14px 14px 90px', position: 'relative' }}
        >
          {content}
        </Box>

        {!pageSheet && pm.showPrimaryAction ? (
          <ButtonBase
            onClick={pm.primaryAction}
            sx={{
              position: 'fixed',
              right: 18,
              bottom: 88,
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              background: t.primary,
              color: t.onPrimary,
              borderRadius: '18px',
              height: 56,
              px: '20px',
              fontSize: '15px',
              fontWeight: 600,
              boxShadow: '0 8px 22px rgba(21,101,192,.4)',
              zIndex: 3,
            }}
          >
            <Sym name={pm.primaryActionIcon} size={24} color={t.onPrimary} />
            {pm.primaryActionLabel}
          </ButtonBase>
        ) : null}

        <Box
          component="nav"
          aria-label={tl('nav.mainNav')}
          sx={{
            flex: '0 0 auto',
            height: 72,
            background: NEUTRAL.sidebar,
            borderTop: `1px solid ${NEUTRAL.line3}`,
            display: 'flex',
            alignItems: 'stretch',
            p: '8px 6px',
          }}
        >
          {bottomDefs.map((n) => {
            const isMore = n.key === '__more';
            const active = !isMore && state.route === n.key;
            const badge = n.badge || 0;
            return (
              <ButtonBase
                key={n.key}
                onClick={() => (isMore ? app.openMore() : app.go(n.key as Route))}
                aria-current={active ? 'page' : undefined}
                sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '4px', p: '4px 0' }}
              >
                <Box
                  component="span"
                  sx={{
                    position: 'relative',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: 58,
                    height: 30,
                    borderRadius: '16px',
                    background: active ? t.secondaryContainer : 'transparent',
                    color: active ? t.onSecondaryContainer : NEUTRAL.onSurfaceVariant,
                  }}
                >
                  <Sym name={n.icon} size={24} />
                  {badge > 0 ? (
                    <Box
                      sx={{
                        position: 'absolute',
                        top: -2,
                        right: 8,
                        minWidth: 18,
                        height: 18,
                        borderRadius: '10px',
                        background: t.primary,
                        color: t.onPrimary,
                        fontSize: '10px',
                        fontWeight: 700,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        px: '5px',
                      }}
                    >
                      {badge}
                    </Box>
                  ) : null}
                </Box>
                <Box
                  component="span"
                  sx={{ fontSize: '11px', fontWeight: 600, color: active ? NEUTRAL.onSurface : NEUTRAL.secondary }}
                >
                  {n.label}
                </Box>
              </ButtonBase>
            );
          })}
        </Box>
      </Box>
    );
  }

  // ===================== DESKTOP =====================
  return (
    <Box sx={{ display: 'flex', height: '100vh', minHeight: 0, background: NEUTRAL.surface }}>
      <Box component="a" href="#main-content" sx={skipLinkSx}>
        {tl('shell.skipToMain')}
      </Box>
      <Box
        sx={{
          flex: '0 0 268px',
          background: NEUTRAL.sidebar,
          borderRight: `1px solid ${NEUTRAL.line3}`,
          display: 'flex',
          flexDirection: 'column',
          minHeight: 0,
        }}
      >
        <ButtonBase
          onClick={app.openTeamSwitcher}
          aria-label={tl('shell.openTeamSwitcher', { name: shortName(team.name) })}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
            p: '16px',
            borderBottom: `1px solid ${NEUTRAL.line}`,
            textAlign: 'left',
            width: '100%',
            justifyContent: 'flex-start',
          }}
        >
          {teamIcon}
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box
              sx={{
                fontSize: '14px',
                fontWeight: 600,
                color: NEUTRAL.onSurface,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {shortName(team.name)}
            </Box>
            <Box
              sx={{
                fontSize: '12px',
                color: NEUTRAL.secondary,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {tl('shell.memberCount', { n: team.memberCount, count: team.memberCount })}
            </Box>
          </Box>
          <Sym name="unfold_more" size={22} color={NEUTRAL.secondary} />
        </ButtonBase>

        <Box
          component="nav"
          aria-label={tl('nav.mainNav')}
          sx={{
            flex: 1,
            minHeight: 0,
            overflow: 'auto',
            p: '12px',
            display: 'flex',
            flexDirection: 'column',
            gap: '3px',
          }}
        >
          {railDefs
            .filter((d) => !d.gate || d.gate())
            .map((n) => {
              const active = state.route === n.key;
              const badge = n.badge || 0;
              return (
                <ButtonBase
                  key={n.key}
                  onClick={() => app.go(n.key as Route)}
                  aria-current={active ? 'page' : undefined}
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '14px',
                    p: '11px 14px',
                    borderRadius: '13px',
                    width: '100%',
                    justifyContent: 'flex-start',
                    background: active ? t.secondaryContainer : 'transparent',
                    color: active ? t.onSecondaryContainer : NEUTRAL.onSurfaceVariant,
                    fontWeight: active ? 700 : 500,
                  }}
                >
                  <Sym name={n.icon} size={22} />
                  <Box component="span" sx={{ fontSize: '14px', fontWeight: 'inherit', flex: 1, textAlign: 'left' }}>
                    {n.label}
                  </Box>
                  {badge > 0 ? (
                    <Box
                      component="span"
                      sx={{
                        minWidth: 18,
                        height: 18,
                        borderRadius: '10px',
                        background: t.primary,
                        color: t.onPrimary,
                        fontSize: '10px',
                        fontWeight: 700,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        px: '5px',
                      }}
                    >
                      {badge}
                    </Box>
                  ) : null}
                </ButtonBase>
              );
            })}
        </Box>

        <ButtonBase
          onClick={app.openProfile}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
            m: '12px',
            p: '10px 12px',
            borderRadius: '16px',
            background: NEUTRAL.card,
            border: `1px solid ${NEUTRAL.line}`,
            textAlign: 'left',
            justifyContent: 'flex-start',
          }}
        >
          {myAvatar}
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box
              sx={{
                fontSize: '13px',
                fontWeight: 600,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {state.user.name}
            </Box>
            <Box sx={{ fontSize: '11px', color: NEUTRAL.secondary }}>{tl('shell.accountAndRoles')}</Box>
          </Box>
          <Sym name="settings" size={20} color={NEUTRAL.secondary} />
        </ButtonBase>
      </Box>

      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          minHeight: 0,
          background: NEUTRAL.surface,
        }}
      >
        <Box
          sx={{
            flex: '0 0 auto',
            display: 'flex',
            alignItems: 'center',
            gap: '14px',
            p: '18px 28px',
            borderBottom: `1px solid ${NEUTRAL.line2}`,
          }}
        >
          {pageSheet ? (
            <ButtonBase
              onClick={app.closeSheet}
              aria-label={tl('shell.back')}
              sx={{
                width: 40,
                height: 40,
                borderRadius: '50%',
                border: `1px solid ${NEUTRAL.line3}`,
                background: NEUTRAL.card,
                color: NEUTRAL.onSurfaceVariant,
                flex: '0 0 auto',
              }}
            >
              <Sym name="arrow_back" size={22} />
            </ButtonBase>
          ) : null}
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box sx={{ fontSize: '22px', fontWeight: 700, letterSpacing: '-.2px' }}>{pm.title}</Box>
            <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary }}>{pm.subtitle}</Box>
          </Box>
          <ButtonBase
            onClick={app.openNotifications}
            aria-label={
              hasUnread ? tl('shell.unreadNotifications', { n: state.notifUnread }) : tl('shell.openNotifications')
            }
            sx={{
              position: 'relative',
              width: 44,
              height: 44,
              borderRadius: '50%',
              border: `1px solid ${NEUTRAL.line3}`,
              background: NEUTRAL.card,
              color: NEUTRAL.onSurfaceVariant,
              flex: '0 0 auto',
            }}
          >
            <Sym name="notifications" size={23} />
            {hasUnread ? (
              <Box
                aria-hidden="true"
                sx={{
                  position: 'absolute',
                  top: -2,
                  right: -2,
                  minWidth: 18,
                  height: 18,
                  borderRadius: '10px',
                  background: t.primary,
                  color: t.onPrimary,
                  fontSize: '10px',
                  fontWeight: 700,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  px: '5px',
                  border: `2px solid ${NEUTRAL.card}`,
                }}
              >
                {notifBadge}
              </Box>
            ) : null}
          </ButtonBase>
          {!pageSheet && pm.showPrimaryAction ? (
            <ButtonBase
              onClick={pm.primaryAction}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                background: t.primary,
                color: t.onPrimary,
                borderRadius: '14px',
                p: '11px 18px',
                fontSize: '14px',
                fontWeight: 600,
                boxShadow: '0 4px 14px rgba(21,101,192,.28)',
              }}
            >
              <Sym name={pm.primaryActionIcon} size={20} color={t.onPrimary} />
              {pm.primaryActionLabel}
            </ButtonBase>
          ) : null}
        </Box>
        <Box component="main" id="main-content" sx={{ flex: 1, minHeight: 0, overflow: 'auto', p: '24px 28px 56px' }}>
          {content}
        </Box>
      </Box>
    </Box>
  );
}
