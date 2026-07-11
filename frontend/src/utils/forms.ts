import type { AppState } from '@/context/AppContext';

/**
 * A typed, read-only view over the global form buffer for a single sheet.
 *
 * The app keeps every in-progress form's values in one shared, heterogeneous
 * `state.form` object (the live editing buffer for whichever sheet is open).
 * That bag is intentionally untyped at the store level, but each sheet knows
 * the concrete shape it edits. `formValues` lets a sheet read the buffer
 * through its own interface instead of the untyped `any` fallback, so field
 * access is checked and renames are caught by the compiler.
 *
 * The result is `Partial<T>` because the buffer may be empty (before the form
 * is initialised) or hold only a subset of fields while the user fills it in.
 * Use `app.setFormVal(patch)` to write back — `patch` can be typed as
 * `Partial<T>` at the call site for symmetric safety.
 */
export function formValues<T extends Record<string, unknown>>(state: Pick<AppState, 'form'>): Partial<T> {
  return state.form as Partial<T>;
}

/**
 * Clears `state.busy` only if it still holds the exact value this action set
 * it to, before starting its own request.
 *
 * `busy` is one shared string across the whole app (every Save button reads
 * `busy === 'save'`, every delete flow sets `busy === 'delete'`, etc.), so an
 * unconditional `setState({ busy: null })` after an awaited request is a
 * race: if a *different* kind of action (e.g. a delete) started and is still
 * in flight when this request resolves, blindly nulling `busy` here would
 * incorrectly re-enable that other action's UI (spinner/disabled state)
 * while it's still pending, inviting a double-submit. Two actions of the
 * *same* kind can't overlap this way in the first place — they'd share one
 * `busy` value and the second one's trigger would already be disabled — so
 * this only needs to guard against a same-string self-clobber, not track a
 * full per-request identity.
 */
export function clearBusyIfOwned(S: () => Pick<AppState, 'busy'>, setState: SetBusy, owner: string): void {
  if (S().busy === owner) setState({ busy: null });
}

type SetBusy = (patch: { busy: null }) => void;
