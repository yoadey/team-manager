import { describe, it, expect, vi } from 'vitest';
import { getErrorMessage, reportActionError } from './errors';

describe('getErrorMessage', () => {
  it('uses Error.message', () => {
    expect(getErrorMessage(new Error('boom'))).toBe('boom');
  });

  it('uses non-empty strings verbatim', () => {
    expect(getErrorMessage('nope')).toBe('nope');
  });

  it('falls back for empty / unknown values', () => {
    expect(getErrorMessage('')).toBe('Unbekannter Fehler');
    expect(getErrorMessage(null)).toBe('Unbekannter Fehler');
    expect(getErrorMessage({})).toBe('Unbekannter Fehler');
  });
});

describe('reportActionError', () => {
  it('clears busy and surfaces a toast with the message', () => {
    const setState = vi.fn();
    const toastMsg = vi.fn();
    reportActionError({ setState, toastMsg }, new Error('kaputt'), 'error.save');
    expect(setState).toHaveBeenCalledWith({ busy: null });
    expect(toastMsg).toHaveBeenCalledTimes(1);
    expect(toastMsg.mock.calls[0][0]).toContain('kaputt');
  });

  it('defaults the fallback context', () => {
    const setState = vi.fn();
    const toastMsg = vi.fn();
    reportActionError({ setState, toastMsg }, new Error('x'));
    expect(toastMsg.mock.calls[0][0]).toContain('Aktion fehlgeschlagen');
  });
});
