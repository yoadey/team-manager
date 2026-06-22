# Contributing

Thanks for contributing to **team-manager**. This guide covers the local
workflow and the quality bar enforced in CI.

## Prerequisites

- **Node.js ≥ 22** and **npm ≥ 10** (see `engines` in `package.json`; `.nvmrc`
  pins the version — `nvm use`).

## Setup

```bash
npm install
cp .env.example .env   # optional — the app works with an empty .env
npm run dev            # http://localhost:5173
```

## Development workflow

1. Branch off `main`.
2. Make your change, following the conventions in
   [`CLAUDE.md`](./CLAUDE.md) (architecture, state, RBAC, i18n, sheets).
3. Keep changes scoped; match the style of the surrounding code.
4. Run the full local check set before pushing:

   ```bash
   npm run lint
   npm run typecheck
   npm test
   npm run build
   ```

5. Open a PR using the template. CODEOWNERS are requested automatically.

A Husky pre-commit hook runs `lint-staged` (ESLint + Prettier on staged files).

## Quality bar (enforced in CI)

CI runs lint → typecheck → test (coverage) → build (bundle-size budget + SBOM)
→ Playwright E2E → Lighthouse. PRs must be green to merge.

- **Tests** live next to source as `*.test.ts(x)`. Add tests for new logic; the
  coverage floors (`vitest.config.ts`) must hold.
- **Accessibility**: components must be keyboard-operable and pass `vitest-axe`
  / `eslint-plugin-jsx-a11y`. Respect dark mode via the `NEUTRAL` tokens — avoid
  hardcoded hex colors for surfaces/text.
- **i18n**: no hardcoded user-facing strings — use `t()` and update **both**
  `src/i18n/de.ts` and `src/i18n/en.ts`.
- **Security**: never commit secrets/PII. `dangerouslySetInnerHTML` is blocked by
  lint; sanitise HTML if you must render it. See [`SECURITY.md`](./SECURITY.md).

## Commit messages

Use clear, imperative messages (Conventional Commits style is welcome, e.g.
`feat(i18n): persist locale`). Group related changes into focused commits.

## Reporting bugs / security issues

Open a GitHub issue for bugs. For security vulnerabilities, follow
[`SECURITY.md`](./SECURITY.md) (private reporting) instead of a public issue.
