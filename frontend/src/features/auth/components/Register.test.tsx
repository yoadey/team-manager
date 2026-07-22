import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { Register } from './Register';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(overrides: { doRegister?: ReturnType<typeof vi.fn>; doResendVerification?: ReturnType<typeof vi.fn> } = {}) {
  const doRegister = overrides.doRegister ?? vi.fn().mockResolvedValue(true);
  const doResendVerification = overrides.doResendVerification ?? vi.fn().mockResolvedValue(true);
  const app = { doRegister, doResendVerification };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

function fillAndSubmit(email: string, password: string, confirmPassword: string) {
  fireEvent.change(document.getElementById('register-email')!, { target: { value: email } });
  fireEvent.change(document.getElementById('register-password')!, { target: { value: password } });
  fireEvent.change(document.getElementById('register-confirm-password')!, { target: { value: confirmPassword } });
  fireEvent.click(screen.getByText('Konto erstellen').closest('button')!);
}

describe('Register', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('rejects a mismatched password confirmation without calling doRegister', () => {
    const app = makeApp();
    render(<Register onBack={vi.fn()} />);

    fillAndSubmit('new@example.com', 'longenoughpassword', 'different-password');

    expect(screen.getByRole('alert').textContent).toContain('Die Passwörter stimmen nicht überein.');
    expect(app.doRegister).not.toHaveBeenCalled();
  });

  it('rejects a too-short password without calling doRegister', () => {
    const app = makeApp();
    render(<Register onBack={vi.fn()} />);

    fillAndSubmit('new@example.com', 'short', 'short');

    expect(screen.getByRole('alert').textContent).toContain('Das Passwort muss mindestens 8 Zeichen lang sein.');
    expect(app.doRegister).not.toHaveBeenCalled();
  });

  it('submits valid input and shows the "check your email" confirmation', async () => {
    const app = makeApp();
    render(<Register onBack={vi.fn()} />);

    fillAndSubmit('new@example.com', 'longenoughpassword', 'longenoughpassword');

    expect(app.doRegister).toHaveBeenCalledWith('new@example.com', 'longenoughpassword');
    await waitFor(() => expect(screen.getByText('E-Mails prüfen')).toBeTruthy());
    expect(screen.getByText((content) => content.includes('new@example.com'))).toBeTruthy();
  });

  it('does not show the confirmation when doRegister fails', async () => {
    const app = makeApp({ doRegister: vi.fn().mockResolvedValue(false) });
    render(<Register onBack={vi.fn()} />);

    fillAndSubmit('new@example.com', 'longenoughpassword', 'longenoughpassword');

    await waitFor(() => expect(app.doRegister).toHaveBeenCalled());
    expect(screen.queryByText('E-Mails prüfen')).toBeNull();
  });

  it('clicking resend after a successful registration calls doResendVerification', async () => {
    const app = makeApp();
    render(<Register onBack={vi.fn()} />);

    fillAndSubmit('new@example.com', 'longenoughpassword', 'longenoughpassword');
    await waitFor(() => expect(screen.getByText('E-Mails prüfen')).toBeTruthy());

    fireEvent.click(screen.getByText('Bestätigungs-E-Mail erneut senden'));
    await waitFor(() => expect(app.doResendVerification).toHaveBeenCalledWith('new@example.com'));
  });

  it('calls onBack when the back link is clicked', () => {
    const onBack = vi.fn();
    makeApp();
    render(<Register onBack={onBack} />);

    fireEvent.click(screen.getByText((content) => content.includes('Zurück')));
    expect(onBack).toHaveBeenCalled();
  });
});
