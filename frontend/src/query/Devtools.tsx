import { lazy, Suspense } from 'react';

// Dynamic import so the devtools bundle (and its dependencies) is only ever
// fetched in dev -- `import.meta.env.DEV` is a build-time constant Vite can
// prove false in a production build, letting Rollup drop this chunk entirely.
const LazyDevtools = import.meta.env.DEV
  ? lazy(() =>
      import('@tanstack/react-query-devtools').then((m) => ({ default: m.ReactQueryDevtools })),
    )
  : null;

export function QueryDevtools() {
  if (!LazyDevtools) return null;
  return (
    <Suspense fallback={null}>
      <LazyDevtools initialIsOpen={false} />
    </Suspense>
  );
}
