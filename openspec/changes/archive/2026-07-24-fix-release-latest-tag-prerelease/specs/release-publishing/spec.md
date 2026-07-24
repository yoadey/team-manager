## ADDED Requirements

### Requirement: The `latest` GHCR tag only tracks stable releases
When the release workflow publishes images for a pushed `v*.*.*` tag, it MUST apply the `latest` tag only when that tag is a stable (non-prerelease) semantic version. A prerelease tag (a version with a `-` suffix, e.g. `-alpha.1`, `-beta.2`, `-rc.1`) MUST NOT be published as `latest`.

#### Scenario: Prerelease tag pushed
- **WHEN** a tag matching `v*.*.*` with a prerelease suffix (e.g. `v0.1.0-alpha.1`) is pushed
- **THEN** the built images are tagged with the exact version (`0.1.0-alpha.1`) and the commit SHA tag, but NOT `latest`

#### Scenario: Stable release tag pushed
- **WHEN** a tag matching `v*.*.*` with no prerelease suffix (e.g. `v1.0.0`) is pushed
- **THEN** the built images are tagged with the exact version, `{major}.{minor}`, the commit SHA tag, and `latest`
</content>
