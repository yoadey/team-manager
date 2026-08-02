## 1. Spec
- [x] 1.1 Add optional `userId` + `membershipId` to `PollOption.voters` in `openapi.yaml`
- [x] 1.2 Run `make generate` + `make generate-ts`; commit generated output

## 2. Backend
- [x] 2.1 `polls` repository/service: populate voter `userId`/`membershipId` only when the poll is non-anonymous; leave identity-free for anonymous polls
- [x] 2.2 Ensure the poll read path enforces `polls` read permission (unchanged gate)

## 3. Frontend
- [x] 3.1 Voter details popup with two views: "by option" (list under each option) and "matrix" (options numbered 1..n across the top, one row per user, marks per selection)
- [x] 3.2 Build the matrix from `options[].voters`; render avatars via the shared avatar component
- [x] 3.3 Hide the popup/entry for anonymous polls; `de`/`en` strings; MSW handlers return voter ids for non-anonymous polls only

## 4. Verification
- [x] 4.1 openapi-drift green (regenerated clients committed)
- [x] 4.2 Backend tests: non-anonymous poll returns voter identities; anonymous poll returns none; permission enforced
- [x] 4.3 `golangci-lint` + `go test ./...` green
- [x] 4.4 Frontend tests: by-option list + matrix render correctly; anonymous poll shows no identities; lint/typecheck/test/build + bundle budget green
