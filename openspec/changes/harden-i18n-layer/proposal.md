## Why

The in-house i18n layer (`frontend/src/i18n/`) has three related gaps
found during review, all rooted in `t()` being a bare module-level
function outside React's render/subscription model:

1. **Locale switches don't propagate to already-mounted components.**
   `activeLocale` is a module-level variable (`i18n/index.ts:56`).
   `LocaleProvider` wraps `<App/>` as a `children` prop created once at
   the root; when `setLocale()` flips `activeLocale` and notifies
   listeners, only `LocaleProvider` itself and the one component calling
   `useLocale()` (`AppearancePanel.tsx`) re-render, since `children`'s
   element reference is unchanged and React bails out of the rest of the
   subtree. Every other component calls bare `t()` in render without
   subscribing to anything, so it keeps showing the *previous* locale's
   strings until it re-renders for an unrelated reason. Concretely: a
   user switches language in Settings, closes the sheet — the app shell,
   toasts, and any screen that hasn't independently re-rendered still
   show the old language. `LocaleProvider.test.tsx` only exercises a
   dedicated `useLocale()` consumer, so this gap is untested.
2. **Plural interpolation requires the same value under two different
   param names, with a silent-failure footgun.** `count` drives plural-
   form selection (`_one`/`_other`) but templates interpolate `{n}`, so
   every call site must pass both, e.g.
   `t('team.membersCount', { n: x, count: x })`. `Params` is
   `Record<string, string|number>` — untyped against the catalog — so a
   call passing only `count` compiles fine, selects the right plural
   form, and then silently renders the literal string `{n}` in the UI
   (`interpolate()` leaves unmatched placeholders verbatim). No current
   call site is missing `n`, but nothing prevents or catches a future
   one.
3. **`t(key: string, …)` is not type-safe.** Any string is accepted; a
   typo'd key type-checks and silently falls back to returning the raw
   key at runtime. This is exactly the failure mode CLAUDE.md's strict-TS
   posture is meant to prevent elsewhere.

## What Changes

- Force the whole tree to re-render on a locale change (e.g.
  `<App key={locale}/>` at the root, or route `t()` through a hook that
  subscribes to `LocaleContext` so every consuming component
  re-subscribes). Prefer whichever keeps the smallest diff against the
  many existing bare `t()` call sites — a design decision documented in
  `design.md`.
- When `n` is omitted but `count` is provided, derive the interpolation
  value from `count` automatically instead of requiring both.
- Generate a literal union of dotted keys from `de.ts` (the single
  source of truth) and type `t`'s first parameter against it, so a
  typo'd key is a compile error.

## Capabilities

### Added Capabilities
- `i18n-layer`: locale switches are reflected across the whole mounted
  tree; plural interpolation doesn't require duplicating the same value
  under two parameter names; translation keys are type-checked at
  compile time.

## Impact

- `frontend/src/i18n/index.ts`, `LocaleProvider.tsx`, `main.tsx`.
- Every call site using `t()` is a potential type-checking touch point
  once keys become a literal union — expect this to surface any existing
  typo'd keys as new compile errors, which should be fixed as part of
  this change.
- `frontend/src/i18n/de.ts` remains the source of truth for both the
  translation catalog and the generated key type.
