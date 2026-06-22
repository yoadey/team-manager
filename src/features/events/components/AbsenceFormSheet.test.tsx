import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AbsenceFormSheet } from './AbsenceFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return {
    useApp,
    // Actions + selector derive from the per-test useApp mock so migrated
    // atoms (TextInput/TextArea via useAppActions/useAppSelector) resolve.
    useAppActions: vi.fn(() => useApp()),
    useAppSelector: (sel: (s: { form: Record<string, unknown> }) => unknown) => sel(useApp().state),
  };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(formOverrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      form: { from: '', to: '', reason: '', ...formOverrides },
      formErrors: {},
      busy: null,
    },
    setFormErrors: vi.fn(),
    onFormInput: vi.fn(),
    saveAbsence: vi.fn(),
  };
}

describe('AbsenceFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheet = {} as never;

  it('renders hint text', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={sheet} />);
    // Hint box is shown
    expect(screen.getByText(/Abwesenheit/i)).toBeTruthy();
  });

  it('renders From and To date fields', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={sheet} />);
    const dateInputs = document.querySelectorAll('input[type="date"]');
    expect(dateInputs.length).toBe(2);
  });

  it('renders reason field', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={sheet} />);
    const inputs = document.querySelectorAll('input[type="text"]');
    expect(inputs.length).toBeGreaterThanOrEqual(1);
  });

  it('shows "Abwesenheit eintragen" label in create mode', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByRole('button', { name: /eintragen/i })).toBeTruthy();
  });

  it('shows "Änderungen speichern" in edit mode', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<AbsenceFormSheet app={app as never} sheet={{ mode: 'edit' } as never} />);
    expect(screen.getByRole('button', { name: /speichern/i })).toBeTruthy();
  });
});
