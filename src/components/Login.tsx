import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '../store/AppContext';
import { NEUTRAL } from '../theme/tokens';
import { Spinner, Sym } from './ui';

export function Login() {
  const { state, doLogin } = useApp();
  const { providers, busy } = state;

  return (
    <Box sx={{ minHeight: '100vh', width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', p: '24px', overflow: 'auto', background: 'radial-gradient(120% 100% at 50% 0%, #EEF0F6 0%, #E0E2EA 100%)', animation: 'tvFade .4s ease' }}>
      <Box sx={{ width: '100%', maxWidth: '420px', background: '#fff', borderRadius: '28px', boxShadow: '0 24px 60px rgba(20,30,55,.18)', p: '40px 32px 28px', animation: 'tvUp .5s ease' }}>
        <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: '6px', mb: '28px' }}>
          <Box sx={{ width: 72, height: 72, borderRadius: '22px', background: '#1A1A1A', color: '#F5C518', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 800, fontSize: '26px', letterSpacing: '1px', boxShadow: '0 8px 20px rgba(0,0,0,.22)' }}>SG</Box>
          <Box sx={{ fontSize: '23px', fontWeight: 700, mt: '14px', color: NEUTRAL.onSurface }}>Teamverwaltung</Box>
          <Box sx={{ fontSize: '14px', color: '#5A5D66', lineHeight: 1.5 }}>Anmeldung über deinen Identity-Provider.<br />Es ist kein Passwort beim Verein gespeichert.</Box>
        </Box>

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {providers.map((p) => {
            const isBusy = busy === 'login:' + p.id;
            const border = p.border ? '1.5px solid #DADCE3' : '1.5px solid transparent';
            const isApple = p.name === 'Apple';
            const glyph = isApple ? 'phone_iphone' : p.glyph;
            return (
              <ButtonBase
                key={p.id}
                onClick={() => doLogin(p.id)}
                disabled={isBusy}
                sx={{ display: 'flex', alignItems: 'center', gap: '14px', width: '100%', p: '12px 16px', borderRadius: '16px', background: p.bg, color: p.fg, border, boxShadow: '0 1px 2px rgba(0,0,0,.06)', opacity: isBusy ? 0.85 : 1, justifyContent: 'flex-start' }}
              >
                <Box component="span" sx={{ width: 34, height: 34, borderRadius: '9px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 800, fontSize: p.name === 'Vereins-SSO' ? '13px' : '18px', background: isApple ? 'transparent' : 'rgba(0,0,0,.05)', color: p.fg, flex: '0 0 auto', fontFamily: isApple ? "'Material Symbols Outlined'" : 'inherit' }}>{glyph}</Box>
                <Box component="span" sx={{ flex: 1, textAlign: 'left' }}>
                  <Box component="span" sx={{ display: 'block', fontSize: '15px', fontWeight: 600, lineHeight: 1.2 }}>{p.name}</Box>
                  <Box component="span" sx={{ display: 'block', fontSize: '12px', opacity: 0.7, fontWeight: 400 }}>{p.sub}</Box>
                </Box>
                {isBusy ? <Spinner size={18} /> : <Sym name="chevron_right" size={20} sx={{ opacity: 0.5 }} />}
              </ButtonBase>
            );
          })}
        </Box>

        <Box sx={{ mt: '24px', p: '14px', borderRadius: '14px', background: '#FFF7E6', border: '1px solid #F0DBA8', display: 'flex', gap: '10px', alignItems: 'flex-start' }}>
          <Sym name="install_mobile" size={18} color="#8A6100" sx={{ lineHeight: 1.2 }} />
          <Box sx={{ fontSize: '12px', color: '#6B5413', lineHeight: 1.5 }}>Als App installierbar (PWA). <b>iOS-Hinweis:</b> Push-Nachrichten funktionieren erst, wenn die App zum Home-Bildschirm hinzugefügt wurde.</Box>
        </Box>
        <Box sx={{ textAlign: 'center', mt: '18px', fontSize: '11px', color: NEUTRAL.faint }}>OIDC · Authorization Code Flow + PKCE</Box>
      </Box>
    </Box>
  );
}
