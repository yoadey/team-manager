import { useSyncExternalStore, type ReactNode } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp, sheetErrorBoundaryKey, type AppContextValue, type Route, type SheetState } from '@/context/AppContext';
import { ROUTE_MODULE } from '@/context/urlState';
import { buildTokens, initials, NEUTRAL } from '@/styles/tokens';
import { todayLocalDate } from '@/utils/date';
import { Sym } from '@/components/ui';
import { useEventsQuery, useEventDetailQuery } from '@/features/events';
import { useMembersQuery } from '@/features/members';
import { useNotificationsQuery } from '@/features/notifications';
import { RouteScreen } from '@/pages';
import { renderSheet } from '@/sheets';
import { useCompact } from './useCompact';
import { t as tl, getLocale, subscribeLocale } from '@/i18n';
import { pageMeta, type PM } from './pageMeta';
import type { TeamForUser } from '@/types';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import { captureError } from '@/monitoring';
export { COMPACT_BP, useCompact, shortName } from './useCompact';

type Tokens = ReturnType<typeof buildTokens>;

interface NavDef {
  key: string;
  label: string;
  icon: string;
  badge?: number;
  gate: () => boolean;
}

// Subscribes to the module-level i18n store directly (see the identical
// helper in components/cards.tsx) so Shell re-renders on a locale switch.
// pageMeta()/tl() read module-level i18n state, not AppContext, so without
// this the page title/subtitle and nav labels stayed in the old language
// until some UNRELATED AppContext change (navigation, a toast) happened to
// force a re-render.
function useLocaleSubscription(): void {
  useSyncExternalStore(subscribeLocale, getLocale);
}

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

function TeamIcon({ team }: { team: TeamForUser }) {
  return (
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
}

function MyAvatar({ user }: { user: AppContextValue['state']['user'] }) {
  if (!user) return null;
  return (
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
        ...(user.photo
          ? { backgroundImage: `url(${user.photo})`, backgroundSize: 'cover', backgroundPosition: 'center' }
          : { background: user.avatarColor }),
      }}
    >
      {user.photo ? '' : initials(user.name)}
    </Box>
  );
}

function NotificationsButton({
  compact,
  tk,
  hasUnread,
  notifUnread,
  notifBadge,
  onOpen,
}: {
  compact: boolean;
  tk: Tokens;
  hasUnread: boolean;
  notifUnread: number;
  notifBadge: string;
  onOpen: () => void;
}) {
  const size = compact ? 38 : 44;
  return (
    <ButtonBase
      onClick={onOpen}
      aria-label={
        hasUnread ? tl('shell.unreadNotifications', { n: notifUnread, count: notifUnread }) : tl('shell.openNotifications')
      }
      sx={{
        position: 'relative',
        width: size,
        height: size,
        borderRadius: '50%',
        border: compact ? 'none' : `1px solid ${NEUTRAL.line3}`,
        background: compact ? 'rgba(255,255,255,.28)' : NEUTRAL.card,
        color: compact ? 'inherit' : NEUTRAL.onSurfaceVariant,
        flex: '0 0 auto',
      }}
    >
      <Sym name="notifications" size={compact ? 21 : 23} />
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
            background: tk.primary,
            color: tk.onPrimary,
            fontSize: '10px',
            fontWeight: 700,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            px: '4px',
            border: `2px solid ${compact ? tk.primaryContainer : NEUTRAL.card}`,
          }}
        >
          {notifBadge}
        </Box>
      ) : null}
    </ButtonBase>
  );
}

function BackButton({ compact, onClick }: { compact: boolean; onClick: () => void }) {
  return (
    <ButtonBase
      onClick={onClick}
      aria-label={tl('shell.back')}
      sx={{
        width: compact ? 38 : 40,
        height: compact ? 38 : 40,
        borderRadius: '50%',
        border: compact ? undefined : `1px solid ${NEUTRAL.line3}`,
        background: compact ? 'rgba(255,255,255,.28)' : NEUTRAL.card,
        color: compact ? 'inherit' : NEUTRAL.onSurfaceVariant,
        flex: '0 0 auto',
      }}
    >
      <Sym name="arrow_back" size={22} />
    </ButtonBase>
  );
}

