import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { SheetHost } from './SheetHost';

const mocks = vi.hoisted(() => ({
  useApp: vi.fn(),
  isPageSheet: vi.fn(),
  renderSheet: vi.fn(),
  sheetMeta: vi.fn(),
  captureError: vi.fn(),
}));

vi.mock('@/context/AppContext', () => ({
  useApp: mocks.useApp,
  isPageSheet: mocks.isPageSheet,
}));

vi.mock('@/sheets', () => ({
  renderSheet: mocks.renderSheet,
  sheetMeta: mocks.sheetMeta,
}));

vi.mock('@/monitoring', () => ({ captureError: (...args: unknown[]) => mocks.captureError(...args) }));

const mockUseApp = mocks.useApp;
const mockIsPageSheet = mocks.isPageSheet;
const mockSheetMetaFn = mocks.sheetMeta;

function makeApp(sheet: unknown = null) {
  return {
    state: {
      sheet,
      primaryColor: '#4285F4',
    },
    closeSheet: vi.fn(),
  };
}

describe('SheetHost', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockIsPageSheet.mockReturnValue(false);
    mocks.renderSheet.mockReturnValue(<div>Sheet Content</div>);
    mockSheetMetaFn.mockReturnValue({ title: 'Teams', subtitle: null, hasBack: false, onBack: null });
  });

  it('renders nothing when no sheet', () => {
    mockUseApp.mockReturnValue(makeApp(null));
    const { container } = render(<SheetHost />);
    expect(container.firstChild).toBeNull();
  });

  it('renders nothing for page sheets', () => {
    mockIsPageSheet.mockReturnValue(true);
    mockUseApp.mockReturnValue(makeApp({ type: 'eventDetail' }));
    const { container } = render(<SheetHost />);
    expect(container.firstChild).toBeNull();
  });

  it('renders modal with title for modal sheets', () => {
    mockUseApp.mockReturnValue(makeApp({ type: 'teams' }));
    render(<SheetHost />);
    expect(screen.getByText('Teams')).toBeTruthy();
    expect(screen.getByText('Sheet Content')).toBeTruthy();
  });

  it('renders with subtitle when provided', () => {
    mockSheetMetaFn.mockReturnValue({ title: 'Profil', subtitle: 'Dein Konto', hasBack: false, onBack: null });
    mockUseApp.mockReturnValue(makeApp({ type: 'profile' }));
    render(<SheetHost />);
    expect(screen.getByText('Dein Konto')).toBeTruthy();
  });

  it('renders back button when hasBack is true', () => {
    mockSheetMetaFn.mockReturnValue({ title: 'Detail', subtitle: null, hasBack: true, onBack: vi.fn() });
    mockUseApp.mockReturnValue(makeApp({ type: 'comment' }));
    render(<SheetHost />);
    expect(screen.getByLabelText('Zurück')).toBeTruthy();
  });

  it('renders close button', () => {
    mockUseApp.mockReturnValue(makeApp({ type: 'teams' }));
    render(<SheetHost />);
    expect(screen.getByLabelText('Schließen')).toBeTruthy();
  });

  // Regression test: a crashing modal sheet (e.g. NotificationsSheet
  // dereferencing an optional field the mock/real backend didn't populate)
  // used to have no ErrorBoundary of its own, unlike page sheets in
  // AppShell -- the throw would bubble past the modal entirely and blank the
  // whole app, caught only by the single top-level boundary. Now it must be
  // contained to the sheet body while the modal chrome (title, close button)
  // stays intact.
  it('contains a crashing sheet body in its own error boundary', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    // renderSheet is a dispatcher that returns a JSX element descriptor
    // without executing the sheet component's body -- React only invokes
    // that body during its own render pass (which the ErrorBoundary wraps),
    // so the throw must happen there too, not inside renderSheet itself.
    function ThrowingSheet(): never {
      throw new Error('boom: sheet crashed');
    }
    mocks.renderSheet.mockImplementation(() => <ThrowingSheet />);
    mockUseApp.mockReturnValue(makeApp({ type: 'notifications' }));

    act(() => {
      render(<SheetHost />);
    });

    expect(screen.getByText('Teams')).toBeTruthy(); // modal title chrome survives
    expect(screen.getByLabelText('Schließen')).toBeTruthy(); // close button survives
    expect(screen.getByText('Neu versuchen')).toBeTruthy(); // fallback retry control
    expect(mocks.captureError).toHaveBeenCalled();
    spy.mockRestore();
  });
});
