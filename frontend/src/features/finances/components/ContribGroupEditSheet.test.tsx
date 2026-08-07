import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ContribGroupEditSheet } from './ContribGroupEditSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return { useApp };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeRows(overrides: Record<string, unknown>[] = [{}, {}]) {
  return overrides.map((o, i) => ({ id: `c${i + 1}`, ...o }));
}

function makeApp(formOverrides: Record<string, unknown> = {}, rows = makeRows()) {
  const app = {
    state: { savingContrib: false },
    editContribGroup: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  const formInitial = { label: 'Monatsbeitrag', amount: '20', description: '', dueDate: '', ...formOverrides };
  return { app, sheet: { formInitial, contribGroupRows: rows } as never };
}

describe('ContribGroupEditSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders prefilled with the group\'s current values', () => {
    const { app, sheet } = makeApp();
    render(<ContribGroupEditSheet app={app as never} sheet={sheet} />);
    const labelInput = screen.getByPlaceholderText('z. B. Mitgliedsbeitrag Januar 2026') as HTMLInputElement;
    expect(labelInput.value).toBe('Monatsbeitrag');
  });

  it('shows label error when label is empty on blur', async () => {
    const { app, sheet } = makeApp({ label: '' });
    render(<ContribGroupEditSheet app={app as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('z. B. Mitgliedsbeitrag Januar 2026');
    fireEvent.blur(input);
    await waitFor(() => {
      expect(screen.getByText('Bezeichnung fehlt.')).toBeTruthy();
    });
  });

  it('submitting fans the update out over every row in the group', async () => {
    const rows = makeRows([{}, {}, {}]);
    const { app, sheet } = makeApp({}, rows);
    render(<ContribGroupEditSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Änderungen speichern'));
    await waitFor(() => {
      expect(app.editContribGroup).toHaveBeenCalledWith(
        rows,
        expect.objectContaining({ label: 'Monatsbeitrag', amount: '20' }),
      );
    });
  });
});
