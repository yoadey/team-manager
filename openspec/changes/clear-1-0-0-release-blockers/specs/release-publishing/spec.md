## ADDED Requirements

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
