# release-publishing Specification

## Purpose
Defines how the release workflow tags published container images for a pushed `v*.*.*` tag: every release gets its exact-version and commit-SHA tags, but the `latest` tag is applied only for a stable (non-prerelease) semantic version, so a prerelease build never becomes the image pulled by default.

## Requirements

### Requirement: The `latest` GHCR tag only tracks stable releases
When the release workflow publishes images for a pushed `v*.*.*` tag, it MUST apply the `latest` tag only when that tag is a stable (non-prerelease) semantic version. A prerelease tag (a version with a `-` suffix, e.g. `-alpha.1`, `-beta.2`, `-rc.1`) MUST NOT be published as `latest`.

#### Scenario: Prerelease tag pushed
- **WHEN** a tag matching `v*.*.*` with a prerelease suffix (e.g. `v0.1.0-alpha.1`) is pushed
- **THEN** the built images are tagged with the exact version (`0.1.0-alpha.1`) and the commit SHA tag, but NOT `latest`

#### Scenario: Stable release tag pushed
- **WHEN** a tag matching `v*.*.*` with no prerelease suffix (e.g. `v1.0.0`) is pushed
- **THEN** the built images are tagged with the exact version, `{major}.{minor}`, the commit SHA tag, and `latest`
</content>

### Requirement: A release publishes no chart without its images
The release workflow MUST NOT publish the Helm chart for a pushed
`v*.*.*` tag unless that same run successfully built, scanned and pushed
both container images. The chart names those images through its
`appVersion`, so publishing it on its own would produce an immutable,
signed artifact pointing at image tags that do not exist.

#### Scenario: Image vulnerability scan fails on a tagged release
- **WHEN** a `v*.*.*` tag is pushed and the images job fails (for
  example, its Trivy scan of the published digest finds a CRITICAL or
  HIGH vulnerability)
- **THEN** the Helm chart job does not run
- **AND** no chart version is pushed to the OCI registry for that tag

#### Scenario: Images publish successfully
- **WHEN** a `v*.*.*` tag is pushed and both component images build,
  scan, push, sign and attest successfully
- **THEN** the Helm chart job runs and publishes the chart at the same
  version
