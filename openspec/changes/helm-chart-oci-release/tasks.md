## 1. release.yml

- [x] 1.1 Add a `helm-chart` job (same triggers as `images`): checkout,
      resolve version (tag / `workflow_dispatch` input / SHA fallback,
      strip leading `v`), install Helm (same pinned, hash-verified installer
      as `ci.yml`'s `helm-lint` job), `helm lint helm/team-manager`,
      `helm package --version --app-version --destination`, `helm registry
      login` to `ghcr.io`, `helm push` to `oci://ghcr.io/<owner>/charts`,
      install cosign, `cosign sign --yes` the pushed digest.
- [x] 1.2 Job permissions: `contents: read`, `packages: write`,
      `id-token: write` (cosign keyless).

## 2. Docs

- [x] 2.1 Add a "Helm chart" subsection to `docs/operations.md`'s
      "Container images & releases" section: what gets pushed/signed and
      where, and the `helm upgrade --install ... oci://...` install/upgrade
      command.

## 3. Verification

- [x] 3.1 `openspec validate helm-chart-oci-release --strict` — passes.
- [x] 3.2 `.github/workflows/release.yml` stays valid YAML (`python3 -c
      "import yaml; yaml.safe_load(...)"`) and the new job's steps follow
      the existing `images` job's conventions (pinned action SHAs,
      hash-verified curl|bash installer).
- [ ] 3.3 Local dry run: `helm lint helm/team-manager` and
      `helm package helm/team-manager --version 0.0.0-test` — not run; the
      sandbox's network proxy blocks `get.helm.sh` (403 on the CONNECT
      tunnel), so Helm couldn't be installed locally. The pinned installer
      matches `ci.yml`'s already-green `helm-lint` job exactly, and the new
      `helm lint`/`helm package` steps use the same chart path; leave this
      for actual CI to confirm.
