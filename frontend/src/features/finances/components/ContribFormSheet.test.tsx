import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ContribFormSheet } from './ContribFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return {
    useApp,
    useAppActions: vi.fn(() => useApp()),
    useAppSelector: (sel: (s: { form: Record<string, unknown> }) => unknown) => sel(useApp().state),
  };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(formOverrides: Record<string, unknown> = {}, errOverrides: Record<string, string> = {}) {
  const setFormErrors = vi.fn();
  const app = {
    state: {
      primaryColor: '#1565C0',
      form: { id: 'c1', label: '', amount: '', ...formOverrides },
      formErrors: { label: '', amount: '', ...errOverrides },
      busy: null,
    },
    setFormErrors,
    setFormVal: vi.fn(),
    onFormInput: vi.fn(),
    saveContrib: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

describe('ContribFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheet = {} as never;

  it('renders label and amount fields', () => {
    makeApp();
    const app = mockUseApp();
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByPlaceholderText('z. B. Monatsbeitrag')).toBeTruthy();
  });

  it('caps the label input at 255 characters matching the backend limit', () => {
    makeApp();
    const app = mockUseApp();
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('z. B. Monatsbeitrag') as HTMLInputElement;
    expect(input.maxLength).toBe(255);
  });

  it('shows label error when label is empty on blur', async () => {
    makeApp({ label: '' });
    const app = mockUseApp();
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('z. B. Monatsbeitrag');
    fireEvent.blur(input);
    await waitFor(() => {
      expect(screen.getByText('Bezeichnung fehlt.')).toBeTruthy();
    });
  });

  it('shows amount required error when amount is blank on blur', async () => {
    makeApp({ label: 'Test', amount: '' });
    const app = mockUseApp();
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const amountInputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(amountInputs[0]);
    await waitFor(() => {
      expect(screen.getByText('Betrag fehlt.')).toBeTruthy();
    });
  });

  it('shows error when amount is negative', async () => {
    makeApp({ label: 'Test', amount: '-5' });
    const app = mockUseApp();
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const amountInputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(amountInputs[0]);
    await waitFor(() => {
      expect(screen.getByText('Betrag muss größer als 0 € sein.')).toBeTruthy();
    });
  });

  it('clears amount error when amount is valid', async () => {
    makeApp({ label: 'Test', amount: '120' });
    const app = mockUseApp();
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const amountInputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(amountInputs[0]);
    await waitFor(() => {
      expect(screen.queryByText('Betrag muss größer als 0 € sein.')).toBeNull();
    });
  });

  it('submit button is disabled when form is empty', () => {
    makeApp({ label: '', amount: '' });
    const app = mockUseApp();
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Änderungen speichern/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when form is valid', () => {
    makeApp({ label: 'Monatsbeitrag', amount: '15' });
    const app = mockUseApp();
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Änderungen speichern/i });
    expect(btn).not.toBeDisabled();
  });

  it('calls saveContrib when save button is clicked and form is valid', async () => {
    const app = makeApp({ label: 'Monatsbeitrag', amount: '15' });
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByRole('button', { name: /Änderungen speichern/i }));
    await waitFor(() => {
      expect(app.saveContrib).toHaveBeenCalled();
    });
  });
});
