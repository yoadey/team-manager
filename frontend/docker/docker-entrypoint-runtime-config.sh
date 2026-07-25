#!/bin/sh
# Runs automatically before nginx starts (nginx-unprivileged's inherited
# ENTRYPOINT executes every executable script in /docker-entrypoint.d/).
#
# Regenerates config.js from the API_BASE_URL/SENTRY_DSN/VAPID_PUBLIC_KEY/
# OPERATOR_* env vars, and re-templates index.html's CSP connect-src from
# API_BASE_URL, at container start — so the same built image can be pointed
# at any backend/Sentry project/VAPID keypair, and carry any operator's own
# legal-notice/privacy-policy identity data, without rebuilding — see
# src/config.ts, src/features/legal/content.ts and docs/operations.md.
# Scoped to exactly the named vars (not nginx's built-in
# envsubst-on-templates mechanism, which substitutes every env var and would
# leak unrelated container env into these public, browser-served files).
# index.html doesn't need any of these substituted -- its CSP already allows
# Sentry's ingest host generically (https://*.ingest.sentry.io) independent
# of which DSN is set, VAPID_PUBLIC_KEY is never used in a
# network-connecting context that CSP governs (it's only passed to
# PushManager.subscribe(), not fetched from), and the OPERATOR_* fields are
# plain text rendered by React, not URLs CSP needs to allow-list.
#
# OPERATOR_* values are operator-supplied deploy-time config (the same trust
# level as API_BASE_URL/SENTRY_DSN already substituted here), not
# third-party/end-user input.
set -eu

: "${API_BASE_URL:=}"
: "${SENTRY_DSN:=}"
: "${VAPID_PUBLIC_KEY:=}"
: "${OPERATOR_NAME:=}"
: "${OPERATOR_LEGAL_FORM:=}"
: "${OPERATOR_STREET:=}"
: "${OPERATOR_POSTAL_CODE:=}"
: "${OPERATOR_CITY:=}"
: "${OPERATOR_REPRESENTED_BY:=}"
: "${OPERATOR_PHONE:=}"
: "${OPERATOR_EMAIL:=}"
: "${OPERATOR_REGISTER_COURT:=}"
: "${OPERATOR_REGISTER_NUMBER:=}"
: "${OPERATOR_VAT_ID:=}"
: "${OPERATOR_DATA_PROTECTION_EMAIL:=}"
: "${OPERATOR_S3_PROVIDER:=}"
: "${OPERATOR_SMTP_PROVIDER:=}"
: "${OPERATOR_SENTRY_PROVIDER:=}"
: "${OPERATOR_OTEL_PROVIDER:=}"

envsubst '${API_BASE_URL} ${SENTRY_DSN} ${VAPID_PUBLIC_KEY} ${OPERATOR_NAME} ${OPERATOR_LEGAL_FORM} ${OPERATOR_STREET} ${OPERATOR_POSTAL_CODE} ${OPERATOR_CITY} ${OPERATOR_REPRESENTED_BY} ${OPERATOR_PHONE} ${OPERATOR_EMAIL} ${OPERATOR_REGISTER_COURT} ${OPERATOR_REGISTER_NUMBER} ${OPERATOR_VAT_ID} ${OPERATOR_DATA_PROTECTION_EMAIL} ${OPERATOR_S3_PROVIDER} ${OPERATOR_SMTP_PROVIDER} ${OPERATOR_SENTRY_PROVIDER} ${OPERATOR_OTEL_PROVIDER}' < /etc/nginx/templates/config.js.template > /usr/share/nginx/html/config.js
envsubst '${API_BASE_URL}' < /etc/nginx/templates/index.html.template > /usr/share/nginx/html/index.html
