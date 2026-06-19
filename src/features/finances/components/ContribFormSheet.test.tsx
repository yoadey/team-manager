import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ContribFormSheet } from './ContribFormSheet';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

function makeApp(formOverrides: Record<string, unknown> = {}, errOverrides: Record<string, string> = {}) {
  const setFormErrors = vi.fn();
  const app = {
    state: {
      primaryColor: '#1565C0',
      form: { label: '', amount: '', ...formOverrides },
      formErrors: { label: '', amount: '', ...errOverrides },
      busy: null,
    },
    setFormErrors,
    setFormVal: vi.fn(),
    onFormInput: vi.fn(),
    saveContrib: vi.fn(),
  };
  mockUseApp.mockReturnValue(app);
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

  it('shows label error when label is empty on blur', () => {
    const app = makeApp({ label: '' });
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('z. B. Monatsbeitrag');
    fireEvent.blur(input);
    expect(app.setFormErrors).toHaveBeenCalledWith({ label: expect.stringMatching(/\S+/) });
  });

  it('clears label error when label has value on blur', () => {
    const app = makeApp({ label: 'Jahresbeitrag' });
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('z. B. Monatsbeitrag');
    fireEvent.blur(input);
    expect(app.setFormErrors).toHaveBeenCalledWith({ label: '' });
  });

  it('shows amount required error when amount is blank on blur', () => {
    const app = makeApp({ label: 'Test', amount: '' });
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const amountInputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(amountInputs[0]);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: expect.stringMatching(/\S+/) });
  });

  it('shows error when amount is negative', () => {
    const app = makeApp({ label: 'Test', amount: '-5' });
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const amountInputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(amountInputs[0]);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: expect.stringMatching(/\S+/) });
  });

  it('clears amount error when amount is valid', () => {
    const app = makeApp({ label: 'Test', amount: '120' });
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    const amountInputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(amountInputs[0]);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: '' });
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

  it('shows error text in field when errors present', () => {
    makeApp({}, { label: 'Fehler!', amount: '' });
    const app = mockUseApp();
    render(<ContribFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Fehler!')).toBeTruthy();
  });
});
