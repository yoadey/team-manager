## 1. `values.yaml`: `frontend` section

- [x] 1.1 Add `frontend.enabled: false`, `frontend.replicaCount`,
      `frontend.image.{repository,pullPolicy,tag,digest}` (repository
      defaults to `ghcr.io/yoadey/team-manager-frontend`).
- [x] 1.2 Add `frontend.apiBaseUrl` (empty default; unset serves the
      frontend's built-in mock backend per `docs/operations.md`),
      `frontend.sentryDsn`, `frontend.vapidPublicKey` (empty default —
      resolved against `.Values.push.publicKey` at template time, not
      hardcoded here).
- [x] 1.3 Add `frontend.operator.{name,legalForm,street,postalCode,city,
      representedBy,phone,email,registerCourt,registerNumber,vatId,
      dataProtectionEmail,s3Provider,smtpProvider,sentryProvider,
      otelProvider}`, all empty strings, with comments cross-referencing
      `docs/operations.md`'s "Legal setup before going public" for which
      are always-required vs. optional-with-omission.
- [x] 1.4 Add `frontend.resources`, `frontend.startupProbe` (omit — nginx
      starts fast, no migration-style slow-start concern),
      `frontend.livenessProbe`/`readinessProbe` (both `httpGet: /healthz`,
      same port).
- [x] 1.5 Add `frontend.podSecurityContext` (`runAsNonRoot: true`,
      `seccompProfile: RuntimeDefault` — no `runAsUser`/`fsGroup`
      override, the nginx-unprivileged image's own default UID 101
      already satisfies `runAsNonRoot`) and `frontend.securityContext`
      (`allowPrivilegeEscalation: false`, capabilities drop `ALL`,
      `seccompProfile: RuntimeDefault` — deliberately no
      `readOnlyRootFilesystem`, see design.md).
- [x] 1.6 Add `frontend.service.{type,port,targetPort}` (`targetPort:
      8080`, matching the frontend image's `EXPOSE`).
- [x] 1.7 Add `frontend.ingress.{enabled,className,annotations,hosts,tls}`,
      shaped identically to the existing top-level `ingress` block.
- [x] 1.8 Add `frontend.autoscaling`, `frontend.podDisruptionBudget`,
      `frontend.networkPolicy.ingress.from` (egress has no equivalent
      knob — see design.md, it's DNS-only and not configurable).
- [x] 1.9 Add `frontend.nodeSelector`/`tolerations`/`affinity` (default
      `podAntiAffinity` keyed to `team-manager.frontend.name`, mirroring
      the backend's), `frontend.priorityClassName`,
      `frontend.topologySpreadConstraints`,
      `frontend.podAnnotations`/`podLabels`,
      `frontend.extraEnv`/`extraVolumes`/`extraVolumeMounts`.
- [x] 1.10 Add a comment above the existing top-level `ingress:` block
      clarifying it is specifically the backend's Ingress, now that a
      second (`frontend.ingress`) exists.

## 2. `values.schema.json`

- [x] 2.1 Add a `frontend` object covering every field from section 1,
      `additionalProperties: false` throughout, matching this schema's
      existing conventions (string/bool/int types, `$ref` reuse of the
      existing `probe`/`egressRule` `$defs` where shapes match).

## 3. `templates/_helpers.tpl`: frontend identity

- [x] 3.1 `team-manager.frontend.name` (`printf "%s-frontend" (include
      "team-manager.name" .)`), `team-manager.frontend.fullname`
      (same pattern against `team-manager.fullname`).
- [x] 3.2 `team-manager.frontend.selectorLabels`
      (`app.kubernetes.io/name`: frontend name, `app.kubernetes.io/instance`:
      release — deliberately distinct from `team-manager.selectorLabels`,
      see design.md).
- [x] 3.3 `team-manager.frontend.labels` (same shape as
      `team-manager.labels` but built on `team-manager.frontend.selectorLabels`
      and `app.kubernetes.io/version` from `frontend.image.tag` (default
      `Chart.AppVersion`), `app.kubernetes.io/component: frontend`).

## 4. `templates/frontend/deployment.yaml`

- [x] 4.1 Single container (no initContainer — nginx has no migration
      step), image via a `team-manager.frontend.image` helper (mirroring
      `team-manager.image`'s digest-takes-precedence-over-tag logic).
- [x] 4.2 `env:` built from `frontend.apiBaseUrl`/`sentryDsn`/
      `vapidPublicKey` (resolved against `.Values.push.publicKey` per
      design.md) → `API_BASE_URL`/`SENTRY_DSN`/`VAPID_PUBLIC_KEY`, and
      every `frontend.operator.*` field → its `OPERATOR_*` env var name,
      plus `frontend.extraEnv` appended (same pattern as the backend's
      `_env.tpl`).
- [x] 4.3 `serviceAccountName: {{ include "team-manager.serviceAccountName" . }}`
      (shared with the backend, see design.md — no separate frontend
      ServiceAccount).
- [x] 4.4 `podLabels`/`podAnnotations`/`extraVolumes`/`extraVolumeMounts`/
      `priorityClassName`/`topologySpreadConstraints`/`nodeSelector`/
      `tolerations`/`affinity` wired the same way the backend Deployment's
      are (task 4-equivalents from `helm-chart-best-practices`).

## 5. `templates/frontend/service.yaml`

- [x] 5.1 `ClusterIP`-by-default Service selecting on
      `team-manager.frontend.selectorLabels`.

## 6. `templates/frontend/ingress.yaml`

- [x] 6.1 Structurally identical to the existing (backend) `ingress.yaml`,
      gated on `frontend.ingress.enabled`, targeting the frontend Service/
      `frontend.service.port`.

## 7. `templates/frontend/hpa.yaml` / `templates/frontend/pdb.yaml`

- [x] 7.1 `autoscaling/v2` HPA gated on `frontend.autoscaling.enabled`,
      targeting the frontend Deployment.
- [x] 7.2 `PodDisruptionBudget` gated on
      `frontend.podDisruptionBudget.enabled`, selecting
      `team-manager.frontend.selectorLabels` (no `component` exclusion
      needed — nothing else shares this selector, per design.md).

## 8. `templates/frontend/networkpolicy.yaml`

- [x] 8.1 Gated on `frontend.networkPolicy.enabled`
      (default `true`, matching the backend's own default),
      `podSelector: team-manager.frontend.selectorLabels`, ingress on
      `frontend.service.targetPort` restrictable via
      `frontend.networkPolicy.ingress.from`, egress limited to DNS
      (port 53 TCP/UDP) only — no S3/SMTP/HTTPS/Postgres rules, per
      design.md.

## 9. Documentation

- [x] 9.1 `README.md`: extend the values table with every `frontend.*`
      key; add a short "Frontend" subsection under "Installing" showing a
      minimal `frontend.enabled=true` example.
- [x] 9.2 `docs/operations.md`: rewrite the "Note: there is currently no
      Helm/Kubernetes manifest for deploying the frontend image itself"
      paragraph to describe `frontend.*` instead.

## 10. `values-staging.yaml` / `values-prod.yaml`

- [x] 10.1 Add example `frontend.enabled: true`, `frontend.ingress.*`
      (mirroring each overlay's existing backend `ingress.hosts`),
      `frontend.apiBaseUrl` (pointing at that overlay's backend ingress
      host). Deliberately leave `frontend.operator.*` unset with a comment
      pointing at setting real values via `--set`/an untracked overlay —
      see proposal.md's Impact section for why real operator PII isn't
      committed here.

## 11. CI

- [x] 11.1 `.github/workflows/ci.yml`'s `helm-lint` job: add a `render`
      invocation with `--set frontend.enabled=true` plus enough of
      `frontend.image.tag`/`frontend.apiBaseUrl`/operator fields to
      exercise the always-required-per-docs `OPERATOR_*` set, covering
      the new templates end to end (kubeconform-validated like every
      other render in that job).
- [x] 11.2 Additional `--set` coverage for `frontend.ingress.enabled=true`,
      `frontend.autoscaling.enabled=true`,
      `frontend.podDisruptionBudget.enabled=false`, and
      `frontend.vapidPublicKey` left unset with `push.publicKey` set
      (proving the derive-by-default behavior renders the resolved value).

## 12. Verification

- [x] 12.1 `helm lint --strict` (default values — `frontend.enabled=false`
      renders no frontend resources — and both overlays with
      `frontend.enabled=true`) passes.
- [x] 12.2 `helm template` for every combination in task 11 renders and
      validates cleanly against `kubeconform`.
- [x] 12.3 A rendered frontend Deployment/Service/NetworkPolicy/PDB is
      inspected to confirm its `app.kubernetes.io/name` differs from the
      backend's, and that the backend's own NetworkPolicy/PDB/Service
      manifests are byte-for-byte unchanged by enabling
      `frontend.enabled=true`.
- [x] 12.4 A rendered frontend Deployment with `push.publicKey` set but
      `frontend.vapidPublicKey` unset shows `VAPID_PUBLIC_KEY` resolved to
      `push.publicKey`'s value.
- [x] 12.5 CI green.
