import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PollFormSheet } from './PollFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return {
    useApp,
    // Actions + selector derive from the per-test useApp mock so migrated
    // atoms (TextInput/TextArea via useAppActions/useAppSelector) resolve.
    useAppActions: vi.fn(() => useApp()),
    useAppSelector: (sel: (s: { form: Record<string, unknown> }) => unknown) => sel(useApp().state),
  };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(formOverrides: Record<string, unknown> = {}, errOverrides: Record<string, string> = {}) {
  const setFormErrors = vi.fn();
  const app = {
    state: {
      primaryColor: '#4285F4',
      form: {
        question: '',
        opt0: '',
        opt1: '',
        opt2: '',
        opt3: '',
        multiple: false,
        anonymous: false,
        ...formOverrides,
      },
      formErrors: { question: '', options: '', ...errOverrides },
      busy: null,
    },
    setFormErrors,
    setFormVal: vi.fn(),
    onFormInput: vi.fn(),
    savePoll: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

describe('PollFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheet = {} as never;

  it('renders question field', () => {
    makeApp();
    const app = mockUseApp();
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByPlaceholderText('Worüber soll abgestimmt werden?')).toBeTruthy();
  });

  it('renders option fields (opt0, opt1, opt2, opt3)', () => {
    makeApp();
    const app = mockUseApp();
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByPlaceholderText('Option 1')).toBeTruthy();
    expect(screen.getByPlaceholderText('Option 2')).toBeTruthy();
    expect(screen.getByPlaceholderText('Option 3 (optional)')).toBeTruthy();
    expect(screen.getByPlaceholderText('Option 4 (optional)')).toBeTruthy();
  });

  it('sets question error on blur when question is empty', () => {
    const app = makeApp({ question: '' });
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('Worüber soll abgestimmt werden?');
    fireEvent.blur(input);
    expect(app.setFormErrors).toHaveBeenCalledWith({ question: expect.stringMatching(/\S+/) });
  });

  it('clears question error on blur when question has value', () => {
    const app = makeApp({ question: 'Welche Farbe bevorzugt ihr?' });
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    const input = screen.getByPlaceholderText('Worüber soll abgestimmt werden?');
    fireEvent.blur(input);
    expect(app.setFormErrors).toHaveBeenCalledWith({ question: '' });
  });

  it('shows options error when fewer than 2 options have values and error is set', () => {
    // Only one option filled; pass the error via errOverrides so the component renders it
    const app = makeApp({ opt0: 'Nur eine Option', opt1: '' }, { options: 'Mindestens zwei Optionen angeben.' });
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByRole('alert')).toBeTruthy();
    expect(screen.getByText('Mindestens zwei Optionen angeben.')).toBeTruthy();
  });

  it('sets options error on blur when fewer than 2 options are filled', () => {
    const app = makeApp({ opt0: 'Einzige Option', opt1: '' });
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    const opt1Input = screen.getByPlaceholderText('Option 2');
    fireEvent.blur(opt1Input);
    expect(app.setFormErrors).toHaveBeenCalledWith({ options: expect.stringMatching(/\S+/) });
  });

  it('renders multiple toggle button', () => {
    makeApp();
    const app = mockUseApp();
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Mehrfachauswahl')).toBeTruthy();
  });

  it('renders anonymous toggle button', () => {
    makeApp();
    const app = mockUseApp();
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Anonym')).toBeTruthy();
  });

  it('submit button is disabled when question is empty', () => {
    makeApp({ question: '', opt0: 'A', opt1: 'B' });
    const app = mockUseApp();
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Umfrage erstellen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is disabled when fewer than 2 options are filled', () => {
    makeApp({ question: 'Eine Frage?', opt0: 'Nur eine', opt1: '' });
    const app = mockUseApp();
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Umfrage erstellen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when question and at least 2 options are filled', () => {
    makeApp({ question: 'Welches Datum passt?', opt0: 'Samstag', opt1: 'Sonntag' });
    const app = mockUseApp();
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Umfrage erstellen/i });
    expect(btn).not.toBeDisabled();
  });

  it('clicking multiple toggle calls setFormVal with toggled value', () => {
    const app = makeApp({ multiple: false });
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    const multipleBtn = screen.getByText('Mehrfachauswahl').closest('button');
    expect(multipleBtn).toBeTruthy();
    fireEvent.click(multipleBtn!);
    expect(app.setFormVal).toHaveBeenCalledWith({ multiple: true });
  });

  it('clicking anonymous toggle calls setFormVal with toggled value', () => {
    const app = makeApp({ anonymous: false });
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    const anonymousBtn = screen.getByText('Anonym').closest('button');
    expect(anonymousBtn).toBeTruthy();
    fireEvent.click(anonymousBtn!);
    expect(app.setFormVal).toHaveBeenCalledWith({ anonymous: true });
  });

  it('calls savePoll when create button is clicked', () => {
    const app = makeApp({ question: 'Lieblingsfarbe?', opt0: 'Rot', opt1: 'Blau' });
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByRole('button', { name: /Umfrage erstellen/i }));
    expect(app.savePoll).toHaveBeenCalled();
  });

  it('triggers options validation on blur of first option field', () => {
    const app = makeApp({ opt0: '', opt1: '' });
    render(<PollFormSheet app={app as never} sheet={sheet} />);
    const inputs = document.querySelectorAll('input');
    fireEvent.blur(inputs[1]);
    expect(app.setFormErrors).toHaveBeenCalled();
  });
});
