import Box from '@mui/material/Box';
import { useApp } from '@/context/AppContext';
import { Sym } from './ui';

export function Toast() {
  const { state } = useApp();
  if (!state.toast) return null;
  const { message, action, kind = 'success' } = state.toast;
  const isError = kind === 'error';
  // Errors get role="alert" (assertive) instead of role="status" (polite) --
  // a permission/network failure should interrupt a screen reader the same
  // way the red icon/color interrupts a sighted user's skim, rather than
  // being announced at the next pause like a routine save confirmation.
  const accentColor = isError ? '#F2A0A0' : '#9FD8A0';
  return (
    <Box role={isError ? 'alert' : 'status'} aria-live={isError ? 'assertive' : 'polite'} aria-atomic="true" sx={{ position: 'fixed', left: '50%', bottom: 26, transform: 'translateX(-50%)', background: '#2A2C33', color: '#fff', p: '13px 20px', borderRadius: '13px', fontSize: '14px', fontWeight: 500, boxShadow: '0 10px 30px rgba(0,0,0,.3)', zIndex: 1500, animation: 'tvUp .3s ease', display: 'flex', alignItems: 'center', gap: '9px', maxWidth: '90vw' }}>
      <Sym name={isError ? 'error' : 'check_circle'} size={19} color={accentColor} />
      <Box component="span" sx={{ minWidth: 0, overflowWrap: 'anywhere' }}>
        {message}
      </Box>
      {action && (
        <Box
          component="button"
          type="button"
          onClick={action.fn}
          sx={{ ml: '4px', color: accentColor, fontWeight: 700, background: 'none', border: 'none', cursor: 'pointer', fontSize: '14px', p: 0, textDecoration: 'underline' }}
        >
          {action.label}
        </Box>
      )}
    </Box>
  );
}