function PrimaryActionButton({ compact, tk, pm }: { compact: boolean; tk: Tokens; pm: PM }) {
  return (
    <ButtonBase
      onClick={pm.primaryAction}
      sx={
        compact
          ? {
              position: 'fixed',
              right: 18,
              bottom: 88,
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              background: tk.primary,
              color: tk.onPrimary,
              borderRadius: '18px',
              height: 56,
              px: '20px',
              fontSize: '15px',
              fontWeight: 600,
              boxShadow: '0 8px 22px rgba(21,101,192,.4)',
              zIndex: 3,
            }
          : {
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              background: tk.primary,
              color: tk.onPrimary,
              borderRadius: '14px',
              p: '11px 18px',
              fontSize: '14px',
              fontWeight: 600,
              boxShadow: '0 4px 14px rgba(21,101,192,.28)',
            }
      }
    >
      <Sym name={pm.primaryActionIcon} size={compact ? 24 : 20} color={tk.onPrimary} />
      {pm.primaryActionLabel}
    </ButtonBase>
  );
}

function MobileNavBar({ defs, route, onNavigate, onMore, tk }: {
  defs: NavDef[];
  route: string;
  onNavigate: (route: Route) => void;
  onMore: () => void;
  tk: Tokens;
}) {
  return (
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
      {defs
        .filter((d) => d.gate())
        .map((n) => {
          const isMore = n.key === '__more';
          const active = !isMore && route === n.key;
          const badge = n.badge || 0;
          return (
            <ButtonBase
              key={n.key}
              onClick={() => (isMore ? onMore() : onNavigate(n.key as Route))}
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
                  background: active ? tk.secondaryContainer : 'transparent',
                  color: active ? tk.onSecondaryContainer : NEUTRAL.onSurfaceVariant,
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
                      background: tk.primary,
                      color: tk.onPrimary,
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
              <Box component="span" sx={{ fontSize: '11px', fontWeight: 600, color: active ? NEUTRAL.onSurface : NEUTRAL.secondary }}>
                {n.label}
              </Box>
            </ButtonBase>
          );
        })}
    </Box>
  );
}

