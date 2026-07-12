import { captureException } from '@/monitoring';
import { t } from '@/i18n';
import { clearBusyIfOwned } from '@/utils/forms';

/** Extracts a human-readable message from an unknown thrown value. */
export function getErrorMessage(err: unknown): string {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'string' && err) return err;
  return t('error.unknown');
}

// ---------------------------------------------------------------------------
// Typed error classes — P2.11
// ---------------------------------------------------------------------------

/** Thrown when a remote call fails due to connectivity or HTTP 5xx. */
export class NetworkError extends Error {
  readonly kind = 'network' as const;
  constructor(message?: string) {
    super(message ?? 'Network error');
    this.name = 'NetworkError';
  }
}

/** Thrown when the server rejects input (HTTP 400 / 422). */
export class ValidationError extends Error {
  readonly kind = 'validation' as const;
  readonly field?: string;
  constructor(message: string, field?: string) {
    super(message);
    this.name = 'ValidationError';
    this.field = field;
  }
}

/** Thrown when the session has expired or credentials are invalid (HTTP 401). */
export class AuthError extends Error {
  readonly kind = 'auth' as const;
  constructor(message?: string) {
    super(message ?? 'Authentication error');
    this.name = 'AuthError';
  }
}

/**
 * Thrown when the authenticated user lacks permission for the action (HTTP
 * 403). Distinct from AuthError: the session itself is still valid, so the
 * user must not be logged out — only told the action isn't allowed.
 */
export class ForbiddenError extends Error {
  readonly kind = 'forbidden' as const;
  constructor(message?: string) {
    super(message ?? 'Forbidden');
    this.name = 'ForbiddenError';
  }
}

// ---------------------------------------------------------------------------
// Retry helper — P2.12
// Only use for idempotent read-like operations; never wrap mutations that
// lack a server-side idempotency key.
// ---------------------------------------------------------------------------

/**
 * Retries `fn` up to `maxRetries` times when it throws a `NetworkError`.
 * Backoff: 300 ms × 2^attempt (300 ms, 600 ms for the default 2 retries).
 * All other error types are re-thrown immediately without retrying.
 */
export async function retryable<T>(fn: () => Promise<T>, maxRetries = 2): Promise<T> {
  let attempt = 0;
  for (;;) {
    try {
      return await fn();
    } catch (err) {
      if (!(err instanceof NetworkError) || attempt >= maxRetries) throw err;
      await new Promise<void>((resolve) => setTimeout(resolve, 300 * 2 ** attempt));
      attempt++;
    }
  }
}

// ---------------------------------------------------------------------------
// Action error reporter
// ---------------------------------------------------------------------------

interface ActionReporter {
  /** Clears any in-flight `busy` flag so the UI is never stuck. */
  setState: (patch: { busy: null }) => void;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'error') => void;
  /**
   * Called when an AuthError (HTTP 401 — session expired/invalid) is caught.
   * Use to trigger logout and redirect to the login screen so the user is
   * never left in a half-authenticated state. Not called for ForbiddenError
   * (HTTP 403), since the session is still valid there.
   */
  onAuthError?: () => void;
  /**
   * Reads the live app state so busyOwner can be checked against the CURRENT
   * `busy` value, not a stale one captured at call time. Required together
   * with busyOwner; omit both if the failing action never set `busy` itself
   * (e.g. a background load) -- see busyOwner's doc comment for why.
   */
  S?: () => { busy: string | null };
  /**
   * The exact value this action's own setState({ busy: '...' }) used before
   * starting its request. When set (with S), busy is only cleared if it
   * still holds this value (clearBusyIfOwned) -- otherwise a DIFFERENT,
   * still-in-flight action that has since taken over `busy` would have its
   * spinner/disabled state incorrectly cleared by this one's failure.
   *
   * Omit for reporters whose action never sets `busy` in the first place
   * (background loads, votes/toggles using their own inFlight Set guard):
   * clearing busy unconditionally here would still risk clobbering an
   * unrelated in-flight action, so those reporters don't touch `busy` at all
   * rather than falling back to the old unconditional clear.
   */
  busyOwner?: string;
}

/**
 * Standard handling for a failed user-triggered action: report to monitoring,
 * release the busy state so dialogs/buttons recover, and surface a toast.
 * Typed errors (NetworkError, AuthError) map to dedicated i18n keys so the
 * message is clean and localised. Other errors append the raw message for
 * debuggability.
 * `fallbackKey` is an i18n key for the leading context (e.g. `error.save`).
 */
export function reportActionError(reporter: ActionReporter, err: unknown, fallbackKey = 'error.action'): void {
  captureException(err);
  if (reporter.S && reporter.busyOwner) {
    clearBusyIfOwned(reporter.S, reporter.setState, reporter.busyOwner);
  }

  if (err instanceof NetworkError) {
    reporter.toastMsg(t('error.network'), undefined, 'error');
  } else if (err instanceof AuthError) {
    reporter.toastMsg(t('error.login'), undefined, 'error');
    reporter.onAuthError?.();
  } else if (err instanceof ForbiddenError) {
    reporter.toastMsg(t('error.forbidden'), undefined, 'error');
  } else {
    reporter.toastMsg(`${t(fallbackKey)}: ${getErrorMessage(err)}`, undefined, 'error');
  }
}
