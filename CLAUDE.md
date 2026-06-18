# Teamverwaltung — Developer Guide

## Quick Start

```bash
npm install
npm run dev          # http://localhost:5173
npm test             # run all tests once
npm run typecheck    # TypeScript check
npm run lint         # ESLint
npm run format       # Prettier (auto-fix)
```

## Architecture

### Technology Stack

- **React 18** + **TypeScript 5** (strict mode)
- **Material UI v6** for components, **Emotion** for styling
- **Vite 6** for bundling, **Vitest 2** for tests
- **State-based routing** (shallow, not URL-based — no router dependency; navigation is driven by `state.route`)
- **i18n** via a lightweight in-house layer (`src/i18n`): locale-aware `Intl` formatting + `t()` catalogs (German default, English skeleton)
- **Error handling**: every async action funnels failures through `reportActionError` (`src/utils/errors.ts`); global `unhandledrejection`/`error` handlers report to Sentry (`src/monitoring.ts`)

### State Management

All application state lives in `src/context/AppContext.tsx` via a single `AppState` object. Feature-specific actions are delegated to hooks in `src/context/useFeatureActions.ts`. Access state via `useApp()`:

```tsx
const { state, can, go, openEventForm } = useApp();
```

### Service Layer

`src/services/serviceLayer.ts` is a **mock backend** with artificial delay (120–320 ms) and localStorage persistence. It mirrors the future Go/PostgreSQL API contract — replace method bodies with `fetch()` calls when connecting a real backend. The `api` object exported from this file is the only entry point for data access.

### Routing

Navigation is state-based (`state.route`). Use `app.go('finances')` to navigate. `src/pages/index.tsx` renders the active route; heavy routes are code-split via `React.lazy()`. Route guards are enforced there (e.g. `finances` requires `can('finances', 'read')`).

### Permissions (RBAC)

Each team member has roles; each role has per-module permission levels (`none | read | write`). Check permissions via:

```tsx
app.can('finances', 'read'); // true if the user can at least read finances
app.can('events', 'write'); // true only if the user has write access
app.isStaff(); // shorthand: can write events OR members
```

### Sheets / Modals

Overlay dialogs are called "sheets". Open via `app.setState({ sheet: { type: 'eventForm', ... } })` or through dedicated action methods (`app.openEventForm(null)`). `src/sheets/DialogSheets.tsx` maps sheet types to components.

## Directory Structure

```
src/
├── components/       Shared UI atoms (ErrorBoundary, Toast, ui.tsx, cards.tsx)
├── context/          Global state (AppContext, useFeatureActions)
├── features/         Feature modules (events, members, finances, news, polls, team, auth, notifications)
│   └── <feature>/
│       ├── *Page.tsx         Route-level component
│       ├── components/       Feature-specific UI
│       ├── hooks/            Feature-specific actions
│       ├── index.ts          Public API exports
│       └── types.ts          Feature types
├── layouts/          AppShell (navigation chrome)
├── monitoring.ts     Sentry initialisation (guarded by VITE_SENTRY_DSN)
├── pages/            RouteScreen (lazy-loads feature pages)
├── services/         Mock service layer + mappers
├── sheets/           Sheet dispatcher
├── styles/           MUI theme builder + design tokens
├── types/            Shared domain types
└── utils/            date.ts, validation.ts
```

## Environment Variables

Copy `.env.example` to `.env` (gitignored). All variables are optional — the app works with an empty `.env`.

| Variable                  | Default          | Purpose                                               |
| ------------------------- | ---------------- | ----------------------------------------------------- |
| `VITE_APP_NAME`           | `Teamverwaltung` | Browser title                                         |
| `VITE_STORAGE_KEY_PREFIX` | `tv_db_`         | localStorage key prefix for the mock DB               |
| `VITE_MOCK_DELAY_MIN/MAX` | `120` / `320`    | Simulated API latency (ms)                            |
| `VITE_SENTRY_DSN`         | _(empty)_        | Sentry DSN; monitoring disabled when empty            |
| `VITE_API_BASE_URL`       | _(empty)_        | Real backend base URL (unused until mock is replaced) |

## Testing

Tests live alongside source files as `*.test.ts(x)`. Currently covering services and utilities:

```bash
npm test                  # single run
npm run test:watch        # watch mode
npm run test:coverage     # whole-app coverage report (floors: 18% statements/lines, 50% functions, 70% branches — raise as tests grow)
```

Add component tests with `@testing-library/react`. The jsdom environment and jest-dom matchers are pre-configured in `src/test/setup.ts`.

## Code Quality

- **Commits** run `lint-staged` via Husky (ESLint + Prettier on staged files)
- **CI** (`.github/workflows/ci.yml`) runs lint → typecheck → test → build on every PR
- ESLint config: `eslint.config.js` (flat config, TypeScript + react-hooks rules)
- Prettier config: `.prettierrc.json` (120-char width, single quotes, trailing commas)

## Replacing the Mock Backend

When connecting a real API:

1. Replace method bodies in `src/services/serviceLayer.ts` with `fetch()` / Axios calls
2. Keep the exported `api` object shape unchanged (the app consumes this contract)
3. Remove the `loadDb` / `seed` / `persist` localStorage functions
4. Set `VITE_API_BASE_URL` in production `.env`
