import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import { t } from '@/i18n';
import type { SettingsPanelProps } from '../settingsCategories';

export function LegalPanel({ app }: SettingsPanelProps) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      <ButtonBase
        onClick={() => app.openLegal('impressum')}
        sx={{
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          p: '13px',
          borderRadius: '14px',
          border: `1px solid ${NEUTRAL.line}`,
          background: NEUTRAL.card,
          color: NEUTRAL.onSurfaceVariant,
          fontWeight: 600,
          fontSize: '14px',
        }}
      >
        {t('sheet.legalImpressum')}
        <Sym name="chevron_right" size={20} color={NEUTRAL.faint} />
      </ButtonBase>
      <ButtonBase
        onClick={() => app.openLegal('datenschutz')}
        sx={{
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          p: '13px',
          borderRadius: '14px',
          border: `1px solid ${NEUTRAL.line}`,
          background: NEUTRAL.card,
          color: NEUTRAL.onSurfaceVariant,
          fontWeight: 600,
          fontSize: '14px',
        }}
      >
        {t('sheet.legalDatenschutz')}
        <Sym name="chevron_right" size={20} color={NEUTRAL.faint} />
      </ButtonBase>
    </Box>
  );
}
