import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { NEUTRAL } from '@/styles/tokens';
import { t } from '@/i18n';

const linkSx = {
  fontSize: '12px',
  fontWeight: 600,
  color: NEUTRAL.secondary,
  p: '4px 2px',
  textDecoration: 'underline',
};

/**
 * §5 DDG requires the legal notice to be "leicht erkennbar, unmittelbar
 * erreichbar und ständig verfügbar" -- reachable from every screen, including
 * pre-login. Uses openLegal (team-independent, unlike openMore) so it works
 * identically here (Login.tsx) and inside the authenticated app (Settings'
 * LegalPanel, via features/settings).
 */
export function LegalFooter() {
  const { openLegal } = useApp();
  return (
    <Box
      component="footer"
      sx={{ mt: '18px', display: 'flex', gap: '10px', alignItems: 'center', justifyContent: 'center', flexWrap: 'wrap' }}
    >
      <Box component="span" sx={{ fontSize: '12px', color: NEUTRAL.faint }}>
        {t('auth.footerLegalHint')}
      </Box>
      <ButtonBase onClick={() => openLegal('impressum')} sx={linkSx}>
        {t('sheet.legalImpressum')}
      </ButtonBase>
      <Box component="span" sx={{ fontSize: '12px', color: NEUTRAL.faint }} aria-hidden="true">
        ·
      </Box>
      <ButtonBase onClick={() => openLegal('datenschutz')} sx={linkSx}>
        {t('sheet.legalDatenschutz')}
      </ButtonBase>
    </Box>
  );
}
