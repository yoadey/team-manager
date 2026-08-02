## 1. CI

- [x] 1.1 Add an `openspec-validate` job to `ci.yml`, running
      `npx --yes @fission-ai/openspec@latest validate --all --strict` and
      failing the build on a non-zero exit code. Runs unconditionally on
      every push/PR (matching every other job in this workflow — the repo
      has no existing path-filtering pattern to gate on `openspec/**`
      alone, and the job is cheap enough not to need one)
- [x] 1.2 Smoke-tested locally: a scratch change with a requirement
      missing a `#### Scenario:` block made the command exit 1; removing
      it restored a clean, 0-exit-code run

## 2. Verification

- [x] 2.1 `npx @fission-ai/openspec@latest validate --all --strict` passes
      locally against the current repo state (job itself will confirm on
      CI once this PR runs)
- [x] 2.2 `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"`
      confirms the edited workflow file is still valid YAML
