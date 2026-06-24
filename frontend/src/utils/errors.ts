import { captureException } from '@/monitoring';
import { t } from '@/i18n';

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

/** Thrown when the session has expired or credentials are invalid (HTTP 401/403). */
export class AuthError extends Error {
  readonly kind = 'auth' as const;
  constructor(message?: string) {
    super(message ?? 'Authentication error');
    this.name = 'AuthError';
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
  toastMsg: (m: string) => void;
  /**
   * Called when an AuthError (HTTP 401/403) is caught. Use to trigger logout
   * and redirect to the login screen so the user is never left in a half-
   * authenticated state.
   */
  onAuthError?: () => void;
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
  reporter.setState({ busy: null });

  if (err instanceof NetworkError) {
    reporter.toastMsg(t('error.network'));
  } else if (err instanceof AuthError) {
    reporter.toastMsg(t('error.login'));
    reporter.onAuthError?.();
  } else {
    reporter.toastMsg(`${t(fallbackKey)}: ${getErrorMessage(err)}`);
  }
}
