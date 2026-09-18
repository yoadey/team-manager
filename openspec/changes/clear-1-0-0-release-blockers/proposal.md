## Why

`main` is red, and has been for every one of the last five runs. All
functional gates pass (backend/frontend lint, typecheck, tests, E2E,
OpenAPI codegen drift, migration rollback, coverage, CodeQL, OpenSpec
validate); the five failures are entirely the security gates:

- `Backend · Security (govulncheck)` — 4 called vulnerabilities:
  `golang.org/x/crypto` GO-2026-6354/6355 (fixed in v0.56.0),
  `google.golang.org/grpc` GO-2026-6348 (v1.83.1), and
  `github.com/moby/go-archive` GO-2026-6253 (v0.3.0).
- `Security · Container scan (Trivy)` — 4 HIGH in the backend image:
  `golang.org/x/crypto` CVE-2026-56854, `golang.org/x/image`
  CVE-2026-46603, `google.golang.org/grpc` CVE-2026-84304/84445.
- `Security · Container scan (Trivy, frontend)` — HIGH in the pinned
  `nginx-unprivileged` base layer (`libssl3` CVE-2026-14456, `libuuid`/
  util-linux CVE-2026-53612/53613/53614/76642/78408/78410).
- `Security · Container scan (Trivy, backup CronJob images)` — Go stdlib
  HIGH in the pinned `postgres:17`/`amazon/aws-cli` digests.
- `Frontend · Security audit` — 8 HIGH npm advisories not covered by
  `npm-audit-allowlist.txt` (`@cyclonedx/cyclonedx-npm`, `extract-zip`,
  `fast-uri` ×4, `js-yaml`), all in devDependencies.

This blocks a 1.0.0 outright, not just cosmetically. `release.yml`'s
images job runs its own Trivy scan of the **published digest** with
`exit-code: 1` on CRITICAL/HIGH — the same finding set that fails CI
today would abort the release after the images are pushed but before
they are signed and attested.

Worse, the `helm-chart` job carries no `needs:`, so it runs independently
of `images`. Tagging `v1.0.0` right now would publish and cosmign-sign a
chart whose `appVersion` names backend/frontend image tags that the
failed images job never finished publishing.

## What Changes

- Bump the four vulnerable Go modules. `golang.org/x/crypto` v0.56.0 (the
  minimum that closes GO-2026-6354/6355) requires `go >= 1.26.0`, so the
  toolchain moves with it: `go.mod`'s directive to `1.26.8` and
  `backend/Dockerfile`'s builder image to the matching `golang:1.26.8`
  digest (CI's every Go job resolves its version from `go-version-file:
  backend/go.mod`, so nothing else needs touching).
- Re-pin the stale base-image digests that the container scans fail on:
  `nginx-unprivileged:1-alpine3.23` (frontend), and `postgres:17` /
  `amazon/aws-cli` (backup CronJob), keeping every duplicated copy of
  those digests in lockstep as ci.yml's own drift check requires.
- Resolve the npm advisories by dependency bump where a non-breaking fix
  exists (`@cyclonedx/cyclonedx-npm` 6, `js-yaml` and `fast-uri`
  overrides), and allowlist only what has no fix on its current major
  (the second `extract-zip` advisory, same @lhci/cli chain and same
  reachability analysis as the one already allowlisted).
- Gate `release.yml`'s `helm-chart` job on `images`, so a release that
  fails its vulnerability scan publishes no chart either.

## Capabilities

### Modified Capabilities
- `release-publishing`: a release never publishes a Helm chart for a
  version whose container images did not publish.

## Impact

- `backend/go.mod`, `backend/go.sum`, `backend/Dockerfile`.
- `frontend/Dockerfile`, `frontend/package.json`,
  `frontend/package-lock.json`, `frontend/npm-audit-allowlist.txt`.
- `docker-compose.yml`, `helm/team-manager/values.yaml`,
  `.github/workflows/ci.yml`, `.github/workflows/scheduled-security-scan.yml`
  (the four places the postgres/aws-cli digests are duplicated).
- `.github/workflows/release.yml` (`helm-chart` gains `needs: images`).
- `CLAUDE.md` and `openspec/config.yaml` (Go 1.25+ → 1.26+).
- No application behavior change; no migration; no API change.
