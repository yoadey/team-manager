## Why

CLAUDE.md mandates: "`openspec validate <name> --strict` (or `--all`) MUST
pass before implementation." No CI job runs it — grepping
`.github/workflows/*.yml` for "openspec" returns nothing. A proposal
merged with a missing `#### Scenario:` block, a malformed delta, or a spec
reference to a nonexistent capability currently goes undetected, silently
undermining the spec-driven workflow the rest of CLAUDE.md is built
around. This mirrors the existing `backend-openapi-drift` job's role for
the OpenAPI-first workflow — that gate exists; the equivalent one for
OpenSpec does not.

## What Changes

- Add an `openspec-validate` CI job that runs
  `npx --yes @fission-ai/openspec@latest validate --all --strict --json`
  and fails the build on any validation error, gated on PRs that touch
  `openspec/**`.

## Capabilities

### Modified Capabilities
- `dev-tooling`: OpenSpec change proposals and specs are validated in CI,
  not only by convention.

## Impact

- `.github/workflows/ci.yml` (new job).
- No application code changes.
