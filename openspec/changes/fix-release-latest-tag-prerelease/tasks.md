## 1. Workflow fix

- [x] 1.1 In `.github/workflows/release.yml`, change the `type=raw,value=latest,...` line's `enable` condition so it's true only for a tag push whose ref name has no prerelease suffix (i.e. no `-` after the `vMAJOR.MINOR.PATCH` core), instead of `${{ github.ref_type == 'tag' }}` alone. Implemented as `enable=${{ github.ref_type == 'tag' && !contains(github.ref_name, '-') }}`.
- [x] 1.2 Leave the `type=semver` entries as-is — their prerelease handling (`{{version}}` only, `{{major}}.{{minor}}` suppressed) is already correct and unaffected by this change.

## 2. Verification

- [x] 2.1 Traced the resulting tag set by hand for both a prerelease ref (`v0.1.0-alpha.1`) and a stable ref (`v1.0.0`): prerelease → `github.ref_name` contains `-` → `latest` disabled, so only `0.1.0-alpha.1` (semver `{{version}}`) + `sha-*` are produced (`{{major}}.{{minor}}` already suppressed by metadata-action for prerelease semver); stable → no `-` in `v1.0.0` → `latest` enabled, producing `1.0.0` + `1.0` + `sha-*` + `latest`.
- [x] 2.2 `openspec validate fix-release-latest-tag-prerelease --strict` — passes
- [x] 2.3 `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"` — YAML parses cleanly (no `actionlint`/`yamllint` available in this sandbox)
</content>
