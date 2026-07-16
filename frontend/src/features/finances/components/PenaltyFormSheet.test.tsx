import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PenaltyFormSheet } from './PenaltyFormSheet';

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

function makeApp(formOverrides: Record<string, unknown> = {}, errOverrides: Record<string, string> = {}) {
  const setFormErrors = vi.fn();
  const app = {
    state: {
      primaryColor: '#1565C0',
      form: { label: '', amount: '', ...formOverrides },
      formErrors: { label: '', amount: '', ...errOverrides },
    },
    setFormErrors,
    setFormVal: vi.fn(),
    onFormInput: vi.fn(),
    savePenalty: vi.fn(),
    deletePenaltyDef: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

describe('PenaltyFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheet = { mode: 'create' } as never;

  it('renders label and amount fields', () => {
    makeApp();
    render(<PenaltyFormSheet app={mockUseApp() as never} sheet={sheet} />);
    expect(screen.getByPlaceholderText('z. B. Zu spät zum Training')).toBeTruthy();
  });

  // Regression test: the label field had no client-side maxLength, matching
  // the backend's 255-char validate.MaxLen bound.
  it('caps the label input at 255 characters matching the backend limit', () => {
    makeApp();
    render(<PenaltyFormSheet app={mockUseApp() as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('z. B. Zu spät zum Training') as HTMLInputElement;
    expect(input.maxLength).toBe(255);
  });

  it('shows label error when label is empty on blur', () => {
    const app = makeApp({ label: '' });
    render(<PenaltyFormSheet app={app as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('z. B. Zu spät zum Training');
    fireEvent.blur(input);
    expect(app.setFormErrors).toHaveBeenCalledWith({ label: expect.stringMatching(/\S+/) });
  });

  it('clears label error when label has value on blur', () => {
    const app = makeApp({ label: 'Zu spät' });
    render(<PenaltyFormSheet app={app as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('z. B. Zu spät zum Training');
    fireEvent.blur(input);
    expect(app.setFormErrors).toHaveBeenCalledWith({ label: '' });
  });

  it('shows amount required error when amount is blank on blur', () => {
    const app = makeApp({ label: 'Test', amount: '' });
    render(<PenaltyFormSheet app={app as never} sheet={sheet} />);
    const inputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(inputs[0]);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: expect.stringMatching(/\S+/) });
  });

  it('shows amount positive error when amount is zero', () => {
    const app = makeApp({ label: 'Test', amount: '0' });
    render(<PenaltyFormSheet app={app as never} sheet={sheet} />);
    const inputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(inputs[0]);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: expect.stringMatching(/\S+/) });
  });

  it('clears amount error when amount is valid positive number', () => {
    const app = makeApp({ label: 'Test', amount: '5' });
    render(<PenaltyFormSheet app={app as never} sheet={sheet} />);
    const inputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(inputs[0]);
    expect(app.setFormErrors).toHaveBeenCalledWith({ amount: '' });
  });

  it('disables submit when form is invalid (empty label and amount)', () => {
    makeApp({ label: '', amount: '' });
    const app = mockUseApp();
    render(<PenaltyFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Strafe hinzufügen/i });
    expect(btn).toBeDisabled();
  });

  it('enables submit when form is valid', () => {
    makeApp({ label: 'Zu spät', amount: '10' });
    const app = mockUseApp();
    render(<PenaltyFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Strafe hinzufügen/i });
    expect(btn).not.toBeDisabled();
  });

  it('shows delete button in edit mode', () => {
    makeApp({ id: 'p1', label: 'Strafe', amount: '5' });
    const app = mockUseApp();
    render(<PenaltyFormSheet app={app as never} sheet={{ mode: 'edit' } as never} />);
    expect(screen.getByText(/Strafe aus Katalog entfernen/i)).toBeTruthy();
  });

  it('shows error text when errors are set', () => {
    makeApp({}, { label: 'Pflichtfeld', amount: '' });
    const app = mockUseApp();
    render(<PenaltyFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Pflichtfeld')).toBeTruthy();
  });

  it('clicking delete button calls deletePenaltyDef', () => {
    const app = makeApp({ id: 'p1', label: 'Strafe', amount: '5' });
    render(<PenaltyFormSheet app={app as never} sheet={{ mode: 'edit' } as never} />);
    fireEvent.click(screen.getByText(/Strafe aus Katalog entfernen/i).closest('button')!);
    expect(app.deletePenaltyDef).toHaveBeenCalledWith('p1');
  });

  it('calls savePenalty when save button is clicked with valid form', () => {
    const app = makeApp({ label: 'Verspätung', amount: '5' });
    render(<PenaltyFormSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByRole('button', { name: /Strafe/i }));
    expect(app.savePenalty).toHaveBeenCalled();
  });
});
