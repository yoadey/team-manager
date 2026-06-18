import { Component, type ReactNode, type ErrorInfo } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
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
      console.error('[ErrorBoundary]', error, info.componentStack);
    }
  }

  render() {
    if (this.state.error) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <DefaultFallback
          error={this.state.error}
          onReset={() => this.setState({ error: null })}
        />
      );
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
      <Box sx={{ fontSize: '16px', fontWeight: 700, color: '#1A1C20' }}>Etwas ist schiefgelaufen</Box>
      {import.meta.env.DEV && (
        <Box
          component="pre"
          sx={{
            fontSize: '12px',
            color: '#BA1A1A',
            background: '#FFDAD6',
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
          background: '#1565C0',
          color: '#fff',
          fontSize: '14px',
          fontWeight: 600,
        }}
      >
        Neu versuchen
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
        background: '#E4E5EC',
      }}
    >
      <Box sx={{ fontSize: '48px' }}>⚠️</Box>
      <Box sx={{ fontSize: '20px', fontWeight: 700, color: '#1A1C20' }}>Die App konnte nicht geladen werden</Box>
      <Box sx={{ fontSize: '14px', color: '#44474E' }}>Bitte lade die Seite neu.</Box>
      {import.meta.env.DEV && (
        <Box
          component="pre"
          sx={{
            fontSize: '12px',
            color: '#BA1A1A',
            background: '#FFDAD6',
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
          background: '#1565C0',
          color: '#fff',
          fontSize: '15px',
          fontWeight: 600,
        }}
      >
        Seite neu laden
      </ButtonBase>
    </Box>
  );
}
