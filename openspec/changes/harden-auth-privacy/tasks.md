## 1. Audit email PII
- [ ] 1.1 Add a keyed-hash helper (HMAC-SHA256 over lowercased email); choose/plumb the key (reuse an existing 32-byte key or add `AUDIT_HMAC_KEY`); document in `CLAUDE.md`
- [ ] 1.2 Replace `slog.String("email", …)` in `auth/handler.go` login success/failure with the hash (`email_hash`)
- [ ] 1.3 (Optional) On `EraseUser`, scrub that user's audit rows' email attrs; keep `actor_id`

## 2. Password length
- [ ] 2.1 Reject `len(password) > 72` bytes with a validation error in login and password-set paths, before bcrypt
- [ ] 2.2 Test both paths for the over-length rejection

## 3. CSRF fallback
- [ ] 3.1 In `CSRFOriginCheck`, also read `Sec-Fetch-Site`; reject mutating requests with neither `Origin` nor `Sec-Fetch-Site`; keep allowing whitelisted `Origin`
- [ ] 3.2 Update/extend CSRF tests (allowed origin, same-origin fetch-site, missing-both rejected)

## 4. Verification
- [ ] 4.1 `cd backend && make test` green (auth, middleware)
- [ ] 4.2 `make lint` green; coverage gate holds
- [ ] 4.3 Grep confirms no `slog.String("email"` plaintext remains in audit calls
