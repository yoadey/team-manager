import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import type { SheetProps } from '@/sheets/types';
import type { Route } from '@/context/AppContext';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Sym, Av, SectionTitle } from '@/components/ui';
import { shortName } from '@/layouts/useCompact';
import { t, type Locale } from '@/i18n';
import { useLocale } from '@/i18n/LocaleProvider';

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
              background: active ? tk.primaryContainer : '#fff',
              justifyContent: 'flex-start',
              textAlign: 'left',
            }}
          >
            <Box
              component="span"
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
                {tm.myRoles.map((r) => r.name).join(', ') + ' · ' + t('team.membersCount', { n: tm.memberCount })}
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
  const team = app.activeTeam()!;
  const S = state;
  const roles = S.roles;
  const myIds = app.myRoles().map((r) => r.id);
  return (
    <Box>
      <Box key="hd" sx={{ display: 'flex', alignItems: 'center', gap: '14px', p: '4px 2px 18px' }}>
        <Box key="av" sx={{ position: 'relative' }}>
          <Av name={S.user!.name} photo={S.user!.photo} color={S.user!.avatarColor} size={60} font={21} />
          <Box
            component="label"
            key="up"
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

      <Box key="rs">
        <SectionTitle>{t('team.myRolesInTeam', { teamName: shortName(team.name) })}</SectionTitle>
        <Box key="l" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {roles.map((r) => {
            const sel = myIds.includes(r.id);
            return (
              <ButtonBase
                key={r.id}
                onClick={() => app.toggleMyRole(r.id)}
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  width: '100%',
                  p: '12px 14px',
                  borderRadius: '14px',
                  cursor: 'pointer',
                  border: '1px solid ' + (sel ? tk.primary : '#E6E7EE'),
                  background: sel ? tk.primaryContainer : '#fff',
                  justifyContent: 'flex-start',
                  textAlign: 'left',
                }}
              >
                <Box
                  component="span"
                  key="c"
                  sx={{
                    width: '22px',
                    height: '22px',
                    borderRadius: '6px',
                    border: '2px solid ' + (sel ? tk.primary : '#B0B3BC'),
                    background: sel ? tk.primary : '#fff',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    flex: '0 0 auto',
                  }}
                >
                  {sel ? <Sym name="check" size={15} color="#fff" /> : null}
                </Box>
                <Box
                  component="span"
                  key="d"
                  sx={{ width: '11px', height: '11px', borderRadius: '50%', background: r.color, flex: '0 0 auto' }}
                />
                <Box component="span" key="n" sx={{ flex: 1, textAlign: 'left', fontSize: '14px', fontWeight: 500 }}>
                  {r.name}
                </Box>
              </ButtonBase>
            );
          })}
        </Box>
        <Box key="hint" sx={{ fontSize: '12px', color: NEUTRAL.faint, m: '8px 2px 0', lineHeight: 1.5 }}>
          {t('team.multiRoleHint')}
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
          background: '#FFF4F3',
          color: NEUTRAL.error,
          fontWeight: 600,
          fontSize: '14px',
          cursor: 'pointer',
        }}
      >
        <Sym name="logout" size={20} color={NEUTRAL.error} />
        {t('team.logout')}
      </ButtonBase>
    </Box>
  );
}

export function MoreSheet({ app }: SheetProps) {
  const items: Array<[Route, string, string, boolean]> = [
    ['finances', t('nav.finances'), 'payments', app.can('finances', 'read')],
    ['stats', t('nav.stats'), 'insights', true],
    ['news', t('nav.news'), 'campaign', true],
    ['polls', t('nav.polls'), 'how_to_vote', true],
    ['team', t('nav.team'), 'shield', true],
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
            <Sym name="chevron_right" size={22} color="#C0C2CA" />
          </ButtonBase>
        ))}
    </Box>
  );
}
