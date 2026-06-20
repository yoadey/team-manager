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
