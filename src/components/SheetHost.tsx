import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import Modal from '@mui/material/Modal';
import { useApp } from '../context/AppContext';
import { isPageSheet } from '../context/AppContext';
import { NEUTRAL } from '../styles/tokens';
import { Sym } from './ui';
import { renderSheet, sheetMeta } from '../sheets';
import { useCompact } from '../layouts/AppShell';

export function SheetHost() {
  const app = useApp();
  const { state } = app;
  const compact = useCompact();
  const cur = state.sheet;
  const modalSheet = cur && !isPageSheet(cur.type) ? cur : null;

  if (!modalSheet) return null;
  const meta = sheetMeta(app, modalSheet);

  return (
    <Modal open onClose={app.closeSheet} closeAfterTransition sx={{ zIndex: 1300 }}>
      <Box
        onClick={app.closeSheet}
        sx={{ position: 'fixed', inset: 0, background: 'rgba(15,20,35,0.45)', display: 'flex', alignItems: compact ? 'flex-end' : 'center', justifyContent: 'center', p: compact ? 0 : '24px', animation: 'tvFade .2s ease' }}
      >
        <Box
          onClick={(e) => e.stopPropagation()}
          sx={{
            width: '100%', maxWidth: compact ? '480px' : '540px', maxHeight: compact ? '92vh' : '88vh',
            background: NEUTRAL.surface, display: 'flex', flexDirection: 'column',
            borderRadius: compact ? '28px 28px 0 0' : '28px', overflow: 'hidden',
            boxShadow: '0 30px 80px rgba(20,30,55,.35)',
            animation: compact ? 'tvSheetUp .28s cubic-bezier(.2,.8,.2,1)' : 'tvScale .24s ease',
          }}
        >
          <Box sx={{ flex: '0 0 auto', display: 'flex', alignItems: 'center', gap: '12px', p: '18px 20px 12px' }}>
            {meta.hasBack ? (
              <ButtonBase onClick={meta.onBack} sx={{ width: 38, height: 38, borderRadius: '50%', background: NEUTRAL.line2, color: NEUTRAL.onSurfaceVariant, flex: '0 0 auto' }}><Sym name="arrow_back" size={22} /></ButtonBase>
            ) : null}
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Box sx={{ fontSize: '18px', fontWeight: 700, letterSpacing: '-.2px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{meta.title}</Box>
              {meta.subtitle ? <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, mt: '2px' }}>{meta.subtitle}</Box> : null}
            </Box>
            <ButtonBase onClick={app.closeSheet} sx={{ width: 38, height: 38, borderRadius: '50%', background: NEUTRAL.line2, color: NEUTRAL.onSurfaceVariant, flex: '0 0 auto' }}><Sym name="close" size={22} /></ButtonBase>
          </Box>
          <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto', p: '4px 20px 22px' }}>{renderSheet(app, modalSheet)}</Box>
        </Box>
      </Box>
    </Modal>
  );
}
