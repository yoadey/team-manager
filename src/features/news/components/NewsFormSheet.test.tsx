import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { NewsFormSheet } from './NewsFormSheet';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(formOverrides: Record<string, unknown> = {}, errOverrides: Record<string, string> = {}) {
  const setFormErrors = vi.fn();
  const app = {
    state: {
      primaryColor: '#4285F4',
      form: { title: '', body: '', pinned: false, ...formOverrides },
      formErrors: { title: '', body: '', ...errOverrides },
      busy: null,
    },
    setFormErrors,
    setFormVal: vi.fn(),
    onFormInput: vi.fn(),
    saveNews: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

describe('NewsFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheetCreate = { mode: 'create' } as never;
  const sheetEdit = { mode: 'edit' } as never;

  it('renders title field', () => {
    makeApp();
    const app = mockUseApp();
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    expect(screen.getByPlaceholderText('Überschrift')).toBeTruthy();
  });

  it('renders body field (textarea)', () => {
    makeApp();
    const app = mockUseApp();
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    expect(screen.getByPlaceholderText('Was gibt es Neues?')).toBeTruthy();
  });

  it('sets title error on blur when title is empty', () => {
    const app = makeApp({ title: '' });
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    const input = screen.getByPlaceholderText('Überschrift');
    fireEvent.blur(input);
    expect(app.setFormErrors).toHaveBeenCalledWith({ title: expect.stringMatching(/\S+/) });
  });

  it('clears title error on blur when title has value', () => {
    const app = makeApp({ title: 'Wichtige Neuigkeit' });
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    const input = screen.getByPlaceholderText('Überschrift');
    fireEvent.blur(input);
    expect(app.setFormErrors).toHaveBeenCalledWith({ title: '' });
  });

  it('sets body error on blur when body is empty', () => {
    const app = makeApp({ body: '' });
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    const textarea = screen.getByPlaceholderText('Was gibt es Neues?');
    fireEvent.blur(textarea);
    expect(app.setFormErrors).toHaveBeenCalledWith({ body: expect.stringMatching(/\S+/) });
  });

  it('clears body error on blur when body has value', () => {
    const app = makeApp({ body: 'Das Training findet statt.' });
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    const textarea = screen.getByPlaceholderText('Was gibt es Neues?');
    fireEvent.blur(textarea);
    expect(app.setFormErrors).toHaveBeenCalledWith({ body: '' });
  });

  it('renders pin toggle', () => {
    makeApp();
    const app = mockUseApp();
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    expect(screen.getByText('Oben anpinnen')).toBeTruthy();
  });

  it('clicking pin toggle calls setFormVal with pinned: true when currently false', () => {
    const app = makeApp({ pinned: false });
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    const pinBtn = screen.getByText('Oben anpinnen').closest('button');
    expect(pinBtn).toBeTruthy();
    fireEvent.click(pinBtn!);
    expect(app.setFormVal).toHaveBeenCalledWith({ pinned: true });
  });

  it('clicking pin toggle calls setFormVal with pinned: false when currently true', () => {
    const app = makeApp({ pinned: true });
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    const pinBtn = screen.getByText('Oben anpinnen').closest('button');
    expect(pinBtn).toBeTruthy();
    fireEvent.click(pinBtn!);
    expect(app.setFormVal).toHaveBeenCalledWith({ pinned: false });
  });

  it('submit button is disabled when title is empty', () => {
    makeApp({ title: '', body: 'Irgendein Text' });
    const app = mockUseApp();
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    const btn = screen.getByRole('button', { name: /Veröffentlichen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is disabled when body is empty', () => {
    makeApp({ title: 'Ein Titel', body: '' });
    const app = mockUseApp();
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    const btn = screen.getByRole('button', { name: /Veröffentlichen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when both title and body are filled', () => {
    makeApp({ title: 'Ein Titel', body: 'Inhalt der Neuigkeit' });
    const app = mockUseApp();
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    const btn = screen.getByRole('button', { name: /Veröffentlichen/i });
    expect(btn).not.toBeDisabled();
  });

  it('shows "Veröffentlichen" in create mode', () => {
    makeApp({ title: 'Titel', body: 'Text' });
    const app = mockUseApp();
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    expect(screen.getByRole('button', { name: /Veröffentlichen/i })).toBeTruthy();
  });

  it('shows "Änderungen speichern" in edit mode', () => {
    makeApp({ title: 'Titel', body: 'Text' });
    const app = mockUseApp();
    render(<NewsFormSheet app={app as never} sheet={sheetEdit} />);
    expect(screen.getByRole('button', { name: /Änderungen speichern/i })).toBeTruthy();
  });

  it('shows field error text when errors are present', () => {
    makeApp({}, { title: 'Titel fehlt.', body: '' });
    const app = mockUseApp();
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    expect(screen.getByText('Titel fehlt.')).toBeTruthy();
  });

  it('calls saveNews when publish button is clicked with valid form', () => {
    const app = makeApp({ title: 'Test Titel', body: 'Test Inhalt' });
    render(<NewsFormSheet app={app as never} sheet={sheetCreate} />);
    fireEvent.click(screen.getByRole('button', { name: /Veröffentlichen/i }));
    expect(app.saveNews).toHaveBeenCalled();
  });
});
