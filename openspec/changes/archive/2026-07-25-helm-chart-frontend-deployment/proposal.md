## Why

`helm/team-manager` deploys the backend API only. `docs/operations.md`'s
"Frontend image: pointing it at a backend" section says so explicitly:
*"there is currently no Helm/Kubernetes manifest for deploying the frontend
image itself (only the backend has one under `helm/team-manager/`); until
one exists, deploy the frontend container by whatever means fits your
infrastructure."* Chart.yaml's own description ("backend API + optional
frontend") already implied the chart should cover both — it never did.

Confirmed directly (PR #110 review feedback): the chart is missing the
frontend entirely, and it does not need to preserve backward compatibility
with the current `values.yaml` shape while fixing this — no tagged release
exists yet (`Chart.yaml` `version: 0.1.0`), so there is no deployed values
file this would break.

The frontend image (`ghcr.io/yoadey/team-manager-frontend`, built by the
same `release.yml` that builds the backend) is a stateless nginx-unprivileged
static-SPA server whose entire per-deployment configuration — which backend
it talks to, its Sentry project, its VAPID keypair, and the operator's
legal-notice identity — is resolved at container start via **plain,
non-secret environment variables** (`frontend/docker/docker-entrypoint-runtime-config.sh`
regenerates `config.js` and re-templates `index.html`'s CSP before nginx
starts). Unlike the backend, nothing here needs a Kubernetes Secret.

## What Changes

- New `frontend.*` values section (mirroring the backend's existing shape:
  `enabled`, `image`, `replicaCount`, `resources`, probes,
  `podSecurityContext`/`securityContext`, `service`, `ingress`,
  `autoscaling`, `podDisruptionBudget`, `networkPolicy`,
  `nodeSelector`/`tolerations`/`affinity`, `priorityClassName`,
  `topologySpreadConstraints`, `podAnnotations`/`podLabels`,
  `extraEnv`/`extraVolumes`/`extraVolumeMounts`), off by default
  (`frontend.enabled: false`) — matching the chart's existing convention
  for infra-cost-affecting optional features (`ingress.enabled`,
  `monitoring.enabled`, `backup.enabled` are all opt-in the same way).
- Frontend-specific config as plain values (no Secret): `apiBaseUrl`,
  `sentryDsn`, `vapidPublicKey` (defaults to `.Values.push.publicKey` when
  unset, so the two can't silently drift out of sync — the exact failure
  mode `docs/operations.md` already warns about), and an `operator.*` block
  for every `OPERATOR_*` env var `docs/operations.md`'s "Legal setup before
  going public" section documents.
- New templates under `templates/frontend/`: `deployment.yaml`,
  `service.yaml`, `ingress.yaml`, `hpa.yaml`, `pdb.yaml`,
  `networkpolicy.yaml` (egress limited to DNS only — nginx never makes
  outbound calls itself; every API call happens browser-side against
  `apiBaseUrl`, not from this pod).
- New `_helpers.tpl` identity helpers
  (`team-manager.frontend.name`/`fullname`/`labels`/`selectorLabels`) so
  frontend pods carry a **distinct** `app.kubernetes.io/name` from the
  backend's — required so the backend's existing NetworkPolicy/
  PodDisruptionBudget/Service selectors (all keyed on
  `team-manager.selectorLabels`) don't silently start matching frontend
  pods too (Kubernetes selectors match on label presence, not exclusivity,
  the same hazard `pdb.yaml`'s existing backup-exclusion comment already
  documents for this exact chart).
- `values.schema.json` coverage for the whole `frontend` block.
- `README.md` values table extended; `docs/operations.md`'s "no
  Helm/Kubernetes manifest" paragraph rewritten to point at the new
  `frontend.*` values instead.
- CI: `helm lint --strict` / `helm template` / `kubeconform` coverage for
  `frontend.enabled=true` and its own conditional branches, following
  `ci.yml`'s existing per-branch-comment convention.

### Explicitly out of scope

- Single-Ingress path-based routing between frontend and backend. The
  backend's OpenAPI routes have no shared prefix (`/auth/login`, `/teams`,
  ... directly at the root, not under `/api`), so unambiguous path-based
  routing on one hostname isn't feasible without hardcoding the entire
  route list into Ingress rules. `frontend.ingress` is therefore its own
  independent Ingress, shaped identically to the existing (backend)
  `ingress` block — typically pointed at a separate hostname (e.g.
  `team-manager.example.com` for the frontend, `api.team-manager.example.com`
  for the backend), the same topology `docs/operations.md` already
  describes for a plain-Docker deployment.
- `readOnlyRootFilesystem` for the frontend container. The runtime-config
  entrypoint script overwrites `config.js`/`index.html` in place under
  `/usr/share/nginx/html` on every container start — a read-only root
  filesystem would break that unless the built static assets were somehow
  re-staged into a writable volume at startup, which would diverge from
  the documented/tested Docker Compose behavior. Every other hardening
  control (`runAsNonRoot` — already the image's own default UID 101 —
  `allowPrivilegeEscalation: false`, all capabilities dropped,
  `seccompProfile: RuntimeDefault`) still applies.
- A ConfigMap/Secret-reference pattern for `frontend.operator.*` (mirroring
  the backend's `create`-or-`existingSecret` convention). These values are
  a club's already-public legal-notice contact details (name, business
  address, phone, email — precisely what German `§5 DDG` requires be shown
  on the page itself), not credentials; plain values match how
  `docs/operations.md`'s own Docker example already passes them
  (`-e OPERATOR_NAME=... -e OPERATOR_STREET=...`, no secret involved).

## Capabilities

### Modified Capabilities

- `helm-deployment`: the chart now also deploys the frontend (Deployment,
  Service, Ingress, optional HPA/PDB, its own NetworkPolicy), off by
  default, with runtime configuration (backend URL, Sentry DSN, VAPID
  public key, operator legal-notice identity) as plain values — no new
  Secret.

## Impact

- `helm/team-manager/`: `values.yaml`, `values.schema.json`,
  `templates/_helpers.tpl`, new `templates/frontend/*.yaml`, `README.md`.
- `values-staging.yaml`/`values-prod.yaml`: example `frontend.*` overrides
  (image tag, ingress host, `apiBaseUrl`) — deliberately **not** including
  real `operator.*` values (a club's real personal/business contact
  details) in a values file committed to this public repository; left as
  placeholders with a pointer to set them via `--set`/an untracked overlay
  at actual deploy time.
- `docs/operations.md`: rewrites the "no Helm/Kubernetes manifest" note.
- `.github/workflows/ci.yml`: new `--set frontend.enabled=true ...`
  coverage in the existing `helm-lint` job.
- No application code, API, or database schema change; no migration.
  Chart-only.
