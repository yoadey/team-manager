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
# Pinned to the alpine3.23 variant (rather than the 1.27-alpine tag previously
# here) for current, patched OS packages — 1.27-alpine's Alpine base carried
# several HIGH/CRITICAL CVEs (openssl, libpng, libxml2, musl) that the newly
# added Trivy frontend scan caught. Re-resolved 2026-07-08 (from
# registry-1.docker.io) after the previous digest's frozen libexpat 2.8.1-r0
# picked up 3 fresh HIGH CVEs disclosed since it was pinned.
FROM nginxinc/nginx-unprivileged:1-alpine3.23@sha256:6320020c7da8714feab524e02c08c5a1958675c4e68700e93a2fd8970b065786 AS runtime

# envsubst (gettext) ships in the base image for its own built-in
# entrypoint.d templating feature; fail the build loudly here rather than
# discovering it's missing only when a container starts.
RUN which envsubst

# Replace the default site with our SPA + security-header config.
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY security-headers.conf /etc/nginx/conf.d/security-headers.conf
# --chown=101:101 (nginx-unprivileged's non-root user): plain COPY defaults
# to root:root regardless of the image's USER directive, since these files
# come from the root-run builder stage. docker-entrypoint-runtime-config.sh
# below runs as uid 101 (containers don't switch users between ENTRYPOINT
# and CMD) and needs write permission on index.html specifically — an
# `envsubst > existing-file` redirect opens the existing inode for writing,
# which fails on a root-owned, group/other-unwritable file even though the
# directory itself is otherwise servable.
COPY --chown=101:101 --from=builder /app/dist /usr/share/nginx/html

# index.html.template is the *built* index.html (correct hashed asset paths)
# re-used as an envsubst template for its CSP connect-src placeholder — see
# docker-entrypoint-runtime-config.sh, which runs before nginx starts (via
# the base image's inherited ENTRYPOINT) and overwrites both this and
# config.js in /usr/share/nginx/html from the API_BASE_URL env var.
COPY --from=builder /app/dist/index.html /etc/nginx/templates/index.html.template
COPY docker/config.js.template /etc/nginx/templates/config.js.template
COPY --chmod=755 docker/docker-entrypoint-runtime-config.sh /docker-entrypoint.d/40-runtime-config.sh

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

CMD ["nginx", "-g", "daemon off;"]
