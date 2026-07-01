import Box from '@mui/material/Box';
import { useApp } from '@/context/AppContext';
import { Sym } from './ui';

export function Toast() {
  const { state } = useApp();
  if (!state.toast) return null;
  const { message, action } = state.toast;
  return (
    <Box role="status" aria-live="polite" aria-atomic="true" sx={{ position: 'fixed', left: '50%', bottom: 26, transform: 'translateX(-50%)', background: '#2A2C33', color: '#fff', p: '13px 20px', borderRadius: '13px', fontSize: '14px', fontWeight: 500, boxShadow: '0 10px 30px rgba(0,0,0,.3)', zIndex: 1500, animation: 'tvUp .3s ease', display: 'flex', alignItems: 'center', gap: '9px', maxWidth: '90vw' }}>
      <Sym name="check_circle" size={19} color="#9FD8A0" />
      {message}
      {action && (
        <Box
          component="button"
          onClick={action.fn}
          sx={{ ml: '4px', color: '#9FD8A0', fontWeight: 700, background: 'none', border: 'none', cursor: 'pointer', fontSize: '14px', p: 0, textDecoration: 'underline' }}
        >
          {action.label}
        </Box>
      )}
    </Box>
  );
}
