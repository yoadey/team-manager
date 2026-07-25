## ADDED Requirements

### Requirement: Optional frontend deployment
The chart MUST be able to deploy the frontend image (Deployment, Service,
Ingress, and optionally an HPA and PodDisruptionBudget) as well as the
backend, gated on `frontend.enabled` (default `false`).

#### Scenario: Frontend disabled (default)
- **WHEN** `frontend.enabled` is `false` (the default)
- **THEN** no frontend Deployment/Service/Ingress/HPA/PDB/NetworkPolicy is
  rendered, and every backend resource renders identically to a chart
  release with no `frontend` values set at all

#### Scenario: Frontend enabled
- **WHEN** `frontend.enabled` is `true`
- **THEN** the chart renders a frontend Deployment running
  `frontend.image.repository` (tag/digest per the same precedence as the
  backend's `image`), a Service on `frontend.service.port` targeting
  `frontend.service.targetPort`, and — when `frontend.ingress.enabled` is
  also `true` — an Ingress for it

### Requirement: Frontend pods use a distinct selector identity
Frontend-managed resources MUST use an `app.kubernetes.io/name` distinct
from the backend's, so that no backend-scoped selector (NetworkPolicy,
PodDisruptionBudget, Service) can incidentally match frontend pods or vice
versa.

#### Scenario: Frontend and backend both enabled
- **WHEN** `frontend.enabled` is `true`
- **THEN** the backend's `NetworkPolicy`, `PodDisruptionBudget`, and
  `Service` selectors match only backend pods, and the frontend's
  corresponding resources match only frontend pods

### Requirement: Frontend runtime configuration as plain values
The chart MUST expose the frontend's runtime configuration
(`API_BASE_URL`, `SENTRY_DSN`, `VAPID_PUBLIC_KEY`, every `OPERATOR_*` env
var) as plain `frontend.*` values rendered directly as container env vars
— no Secret is created or referenced for these.

#### Scenario: VAPID public key not explicitly set on the frontend
- **WHEN** `push.publicKey` is set and `frontend.vapidPublicKey` is left
  at its empty default
- **THEN** the frontend container's `VAPID_PUBLIC_KEY` env var resolves to
  `push.publicKey`'s value

#### Scenario: VAPID public key explicitly overridden
- **WHEN** both `push.publicKey` and `frontend.vapidPublicKey` are set to
  different values
- **THEN** the frontend container's `VAPID_PUBLIC_KEY` env var uses
  `frontend.vapidPublicKey`'s value, not `push.publicKey`'s

### Requirement: Frontend NetworkPolicy limited to DNS egress
When `frontend.networkPolicy.enabled` is `true`, the frontend's
NetworkPolicy MUST NOT permit any egress beyond DNS resolution — the
frontend container never originates outbound application traffic itself.

#### Scenario: Frontend NetworkPolicy rendered
- **WHEN** `frontend.enabled` and `frontend.networkPolicy.enabled` are
  both `true`
- **THEN** the rendered NetworkPolicy's only egress rule is DNS
  (port 53, TCP and UDP)
