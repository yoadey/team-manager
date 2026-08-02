## 1. Mailer CRLF defense-in-depth

- [x] 1.1 `smtp.go`: `buildMessage` now returns `([]byte, error)` and
      rejects `from`/`to`/`subject` containing `\r`/`\n` via a new
      `ErrHeaderInjection` sentinel, checked before any network I/O
- [x] 1.2 `mailer_test.go`: cover a `to` address containing `\r\n` being
      rejected by `SendVerificationEmail`/`SendPasswordResetEmail`
      themselves (not relying on upstream validation) — no real SMTP
      server needed since the check runs before `send` dials out

## 2. Retention job lint noise

- [x] 2.1 Investigated and skipped: `golangci-lint run ./internal/jobs/...`
      is already clean (gosec doesn't flag this `fmt.Sprintf` pattern in
      this repo's configuration). Empirically confirmed that adding the
      proposed `//nolint:gosec` comment anyway makes `nolintlint` fail
      the build with "directive is unused for linter gosec" — the
      opposite of the intended fix. No change made; leaving `retention.go`
      as-is is correct.

## 3. Auth handler status code

- [x] 3.1 `handler.go`: `GetMyPhoto` returns `apierror.Unauthorized`
      (401) instead of `apierror.NotFound` (404) when `UserFromContext`
      fails
- [x] 3.2 `handler_test.go`: added
      `TestHandler_GetMyPhoto_NoAuthContext_Returns401`
- [x] 3.3 (found during implementation) `openapi.yaml`: `GET
      /auth/me/photo` was missing a documented `401` response entirely
      (unlike its sibling `/auth/me`) — added
      `$ref: "#/components/responses/Unauthorized"`, then regenerated
      `internal/gen/api.gen.go`, `frontend/src/api/types.gen.ts`, and
      `frontend/src/api/zod.gen.ts` via `make generate` / `make
      generate-ts`

## 4. OpenAPI version field

- [x] 4.1 Build-time templating of `info.version` would be a
      disproportionate change to the spec-first codegen pipeline for
      this nit, and the field can't simply be removed (required by the
      OpenAPI 3.0 schema). Added an explanatory YAML comment instead,
      pointing at the real versioning mechanism (`API-Version` header,
      `API_DEPRECATION_DATE`) — comments don't affect generated code, so
      no regeneration needed for this part.

## 5. Verification

- [x] 5.1 `cd backend && go test -short ./...` (all green)
- [x] 5.2 `cd backend && golangci-lint run ./...` (0 issues)
- [x] 5.3 `make generate && make generate-ts` clean, diff limited to the
      new 401 response type/schema as expected
