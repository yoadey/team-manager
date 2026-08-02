## 1. CI

- [ ] 1.1 Add an `openspec-validate` job to `ci.yml`, triggered on PRs that
      touch `openspec/**`, running
      `npx --yes @fission-ai/openspec@latest validate --all --strict --json`
      and failing the build on a non-zero exit code
- [ ] 1.2 Smoke-test locally: deliberately break a proposal (e.g. drop a
      `#### Scenario:` block from a scratch change), confirm the command
      exits non-zero, then confirm it's clean again after reverting

## 2. Verification

- [ ] 2.1 CI green on a PR that touches `openspec/**`
- [ ] 2.2 `npx @fission-ai/openspec@latest validate --all --strict` passes
      locally against the current repo state
