## Why

`helm/team-manager` has CI coverage (`ci.yml`'s `helm-lint` job: lint + template
render) but is never packaged or published anywhere. Deploying it means
cloning the repo and running `helm install helm/team-manager ...` against
some commit — there's no immutable, versioned chart artifact matching a
release tag, the way the backend/frontend container images already get via
`.github/workflows/release.yml`'s `images` job. That also blocks anything
that expects to `helm install`/`helm upgrade` from an OCI reference (a
GitOps tool such as ArgoCD/Flux, or a deployer who wants to pin/roll back to
a specific chart version without a git checkout).

## What Changes

- Add a `helm-chart` job to `.github/workflows/release.yml` (same triggers as
  the existing `images` job: `v*.*.*` tag push, or `workflow_dispatch`) that:
  - Resolves the release version the same way the `images` job does (tag /
    `workflow_dispatch` input / commit-SHA fallback), stripping the leading
    `v` since Helm requires strict SemVer with no prefix.
  - `helm lint`s the chart before packaging.
  - `helm package`s `helm/team-manager` with `--version`/`--app-version` set
    to the resolved release version.
  - Logs in to `ghcr.io` via `helm registry login` (Helm's own OCI credential
    store, distinct from the Docker config `docker/login-action` writes) and
    pushes the packaged chart to `oci://ghcr.io/<owner>/charts`.
  - Signs the pushed chart digest with keyless cosign, matching how the
    `images` job signs each container image digest.
- Document the new OCI chart artifact and its install/upgrade command in
  `docs/operations.md`'s "Container images & releases" section.

## Capabilities

### Modified Capabilities
- `helm-deployment`: adds a release channel — the chart is now packaged and
  published to an OCI registry on tag, versioned and signed, not just
  consumable from a local checkout.

## Impact

- `.github/workflows/release.yml`: new `helm-chart` job.
- `docs/operations.md`: new subsection under "Container images & releases".
- No chart template/values changes; `Chart.yaml`'s checked-in
  `version`/`appVersion` (`0.1.0`) stay dev placeholders overridden at
  package time — the same way the images job never bakes a version into any
  source file either.
