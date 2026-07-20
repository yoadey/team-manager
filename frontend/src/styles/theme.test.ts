import { describe, it, expect } from 'vitest';
import { alpha } from '@mui/material/styles';
import { buildMuiTheme } from './theme';

describe('buildMuiTheme', () => {
  it('builds a valid MUI theme for the default preset', () => {
    const theme = buildMuiTheme('#4285F4');
    expect(theme).toBeTruthy();
    expect(theme.palette.primary.main).toBeTruthy();
    expect(theme.shape.borderRadius).toBe(14);
  });

  it('builds themes for different preset colors', () => {
    const presets = ['#4285F4', '#E91E63', '#4CAF50', '#FF9800', '#9C27B0'];
    presets.forEach((key) => {
      const theme = buildMuiTheme(key);
      expect(theme.palette).toBeTruthy();
    });
  });

  it('sets typography to Roboto', () => {
    const theme = buildMuiTheme('#4285F4');
    expect(theme.typography.fontFamily).toContain('Roboto');
  });

  it('sets correct h1 font size', () => {
    const theme = buildMuiTheme('#4285F4');
    expect((theme.typography.h1 as { fontSize: number }).fontSize).toBe(22);
  });

  it('sets background colors', () => {
    const theme = buildMuiTheme('#4285F4');
    expect(theme.palette.background.default).toBeTruthy();
  });

  // Regression test: palette.text/background/divider used to be set to
  // NEUTRAL's var(--tv-neutral-*) CSS custom-property strings. MUI computes
  // several components' default styles by calling alpha()/darken()/
  // lighten() on those exact palette values internally -- e.g. MuiSkeleton's
  // default background is alpha(theme.palette.text.primary, 0.11) -- and
  // MUI's colorManipulator throws "MUI: Unsupported `var(...)` color" when
  // asked to alpha-blend a CSS variable reference instead of a literal,
  // parseable color. This reproduces exactly the call MuiSkeleton makes
  // (see @mui/material/Skeleton's styled() definition), which is what a
  // user hit by navigating to any page (e.g. Finances) while its data query
  // was still loading, since SkeletonList renders MUI's real Skeleton.
  it('palette.text.primary/secondary and background/divider are literal colors MUI can alpha-blend', () => {
    const theme = buildMuiTheme('#4285F4');
    expect(() => alpha(theme.palette.text.primary, 0.11)).not.toThrow();
    expect(() => alpha(theme.palette.text.secondary, 0.11)).not.toThrow();
    expect(() => alpha(theme.palette.background.default, 0.11)).not.toThrow();
    expect(() => alpha(theme.palette.background.paper, 0.11)).not.toThrow();
    expect(() => alpha(theme.palette.divider, 0.11)).not.toThrow();
  });
});
