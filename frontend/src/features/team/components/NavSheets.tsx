import { useState, type ChangeEvent } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import type { SheetProps } from '@/sheets/types';
import type { Route } from '@/context/AppContext';
import { ROUTE_MODULE } from '@/context/urlState';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Sym, Av } from '@/components/ui';
import { t, type Locale } from '@/i18n';
import { useLocale } from '@/i18n/LocaleProvider';
import { captureException } from '@/monitoring';
import { reportActionError, AuthError } from '@/utils/errors';
import { usePushActions } from '@/features/notifications';

/** Each language is shown in its own name (endonym), independent of UI locale. */
const LANGUAGE_LABELS: Record<Locale, string> = { de: 'Deutsch', en: 'English' };

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
                {[tm.myRoles.map((r) => r.name).join(', '), t('team.membersCount', { n: tm.memberCount, count: tm.memberCount })]
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

export function ProfileSheet({ app }: SheetProps) {
  const { state } = app;
  const { locale, setLocale, supported } = useLocale();
  const tk = buildTokens(state.primaryColor);
  const S = state;
  const push = usePushActions(app.api, app.toastMsg);

  // Account erasure (GDPR Art. 17): a destructive, irreversible action gated by
  // retyping the account email — no password, so the same flow also covers a
  // future OIDC-only account (no OIDC integration exists yet).
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [confirmEmail, setConfirmEmail] = useState('');
  const [deleting, setDeleting] = useState(false);
  const [deleteErr, setDeleteErr] = useState(false);
  const accountEmail = S.user?.email ?? '';
  const canConfirmDelete = confirmEmail.trim().toLowerCase() === accountEmail.toLowerCase() && accountEmail !== '';
  return (
    <Box>
      <Box key="hd" sx={{ display: 'flex', alignItems: 'center', gap: '14px', p: '4px 2px 18px' }}>
        <Box key="av" sx={{ position: 'relative' }}>
          <Av name={S.user!.name} photo={S.user!.photo} color={S.user!.avatarColor} size={60} font={21} />
          <Box
            component="label"
            key="up"
            aria-label={t('team.changeProfilePhoto')}
            sx={{
              position: 'absolute',
              right: '-4px',
              bottom: '-4px',
              width: '28px',
              height: '28px',
              borderRadius: '50%',
              background: tk.primary,
              color: tk.onPrimary,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              boxShadow: '0 2px 6px rgba(0,0,0,.3)',
            }}
          >
            <Sym name="photo_camera" size={16} color={tk.onPrimary} />
            <input
              key="f"
              type="file"
              accept="image/*"
              onChange={(e) => app.onFile(e, (d) => app.uploadMyPhoto(d))}
              style={{ display: 'none' }}
            />
          </Box>
        </Box>
        <Box key="m" sx={{ minWidth: 0 }}>
          <Box key="n" sx={{ fontSize: '17px', fontWeight: 700 }}>
            {S.user!.name}
          </Box>
          <Box
            key="e"
            sx={{ fontSize: '13px', color: NEUTRAL.secondary, display: 'flex', alignItems: 'center', gap: '6px' }}
          >
            <Sym name="mail" size={15} />
            {S.user!.email}
          </Box>
        </Box>
      </Box>

      <Box key="cs" sx={{ mt: '18px' }}>
        <Box sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, mb: '8px' }}>
          {t('team.colorScheme')}
        </Box>
        <Box sx={{ display: 'flex', gap: '6px' }}>
          {(['system', 'light', 'dark'] as const).map((scheme) => {
            const active = state.colorScheme === scheme;
            const label = t(`team.colorScheme${scheme.charAt(0).toUpperCase() + scheme.slice(1)}`);
            const icon = scheme === 'system' ? 'brightness_auto' : scheme === 'light' ? 'light_mode' : 'dark_mode';
            return (
              <ButtonBase
                key={scheme}
                onClick={() => app.setColorScheme(scheme)}
                aria-pressed={active}
                sx={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: '4px',
                  p: '10px 6px',
                  borderRadius: '12px',
                  border: active ? `2px solid ${tk.primary}` : `1px solid ${NEUTRAL.line}`,
                  background: active ? tk.primaryContainer : NEUTRAL.card,
                  color: active ? tk.primary : NEUTRAL.onSurfaceVariant,
                  fontSize: '11px',
                  fontWeight: 600,
                }}
              >
                <Sym name={icon} size={20} color={active ? tk.primary : NEUTRAL.secondary} />
                {label}
              </ButtonBase>
            );
          })}
        </Box>
      </Box>

      <Box key="lang" sx={{ mt: '18px' }}>
        <Box sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, mb: '8px' }}>{t('team.language')}</Box>
        <Box sx={{ display: 'flex', gap: '6px' }}>
          {supported.map((lng) => {
            const active = locale === lng;
            return (
              <ButtonBase
                key={lng}
                onClick={() => setLocale(lng)}
                aria-pressed={active}
                sx={{
                  flex: 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  p: '10px 6px',
                  borderRadius: '12px',
                  border: active ? `2px solid ${tk.primary}` : `1px solid ${NEUTRAL.line}`,
                  background: active ? tk.primaryContainer : NEUTRAL.card,
                  color: active ? tk.primary : NEUTRAL.onSurfaceVariant,
                  fontSize: '13px',
                  fontWeight: 600,
                }}
              >
                {LANGUAGE_LABELS[lng]}
              </ButtonBase>
            );
          })}
        </Box>
      </Box>

      {push.support === 'supported' && (
        <Box key="push" sx={{ mt: '18px' }}>
          <Box sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, mb: '8px' }}>
            {t('push.title')}
          </Box>
          <ButtonBase
            onClick={() => (push.subscribed ? push.disablePush() : push.enablePush())}
            disabled={push.busy || push.subscribed === null}
            aria-pressed={push.subscribed === true}
            sx={{
              width: '100%',
              display: 'flex',
              alignItems: 'center',
              gap: '10px',
              p: '12px 14px',
              borderRadius: '14px',
              border: push.subscribed ? `2px solid ${tk.primary}` : `1px solid ${NEUTRAL.line}`,
              background: push.subscribed ? tk.primaryContainer : NEUTRAL.card,
              color: push.subscribed ? tk.primary : NEUTRAL.onSurfaceVariant,
              opacity: push.busy ? 0.6 : 1,
              textAlign: 'left',
            }}
          >
            <Sym
              name={push.subscribed ? 'notifications_active' : 'notifications_off'}
              size={20}
              color={push.subscribed ? tk.primary : NEUTRAL.secondary}
            />
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Box sx={{ fontSize: '13px', fontWeight: 600 }}>{t('push.title')}</Box>
              <Box sx={{ fontSize: '11px', color: NEUTRAL.secondary, lineHeight: 1.4 }}>{t('push.description')}</Box>
            </Box>
          </ButtonBase>
        </Box>
      )}

      <ButtonBase
        key="lo"
        onClick={() => app.logout()}
        sx={{
          mt: '18px',
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '8px',
          p: '13px',
          borderRadius: '14px',
          border: '1px solid #F0C4C0',
          background: NEUTRAL.errorBg,
          color: NEUTRAL.error,
          fontWeight: 600,
          fontSize: '14px',
          cursor: 'pointer',
        }}
      >
        <Sym name="logout" size={20} color={NEUTRAL.error} />
        {t('team.logout')}
      </ButtonBase>

      <Box key="privacy" sx={{ mt: '24px', pt: '18px', borderTop: `1px solid ${NEUTRAL.line}` }}>
        <Box sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, mb: '8px' }}>
          {t('team.dataPrivacy')}
        </Box>
        <ButtonBase
          onClick={async () => {
            try {
              await app.exportMyData();
            } catch (err) {
              reportActionError({ setState: app.setState, toastMsg: app.toastMsg, onAuthError: app.logout }, err, 'team.exportDataError');
            }
          }}
          sx={{
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '13px',
            mb: '10px',
            borderRadius: '14px',
            border: `1px solid ${NEUTRAL.line}`,
            background: NEUTRAL.card,
            color: NEUTRAL.onSurfaceVariant,
            fontWeight: 600,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          <Sym name="download" size={20} color={NEUTRAL.secondary} />
          {t('team.exportData')}
        </ButtonBase>
        {!deleteOpen ? (
          <ButtonBase
            onClick={() => {
              setDeleteOpen(true);
              setConfirmEmail('');
              setDeleteErr(false);
            }}
            sx={{
              width: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              p: '13px',
              borderRadius: '14px',
              border: '1px solid #F0C4C0',
              background: NEUTRAL.errorBg,
              color: NEUTRAL.error,
              fontWeight: 600,
              fontSize: '14px',
              cursor: 'pointer',
            }}
          >
            <Sym name="delete_forever" size={20} color={NEUTRAL.error} />
            {t('team.deleteAccount')}
          </ButtonBase>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>
              {t('team.deleteAccountWarning')}
            </Box>
            <Box component="label" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary }}>
              {t('team.deleteAccountConfirmLabel')}
            </Box>
            <Box
              component="input"
              type="email"
              autoComplete="off"
              value={confirmEmail}
              placeholder={accountEmail}
              onChange={(e: ChangeEvent<HTMLInputElement>) => {
                setConfirmEmail(e.target.value);
                setDeleteErr(false);
              }}
              sx={{
                p: '11px 12px',
                borderRadius: '12px',
                border: `1px solid ${deleteErr ? NEUTRAL.error : NEUTRAL.line}`,
                background: NEUTRAL.card,
                fontSize: '14px',
                width: '100%',
                boxSizing: 'border-box',
              }}
            />
            {deleteErr ? (
              <Box sx={{ fontSize: '12px', color: NEUTRAL.error }}>{t('team.deleteAccountError')}</Box>
            ) : null}
            <Box sx={{ display: 'flex', gap: '8px' }}>
              <ButtonBase
                onClick={() => {
                  setDeleteOpen(false);
                  setConfirmEmail('');
                  setDeleteErr(false);
                }}
                disabled={deleting}
                sx={{
                  flex: 1,
                  p: '12px',
                  borderRadius: '14px',
                  border: `1px solid ${NEUTRAL.line}`,
                  background: NEUTRAL.card,
                  color: NEUTRAL.secondary,
                  fontWeight: 600,
                  fontSize: '14px',
                }}
              >
                {t('common.cancel')}
              </ButtonBase>
              <ButtonBase
                disabled={!canConfirmDelete || deleting}
                onClick={async () => {
                  setDeleting(true);
                  try {
                    await app.deleteAccount(confirmEmail.trim());
                    // On success the app resets to the login screen and this sheet unmounts.
                  } catch (err) {
                    if (err instanceof AuthError) {
                      reportActionError({ setState: app.setState, toastMsg: app.toastMsg, onAuthError: app.logout }, err);
                    } else {
                      captureException(err);
                      setDeleteErr(true);
                    }
                    setDeleting(false);
                  }
                }}
                sx={{
                  flex: 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '6px',
                  p: '12px',
                  borderRadius: '14px',
                  border: 'none',
                  background: NEUTRAL.error,
                  color: '#fff',
                  fontWeight: 700,
                  fontSize: '14px',
                  opacity: !canConfirmDelete || deleting ? 0.5 : 1,
                }}
              >
                <Sym name="delete_forever" size={18} color="#fff" />
                {t('team.deleteAccountConfirmButton')}
              </ButtonBase>
            </Box>
          </Box>
        )}
      </Box>
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
                fontFamily: "'Material Symbols Outlined'",
                fontSize: '21px',
                flex: '0 0 auto',
              }}
            >
              {i[2]}
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
