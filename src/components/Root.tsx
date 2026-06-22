import Box from '@mui/material/Box';
import { useApp } from '@/context/AppContext';
import { NEUTRAL } from '@/styles/tokens';
import { Login } from '@/features/auth';
import { Shell } from '@/layouts/AppShell';
import { SheetHost } from './SheetHost';
import { Toast } from './Toast';
import { t } from '@/i18n';

export function Root() {
  const { state } = useApp();

  if (state.phase === 'loading') {
    return (
      <Box
        sx={{
          height: '100vh',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '18px',
          color: NEUTRAL.secondary,
          background: NEUTRAL.appBg,
        }}
      >
        <Box
          role="status"
          aria-label={t('common.loading')}
          sx={{
            width: 42,
            height: 42,
            border: `4px solid ${NEUTRAL.line3}`,
            borderTopColor: state.primaryColor,
            borderRadius: '50%',
            animation: 'tvSpin .8s linear infinite',
          }}
        />
        <Box aria-live="polite" sx={{ fontSize: '14px', letterSpacing: '.3px' }}>
          {t('shell.connecting')}
        </Box>
      </Box>
    );
  }

  return (
    <>
      {state.phase === 'login' ? <Login /> : <Shell />}
      <SheetHost />
      <Toast />
    </>
  );
}
