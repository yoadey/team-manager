import { captureException } from '@/monitoring';
import { t } from '@/i18n';

/** Extracts a human-readable message from an unknown thrown value. */
export function getErrorMessage(err: unknown): string {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'string' && err) return err;
  return t('error.unknown');
}

interface ActionReporter {
  /** Clears any in-flight `busy` flag so the UI is never stuck. */
  setState: (patch: { busy: null }) => void;
  toastMsg: (m: string) => void;
}

/**
 * Standard handling for a failed user-triggered action: report to monitoring,
 * release the busy state so dialogs/buttons recover, and surface a toast.
 * `fallbackKey` is an i18n key for the leading context (e.g. `error.save`).
 */
export function reportActionError(reporter: ActionReporter, err: unknown, fallbackKey = 'error.action'): void {
  captureException(err);
  reporter.setState({ busy: null });
  reporter.toastMsg(`${t(fallbackKey)}: ${getErrorMessage(err)}`);
}
