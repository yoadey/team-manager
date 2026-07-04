#!/bin/sh
# Runs automatically before nginx starts (nginx-unprivileged's inherited
# ENTRYPOINT executes every executable script in /docker-entrypoint.d/).
#
# Regenerates config.js from the API_BASE_URL/SENTRY_DSN env vars, and
# re-templates index.html's CSP connect-src from API_BASE_URL, at container
# start — so the same built image can be pointed at any backend/Sentry
# project without rebuilding — see src/config.ts and docs/operations.md.
# Scoped to exactly the named vars (not nginx's built-in envsubst-on-templates
# mechanism, which substitutes every env var and would leak unrelated
# container env into these public, browser-served files). index.html doesn't
# need SENTRY_DSN substituted — its CSP already allows Sentry's ingest host
# generically (https://*.ingest.sentry.io), independent of which DSN is set.
set -eu

: "${API_BASE_URL:=}"
: "${SENTRY_DSN:=}"

envsubst '${API_BASE_URL} ${SENTRY_DSN}' < /etc/nginx/templates/config.js.template > /usr/share/nginx/html/config.js
envsubst '${API_BASE_URL}' < /etc/nginx/templates/index.html.template > /usr/share/nginx/html/index.html