function DesktopNavRail({ defs, route, onNavigate, tk }: {
  defs: NavDef[];
  route: string;
  onNavigate: (route: Route) => void;
  tk: Tokens;
}) {
  return (
    <Box
      component="nav"
      aria-label={tl('nav.mainNav')}
      sx={{ flex: 1, minHeight: 0, overflow: 'auto', p: '12px', display: 'flex', flexDirection: 'column', gap: '3px' }}
    >
      {defs
        .filter((d) => d.gate())
        .map((n) => {
          const active = route === n.key;
          const badge = n.badge || 0;
          return (
            <ButtonBase
              key={n.key}
              onClick={() => onNavigate(n.key as Route)}
              aria-current={active ? 'page' : undefined}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '14px',
                p: '11px 14px',
                borderRadius: '13px',
                width: '100%',
                justifyContent: 'flex-start',
                background: active ? tk.secondaryContainer : 'transparent',
                color: active ? tk.onSecondaryContainer : NEUTRAL.onSurfaceVariant,
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
                    background: tk.primary,
                    color: tk.onPrimary,
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
  );
}

interface ShellChromeProps {
  app: AppContextValue;
  team: TeamForUser;
  tk: Tokens;
  pageSheet: SheetState | null;
  pm: PM;
  content: ReactNode;
  notifUnread: number;
  notifBadge: string;
  hasUnread: boolean;
  navDefs: NavDef[];
}

function MobileShell({ app, team, tk, pageSheet, pm, content, notifUnread, notifBadge, hasUnread, navDefs }: ShellChromeProps) {
  const { state } = app;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', minHeight: 0, background: NEUTRAL.surface }}>
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
          background: tk.primaryContainer,
          color: tk.onPrimaryContainer,
        }}
      >
        {pageSheet ? <BackButton compact onClick={app.closeSheet} /> : null}
        <ButtonBase
          onClick={app.openTeamSwitcher}
          aria-label={tl('shell.openTeamSwitcher', { name: team.name })}
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
          <TeamIcon team={team} />
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box sx={{ fontSize: '14px', fontWeight: 700, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {team.name}
            </Box>
            <Box sx={{ fontSize: '11px', opacity: 0.8, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {pm.title}
            </Box>
          </Box>
          <Sym name="unfold_more" size={20} sx={{ opacity: 0.8 }} />
        </ButtonBase>
        <NotificationsButton
          compact
          tk={tk}
          hasUnread={hasUnread}
          notifUnread={notifUnread}
          notifBadge={notifBadge}
          onOpen={app.openNotifications}
        />
        <ButtonBase onClick={app.openProfile} aria-label={`${state.user!.name} – ${tl('shell.openProfile')}`} sx={{ borderRadius: '50%' }}>
          <MyAvatar user={state.user} />
        </ButtonBase>
      </Box>

      <Box component="main" id="main-content" sx={{ flex: 1, minHeight: 0, overflow: 'auto', p: '14px 14px 90px', position: 'relative' }}>
        {content}
      </Box>

      {!pageSheet && pm.showPrimaryAction ? <PrimaryActionButton compact tk={tk} pm={pm} /> : null}

      <MobileNavBar defs={navDefs} route={state.route} onNavigate={app.go} onMore={app.openMore} tk={tk} />
    </Box>
  );
}

function DesktopShell({ app, team, tk, pageSheet, pm, content, notifUnread, notifBadge, hasUnread, navDefs }: ShellChromeProps) {
  const { state } = app;
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
          aria-label={tl('shell.openTeamSwitcher', { name: team.name })}
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
          <TeamIcon team={team} />
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
              {team.name}
            </Box>
            <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {tl('shell.memberCount', { n: team.memberCount, count: team.memberCount })}
            </Box>
          </Box>
          <Sym name="unfold_more" size={22} color={NEUTRAL.secondary} />
        </ButtonBase>

        <DesktopNavRail defs={navDefs} route={state.route} onNavigate={app.go} tk={tk} />

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
          <MyAvatar user={state.user} />
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box sx={{ fontSize: '13px', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {state.user!.name}
            </Box>
            <Box sx={{ fontSize: '11px', color: NEUTRAL.secondary }}>{tl('shell.accountAndRoles')}</Box>
          </Box>
          <Sym name="settings" size={20} color={NEUTRAL.secondary} />
        </ButtonBase>
      </Box>

      <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', minHeight: 0, background: NEUTRAL.surface }}>
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
          {pageSheet ? <BackButton onClick={app.closeSheet} compact={false} /> : null}
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box sx={{ fontSize: '22px', fontWeight: 700, letterSpacing: '-.2px' }}>{pm.title}</Box>
            <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary }}>{pm.subtitle}</Box>
          </Box>
          <NotificationsButton
            compact={false}
            tk={tk}
            hasUnread={hasUnread}
            notifUnread={notifUnread}
            notifBadge={notifBadge}
            onOpen={app.openNotifications}
          />
          {!pageSheet && pm.showPrimaryAction ? <PrimaryActionButton compact={false} tk={tk} pm={pm} /> : null}
        </Box>
        <Box component="main" id="main-content" sx={{ flex: 1, minHeight: 0, overflow: 'auto', p: '24px 28px 56px' }}>
          {content}
        </Box>
      </Box>
    </Box>
  );
}

export function Shell() {
  useLocaleSubscription();
  const app = useApp();
  const { state } = app;
  const compact = useCompact();
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam();
  const pageSheet = app.activePageSheet();
  // Hooks must run unconditionally on every render (before the `!team` early
  // return below), so the events queries are called here regardless of
  // whether a team/user is present yet -- `enabled` on each query gates the
  // actual fetch.
  const { data: events } = useEventsQuery(app.api, state.activeTeamId);
  const detailEventId = pageSheet?.type === 'eventDetail' ? (pageSheet.eventId ?? null) : null;
  const { data: detailData } = useEventDetailQuery(app.api, state.activeTeamId, detailEventId);
  const { data: members } = useMembersQuery(app.api, state.activeTeamId);
  const { data: notifData } = useNotificationsQuery(app.api, state.activeTeamId);
  if (!team || !state.user) return null;

  const today = todayLocalDate();
  const pending = (events ?? []).filter(
    (e) => e.date >= today && e.myStatus === 'pending' && e.status !== 'cancelled',
  ).length;

  const pm = pageMeta(app, detailData?.event, members?.length);
  const notifUnread = notifData?.unreadCount ?? 0;
  const notifBadge = notifUnread > 9 ? '9+' : String(notifUnread);
  const hasUnread = notifUnread > 0;

  const content = pageSheet ? (
    <Box sx={{ maxWidth: '860px' }}>
      <ErrorBoundary key={sheetErrorBoundaryKey(pageSheet)} onError={captureError}>
        {renderSheet(app, pageSheet)}
      </ErrorBoundary>
    </Box>
  ) : (
    <RouteScreen />
  );

  // Derives each nav entry's gate from the shared ROUTE_MODULE map (same one
  // RouteScreen's per-route content gate uses) rather than hand-rolling
  // per-entry app.can() checks, so nav visibility and page content can't
  // drift apart the way they previously did (only 'finances' was gated here).
  // Always returns a real predicate (never undefined) -- a route with no
  // module mapping is simply always visible -- so NavDef.gate can stay a
  // required `() => boolean` instead of an optional field callers must
  // null-check.
  const navGate = (route: Route): (() => boolean) => {
    const module = ROUTE_MODULE[route];
    return module ? () => app.can(module, 'read') : () => true;
  };

  const railDefs: NavDef[] = [
    { key: 'home', label: tl('nav.home'), icon: 'home', gate: navGate('home') },
    { key: 'events', label: tl('nav.events'), icon: 'event', badge: pending, gate: navGate('events') },
    { key: 'members', label: tl('nav.members'), icon: 'group', gate: navGate('members') },
    { key: 'finances', label: tl('nav.finances'), icon: 'payments', gate: navGate('finances') },
    { key: 'stats', label: tl('nav.stats'), icon: 'insights', gate: navGate('stats') },
    { key: 'news', label: tl('nav.news'), icon: 'campaign', gate: navGate('news') },
    { key: 'polls', label: tl('nav.polls'), icon: 'how_to_vote', gate: navGate('polls') },
    { key: 'team', label: tl('nav.team'), icon: 'shield', gate: navGate('team') },
  ];
  const bottomDefs: NavDef[] = [
    { key: 'home', label: tl('nav.home'), icon: 'home', gate: navGate('home') },
    { key: 'events', label: tl('nav.events'), icon: 'event', badge: pending, gate: navGate('events') },
    { key: 'members', label: tl('nav.members'), icon: 'group', gate: navGate('members') },
    { key: '__more', label: tl('nav.more'), icon: 'apps', gate: () => true },
  ];

  const chromeProps: ShellChromeProps = {
    app,
    team,
    tk,
    pageSheet,
    pm,
    content,
    notifUnread,
    notifBadge,
    hasUnread,
    navDefs: compact ? bottomDefs : railDefs,
  };

  return compact ? <MobileShell {...chromeProps} /> : <DesktopShell {...chromeProps} />;
}
