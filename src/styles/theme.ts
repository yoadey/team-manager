import { createTheme, type Theme } from '@mui/material/styles';
import { buildTokens, NEUTRAL } from './tokens';

// Extend the MUI palette with the Material-3 "container" colours so they are
// available everywhere via theme.palette.*.
declare module '@mui/material/styles' {
  interface Palette {
    primaryContainer: Palette['primary'];
    secondaryContainer: Palette['primary'];
  }
  interface PaletteOptions {
    primaryContainer?: PaletteOptions['primary'];
    secondaryContainer?: PaletteOptions['primary'];
  }
  interface TypeBackground {
    sidebar: string;
    card: string;
  }
}

/** Build a MUI theme from one of the 5 design presets. */
export function buildMuiTheme(presetKey: string): Theme {
  const t = buildTokens(presetKey);
  return createTheme({
    palette: {
      mode: 'light',
      primary: { main: t.primary, contrastText: t.onPrimary },
      secondary: { main: t.onSecondaryContainer, contrastText: t.onPrimary },
      primaryContainer: { main: t.primaryContainer, contrastText: t.onPrimaryContainer },
      secondaryContainer: { main: t.secondaryContainer, contrastText: t.onSecondaryContainer },
      success: { main: t.success },
      error: { main: t.error },
      warning: { main: t.warn },
      background: { default: NEUTRAL.appBg, paper: NEUTRAL.card, sidebar: NEUTRAL.sidebar, card: NEUTRAL.card },
      text: { primary: NEUTRAL.onSurface, secondary: NEUTRAL.secondary },
      divider: NEUTRAL.line,
    },
    shape: { borderRadius: 14 },
    typography: {
      fontFamily: "'Roboto','Roboto Flex',system-ui,sans-serif",
      h1: { fontSize: 22, fontWeight: 700, letterSpacing: '-0.2px' },
      h2: { fontSize: 18, fontWeight: 700, letterSpacing: '-0.2px' },
      button: { textTransform: 'none', fontWeight: 600 },
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: {
          body: { backgroundColor: NEUTRAL.appBg, color: NEUTRAL.onSurface },
          '*': { boxSizing: 'border-box', WebkitTapHighlightColor: 'transparent' },
          '::-webkit-scrollbar': { width: 10, height: 10 },
          '::-webkit-scrollbar-thumb': { background: '#C8CAD2', borderRadius: 10, border: '2px solid transparent', backgroundClip: 'content-box' },
          '::-webkit-scrollbar-track': { background: 'transparent' },
          '@keyframes tvFade': { from: { opacity: 0 }, to: { opacity: 1 } },
          '@keyframes tvUp': { from: { opacity: 0, transform: 'translateY(16px)' }, to: { opacity: 1, transform: 'translateY(0)' } },
          '@keyframes tvScale': { from: { opacity: 0, transform: 'scale(.96)' }, to: { opacity: 1, transform: 'scale(1)' } },
          '@keyframes tvSpin': { to: { transform: 'rotate(360deg)' } },
          '@keyframes tvSheetUp': { from: { transform: 'translateY(100%)' }, to: { transform: 'translateY(0)' } },
        },
      },
      MuiButtonBase: { defaultProps: { disableRipple: false } },
    },
  });
}
