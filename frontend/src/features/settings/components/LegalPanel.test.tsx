import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { LegalPanel } from './LegalPanel';
import type { AppContextValue } from '@/context/AppContext';

function makeApp(): AppContextValue {
  return { openLegal: vi.fn() } as unknown as AppContextValue;
}

describe('LegalPanel', () => {
  it('clicking "Impressum" calls openLegal with impressum', () => {
    const app = makeApp();
    render(<LegalPanel app={app} />);
    fireEvent.click(screen.getByText('Impressum'));
    expect(app.openLegal).toHaveBeenCalledWith('impressum');
  });

  it('clicking "Datenschutzerklärung" calls openLegal with datenschutz', () => {
    const app = makeApp();
    render(<LegalPanel app={app} />);
    fireEvent.click(screen.getByText('Datenschutzerklärung'));
    expect(app.openLegal).toHaveBeenCalledWith('datenschutz');
  });
});
