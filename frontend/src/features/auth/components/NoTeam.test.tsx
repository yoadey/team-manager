import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { NoTeam } from './NoTeam';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp() {
  const openCreateTeam = vi.fn();
  const logout = vi.fn();
  const app = { openCreateTeam, logout };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

describe('NoTeam', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the no-team hint copy', () => {
    makeApp();
    render(<NoTeam />);
    expect(screen.getByText('Noch kein Team')).toBeTruthy();
  });

  it('clicking "create team" calls app.openCreateTeam', () => {
    const app = makeApp();
    render(<NoTeam />);
    const createBtn = screen.getByText('Team anlegen').closest('button');
    expect(createBtn).toBeTruthy();
    fireEvent.click(createBtn!);
    expect(app.openCreateTeam).toHaveBeenCalledTimes(1);
    expect(app.logout).not.toHaveBeenCalled();
  });

  it('clicking "logout" calls app.logout', () => {
    const app = makeApp();
    render(<NoTeam />);
    const logoutBtn = screen.getByText('Abmelden').closest('button');
    expect(logoutBtn).toBeTruthy();
    fireEvent.click(logoutBtn!);
    expect(app.logout).toHaveBeenCalledTimes(1);
    expect(app.openCreateTeam).not.toHaveBeenCalled();
  });
});
