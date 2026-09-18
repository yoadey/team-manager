## 1. Backend dependencies and toolchain

- [x] 1.1 `go get golang.org/x/crypto@v0.56.0 golang.org/x/image@v0.45.0
      google.golang.org/grpc@v1.83.2 github.com/moby/go-archive@v0.3.0`
- [x] 1.2 `go.mod`'s `go` directive to `1.26.8` (required by x/crypto
      v0.56.0) and `backend/Dockerfile`'s builder to the matching
      `golang:1.26.8` digest
- [x] 1.3 `DeleteEvent`'s series branch extracted into
      `deleteSeriesRemainder` to stay under the gocognit threshold the
      Go bump's sibling change pushed it over
- [x] 1.4 `CLAUDE.md` and `openspec/config.yaml` say Go 1.26+

## 2. Base image digests

- [x] 2.1 `frontend/Dockerfile`: `nginx-unprivileged:1-alpine3.23` and
      `node:22-alpine` re-pinned to current digests
- [x] 2.2 `postgres:17` re-pinned in all four synchronized locations
      (`docker-compose.yml`, `helm/team-manager/values.yaml`,
      `ci.yml` scan + service containers, `scheduled-security-scan.yml`)
- [x] 2.3 `amazon/aws-cli` re-pinned to 2.36.48 in its three locations,
      pin comment refreshed
- [x] 2.4 ci.yml's own digest drift check re-run by hand — all groups
      report in sync

## 3. Frontend advisories

- [x] 3.1 `@cyclonedx/cyclonedx-npm` bumped to ^6.0.1 (closes
      GHSA-q69g-4hcv-6jg4); `npm run sbom` still emits a CycloneDX 1.6
      document
- [x] 3.2 `js-yaml` override hoisted to top level at ^4.3.2, with
      `@lhci/utils` scoped to ^3.15.2; `ajv` → `fast-uri` ^3.1.6
- [x] 3.3 `npm-audit-allowlist.txt` gains GHSA-7pqw-9j4j-h8q3 with a
      written justification, matching its already-allowlisted sibling
- [x] 3.4 `npm run audit:ci` passes

## 4. Release workflow

- [x] 4.1 `release.yml`'s `helm-chart` job gains `needs: images`

## 5. Suppression hygiene

- [x] 5.1 `.github/trivyignore-frontend.txt` emptied: its three entries
      were written against the alpine3.23 base this change replaces. The
      two curl ones are demonstrably obsolete (the eight unsuppressed curl
      CVEs that were failing need 8.22.0-r0, past the 8.20.0-r0 that fixed
      them), and a suppression cannot report whether it still applies, so
      c-ares went too rather than being carried forward against a base
      nobody re-checked it against
- [x] 5.2 The file and release.yml's matrix comment now say when adding an
      entry back is warranted, and that re-pinning the base is preferred

## 6. Verification

- [x] 6.1 `cd backend && go build ./... && golangci-lint run ./...`
- [x] 6.2 Full backend test suite (unit + integration) against a real
      PostgreSQL — Docker is unavailable in this environment, so
      testcontainers was pointed at a local server for the run
- [x] 6.3 `cd frontend && npm run typecheck && npm run lint && npm test
      && npm run build && npm run check:bundle`
- [x] 6.4 `cd frontend && npm ci` resolves the updated lockfile cleanly
- [x] 6.5 `openspec validate --all --strict`
- [x] 6.6 CI green on the PR — `govulncheck` and the Trivy container
      scans cannot run in this environment (`vuln.go.dev`, the registry
      blob CDN and Alpine's package index are all blocked by its network
      policy), so they were confirmed on the PR: both release gates (the
      backend and frontend published-image scans) pass, the frontend one
      only after moving that image to the alpine3.24 base
