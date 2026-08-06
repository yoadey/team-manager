## 1. DB pool sizing

- [ ] 1.1 `config.go`: add `DB_MAX_CONNS`/`DB_MIN_CONNS` env vars
      (defaults `25`/`2`, matching current hardcoded values)
- [ ] 1.2 `db.go`: read pool size from config instead of hardcoded
      constants
- [ ] 1.3 `config_test.go`/`db_test.go`: cover defaults and explicit
      overrides
- [ ] 1.4 CLAUDE.md: add `DB_MAX_CONNS`/`DB_MIN_CONNS` to the backend
      env-var table, with a note on sizing relative to
      `autoscaling.maxReplicas` × `DB_MAX_CONNS` vs. Postgres
      `max_connections` (or a required pooler)
- [ ] 1.5 `docs/operations.md`: add the same sizing guidance to whatever
      section covers production database configuration
- [ ] 1.6 `helm/team-manager/values-prod.yaml` comment (near the
      existing pooler mention at `values.yaml:441`): cross-reference the
      new sizing guidance

## 2. Rate-limit scaling documentation

- [ ] 2.1 CLAUDE.md: note next to `RATE_LIMIT_RPS`,
      `LOGIN_RATE_LIMIT_PER_MIN`, `REGISTER_RATE_LIMIT_PER_MIN`,
      `RESEND_VERIFICATION_RATE_LIMIT_PER_MIN`,
      `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN` that these are enforced
      per-process, so the effective limit scales with replica count
- [ ] 2.2 `docs/operations.md`: same note in whatever section discusses
      brute-force protection / rate limiting, if one exists

## 3. Verification

- [ ] 3.1 `cd backend && make test-unit`
- [ ] 3.2 `cd backend && make lint`
- [ ] 3.3 `helm lint`/chart render check still passes if values files
      changed
