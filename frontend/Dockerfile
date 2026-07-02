# ── Build stage ───────────────────────────────────────────────────────────────
# Vite inlines import.meta.env.* at build time, so the backend URL and release
# metadata are passed as build args (not runtime env).
FROM node:22-alpine@sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2 AS builder
WORKDIR /app

# Install dependencies against the lockfile for reproducible builds. The
# postinstall hook (patch-react-transition-group.mjs) needs scripts/ present
# before `npm ci` runs it, so that directory is copied alongside the
# manifests rather than deferred to the later `COPY . .` — otherwise npm ci
# fails with MODULE_NOT_FOUND.
COPY package.json package-lock.json ./
COPY scripts/ ./scripts/
RUN npm ci

COPY . .

ARG VITE_API_BASE_URL=""
ARG VITE_BUILD_VERSION="dev"
ARG VITE_BUILD_COMMIT="unknown"
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL \
    VITE_BUILD_VERSION=$VITE_BUILD_VERSION \
    VITE_BUILD_COMMIT=$VITE_BUILD_COMMIT

RUN npm run build && find dist -name '*.map' -delete

# ── Runtime stage ───────────────────────────────────────────────────────────────
# Serve the static bundle with nginx. The image ships no Node runtime or source.
# nginx-unprivileged runs as a non-root user (uid 101) and listens on :8080 out
# of the box, avoiding the pid/temp-dir permission issues of the stock image.
FROM nginxinc/nginx-unprivileged:1.27-alpine@sha256:65e3e85dbaed8ba248841d9d58a899b6197106c23cb0ff1a132b7bfe0547e4c0 AS runtime

# Replace the default site with our SPA + security-header config.
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY security-headers.conf /etc/nginx/conf.d/security-headers.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

CMD ["nginx", "-g", "daemon off;"]
