import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { App } from './App';

vi.mock('./context/AppContext', () => ({
  AppProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  useApp: vi.fn().mockReturnValue({ state: { primaryColor: '#4285F4', phase: 'loading' } }),
}));

vi.mock('./styles/theme', () => ({
  buildMuiTheme: vi.fn().mockReturnValue({ palette: {}, components: {}, typography: {} }),
}));

vi.mock('./components/Root', () => ({
  Root: () => <div role="status">Loading</div>,
}));

vi.mock('@mui/material/styles', () => ({
  ThemeProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock('@mui/material/CssBaseline', () => ({
  default: () => null,
}));

describe('App', () => {
  it('renders without crashing', () => {
    render(<App />);
    expect(screen.getByRole('status')).toBeTruthy();
  });
});
