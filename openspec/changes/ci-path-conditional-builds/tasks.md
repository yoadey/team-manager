## 1. CI workflow

- [x] 1.1 Add a `changes` job to `.github/workflows/ci.yml`: computes
      `frontend`/`backend`/`helm` boolean outputs via `git diff` (PR:
      `origin/<base>...HEAD`; push: `<before>..HEAD`), short-circuits to
      "run everything" when `.github/workflows/ci.yml`, `docker-compose.yml`,
      `.zap/rules.tsv`, or a `.github/trivyignore-*.txt` file changed, and
      fails open (all outputs `true`) if the base ref can't be resolved or
      the diff otherwise fails.
- [x] 1.2 Gate `frontend-lint`, `frontend-typecheck`, `frontend-test`,
      `frontend-security`, `frontend-licenses`, `frontend-build`,
      `frontend-e2e`, `frontend-lighthouse`, and
      `security-container-frontend` on `needs.changes.outputs.frontend`.
- [x] 1.3 Gate `backend-openapi-drift`, `backend-lint`, `backend-test`,
      `backend-build`, `backend-licenses`, `backend-security`,
      `backend-coverage`, `backend-migration-rollback`,
      `backend-migration-safety`, `security-container`, and `dast` on
      `needs.changes.outputs.backend` (combined with
      `backend-migration-safety`'s existing `pull_request`-only condition).
- [x] 1.4 Gate `helm-lint` and `security-container-backup` on
      `needs.changes.outputs.helm`.
- [x] 1.5 Replace `security-sast`'s `language: [go, typescript]` matrix
      with two fixed jobs (`security-sast` for go, `security-sast-typescript`
      for typescript) gated on `needs.changes.outputs.backend` /
      `needs.changes.outputs.frontend` respectively — a job-level `if:`
      cannot read the `matrix` context, so a single matrix job can't
      express a per-language gate.
- [x] 1.6 Leave `security-secrets` unconditional.

## 2. Verification

- [x] 2.1 `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml'))"`
      confirms the file is still valid YAML.
- [x] 2.2 `actionlint` (installed via `go install
      github.com/rhysd/actionlint/cmd/actionlint@latest`) against
      `.github/workflows/ci.yml` — this is what actually caught the
      `matrix`-context-in-`if:` mistake behind 1.5 after the first push
      made GitHub Actions reject the whole file (0 jobs scheduled); plain
      YAML validity alone doesn't catch GitHub Actions schema/expression
      errors.
- [x] 2.3 Trace each job's new `if`/`needs` by hand against the existing
      dependency graph to confirm no required job is left permanently
      skipped or orphaned.
- [x] 2.4 This PR's own CI run exercises every job — modifying
      `.github/workflows/ci.yml` itself forces the `changes` job's outputs
      to all-true — serving as an end-to-end check that the new job graph
      is both syntactically and semantically valid.
