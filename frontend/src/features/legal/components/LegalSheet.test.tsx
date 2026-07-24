import { describe, it, expect, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { LegalSheet } from './LegalSheet';
import { setLocale } from '@/i18n';
import type { SheetState } from '@/context/AppContext';

function sheet(legalPage: 'impressum' | 'datenschutz'): SheetState {
  return { type: 'legal', legalPage };
}

describe('LegalSheet', () => {
  afterEach(() => {
    setLocale('de');
  });

  it('renders the German legal-notice content by default', () => {
    render(<LegalSheet app={{} as never} sheet={sheet('impressum')} />);
    expect(screen.getByText('Angaben gemäß § 5 DDG')).toBeTruthy();
    expect(screen.getAllByText((c) => c.includes('[BETREIBER:')).length).toBeGreaterThan(0);
  });

  it('renders the German privacy-policy content', () => {
    render(<LegalSheet app={{} as never} sheet={sheet('datenschutz')} />);
    expect(screen.getByText('Verantwortlicher')).toBeTruthy();
    expect(screen.getByText('Deine Rechte')).toBeTruthy();
  });

  it('renders the English legal-notice content when the locale is en', () => {
    setLocale('en');
    render(<LegalSheet app={{} as never} sheet={sheet('impressum')} />);
    expect(screen.getByText('Information pursuant to § 5 DDG (Germany)')).toBeTruthy();
  });

  it('defaults to impressum when no legalPage is set', () => {
    render(<LegalSheet app={{} as never} sheet={{ type: 'legal' }} />);
    expect(screen.getByText('Angaben gemäß § 5 DDG')).toBeTruthy();
  });
});
