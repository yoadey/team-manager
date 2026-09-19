## Context

The repository has never cut a stable release — the only tag is
`v0.1.0-alpha.1` (2026-07-22). The release machinery itself is complete
and well-tested (`release.yml` builds both images, scans the published
digest, signs keylessly, attests an SBOM, and packages/pushes the Helm
chart as a signed OCI artifact), and `release-publishing`'s existing
requirement already keeps `latest` off prereleases. What is missing for a
`v1.0.0` is not machinery but a green commit to cut it from.

## Goals

- Every security gate green on `main`, so the release workflow's own
  vulnerability gate cannot abort a tagged release halfway through.
- No release can publish a chart without the images it names.

## Non-Goals

- Widening what the gates check, or relaxing any of them to get green.
- The five other open change proposals. They are genuine improvements
  but none of them blocks a release.

## Decisions

- **Take the Go 1.26 toolchain bump rather than only the minimum Trivy
  asks for.** Trivy's backend finding (CVE-2026-56854) is satisfied by
  `golang.org/x/crypto` v0.55.0, which stays on Go 1.25. But govulncheck
  additionally flags GO-2026-6354/6355 against the same module, fixed
  only in v0.56.0, which declares `go >= 1.26.0`. Stopping at v0.55.0
  would unblock the release workflow while leaving CI red — trading a
  visible, permanent red build for a smaller diff. Since CI resolves its
  Go version from `go.mod` itself, the bump's blast radius is two lines
  (the directive and the builder image digest), and `golangci-lint`
  v2.12.2 — the version both the Makefile and ci.yml pin — builds and
  runs clean on 1.26.
- **Pin the toolchain to a patch version (`1.26.8`), not `1.26.0`.**
  `go get` writes the minimum that satisfies the requirement; the repo's
  existing convention (and `backend/Dockerfile`'s own comment) is that
  the directive and the builder image name the *same exact* version, so
  the binary CI tests and scans is the binary the image ships.
- **Fix npm advisories by dependency resolution first, allowlist last.**
  Of the eight failing advisories, seven had a fix reachable without a
  breaking change to a tool this repo depends on: `@cyclonedx/cyclonedx-npm`
  6.0.1 (a major bump of a dev-only SBOM generator, verified to still
  produce a CycloneDX 1.6 document), plus `js-yaml`/`fast-uri` overrides.
  Only `extract-zip` has no fix on `@lhci/cli`'s current major, and it
  joins the sibling advisory already allowlisted for the identical
  dependency path and reachability argument. This keeps the allowlist a
  record of genuinely unfixable findings rather than a dumping ground.
- **`js-yaml` is overridden at the top level, with `@lhci/utils` scoped
  separately.** A nested `@eslint/eslintrc` override was resolved but not
  applied by npm (the nested copy stayed at 4.3.1, reported `invalid`),
  so the 4.x override is hoisted to the top level. `@lhci/utils` needs
  the 3.x line (js-yaml 4 dropped `safeLoad`), so it keeps its own nested
  pin at the patched 3.15.2.
- **`helm-chart` gains `needs: images` rather than the two jobs being
  merged.** They still publish to different registries with different
  credentials and deserve separate logs; the ordering constraint is the
  only thing that was missing.

## Risks

- **The base-image re-pins are unverified locally.** This environment's
  network policy blocks the registry blob CDN, so Trivy cannot pull and
  scan an image here, and `vuln.go.dev` is blocked so govulncheck cannot
  run either. The Go module bumps are verified against the advisories'
  own stated fixed-versions, and the digests are the current ones each
  tag resolves to — but whether the rebuilt `nginx-unprivileged` layer
  actually carries the patched `libssl3`/`util-linux`, and whether the
  new `postgres:17`/`aws-cli` digests clear the Go stdlib findings, is
  confirmed only by CI. Treat a still-red container scan on the PR as
  expected-possible, needing another digest bump or a justified
  `trivyignore` entry, not as a surprise.
- **The frontend `trivyignore` entries may now be stale.** The three
  entries (c-ares, curl ×2) were written against a base image digest
  this change replaces. A stale ignore entry does not fail a scan, so
  they are left in place rather than removed speculatively; they carry
  their own "remove once upstream rebuilds" note already.
- **Go 1.26 is a toolchain major-minor bump immediately before a 1.0.0.**
  Mitigated by the full backend suite (unit + every integration test)
  passing against a real PostgreSQL on 1.26.8, plus a clean
  `golangci-lint` run — but it is a larger change than a release-prep
  commit would ideally carry.
