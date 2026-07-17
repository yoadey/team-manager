import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PenaltyFormSheet } from './PenaltyFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return { useApp };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(formOverrides: Record<string, unknown> = {}) {
  const app = {
    state: {
      primaryColor: '#1565C0',
    },
    savePenalty: vi.fn(),
    deletePenaltyDef: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return { app, formInitial: { label: '', amount: '', ...formOverrides } };
}

function makeSheet(mode: 'create' | 'edit', formInitial: Record<string, unknown>) {
  return { mode, formInitial } as never;
}

describe('PenaltyFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders label and amount fields', () => {
    const { app, formInitial } = makeApp();
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.getByPlaceholderText('z. B. Zu spät zum Training')).toBeTruthy();
  });

  it('caps the label input at 255 characters matching the backend limit', () => {
    const { app, formInitial } = makeApp();
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const input = screen.getByPlaceholderText('z. B. Zu spät zum Training') as HTMLInputElement;
    expect(input.maxLength).toBe(255);
  });

  it('shows label error when label is empty on blur', async () => {
    const { app, formInitial } = makeApp({ label: '' });
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const input = screen.getByPlaceholderText('z. B. Zu spät zum Training');
    fireEvent.blur(input);
    await waitFor(() => {
      expect(screen.getByText('Bezeichnung fehlt.')).toBeTruthy();
    });
  });

  it('shows amount required error when amount is blank on blur', async () => {
    const { app, formInitial } = makeApp({ label: 'Test', amount: '' });
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const inputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(inputs[0]);
    await waitFor(() => {
      expect(screen.getByText('Betrag fehlt.')).toBeTruthy();
    });
  });

  it('shows amount positive error when amount is zero', async () => {
    const { app, formInitial } = makeApp({ label: 'Test', amount: '0' });
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const inputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(inputs[0]);
    await waitFor(() => {
      expect(screen.getByText('Betrag muss größer als 0 € sein.')).toBeTruthy();
    });
  });

  it('clears amount error when amount is valid positive number', async () => {
    const { app, formInitial } = makeApp({ label: 'Test', amount: '5' });
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const inputs = screen.getAllByRole('spinbutton');
    fireEvent.blur(inputs[0]);
    await waitFor(() => {
      expect(screen.queryByText('Betrag muss größer als 0 € sein.')).toBeNull();
    });
  });

  it('disables submit when form is invalid (empty label and amount)', () => {
    const { app, formInitial } = makeApp({ label: '', amount: '' });
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const btn = screen.getByRole('button', { name: /Strafe hinzufügen/i });
    expect(btn).toBeDisabled();
  });

  it('enables submit when form is valid', () => {
    const { app, formInitial } = makeApp({ label: 'Zu spät', amount: '10' });
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const btn = screen.getByRole('button', { name: /Strafe hinzufügen/i });
    expect(btn).not.toBeDisabled();
  });

  it('shows delete button in edit mode', () => {
    const { app, formInitial } = makeApp({ id: 'p1', label: 'Strafe', amount: '5' });
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('edit', formInitial)} />);
    expect(screen.getByText(/Strafe aus Katalog entfernen/i)).toBeTruthy();
  });

  it('clicking delete button calls deletePenaltyDef', () => {
    const { app, formInitial } = makeApp({ id: 'p1', label: 'Strafe', amount: '5' });
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('edit', formInitial)} />);
    fireEvent.click(screen.getByText(/Strafe aus Katalog entfernen/i).closest('button')!);
    expect(app.deletePenaltyDef).toHaveBeenCalledWith('p1');
  });

  it('calls savePenalty when save button is clicked with valid form', async () => {
    const { app, formInitial } = makeApp({ label: 'Verspätung', amount: '5' });
    render(<PenaltyFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    fireEvent.click(screen.getByRole('button', { name: /Strafe hinzufügen/i }));
    await waitFor(() => {
      expect(app.savePenalty).toHaveBeenCalled();
    });
  });
});
