import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ConfirmSheet, SeriesActionSheet, CommentSheet } from './DialogSheets';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

// Mock MUI components to avoid style-injection issues in jsdom
vi.mock('@mui/material/Box', () => ({
  default: ({
    children,
    component,
    ...rest
  }: React.PropsWithChildren<{ component?: string; [k: string]: unknown }>) => {
    const Tag = (component as 'span') || 'div';
    return (
      <Tag data-testid="box" {...(rest as object)}>
        {children}
      </Tag>
    );
  },
}));

vi.mock('@mui/material/ButtonBase', () => ({
  default: ({
    children,
    onClick,
    ...rest
  }: React.PropsWithChildren<{ onClick?: () => void; [k: string]: unknown }>) => (
    <button onClick={onClick} {...(rest as object)}>
      {children}
    </button>
  ),
}));

// Mock tokens so we don't drag in Intl formatting
vi.mock('@/styles/tokens', () => ({
  buildTokens: vi.fn().mockReturnValue({
    primary: '#4285F4',
    primaryContainer: '#E8F0FE',
  }),
  statusMeta: vi.fn().mockReturnValue({
    label: 'Zugesagt',
    color: '#2E7D32',
    bg: '#D7F0D8',
    icon: 'check_circle',
  }),
  NEUTRAL: {
    error: '#BA1A1A',
    errorBg: '#FFDAD6',
    secondary: '#6A6D76',
    faint: '#767676',
    success: '#2E7D32',
    successBg: '#D7F0D8',
  },
}));

// Mock shared UI atoms so tests don't need Material Symbols font etc.
vi.mock('@/components/ui', () => ({
  Sym: ({ name }: { name: string }) => <span data-testid={`sym-${name}`}>{name}</span>,
  Chip: ({ label }: { label: string }) => <span data-testid="chip">{label}</span>,
  PrimaryButton: ({ label, onClick, busy }: { label: string; onClick: () => void; busy?: boolean }) => (
    <button onClick={onClick} disabled={!!busy} data-testid="primary-btn">
      {label}
    </button>
  ),
  inputSx: {},
}));

// ────────────────────────────────────────────────────────────────────────────
// Shared mock app factory
// ────────────────────────────────────────────────────────────────────────────

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      form: { commentText: '' },
      user: { id: 'u1' },
      busy: null,
    },
    cancelConfirm: vi.fn(),
    runConfirm: vi.fn(),
    runEventAction: vi.fn(),
    onFormInput: vi.fn(),
    submitComment: vi.fn(),
    ...overrides,
  };
}

// ────────────────────────────────────────────────────────────────────────────
// ConfirmSheet
// ────────────────────────────────────────────────────────────────────────────

