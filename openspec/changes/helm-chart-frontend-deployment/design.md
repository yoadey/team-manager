## Context

The backend-only chart already has a rich, opinionated set of conventions
(per-area externally-managed Secrets with kebab-case `secret.keys`
overrides, `team-manager.labels`/`selectorLabels` helpers, `component`
labeling, escape hatches, digest-pinnable images) established across this
change and the immediately preceding `helm-chart-best-practices` change.
The frontend addition should read as a natural extension of those
conventions, not a bolted-on second chart.

## Goals / Non-Goals

**Goals:**
- The chart can deploy the frontend end to end: Deployment, Service,
  Ingress, with the same production-readiness knobs (HPA, PDB, probes,
  security context, escape hatches) the backend already has.
- Frontend pods are unambiguously selectable/excludable from every
  backend-scoped selector (NetworkPolicy, PDB, Service) — no accidental
  cross-matching.
- `frontend.vapidPublicKey` can't silently drift from the backend's
  `push.publicKey` unless a deployer deliberately overrides it.
- Off by default, so `helm template`/`install` with no overrides renders
  exactly what it did before this change (backend only) — not for
  backward-compatibility's sake (explicitly not a constraint here) but
  because an opt-in default is this chart's existing convention for every
  other infra-cost-affecting feature (`ingress`, `monitoring`, `backup`).

**Non-Goals:**
- Restructuring the existing top-level values (`image`, `resources`,
  `service`, `ingress`, ...) into a `backend.*` namespace for symmetry with
  the new `frontend.*` section. Tempting for consistency, but it would
  touch every template in the chart a second time in the same PR cycle
  for a purely cosmetic rename, with no behavioral benefit — the existing
  top-level values already unambiguously mean "the backend" (nothing else
  existed before this change), and `frontend.*` being the new, explicitly
  namespaced section is enough to disambiguate going forward. Revisit only
  if a real second reason to rename shows up.
- Single-Ingress path-based frontend/backend routing (see proposal.md's
  "Explicitly out of scope").
- `readOnlyRootFilesystem` for the frontend container (see proposal.md).

## Decisions

### Distinct pod identity (`team-manager.frontend.*` helpers)

`team-manager.selectorLabels` (`{name: <chart-name>, instance:
<release>}`) is reused as the `podSelector`/`matchLabels` for the
backend's `NetworkPolicy`, `PodDisruptionBudget`, and `Service` — none of
which are scoped to exclude a second Deployment sharing those same two
labels, the same "selectors match on presence, not exclusivity" hazard
`pdb.yaml`'s own comment already documents for the backup CronJob's pods
(solved there with a `component NotIn [backup]` `matchExpressions`
exclusion). Bolting frontend pods onto the *same* `name`/`instance` labels
would require retrofitting that same exclusion onto three more resources
(NetworkPolicy, PDB, Service) — and would still leave the *Service*
ambiguous, since a Service's `spec.selector` doesn't support
`matchExpressions` at all (only a flat label-equality map), so there is no
way to exclude by `component` there.

Instead: new helpers give the frontend Deployment/Service/etc. their own
`app.kubernetes.io/name` (`team-manager.frontend.name`, e.g.
`team-manager-frontend`) and therefore their own, naturally
non-overlapping `selectorLabels`. This is the same mechanism Kubernetes
itself expects for two independently-scaled components of one logical
application (see `app.kubernetes.io/name` in the Kubernetes recommended
labels docs) — cleaner than a shared name plus per-resource exclusion
logic, and it means the backend's existing `templates/networkpolicy.yaml`,
`templates/pdb.yaml`, and `templates/service.yaml` need **zero changes**
for this to be correct.

### No separate frontend ServiceAccount

Unlike `backup.serviceAccount` (which exists because the backup CronJob
may need IRSA-style S3 write credentials the main app must never inherit),
the frontend nginx container makes no Kubernetes API calls and no calls to
any AWS/cloud API — there's no credential-scoping reason for it to have
its own identity. It reuses `team-manager.serviceAccountName` (the same
ServiceAccount the backend Deployment uses), with
`automountServiceAccountToken: false` already the shared default.

### `vapidPublicKey` derives from `push.publicKey` by default

`docs/operations.md` already warns that mismatching the frontend's
`VAPID_PUBLIC_KEY` against the backend's fails `PushManager.subscribe()`
client-side with "a benign-looking error, not a clear 'wrong key'
message." Rather than ship two independent values a deployer has to
remember to keep in sync, `frontend.vapidPublicKey` defaults to
`.Values.push.publicKey` (only actually rendered as an env var when the
resolved value is non-empty) — a deployer who sets `push.publicKey` once
gets a correctly-wired frontend for free; overriding
`frontend.vapidPublicKey` explicitly still works for the (unusual) case of
intentionally pointing this frontend release at a different backend/VAPID
keypair than the one this same chart release's `push.*` section describes.

### No Secret for `frontend.*`

Every frontend runtime value (`apiBaseUrl`, `sentryDsn`, `vapidPublicKey`,
every `operator.*` field) is either not sensitive at all (a public API
URL, a public VAPID key, a club's own already-public legal contact
details) or — for `sentryDsn` — already sent to the browser as plain
JavaScript regardless of how it's injected server-side, so gating it
behind a Kubernetes Secret would add the `create`/`existingSecret`
machinery's complexity without adding any actual confidentiality. Plain
`values.yaml` strings, rendered directly as container `env:` entries.

### Frontend `NetworkPolicy`: egress limited to DNS

The frontend container never originates outbound traffic of its own —
every API call the SPA makes happens from the **browser**, directly
against `apiBaseUrl`, not proxied through this nginx pod server-side. Its
`networkpolicy.yaml` therefore only needs the same DNS-resolution egress
rule the backend's does, and an ingress rule scoped the same way the
backend's is (`frontend.networkPolicy.ingress.from`, open by default,
restrictable).

## Risks / Trade-offs

- Two independent `ingress` value shapes (existing top-level `ingress.*`
  for the backend, new `frontend.ingress.*`) rather than one unified
  multi-service ingress block. Slightly more values-file boilerplate for a
  deployer wiring up both, but avoids modeling path-based routing this
  API's route shape can't actually support cleanly (see proposal.md).
- `frontend.enabled: false` by default means this change, by itself, does
  not make `docs/operations.md`'s "no Helm/Kubernetes manifest for the
  frontend" problem go away for an existing deployer who upgrades the
  chart without opting in — they have to notice and set
  `frontend.enabled: true`. Accepted: matches this chart's existing
  opt-in convention, and the alternative (defaulting to enabled) would
  make every `helm template`/`lint` invocation in this chart's own CI
  suite start rendering frontend resources unconditionally, including in
  test matrices that don't set a frontend image tag.
