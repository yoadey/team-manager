import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { getErrorMessage, reportActionError, NetworkError, ValidationError, AuthError, ForbiddenError, retryable } from './errors';

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
  it('surfaces a toast with the message', () => {
    const setState = vi.fn();
    const toastMsg = vi.fn();
    reportActionError({ setState, toastMsg }, new Error('kaputt'), 'error.save');
    expect(toastMsg).toHaveBeenCalledTimes(1);
    expect(toastMsg.mock.calls[0]?.[0]).toContain('kaputt');
  });

  it('does not touch state (no busy flag left to manage)', () => {
    const setState = vi.fn();
    const toastMsg = vi.fn();
    reportActionError({ setState, toastMsg }, new Error('kaputt'));
    expect(setState).not.toHaveBeenCalled();
  });

  it('defaults the fallback context', () => {
    const setState = vi.fn();
    const toastMsg = vi.fn();
    reportActionError({ setState, toastMsg }, new Error('x'));
    expect(toastMsg.mock.calls[0]?.[0]).toContain('Aktion fehlgeschlagen');
  });

  it('maps NetworkError to error.network i18n key', () => {
    const setState = vi.fn();
    const toastMsg = vi.fn();
    reportActionError({ setState, toastMsg }, new NetworkError());
    expect(toastMsg.mock.calls[0]?.[0]).toBe('Verbindung zum Service fehlgeschlagen');
  });

  it('maps AuthError to error.login i18n key', () => {
    const setState = vi.fn();
    const toastMsg = vi.fn();
    reportActionError({ setState, toastMsg }, new AuthError());
    expect(toastMsg.mock.calls[0]?.[0]).toBe('Anmeldung fehlgeschlagen');
  });

  it('AuthError triggers onAuthError (session must be cleared)', () => {
    const setState = vi.fn();
    const toastMsg = vi.fn();
    const onAuthError = vi.fn();
    reportActionError({ setState, toastMsg, onAuthError }, new AuthError());
    expect(onAuthError).toHaveBeenCalledTimes(1);
  });

  it('maps ForbiddenError to error.forbidden i18n key without logging out', () => {
    const setState = vi.fn();
    const toastMsg = vi.fn();
    const onAuthError = vi.fn();
    reportActionError({ setState, toastMsg, onAuthError }, new ForbiddenError());
    expect(toastMsg.mock.calls[0]?.[0]).toBe('Dafür fehlt dir die Berechtigung');
    expect(onAuthError).not.toHaveBeenCalled();
  });

  // Regression test: every reportActionError toast (network/auth/forbidden/
  // generic) previously called toastMsg with no kind, so Toast.tsx always
  // rendered the green success checkmark even for "You don't have permission
  // to do that" -- the single most misleading case, since the text reads as
  // a failure while the icon/color scream success.
  it('passes kind: "error" to toastMsg for every error branch', () => {
    const setState = vi.fn();
    const cases: unknown[] = [new Error('kaputt'), new NetworkError(), new AuthError(), new ForbiddenError()];
    for (const err of cases) {
      const toastMsg = vi.fn();
      reportActionError({ setState, toastMsg }, err);
      expect(toastMsg.mock.calls[0]?.[2]).toBe('error');
    }
  });
});

describe('typed error classes', () => {
  it('NetworkError has kind "network" and correct name', () => {
    const err = new NetworkError('down');
    expect(err.kind).toBe('network');
    expect(err.name).toBe('NetworkError');
    expect(err.message).toBe('down');
    expect(err instanceof NetworkError).toBe(true);
    expect(err instanceof Error).toBe(true);
  });

  it('ValidationError carries optional field', () => {
    const err = new ValidationError('too short', 'title');
    expect(err.kind).toBe('validation');
    expect(err.field).toBe('title');
    expect(err instanceof ValidationError).toBe(true);
  });

  it('AuthError has kind "auth"', () => {
    const err = new AuthError();
    expect(err.kind).toBe('auth');
    expect(err instanceof AuthError).toBe(true);
  });

  it('ForbiddenError has kind "forbidden" and is distinct from AuthError', () => {
    const err = new ForbiddenError();
    expect(err.kind).toBe('forbidden');
    expect(err instanceof ForbiddenError).toBe(true);
    expect(err instanceof AuthError).toBe(false);
  });
});

describe('retryable', () => {
  beforeEach(() => {
    // Make setTimeout execute the callback immediately so tests run fast
    // without the real backoff delay and without fake-timer/unhandledRejection issues.
    vi.spyOn(globalThis, 'setTimeout').mockImplementation((cb: TimerHandler) => {
      if (typeof cb === 'function') cb();
      return 0 as unknown as ReturnType<typeof setTimeout>;
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('resolves immediately when fn succeeds on the first try', async () => {
    const fn = vi.fn().mockResolvedValue('ok');
    await expect(retryable(fn)).resolves.toBe('ok');
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('retries up to maxRetries times on NetworkError then resolves', async () => {
    const fn = vi
      .fn()
      .mockRejectedValueOnce(new NetworkError())
      .mockRejectedValueOnce(new NetworkError())
      .mockResolvedValue('recovered');

    await expect(retryable(fn, 2)).resolves.toBe('recovered');
    expect(fn).toHaveBeenCalledTimes(3);
  });

  it('throws NetworkError when all retries are exhausted', async () => {
    const err = new NetworkError('still down');
    const fn = vi.fn().mockRejectedValueOnce(err).mockRejectedValueOnce(err).mockRejectedValueOnce(err);

    await expect(retryable(fn, 2)).rejects.toBe(err);
    expect(fn).toHaveBeenCalledTimes(3);
  });

  it('does not retry non-NetworkError errors', async () => {
    const fn = vi.fn().mockRejectedValue(new ValidationError('bad input'));
    await expect(retryable(fn)).rejects.toBeInstanceOf(ValidationError);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('does not retry AuthError', async () => {
    const fn = vi.fn().mockRejectedValue(new AuthError());
    await expect(retryable(fn)).rejects.toBeInstanceOf(AuthError);
    expect(fn).toHaveBeenCalledTimes(1);
  });
});
