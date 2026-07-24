## ADDED Requirements

### Requirement: Helm chart published to an OCI registry on release
Tagging a release (`vX.Y.Z`) MUST package `helm/team-manager` and push it as
a versioned OCI artifact to GHCR, signed the same way as the release's
container images.

#### Scenario: Tagged release
- **WHEN** a `vX.Y.Z` tag is pushed
- **THEN** `.github/workflows/release.yml`'s `helm-chart` job packages the
  chart with `version`/`appVersion` set to `X.Y.Z`, pushes it to
  `oci://ghcr.io/<owner>/charts/team-manager`, and signs the pushed digest
  with keyless cosign

#### Scenario: Manual dispatch
- **WHEN** the workflow is run manually via `workflow_dispatch` with a
  `version` input
- **THEN** the chart is packaged and pushed using that input as the version,
  mirroring how the `images` job resolves `workflow_dispatch` versions
