## Context

`frontend/src/i18n/index.ts` exposes `t()` as a plain function reading a
module-level `activeLocale`. `LocaleProvider` (`i18n/LocaleProvider.tsx`)
wraps the app and re-renders on `setLocale()`, but since `<App/>` is
passed to it as a `children` prop created once at the root
(`main.tsx:57-68`), React sees an unchanged element reference for
everything below `LocaleProvider` and bails out of re-rendering that
subtree. Only components that call the `useLocale()` hook directly
re-render on locale change; every other component's `t()` calls read
stale strings until an unrelated re-render happens to touch them.

## Goals

- A locale switch must be visible everywhere in the UI without requiring
  the user to navigate or trigger an unrelated re-render.
- Keep the change proportional: hundreds of call sites use bare `t()`
  today; the fix should not require a mechanical rewrite of every one of
  them if avoidable.
- Also close the two adjacent i18n gaps (plural double-param footgun,
  untyped keys) discovered in the same review pass, since they touch the
  same small set of files.

## Decisions

- **Force a full remount on locale change via `<App key={locale}/>` at
  the root**, rather than converting `t()` into a hook that every call
  site must adopt. A remount is a bigger hammer than a
  `useSyncExternalStore`-based `useT()` hook, but it requires touching
  only `main.tsx`, is trivially correct (nothing can miss the
  re-subscription), and the app's own `AppContext` reducer/state design
  already tolerates a full subtree remount (e.g. it happens today on
  team switch in some flows). If a future profiling pass shows the
  remount cost is visible to users (e.g. losing scroll position or
  transient UI state on language switch, which is rare enough not to be
  disruptive), revisit with a scoped `useT()` hook instead — noted as a
  fallback, not pursued now to keep this change's diff proportional to
  the bug.
- **Auto-derive `n` from `count` when `n` is omitted**, rather than
  requiring both at every call site. `count` is already mandatory for
  plural selection; making `n` implicit removes the duplication without
  removing the ability to pass a different `n` when a template
  legitimately needs to interpolate a value other than the count driving
  pluralization (e.g. "3 of 12 selected" — keep explicit `n` support for
  that case, just make it optional when `n === count`).
- **Generate the key union from `de.ts` via TypeScript's `keyof`/template
  literal types**, not a build-time codegen script — `de.ts` is already
  a plain nested object literal, so `as const` plus a recursive
  dotted-path type is sufficient without adding a new build step.

## Risks

- **Remount cost.** A full `<App key={locale}/>` remount discards
  component-local state (open sheets, scroll position, form drafts) on
  every language switch. Mitigated by scope: language switching is a
  deliberate, infrequent action from Settings, not a hot path, and the
  user has just closed whatever they were doing to get there.
- **New compile errors from the typed `t()` key union.** Expected and
  desirable — any existing typo'd key becomes a build failure. Budget
  time in this change's tasks to fix any that surface rather than
  suppressing the type.
