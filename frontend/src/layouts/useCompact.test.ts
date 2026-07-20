import { describe, it, expect } from 'vitest';
import { shortName } from './useCompact';

describe('shortName', () => {
  // Regression test: shortName used to only strip one hardcoded literal (the
  // MSW demo team name), making it a silent no-op for every real,
  // user-created team name even though it's used broadly (team-switcher
  // header, its aria-label, team-settings title, invite-description
  // highlight).
  it('shortens a real, user-created team name to its first word', () => {
    expect(shortName('SG Muster')).toBe('SG');
  });

  it('shortens a multi-word club name to its first word', () => {
    expect(shortName('TSV 1899 Rot-Weiss Musterstadt Handballabteilung')).toBe('TSV');
  });

  it('still shortens the former hardcoded demo team name the same way', () => {
    expect(shortName('A-Team TSC Schwarz-Gelb Aachen')).toBe('A-Team');
  });

  it('returns a single-word name unchanged', () => {
    expect(shortName('Adler')).toBe('Adler');
  });

  it('trims surrounding whitespace before shortening', () => {
    expect(shortName('  SG Muster  ')).toBe('SG');
  });

  it('collapses repeated internal whitespace', () => {
    expect(shortName('SG   Muster')).toBe('SG');
  });
});
