import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PrivacyPanel } from './PrivacyPanel';
import { AuthError } from '@/utils/errors';
import type { AppContextValue } from '@/context/AppContext';

const MOCK_USER = { id: 'u1', name: 'Max Mustermann', email: 'max@example.com', photo: null, avatarColor: '#1565C0' };

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: { user: MOCK_USER, ...overrides },
    logout: vi.fn(),
    deleteAccount: vi.fn().mockResolvedValue(undefined),
    exportMyData: vi.fn().mockResolvedValue(undefined),
    toastMsg: vi.fn(),
    setState: vi.fn(),
  } as unknown as AppContextValue;
}

describe('PrivacyPanel', () => {
  it('clicking "export my data" calls exportMyData', () => {
    const app = makeApp();
    render(<PrivacyPanel app={app} />);
    fireEvent.click(screen.getByText('Meine Daten exportieren'));
    expect(app.exportMyData).toHaveBeenCalledTimes(1);
  });

  it('account deletion requires the matching email before it can be confirmed', () => {
    const app = makeApp();
    render(<PrivacyPanel app={app} />);

    // Reveal the confirm flow.
    fireEvent.click(screen.getByText('Konto löschen'));
    const confirmBtn = screen.getByText('Endgültig löschen').closest('button')!;
    expect(confirmBtn.disabled).toBe(true);

    // A wrong email keeps it disabled.
    const input = screen.getByPlaceholderText('max@example.com');
    fireEvent.change(input, { target: { value: 'wrong@example.com' } });
    expect(confirmBtn.disabled).toBe(true);

    // The matching email (case-insensitive) enables it.
    fireEvent.change(input, { target: { value: 'MAX@example.com' } });
    expect(confirmBtn.disabled).toBe(false);
  });

  it('confirming account deletion calls deleteAccount with the typed email', () => {
    const app = makeApp();
    render(<PrivacyPanel app={app} />);

    fireEvent.click(screen.getByText('Konto löschen'));
    fireEvent.change(screen.getByPlaceholderText('max@example.com'), {
      target: { value: 'max@example.com' },
    });
    fireEvent.click(screen.getByText('Endgültig löschen'));

    expect(app.deleteAccount).toHaveBeenCalledWith('max@example.com');
  });

  it('exportMyData triggers logout on a 401 (expired session)', async () => {
    const app = makeApp();
    (app.exportMyData as ReturnType<typeof vi.fn>).mockRejectedValue(new AuthError());
    render(<PrivacyPanel app={app} />);

    fireEvent.click(screen.getByText('Meine Daten exportieren'));

    await waitFor(() => expect(app.logout).toHaveBeenCalledTimes(1));
  });

  it('account deletion triggers logout on a 401 instead of showing the wrong-email error', async () => {
    const app = makeApp();
    (app.deleteAccount as ReturnType<typeof vi.fn>).mockRejectedValue(new AuthError());
    render(<PrivacyPanel app={app} />);

    fireEvent.click(screen.getByText('Konto löschen'));
    fireEvent.change(screen.getByPlaceholderText('max@example.com'), {
      target: { value: 'max@example.com' },
    });
    fireEvent.click(screen.getByText('Endgültig löschen'));

    await waitFor(() => expect(app.logout).toHaveBeenCalledTimes(1));
    expect(screen.queryByText('Konto konnte nicht gelöscht werden. Stimmt die E-Mail-Adresse?')).toBeNull();
  });

  it('account deletion shows the wrong-email error for a non-auth failure', async () => {
    const app = makeApp();
    (app.deleteAccount as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('email mismatch'));
    render(<PrivacyPanel app={app} />);

    fireEvent.click(screen.getByText('Konto löschen'));
    fireEvent.change(screen.getByPlaceholderText('max@example.com'), {
      target: { value: 'max@example.com' },
    });
    fireEvent.click(screen.getByText('Endgültig löschen'));

    await waitFor(() =>
      expect(screen.getByText('Konto konnte nicht gelöscht werden. Stimmt die E-Mail-Adresse?')).toBeTruthy(),
    );
    expect(app.logout).not.toHaveBeenCalled();
  });
});
