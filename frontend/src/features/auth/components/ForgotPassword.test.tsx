import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ForgotPassword } from './ForgotPassword';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(overrides: { doForgotPassword?: ReturnType<typeof vi.fn> } = {}) {
  const doForgotPassword = overrides.doForgotPassword ?? vi.fn().mockResolvedValue(true);
  const app = { doForgotPassword };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

function fillAndSubmit(email: string) {
  fireEvent.change(document.getElementById('forgot-password-email')!, { target: { value: email } });
  fireEvent.click(screen.getByText('Link zum Zurücksetzen senden').closest('button')!);
}

describe('ForgotPassword', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('submits the email and shows the "check your email" confirmation', async () => {
    const app = makeApp();
    render(<ForgotPassword onBack={vi.fn()} />);

    fillAndSubmit('someone@example.com');

    expect(app.doForgotPassword).toHaveBeenCalledWith('someone@example.com');
    await waitFor(() => expect(screen.getByText('E-Mails prüfen')).toBeTruthy());
    expect(screen.getByText((content) => content.includes('someone@example.com'))).toBeTruthy();
  });

  it('does not show the confirmation when doForgotPassword fails', async () => {
    const app = makeApp({ doForgotPassword: vi.fn().mockResolvedValue(false) });
    render(<ForgotPassword onBack={vi.fn()} />);

    fillAndSubmit('someone@example.com');

    await waitFor(() => expect(app.doForgotPassword).toHaveBeenCalled());
    expect(screen.queryByText('E-Mails prüfen')).toBeNull();
  });

  it('calls onBack when the back link is clicked', () => {
    const onBack = vi.fn();
    makeApp();
    render(<ForgotPassword onBack={onBack} />);

    fireEvent.click(screen.getByText((content) => content.includes('← Zurück')));
    expect(onBack).toHaveBeenCalled();
  });
});
