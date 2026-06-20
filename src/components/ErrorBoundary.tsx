import { NEUTRAL } from '@/styles/tokens';
import { Component, type ReactNode, type ErrorInfo } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { t } from '@/i18n';

interface Props {
  children: ReactNode;
  /** Static fallback element rendered on error. */
  fallback?: ReactNode;
  /** Render-prop fallback that receives the caught error (takes precedence). */
  renderFallback?: (error: Error, reset: () => void) => ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
}
interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    this.props.onError?.(error, info);
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.error('[ErrorBoundary]', error, info.componentStack);
    }
  }

  render() {
    if (this.state.error) {
      const reset = () => this.setState({ error: null });
      if (this.props.renderFallback) return this.props.renderFallback(this.state.error, reset);
      if (this.props.fallback) return this.props.fallback;
      return <DefaultFallback error={this.state.error} onReset={reset} />;
    }
    return this.props.children;
  }
}

function DefaultFallback({ error, onReset }: { error: Error; onReset: () => void }) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '16px',
        minHeight: '200px',
        p: '32px',
        textAlign: 'center',
      }}
    >
      <Box sx={{ fontSize: '32px' }}>⚠️</Box>
      <Box sx={{ fontSize: '16px', fontWeight: 700, color: NEUTRAL.onSurface }}>
        {t('error_boundary.componentTitle')}
      </Box>
      {import.meta.env.DEV && (
        <Box
          component="pre"
          sx={{
            fontSize: '12px',
            color: NEUTRAL.error,
            background: NEUTRAL.errorBg,
            p: '12px',
            borderRadius: '8px',
            maxWidth: '600px',
            overflow: 'auto',
            textAlign: 'left',
          }}
        >
          {error.message}
        </Box>
      )}
      <ButtonBase
        onClick={onReset}
        sx={{
          px: '20px',
          height: '40px',
          borderRadius: '10px',
          background: NEUTRAL.error,
          color: '#fff',
          fontSize: '14px',
          fontWeight: 600,
        }}
      >
        {t('error_boundary.retry')}
      </ButtonBase>
    </Box>
  );
}

/** Full-screen fallback for the app-level boundary. */
export function AppErrorFallback({ error }: { error: Error }) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100vh',
        gap: '16px',
        p: '32px',
        textAlign: 'center',
        background: NEUTRAL.appBg,
      }}
    >
      <Box sx={{ fontSize: '48px' }}>⚠️</Box>
      <Box sx={{ fontSize: '20px', fontWeight: 700, color: NEUTRAL.onSurface }}>{t('error_boundary.appTitle')}</Box>
      <Box sx={{ fontSize: '14px', color: NEUTRAL.onSurfaceVariant }}>{t('error_boundary.appSubtitle')}</Box>
      {import.meta.env.DEV && (
        <Box
          component="pre"
          sx={{
            fontSize: '12px',
            color: NEUTRAL.error,
            background: NEUTRAL.errorBg,
            p: '12px',
            borderRadius: '8px',
            maxWidth: '600px',
            overflow: 'auto',
            textAlign: 'left',
          }}
        >
          {error.message}
        </Box>
      )}
      <ButtonBase
        onClick={() => location.reload()}
        sx={{
          px: '24px',
          height: '44px',
          borderRadius: '12px',
          background: NEUTRAL.error,
          color: '#fff',
          fontSize: '15px',
          fontWeight: 600,
        }}
      >
        {t('error_boundary.reload')}
      </ButtonBase>
    </Box>
  );
}
