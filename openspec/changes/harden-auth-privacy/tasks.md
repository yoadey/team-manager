## 1. Audit email PII
- [x] 1.1 Add `auth.HashEmailForAudit` (one-way SHA-256 hex of the lowercased email; `crypto/sha256`/`hex`/`strings` already imported) — keyless, so no config plumbing; correlatable without plaintext
- [x] 1.2 Replace both `slog.String("email", …)` in `auth/handler.go` login success/failure with `slog.String("email_hash", …)`
- [ ] 1.3 (Optional erase-time scrub) Not needed: audit rows now carry only a hash, so nothing plaintext survives erasure or the retention window

## 2. Password length
- [x] 2.1 `maxPasswordBytes = 72`; `HashPassword` rejects over-length with `ErrPasswordTooLong`; `Login` rejects over-length as invalid credentials (with a dummy compare to keep timing) before any DB lookup
- [x] 2.2 Tests: `HashPassword` rejects 73 bytes / accepts 72; `Login` rejects over-length before the repo is consulted

## 3. CSRF fallback
- [x] 3.1 `CSRFOriginCheck` now blocks mutating requests with `Sec-Fetch-Site: cross-site` (authoritative browser signal), in addition to the disallowed-Origin check; header-less/same-origin requests still allowed (non-browser clients keep working)
- [x] 3.2 Tests: cross-site metadata (no Origin) → 403; same-origin → allowed; existing allowed-Origin / missing-Origin tests still green

## 4. Verification
- [x] 4.1 `go test ./internal/auth/... ./internal/middleware/... -short` green
- [x] 4.2 `golangci-lint run` 0 issues; gofmt/gofumpt clean
- [x] 4.3 Grep confirms no `slog.String("email"` plaintext remains in audit calls
