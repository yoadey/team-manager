## 1. Locale-switch propagation (fastest, highest impact)

- [ ] 1.1 `main.tsx`: key the root `<App/>` (or the tree below
      `LocaleProvider`) on the active locale so a `setLocale()` call
      forces a full remount
- [ ] 1.2 `LocaleProvider.test.tsx`: add a regression test asserting a
      component that only calls bare `t()` (not `useLocale()`) shows the
      new locale's text after `setLocale()`, without requiring an
      unrelated re-render

## 2. Plural interpolation footgun

- [ ] 2.1 `i18n/index.ts`: when `t()` is called with `count` but no `n`,
      default `n` to `count` before interpolating
- [ ] 2.2 Add a test covering a plural call site passing only `count`
      rendering the numeric value correctly (no literal `{n}` in output)
- [ ] 2.3 Audit existing call sites passing both `n: x, count: x`
      identically; simplify to `count: x` alone where `n` and `count`
      are the same value (leave call sites that intentionally differ
      unchanged)

## 3. Type-safe translation keys

- [ ] 3.1 Derive a literal union of dotted keys from `de.ts` (the
      canonical catalog) using `as const` + a recursive dotted-path
      mapped type
- [ ] 3.2 Type `t()`'s first parameter against that union
- [ ] 3.3 Fix any call sites that surface as new compile errors (typo'd
      or stale keys)
- [ ] 3.4 Confirm `en.ts` is still checked for parity against the same
      key set (existing or new test)

## 4. Verification

- [ ] 4.1 `npm run typecheck`
- [ ] 4.2 `npm run lint`
- [ ] 4.3 `npm test`
- [ ] 4.4 `npm run build` (confirms the key-union type doesn't blow up
      build time on the full catalog)
