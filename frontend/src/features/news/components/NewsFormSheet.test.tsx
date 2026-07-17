import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { NewsFormSheet } from './NewsFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return { useApp };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(formOverrides: Record<string, unknown> = {}) {
  const app = {
    state: {
      primaryColor: '#4285F4',
      busy: null,
    },
    saveNews: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return { app, formInitial: { title: '', body: '', pinned: false, ...formOverrides } };
}

function makeSheet(mode: 'create' | 'edit', formInitial: Record<string, unknown>) {
  return { mode, formInitial } as never;
}

describe('NewsFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders title field', () => {
    const { app, formInitial } = makeApp();
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.getByPlaceholderText('Überschrift')).toBeTruthy();
  });

  it('renders body field (textarea)', () => {
    const { app, formInitial } = makeApp();
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.getByPlaceholderText('Was gibt es Neues?')).toBeTruthy();
  });

  it('sets title error on blur when title is empty', async () => {
    const { app, formInitial } = makeApp({ title: '' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const input = screen.getByPlaceholderText('Überschrift');
    fireEvent.blur(input);
    await waitFor(() => {
      expect(screen.getByText('Titel fehlt.')).toBeTruthy();
    });
  });

  it('clears title error on blur when title has value', async () => {
    const { app, formInitial } = makeApp({ title: 'Wichtige Neuigkeit' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const input = screen.getByPlaceholderText('Überschrift');
    fireEvent.blur(input);
    await waitFor(() => {
      expect(screen.queryByText('Titel fehlt.')).toBeNull();
    });
  });

  it('sets body error on blur when body is empty', async () => {
    const { app, formInitial } = makeApp({ body: '' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const textarea = screen.getByPlaceholderText('Was gibt es Neues?');
    fireEvent.blur(textarea);
    await waitFor(() => {
      expect(screen.getByText('Text fehlt.')).toBeTruthy();
    });
  });

  it('clears body error on blur when body has value', async () => {
    const { app, formInitial } = makeApp({ body: 'Das Training findet statt.' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const textarea = screen.getByPlaceholderText('Was gibt es Neues?');
    fireEvent.blur(textarea);
    await waitFor(() => {
      expect(screen.queryByText('Text fehlt.')).toBeNull();
    });
  });

  it('caps title and body inputs matching the backend limits', () => {
    const { app, formInitial } = makeApp();
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const title = screen.getByPlaceholderText('Überschrift') as HTMLInputElement;
    const body = screen.getByPlaceholderText('Was gibt es Neues?') as HTMLTextAreaElement;
    expect(title.maxLength).toBe(255);
    expect(body.maxLength).toBe(10000);
  });

  it('renders pin toggle', () => {
    const { app, formInitial } = makeApp();
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.getByText('Oben anpinnen')).toBeTruthy();
  });

  it('clicking pin toggle updates the toggle switch', () => {
    const { app, formInitial } = makeApp({ pinned: false });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const pinBtn = screen.getByText('Oben anpinnen').closest('button')!;
    fireEvent.click(pinBtn);
    expect(pinBtn.getAttribute('aria-checked')).toBe('true');
  });

  it('submit button is disabled when title is empty', () => {
    const { app, formInitial } = makeApp({ title: '', body: 'Irgendein Text' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const btn = screen.getByRole('button', { name: /Veröffentlichen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is disabled when body is empty', () => {
    const { app, formInitial } = makeApp({ title: 'Ein Titel', body: '' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const btn = screen.getByRole('button', { name: /Veröffentlichen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when both title and body are filled', () => {
    const { app, formInitial } = makeApp({ title: 'Ein Titel', body: 'Inhalt der Neuigkeit' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    const btn = screen.getByRole('button', { name: /Veröffentlichen/i });
    expect(btn).not.toBeDisabled();
  });

  it('shows "Veröffentlichen" in create mode', () => {
    const { app, formInitial } = makeApp({ title: 'Titel', body: 'Text' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    expect(screen.getByRole('button', { name: /Veröffentlichen/i })).toBeTruthy();
  });

  it('shows "Änderungen speichern" in edit mode', () => {
    const { app, formInitial } = makeApp({ title: 'Titel', body: 'Text' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('edit', formInitial)} />);
    expect(screen.getByRole('button', { name: /Änderungen speichern/i })).toBeTruthy();
  });

  it('calls saveNews when publish button is clicked with valid form', async () => {
    const { app, formInitial } = makeApp({ title: 'Test Titel', body: 'Test Inhalt' });
    render(<NewsFormSheet app={app as never} sheet={makeSheet('create', formInitial)} />);
    fireEvent.click(screen.getByRole('button', { name: /Veröffentlichen/i }));
    await waitFor(() => {
      expect(app.saveNews).toHaveBeenCalled();
    });
  });
});
