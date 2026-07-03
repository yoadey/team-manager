// Runtime configuration, loaded before the app bundle (see index.html). This
// checked-in default keeps local dev, tests, and `vite preview` on the mock
// backend, matching today's behavior. The production Docker image
// regenerates this file from the container's API_BASE_URL env var at startup
// (see frontend/docker/) so one built image can point at any backend without
// rebuilding — see src/config.ts for how it's consumed.
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
};
