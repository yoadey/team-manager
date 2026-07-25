import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { RsvpCountdown } from './RsvpCountdown';
import { setLocale } from '@/i18n';

describe('RsvpCountdown', () => {
  beforeEach(() => {
    setLocale('de');
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-01T12:00:00.000Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders nothing when the deadline is more than 24h away', () => {
    const { container } = render(<RsvpCountdown deadline="2026-07-03T12:00:00.000Z" />);
    expect(container.textContent).toBe('');
  });

  it('renders nothing once the deadline has already passed', () => {
    const { container } = render(<RsvpCountdown deadline="2026-07-01T11:00:00.000Z" />);
    expect(container.textContent).toBe('');
  });

  it('shows a countdown once less than 24h remain, with hours and minutes', () => {
    render(<RsvpCountdown deadline="2026-07-02T03:30:00.000Z" />);
    // 15h30m remaining from 12:00 to next day 03:30.
    expect(screen.getByText(/15 Std\. 30 Min\./)).toBeTruthy();
  });

  it('shows minutes only once under an hour remains', () => {
    render(<RsvpCountdown deadline="2026-07-01T12:45:00.000Z" />);
    expect(screen.getByText(/45 Min\./)).toBeTruthy();
    expect(screen.queryByText(/Std\./)).toBeNull();
  });
});
