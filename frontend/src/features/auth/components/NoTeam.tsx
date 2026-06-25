import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import { t } from '@/i18n';

export function NoTeam() {
  const app = useApp();

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
          textAlign: 'center',
          animation: 'tvUp .5s ease',
        }}
      >
        <Box
          sx={{
            width: 72,
            height: 72,
            borderRadius: '22px',
            mx: 'auto',
            background: NEUTRAL.line2,
            color: NEUTRAL.secondary,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Sym name="groups" size={36} />
        </Box>
        <Box sx={{ fontSize: '21px', fontWeight: 700, mt: '20px', color: NEUTRAL.onSurface }}>{t('noTeam.title')}</Box>
        <Box sx={{ fontSize: '14px', color: NEUTRAL.secondary, lineHeight: 1.5, mt: '8px' }}>{t('noTeam.hint')}</Box>

        <ButtonBase
          onClick={() => app.openCreateTeam()}
          sx={{
            width: '100%',
            mt: '28px',
            p: '12px 16px',
            borderRadius: '16px',
            background: '#1565C0',
            color: '#fff',
            fontWeight: 600,
            fontSize: '15px',
            gap: '8px',
            justifyContent: 'center',
          }}
        >
          <Sym name="add" size={20} color="#fff" />
          {t('noTeam.createBtn')}
        </ButtonBase>

        <ButtonBase
          onClick={() => app.logout()}
          sx={{ mt: '16px', fontSize: '13px', color: NEUTRAL.secondary, py: '4px', justifyContent: 'center' }}
        >
          {t('noTeam.logout')}
        </ButtonBase>
      </Box>
    </Box>
  );
}
