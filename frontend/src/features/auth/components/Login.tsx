import React, { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { NEUTRAL } from '@/styles/tokens';
import { Spinner, Sym } from '@/components/ui';
import { t } from '@/i18n';
import { config } from '@/config';
import { Register } from './Register';

export function Login() {
  const { state, doLogin, doPasswordLogin } = useApp();
  const { providers, busy, error } = state;

  // 'register' is a sibling view alongside the provider list / password form
  // (not a separate top-level Phase) -- mirrors how invite acceptance is
  // folded into the login flow rather than getting its own route.
  const [view, setView] = useState<'providers' | 'password' | 'register'>('providers');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  // busy is unused before authentication beyond the 'login:*' values this
  // screen itself sets, so any truthy busy means a login is in flight --
  // every control must disable while ANY one is pending, not just its own.
  // Without this, clicking a second provider (or the password form) while
  // the first is still resolving starts a second, overlapping login: the
  // shared `busy` field gets silently clobbered, and whichever request
  // resolves last wins establishSession's final user/team state even though
  // the session cookie may belong to the OTHER provider's login.
  const loginInFlight = !!busy;

  function handleProviderClick(p: (typeof providers)[number]) {
    if (loginInFlight) return;
    if (p.id === 'password') {
      setView('password');
    } else {
      doLogin(p.id);
    }
  }

  function handlePasswordSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (loginInFlight) return;
    doPasswordLogin(email, password);
  }

  const inputSx = {
    width: '100%',
    p: '10px 14px',
    borderRadius: '12px',
    border: `1.5px solid ${NEUTRAL.inputBorder}`,
    background: NEUTRAL.surface,
    color: NEUTRAL.onSurface,
    fontSize: '14px',
    outline: 'none',
    fontFamily: 'inherit',
  };

  return (
    <Box
      sx={{
        minHeight: '100vh',
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        p: '24px',
        overflow: 'auto',
        background: `radial-gradient(120% 100% at 50% 0%, ${NEUTRAL.surface} 0%, ${NEUTRAL.appBg} 100%)`,
        animation: 'tvFade .4s ease',
      }}
    >
      <Box
        sx={{
          width: '100%',
          maxWidth: '420px',
          background: NEUTRAL.card,
          borderRadius: '28px',
          boxShadow: '0 24px 60px rgba(20,30,55,.18)',
          p: '40px 32px 28px',
          animation: 'tvUp .5s ease',
        }}
      >
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            textAlign: 'center',
            gap: '6px',
            mb: '28px',
          }}
        >
          <Box
            sx={{
              width: 72,
              height: 72,
              borderRadius: '22px',
              background: '#1A1A1A',
              color: '#F5C518',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontWeight: 800,
              fontSize: '26px',
              letterSpacing: '1px',
              boxShadow: '0 8px 20px rgba(0,0,0,.22)',
            }}
          >
            SG
          </Box>
          <Box sx={{ fontSize: '23px', fontWeight: 700, mt: '14px', color: NEUTRAL.onSurface }}>{config.appName}</Box>
          <Box sx={{ fontSize: '14px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>
            {t('auth.loginHint')}
            <br />
            {t('auth.loginHintNote')}
          </Box>
        </Box>

        {error && (
          <Box
            role="alert"
            sx={{
              mb: '16px',
              p: '12px 14px',
              borderRadius: '14px',
              background: NEUTRAL.errorBg,
              color: NEUTRAL.error,
              fontSize: '13px',
              lineHeight: 1.5,
              display: 'flex',
              gap: '10px',
              alignItems: 'flex-start',
            }}
          >
            <Sym name="error" size={18} color={NEUTRAL.error} sx={{ lineHeight: 1.2 }} />
            {error}
          </Box>
        )}

        {view === 'register' ? (
          <Register onBack={() => setView('password')} />
        ) : view === 'password' ? (
          <Box component="form" onSubmit={handlePasswordSubmit} sx={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            <Box component="label" htmlFor="login-email" sx={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
              <Box component="span" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, px: '2px' }}>
                {t('auth.emailLabel')}
              </Box>
              <Box
                id="login-email"
                component="input"
                type="email"
                autoComplete="email"
                value={email}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEmail(e.target.value)}
                required
                sx={inputSx}
              />
            </Box>
            <Box component="label" htmlFor="login-password" sx={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
              <Box component="span" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, px: '2px' }}>
                {t('auth.passwordLabel')}
              </Box>
              <Box
                id="login-password"
                component="input"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPassword(e.target.value)}
                required
                sx={inputSx}
              />
            </Box>
            <ButtonBase
              component="button"
              type="submit"
              disabled={loginInFlight}
              sx={{
                width: '100%',
                p: '12px 16px',
                borderRadius: '16px',
                background: '#1565C0',
                color: '#fff',
                fontWeight: 600,
                fontSize: '15px',
                justifyContent: 'center',
                opacity: busy === 'login:password' ? 0.7 : 1,
              }}
            >
              {busy === 'login:password' ? <Spinner size={18} /> : t('auth.signIn')}
            </ButtonBase>
            <ButtonBase
              onClick={() => setView('register')}
              sx={{ fontSize: '13px', color: '#1565C0', py: '4px', justifyContent: 'center' }}
            >
              {t('auth.createAccount')}
            </ButtonBase>
            <ButtonBase
              onClick={() => setView('providers')}
              sx={{ fontSize: '13px', color: NEUTRAL.secondary, py: '4px', justifyContent: 'center' }}
            >
              ← {t('auth.back')}
            </ButtonBase>
          </Box>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            {providers.map((p) => {
              const isBusy = busy === 'login:' + p.id;
              const border = p.border ? '1.5px solid #DADCE3' : '1.5px solid transparent';
              const isApple = p.name === 'Apple';
              const glyph = isApple ? 'phone_iphone' : p.glyph;
              return (
                <ButtonBase
                  key={p.id}
                  onClick={() => handleProviderClick(p)}
                  disabled={loginInFlight}
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '14px',
                    width: '100%',
                    p: '12px 16px',
                    borderRadius: '16px',
                    background: p.bg,
                    color: p.fg,
                    border,
                    boxShadow: '0 1px 2px rgba(0,0,0,.06)',
                    opacity: isBusy ? 0.85 : 1,
                    justifyContent: 'flex-start',
                  }}
                >
                  <Box
                    component="span"
                    sx={{
                      width: 34,
                      height: 34,
                      borderRadius: '9px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontWeight: 800,
                      fontSize: p.name === 'Vereins-SSO' ? '13px' : '18px',
                      background: isApple ? 'transparent' : 'rgba(0,0,0,.05)',
                      color: p.fg,
                      flex: '0 0 auto',
                      fontFamily: isApple ? "'Material Symbols Outlined'" : 'inherit',
                    }}
                  >
                    {glyph}
                  </Box>
                  <Box component="span" sx={{ flex: 1, textAlign: 'left' }}>
                    <Box component="span" sx={{ display: 'block', fontSize: '15px', fontWeight: 600, lineHeight: 1.2 }}>
                      {p.name}
                    </Box>
                    <Box component="span" sx={{ display: 'block', fontSize: '12px', opacity: 0.7, fontWeight: 400 }}>
                      {p.sub}
                    </Box>
                  </Box>
                  {isBusy ? <Spinner size={18} /> : <Sym name="chevron_right" size={20} sx={{ opacity: 0.5 }} />}
                </ButtonBase>
              );
            })}
          </Box>
        )}

        <Box
          sx={{
            mt: '24px',
            p: '14px',
            borderRadius: '14px',
            background: NEUTRAL.warnBg,
            border: '1px solid #F0DBA8',
            display: 'flex',
            gap: '10px',
            alignItems: 'flex-start',
          }}
        >
          <Sym name="install_mobile" size={18} color={NEUTRAL.warn} sx={{ lineHeight: 1.2 }} />
          <Box sx={{ fontSize: '12px', color: NEUTRAL.warn, lineHeight: 1.5 }}>
            {t('auth.pwaInstallable')} <b>{t('auth.pwaIosLabel')}</b> {t('auth.pwaIosHint')}
          </Box>
        </Box>
        <Box sx={{ textAlign: 'center', mt: '18px', fontSize: '11px', color: NEUTRAL.faint }}>
          OIDC · Authorization Code Flow + PKCE
        </Box>
      </Box>
    </Box>
  );
}
