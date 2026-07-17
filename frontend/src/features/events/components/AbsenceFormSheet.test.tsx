import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AbsenceFormSheet } from './AbsenceFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return { useApp };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp() {
  return {
    state: {
      primaryColor: '#4285F4',
      busy: null,
    },
    saveAbsence: vi.fn(),
  };
}

function makeSheet(formOverrides: Record<string, unknown> = {}, extra: Record<string, unknown> = {}) {
  return {
    formInitial: { from: '', to: '', reason: '', ...formOverrides },
    ...extra,
  } as never;
}

describe('AbsenceFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders hint text', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={makeSheet()} />);
    // Hint box is shown
    expect(screen.getByText(/Abwesenheit/i)).toBeTruthy();
  });

  it('renders From and To date fields', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={makeSheet()} />);
    const dateInputs = document.querySelectorAll('input[type="date"]');
    expect(dateInputs.length).toBe(2);
  });

  it('renders reason field', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={makeSheet()} />);
    const inputs = document.querySelectorAll('input[type="text"]');
    expect(inputs.length).toBeGreaterThanOrEqual(1);
  });

  // Regression test: the reason field had no client-side maxLength, unlike
  // every other create/edit form field, matching the backend's 500-char
  // validate.MaxLen bound.
  it('caps the reason input at 500 characters matching the backend limit', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={makeSheet()} />);
    const input = document.querySelector('input[type="text"]') as HTMLInputElement;
    expect(input.maxLength).toBe(500);
  });

  // Regression test: the from/to date inputs had no cross-field min/max, so
  // the browser's native date picker let a user pick a "to" date before
  // "from", unlike Stats.tsx's date-range picker which already constrains
  // its two inputs against each other -- the mismatch was only ever caught
  // after the fact by validateDateRange on Save.
  it('constrains the "to" date input by the current "from" value', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={makeSheet({ from: '2026-01-05' })} />);
    const dateInputs = document.querySelectorAll('input[type="date"]') as NodeListOf<HTMLInputElement>;
    expect(dateInputs[1].min).toBe('2026-01-05');
  });

  it('constrains the "from" date input by the current "to" value', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={makeSheet({ to: '2026-01-10' })} />);
    const dateInputs = document.querySelectorAll('input[type="date"]') as NodeListOf<HTMLInputElement>;
    expect(dateInputs[0].max).toBe('2026-01-10');
  });

  it('shows "Abwesenheit eintragen" label in create mode', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={makeSheet()} />);
    expect(screen.getByRole('button', { name: /eintragen/i })).toBeTruthy();
  });

  it('shows "Änderungen speichern" in edit mode', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={makeSheet({}, { mode: 'edit' })} />);
    expect(screen.getByRole('button', { name: /speichern/i })).toBeTruthy();
  });
});