describe('ConfirmSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders message text from sheet.cfg.message', () => {
    const app = makeApp();
    render(
      <ConfirmSheet
        app={app as never}
        sheet={{ type: 'confirm', cfg: { message: 'Möchtest du das wirklich tun?' } }}
      />,
    );
    expect(screen.getByText('Möchtest du das wirklich tun?')).toBeTruthy();
  });

  it('renders confirm label from sheet.cfg.confirmLabel', () => {
    const app = makeApp();
    render(<ConfirmSheet app={app as never} sheet={{ type: 'confirm', cfg: { confirmLabel: 'Ja, löschen' } }} />);
    expect(screen.getByText('Ja, löschen')).toBeTruthy();
  });

  it('Cancel button calls cancelConfirm', () => {
    const app = makeApp();
    render(<ConfirmSheet app={app as never} sheet={{ type: 'confirm', cfg: {} }} />);
    fireEvent.click(screen.getByText('Abbrechen'));
    expect(app.cancelConfirm).toHaveBeenCalledTimes(1);
  });

  it('Confirm button calls runConfirm', () => {
    const app = makeApp();
    render(<ConfirmSheet app={app as never} sheet={{ type: 'confirm', cfg: {} }} />);
    // The confirm button renders the confirmLabel or default 'Bestätigen'
    fireEvent.click(screen.getByText('Bestätigen'));
    expect(app.runConfirm).toHaveBeenCalledTimes(1);
  });

  it('danger mode shows warning icon text', () => {
    const app = makeApp();
    render(<ConfirmSheet app={app as never} sheet={{ type: 'confirm', cfg: { danger: true } }} />);
    // In danger mode the icon box renders the text 'warning'
    expect(screen.getByText('warning')).toBeTruthy();
  });

  it('non-danger default shows "Bist du sicher?" when no message provided', () => {
    const app = makeApp();
    render(<ConfirmSheet app={app as never} sheet={{ type: 'confirm', cfg: {} }} />);
    expect(screen.getByText('Bist du sicher?')).toBeTruthy();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// SeriesActionSheet
// ────────────────────────────────────────────────────────────────────────────

describe('SeriesActionSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const mockEvent = {
    id: 'e1',
    title: 'Training',
    date: '2026-06-25',
    seriesId: 's1',
  };

  it('renders "Nur diesen Termin" option', () => {
    const app = makeApp();
    render(
      <SeriesActionSheet
        app={app as never}
        sheet={{ type: 'seriesAction', action: 'cancel', event: mockEvent as never }}
      />,
    );
    expect(screen.getByText('Nur diesen Termin')).toBeTruthy();
  });

  it('renders "Ganze Serie" option', () => {
    const app = makeApp();
    render(
      <SeriesActionSheet
        app={app as never}
        sheet={{ type: 'seriesAction', action: 'cancel', event: mockEvent as never }}
      />,
    );
    expect(screen.getByText('Ganze Serie')).toBeTruthy();
  });

  it('clicking single option calls runEventAction with scope "single"', () => {
    const app = makeApp();
    render(
      <SeriesActionSheet
        app={app as never}
        sheet={{ type: 'seriesAction', action: 'cancel', event: mockEvent as never }}
      />,
    );
    fireEvent.click(screen.getByText('Nur diesen Termin'));
    expect(app.runEventAction).toHaveBeenCalledWith('cancel', mockEvent, 'single');
  });

  it('clicking series option calls runEventAction with scope "series"', () => {
    const app = makeApp();
    render(
      <SeriesActionSheet
        app={app as never}
        sheet={{ type: 'seriesAction', action: 'cancel', event: mockEvent as never }}
      />,
    );
    fireEvent.click(screen.getByText('Ganze Serie'));
    expect(app.runEventAction).toHaveBeenCalledWith('cancel', mockEvent, 'series');
  });
});

// ────────────────────────────────────────────────────────────────────────────
// CommentSheet
// ────────────────────────────────────────────────────────────────────────────

describe('CommentSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders textarea', () => {
    const app = makeApp();
    render(
      <CommentSheet app={app as never} sheet={{ type: 'comment', status: 'yes', userId: 'u1', name: 'Anna Müller' }} />,
    );
    expect(screen.getByRole('textbox')).toBeTruthy();
  });

  it('renders save button', () => {
    const app = makeApp();
    render(
      <CommentSheet app={app as never} sheet={{ type: 'comment', status: 'yes', userId: 'u1', name: 'Anna Müller' }} />,
    );
    expect(screen.getByTestId('primary-btn')).toBeTruthy();
    expect(screen.getByText('Kommentar speichern')).toBeTruthy();
  });

  it('shows "Dein Kommentar" when sheet.userId matches current user id', () => {
    const app = makeApp();
    render(
      <CommentSheet app={app as never} sheet={{ type: 'comment', status: 'yes', userId: 'u1', name: 'Anna Müller' }} />,
    );
    expect(screen.getByText('Dein Kommentar')).toBeTruthy();
  });

  it('shows "Kommentar für {name}" when sheet.userId differs from current user id', () => {
    const app = makeApp();
    render(
      <CommentSheet app={app as never} sheet={{ type: 'comment', status: 'yes', userId: 'u2', name: 'Anna Müller' }} />,
    );
    expect(screen.getByText('Kommentar für Anna Müller')).toBeTruthy();
  });

  it('shows visibility hint when status is "no"', () => {
    const app = makeApp();
    render(
      <CommentSheet app={app as never} sheet={{ type: 'comment', status: 'no', userId: 'u1', name: 'Anna Müller' }} />,
    );
    expect(screen.getByText(/Absage-Kommentare/)).toBeTruthy();
  });
});
