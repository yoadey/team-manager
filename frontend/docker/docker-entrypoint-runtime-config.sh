#!/bin/sh
# Runs automatically before nginx starts (nginx-unprivileged's inherited
# ENTRYPOINT executes every executable script in /docker-entrypoint.d/).
#
# Regenerates config.js and re-templates index.html's CSP connect-src from
# the API_BASE_URL env var at container start, so the same built image can
# be pointed at any backend without rebuilding — see src/config.ts and
# docs/operations.md. Scoped to exactly ${API_BASE_URL} (not nginx's
# built-in envsubst-on-templates mechanism, which substitutes every env var
# and would leak unrelated container env into these public, browser-served
# files).
set -eu

: "${API_BASE_URL:=}"

envsubst '${API_BASE_URL}' < /etc/nginx/templates/config.js.template > /usr/share/nginx/html/config.js
envsubst '${API_BASE_URL}' < /etc/nginx/templates/index.html.template > /usr/share/nginx/html/index.html
