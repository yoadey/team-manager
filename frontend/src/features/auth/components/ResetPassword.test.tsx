import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ResetPassword } from './ResetPassword';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(overrides: { doResetPassword?: ReturnType<typeof vi.fn> } = {}) {
  const doResetPassword = overrides.doResetPassword ?? vi.fn().mockResolvedValue(true);
  const app = { doResetPassword };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

function fillAndSubmit(password: string, confirmPassword: string) {
  fireEvent.change(document.getElementById('reset-password-password')!, { target: { value: password } });
  fireEvent.change(document.getElementById('reset-password-confirm')!, { target: { value: confirmPassword } });
  fireEvent.click(screen.getByText('Neues Passwort speichern').closest('button')!);
}

describe('ResetPassword', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('rejects a mismatched password confirmation without calling doResetPassword', () => {
    const app = makeApp();
    render(<ResetPassword token="raw-token" />);

    fillAndSubmit('longenoughpassword', 'different-password');

    expect(screen.getByRole('alert').textContent).toContain('Die Passwörter stimmen nicht überein.');
    expect(app.doResetPassword).not.toHaveBeenCalled();
  });

  it('rejects a too-short password without calling doResetPassword', () => {
    const app = makeApp();
    render(<ResetPassword token="raw-token" />);

    fillAndSubmit('short', 'short');

    expect(screen.getByRole('alert').textContent).toContain('Das Passwort muss mindestens 8 Zeichen lang sein.');
    expect(app.doResetPassword).not.toHaveBeenCalled();
  });

  it('submits the token and new password', async () => {
    const app = makeApp();
    render(<ResetPassword token="raw-token" />);

    fillAndSubmit('brandnewpassword', 'brandnewpassword');

    await waitFor(() => expect(app.doResetPassword).toHaveBeenCalledWith('raw-token', 'brandnewpassword'));
  });
});
