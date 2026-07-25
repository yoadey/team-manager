// Runtime configuration, loaded before the app bundle (see index.html). This
// checked-in default keeps local dev, tests, and `vite preview` on the mock
// backend, matching today's behavior. The production Docker image
// regenerates this file from the container's API_BASE_URL/SENTRY_DSN/
// VAPID_PUBLIC_KEY env vars at startup (see frontend/docker/) so one built
// image can point at any backend/Sentry project/VAPID keypair without
// rebuilding — see src/config.ts for how it's consumed. SENTRY_DSN and
// VAPID_PUBLIC_KEY have no build-time equivalent that reaches the release
// image (the Dockerfile/release.yml only ever pass VITE_API_BASE_URL/
// VITE_BUILD_VERSION/VITE_BUILD_COMMIT as build args), so this runtime path
// is the only way to enable Sentry, or Web Push, in a released frontend
// image at all -- and the only way a rotated backend VAPID keypair reaches
// an already-built frontend image without a rebuild.
//
// Do not delete this file: Vite's default publicDir copy is what puts it in
// dist/, which is what makes it already owned by the image's non-root user
// (frontend/Dockerfile's `COPY --chown=101:101 ... /usr/share/nginx/html`)
// before the container-start entrypoint script overwrites it. Without a
// pre-existing config.js here, that write would instead be creating a new
// file in a root-owned directory, which fails as the non-root user and
// crashes the container at startup (caught by the security-container-frontend
// CI job's boot-test step, but only there — not at build time).
window.__RUNTIME_CONFIG__ = {
  API_BASE_URL: '',
  SENTRY_DSN: '',
  VAPID_PUBLIC_KEY: '',
  // Operator identity/contact + processor-disclosure fields for the legal-notice
  // and privacy-policy pages (frontend/src/features/legal/content.ts). Blank
  // here (as in local dev) renders the [BETREIBER:]/[OPERATOR:] placeholder
  // markers for the always-required fields and omits the optional sections —
  // see docs/operations.md's "Legal setup before going public" section.
  OPERATOR_NAME: '',
  OPERATOR_LEGAL_FORM: '',
  OPERATOR_STREET: '',
  OPERATOR_POSTAL_CODE: '',
  OPERATOR_CITY: '',
  OPERATOR_REPRESENTED_BY: '',
  OPERATOR_PHONE: '',
  OPERATOR_EMAIL: '',
  OPERATOR_REGISTER_COURT: '',
  OPERATOR_REGISTER_NUMBER: '',
  OPERATOR_VAT_ID: '',
  OPERATOR_DATA_PROTECTION_EMAIL: '',
  OPERATOR_S3_PROVIDER: '',
  OPERATOR_SMTP_PROVIDER: '',
  OPERATOR_SENTRY_PROVIDER: '',
  OPERATOR_OTEL_PROVIDER: '',
};
