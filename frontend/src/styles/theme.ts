import { createTheme, type Theme } from '@mui/material/styles';
import { buildTokens, NEUTRAL, NEUTRAL_LIGHT, neutralCssVars } from './tokens';

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
      // NEUTRAL_LIGHT (literal hex), not NEUTRAL (CSS var(--tv-neutral-*)
      // strings), for every palette slot below: MUI computes hover/disabled/
      // Skeleton overlays by calling alpha()/darken()/lighten() on
      // palette.text/background/divider internally (e.g. MuiSkeleton's
      // default background is alpha(theme.palette.text.primary, 0.11)),
      // which throws/warns ("Unsupported `var(...)` color") on anything
      // that isn't a literal, parseable color. Dark mode is handled
      // independently -- by the CSS custom properties neutralCssVars seeds
      // below and by components referencing NEUTRAL directly in their own
      // sx/styles -- not by this theme's palette.mode (always 'light').
      background: {
        default: NEUTRAL_LIGHT.appBg,
        paper: NEUTRAL_LIGHT.card,
        sidebar: NEUTRAL_LIGHT.sidebar,
        card: NEUTRAL_LIGHT.card,
      },
      text: { primary: NEUTRAL_LIGHT.onSurface, secondary: NEUTRAL_LIGHT.secondary },
      divider: NEUTRAL_LIGHT.line,
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
          ':root': neutralCssVars(false),
          '@media (prefers-color-scheme: dark)': {
            ':root:not([data-color-scheme="light"])': neutralCssVars(true),
          },
          '[data-color-scheme="dark"]': neutralCssVars(true),
          '[data-color-scheme="light"]': neutralCssVars(false),
          body: { backgroundColor: NEUTRAL.appBg, color: NEUTRAL.onSurface },
          '*': { boxSizing: 'border-box', WebkitTapHighlightColor: 'transparent' },
          '::-webkit-scrollbar': { width: 10, height: 10 },
          '::-webkit-scrollbar-thumb': {
            background: '#C8CAD2',
            borderRadius: 10,
            border: '2px solid transparent',
            backgroundClip: 'content-box',
          },
          '::-webkit-scrollbar-track': { background: 'transparent' },
          '@keyframes tvFade': { from: { opacity: 0 }, to: { opacity: 1 } },
          '@keyframes tvUp': {
            from: { opacity: 0, transform: 'translateY(16px)' },
            to: { opacity: 1, transform: 'translateY(0)' },
          },
          '@keyframes tvScale': {
            from: { opacity: 0, transform: 'scale(.96)' },
            to: { opacity: 1, transform: 'scale(1)' },
          },
          '@keyframes tvSpin': { to: { transform: 'rotate(360deg)' } },
          '@keyframes tvSheetUp': { from: { transform: 'translateY(100%)' }, to: { transform: 'translateY(0)' } },
        },
      },
      MuiButtonBase: { defaultProps: { disableRipple: false } },
    },
  });
}
