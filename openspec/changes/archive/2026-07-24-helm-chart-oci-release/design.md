## Context

`.github/workflows/release.yml` already builds, signs, and publishes
versioned backend/frontend images to GHCR on `v*.*.*` tags. The Helm chart
(`helm/team-manager`) is only ever linted/rendered in CI (`ci.yml`'s
`helm-lint` job) — it is never packaged or published, so there is no
immutable, versioned chart artifact to deploy from; deploying means cloning
the repo at some commit and running `helm install helm/team-manager ...`
directly against the working tree.

## Goals / Non-Goals

**Goals:**
- A tagged release publishes the Helm chart as a versioned OCI artifact to
  GHCR, mirroring the `images` job's version resolution and signing.
- The chart version pushed matches the release tag; `Chart.yaml`'s checked-in
  `version` stays a dev placeholder (mirrors how image tags aren't baked
  into any source file either).
- Deployers can `helm install`/`upgrade` directly from
  `oci://ghcr.io/<owner>/charts/team-manager` pinned to a release version
  instead of a git checkout.

**Non-Goals:**
- No chart template/values changes.
- No SBOM/vulnerability scan of the chart artifact — unlike a container
  image, a Helm chart has no OS/language dependency layer for
  Trivy/anchore-sbom-action to meaningfully scan; `helm lint` (run here) and
  `ci.yml`'s existing `helm-lint` job (schema validation via
  `values.schema.json`) are the correctness gates, and cosign signing (added
  here) covers provenance/tamper-evidence — the "did this exact artifact
  come from this workflow" guarantee the images job's cosign step provides.
- No GHCR package-visibility automation — same as the existing image
  packages, a first push to a new package path may default to private;
  making it public is a one-time manual step in GitHub package settings,
  unrelated to this workflow.

## Decisions

### Where in the registry: `oci://ghcr.io/<owner>/charts`, not alongside the image repos
Pushing to `ghcr.io/<owner>/charts/team-manager` (a `charts/` sub-path)
instead of `ghcr.io/<owner>/team-manager` avoids colliding with a future
top-level package name and matches the common convention of separating
chart OCI artifacts from image OCI artifacts in the same registry namespace
(`ghcr.io/<owner>/team-manager-backend` / `-frontend` for images vs.
`ghcr.io/<owner>/charts/team-manager` for the chart).

### Helm's own registry login, not `docker/login-action`
The `images` job authenticates to GHCR via `docker/login-action`, which
writes credentials to the Docker CLI's config (`~/.docker/config.json`).
Helm's OCI registry client resolves credentials from its own config file
(`~/.config/helm/registry/config.json`, written by `helm registry login`),
not Docker's — reusing the `images` job's login step would silently leave
`helm push` unauthenticated. The new job runs its own `helm registry login`
instead.

### Version resolution duplicated, not shared via job outputs
The `images` job's version-resolution logic (tag → `workflow_dispatch` input
→ SHA fallback) is duplicated in the new `helm-chart` job rather than
factored into a shared step: GitHub Actions job outputs require an explicit
`needs:`/`outputs:` wire-up across jobs, and the two jobs otherwise have no
dependency on each other (the chart push doesn't need the images to exist
first, or vice versa) — introducing one only to share four lines of shell
would serialize two independently-fast jobs for no benefit. The chart
version also strips a piece the image-tag logic doesn't need to (a leading
`v`, since Helm requires strict SemVer with no prefix), so it isn't a pure
copy either.

## Risks / Trade-offs

- **Duplicated version-resolution shell** between the `images` and
  `helm-chart` jobs — accepted per the Decision above; if a third job needs
  the same logic, that's the trigger to factor it into a composite action.
- **First push to a new GHCR package path may land private** — same
  operational characteristic the image packages already have; not a
  regression introduced by this change.
