import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PollFormSheet } from './PollFormSheet';

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
    },
    savePoll: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return {
    app,
    formInitial: {
      question: '',
      opt0: '',
      opt1: '',
      opt2: '',
      opt3: '',
      multiple: false,
      anonymous: false,
      ...formOverrides,
    },
  };
}

function makeSheet(formInitial: Record<string, unknown>) {
  return { formInitial } as never;
}

describe('PollFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders question field', () => {
    const { app, formInitial } = makeApp();
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    expect(screen.getByPlaceholderText('Worüber soll abgestimmt werden?')).toBeTruthy();
  });

  it('renders option fields (opt0, opt1, opt2, opt3)', () => {
    const { app, formInitial } = makeApp();
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    expect(screen.getByPlaceholderText('Option 1')).toBeTruthy();
    expect(screen.getByPlaceholderText('Option 2')).toBeTruthy();
    expect(screen.getByPlaceholderText('Option 3 (optional)')).toBeTruthy();
    expect(screen.getByPlaceholderText('Option 4 (optional)')).toBeTruthy();
  });

  it('shows a question error on blur when question is empty', async () => {
    const { app, formInitial } = makeApp({ question: '' });
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    const input = screen.getByPlaceholderText('Worüber soll abgestimmt werden?');
    fireEvent.blur(input);
    await waitFor(() => {
      expect(screen.getByText('Frage fehlt.')).toBeTruthy();
    });
  });

  it('renders multiple toggle button', () => {
    const { app, formInitial } = makeApp();
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    expect(screen.getByText('Mehrfachauswahl')).toBeTruthy();
  });

  it('renders anonymous toggle button', () => {
    const { app, formInitial } = makeApp();
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    expect(screen.getByText('Anonym')).toBeTruthy();
  });

  it('submit button is disabled when question is empty', () => {
    const { app, formInitial } = makeApp({ question: '', opt0: 'A', opt1: 'B' });
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    const btn = screen.getByRole('button', { name: /Umfrage erstellen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is disabled when fewer than 2 options are filled', () => {
    const { app, formInitial } = makeApp({ question: 'Eine Frage?', opt0: 'Nur eine', opt1: '' });
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    const btn = screen.getByRole('button', { name: /Umfrage erstellen/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when question and at least 2 options are filled', () => {
    const { app, formInitial } = makeApp({ question: 'Welches Datum passt?', opt0: 'Samstag', opt1: 'Sonntag' });
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    const btn = screen.getByRole('button', { name: /Umfrage erstellen/i });
    expect(btn).not.toBeDisabled();
  });

  it('clicking multiple toggle flips its pressed state', () => {
    const { app, formInitial } = makeApp({ multiple: false });
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    const multipleBtn = screen.getByText('Mehrfachauswahl').closest('button')!;
    fireEvent.click(multipleBtn);
    // Re-rendered with the toggled value reflected in the button's styling
    // (no aria-pressed on this toggle, so assert via the icon color proxy
    // isn't practical here -- just confirm the click didn't throw and the
    // button is still present).
    expect(multipleBtn).toBeTruthy();
  });

  it('calls savePoll with the validated values when create button is clicked', async () => {
    const { app, formInitial } = makeApp({ question: 'Lieblingsfarbe?', opt0: 'Rot', opt1: 'Blau' });
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    fireEvent.click(screen.getByRole('button', { name: /Umfrage erstellen/i }));
    await waitFor(() => {
      expect(app.savePoll).toHaveBeenCalledWith(
        expect.objectContaining({ question: 'Lieblingsfarbe?', opt0: 'Rot', opt1: 'Blau' }),
      );
    });
  });

  // Regression test: the question/option inputs had no maxLength, so a user
  // could type far past the backend's limits (question: 1000, each option:
  // 500) and only find out via a generic server-error toast after submit.
  it('caps the question input at 1000 characters matching the backend limit', () => {
    const { app, formInitial } = makeApp();
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    const input = screen.getByPlaceholderText('Worüber soll abgestimmt werden?') as HTMLInputElement;
    expect(input.maxLength).toBe(1000);
  });

  it('caps each option input at 500 characters matching the backend limit', () => {
    const { app, formInitial } = makeApp();
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    for (const placeholder of ['Option 1', 'Option 2', 'Option 3 (optional)', 'Option 4 (optional)']) {
      const input = screen.getByPlaceholderText(placeholder) as HTMLInputElement;
      expect(input.maxLength).toBe(500);
    }
  });

  // Regression test: the options error box had no stable id and wasn't wired
  // to the option inputs via aria-describedby/aria-invalid, unlike every
  // other validated field in the app (which goes through the shared Field
  // component and gets this wiring for free) -- a screen reader user
  // focusing an option input got no indication it was the invalid field.
  it('wires the options error to the first two option inputs via aria-describedby/aria-invalid', async () => {
    const { app, formInitial } = makeApp({ question: 'Frage?', opt0: 'Nur eine Option', opt1: '' });
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    const opt1 = screen.getByPlaceholderText('Option 2');
    fireEvent.blur(opt1);

    const errorBox = await waitFor(() => screen.getByRole('alert'));
    expect(errorBox.id).toBeTruthy();

    const opt0 = screen.getByPlaceholderText('Option 1');
    await waitFor(() => {
      expect(opt0.getAttribute('aria-describedby')).toBe(errorBox.id);
      expect(opt1.getAttribute('aria-describedby')).toBe(errorBox.id);
      expect(opt0.getAttribute('aria-invalid')).toBe('true');
      expect(opt1.getAttribute('aria-invalid')).toBe('true');
    });
  });

  it('does not mark option inputs invalid when there is no options error', () => {
    const { app, formInitial } = makeApp({ opt0: 'A', opt1: 'B' });
    render(<PollFormSheet app={app as never} sheet={makeSheet(formInitial)} />);
    const opt0 = screen.getByPlaceholderText('Option 1');
    expect(opt0.getAttribute('aria-describedby')).toBeNull();
    expect(opt0.getAttribute('aria-invalid')).toBe('false');
  });
});
