## Context

`config.Config.ImageDeliveryProxyEnabled` toggles whether image-delivery endpoints stream object-store bytes through the backend (proxy mode, for deployments where the object store isn't reachable directly from the browser) or redirect to a presigned URL (default). `teams.Handler` and `members.Handler` both implement this per their own `imageDeliveryProxyEnabled bool` field + `SetImageDeliveryProxyEnabled` method, wired from `cmd/server/main.go`. `auth.Handler.GetMyPhoto` (the signed-in user's own photo) predates this flag's rollout to `auth` and was never updated — it always redirects.

## Goals / Non-Goals

**Goals:**
- `GET /auth/me/photo` respects `imageDeliveryProxyEnabled` exactly like `GetTeamPhoto`/`GetMemberPhoto`/`GetTeamLogo`.

**Non-Goals:**
- Refactoring the three packages' near-identical branching into one shared helper across package boundaries (`teams.deliverImage` is already local-generic within `teams` for its two photo/logo call sites; `members` and `auth` each have a single call site and follow `members`' simpler inline-branch style — consistent with the existing convention of not sharing this logic across packages).

## Decisions

- Mirror `members.Handler.GetMemberPhoto`/`members.Service.GetMemberPhotoBytes` exactly: add `auth.Service.GetMyPhotoBytes(ctx, userID) (io.ReadCloser, string, error)` (object-store `Get` on the same key `GetMyPhotoURL` presigns), add `imageDeliveryProxyEnabled` + `SetImageDeliveryProxyEnabled` to `auth.Handler`, branch in `GetMyPhoto`.
- `openapi.yaml`: add `"200": $ref: "#/components/responses/PhotoBytes"` to `getMyPhoto`, identical to `getTeamPhoto`. Regenerate via `make generate` (Go) and `make generate-ts` (TS) at the repo root.
- Wire `authHandler.SetImageDeliveryProxyEnabled(cfg.ImageDeliveryProxyEnabled)` in `cmd/server/main.go` right after `initAuthComponents`, matching where `teamsHandler`/`membersHandler` are wired.

## Risks / Trade-offs

- None beyond the standard spec-first regen risk (drift between `openapi.yaml` and generated clients) — covered by the `backend-openapi-drift` CI gate.
