import { describe, it, expect } from 'vitest';
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
});
