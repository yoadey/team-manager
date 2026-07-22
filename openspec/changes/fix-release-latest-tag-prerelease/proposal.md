## Why

`.github/workflows/release.yml` derives its GHCR image tags via `docker/metadata-action`. Alongside the `type=semver` entries (which correctly skip `{{major}}.{{minor}}` for prerelease refs, per the action's documented default), the workflow adds its own explicit rule:

```yaml
type=raw,value=latest,enable=${{ github.ref_type == 'tag' }}
```

This is unconditional on *any* tag push — it does not distinguish a stable release (`v1.2.3`) from a prerelease (`v0.1.0-alpha.1`, `v1.0.0-rc.1`). With the first alpha tag (`v0.1.0-alpha.1`) about to be cut, pushing it today would publish that alpha build as `ghcr.io/yoadey/team-manager-backend:latest` / `-frontend:latest`, overwriting whatever meaning `:latest` is expected to carry. Anything that later pulls `:latest` (docs, ad-hoc scripts, a future automation) would silently get an alpha build instead of the newest stable one.

## What Changes

- Gate the `raw,value=latest` tag entry in `.github/workflows/release.yml` so it only applies to non-prerelease semver tags (a tag with no `-` suffix), leaving prerelease tags (`-alpha.`, `-beta.`, `-rc.`, …) publishing only their exact `{{version}}` tag and the `sha-*` tag.
- No change to the `type=semver` entries themselves — their prerelease handling (`{{version}}` only, no `{{major}}.{{minor}}`) is already correct.

## Capabilities

### New Capabilities
- `release-publishing`: rules governing which GHCR image tags a given release-tag push produces.

### Modified Capabilities
<!-- none -->

## Impact

- `.github/workflows/release.yml` only. No API/schema change, no migration, no frontend/backend runtime code touched.
- Affects future tag pushes only — does not retroactively fix any already-published tag (none exist yet; this is ahead of the first release).
