// Runtime configuration, loaded before the app bundle (see index.html). This
// checked-in default keeps local dev, tests, and `vite preview` on the mock
// backend, matching today's behavior. The production Docker image
// regenerates this file from the container's API_BASE_URL env var at startup
// (see frontend/docker/) so one built image can point at any backend without
// rebuilding — see src/config.ts for how it's consumed.
window.__RUNTIME_CONFIG__ = {
  API_BASE_URL: '',
};
