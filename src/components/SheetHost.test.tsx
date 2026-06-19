import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SheetHost } from './SheetHost';

const mocks = vi.hoisted(() => ({
  useApp: vi.fn(),
  isPageSheet: vi.fn(),
  renderSheet: vi.fn(),
  sheetMeta: vi.fn(),
}));

vi.mock('@/context/AppContext', () => ({
  useApp: mocks.useApp,
  isPageSheet: mocks.isPageSheet,
}));

vi.mock('@/sheets', () => ({
  renderSheet: mocks.renderSheet,
  sheetMeta: mocks.sheetMeta,
}));

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
});
