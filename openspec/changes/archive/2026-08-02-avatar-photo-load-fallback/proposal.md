## Why

Reported bug: on mobile, the user's own photo is missing both in the top-right avatar (`frontend/src/layouts/AppShell.tsx`'s `MyAvatar`, used in `MobileShell`) and in the "Mein Konto" / account settings sheet (`frontend/src/features/team/components/NavSheets.tsx`'s `ProfileSheet`, via the shared `Av` component in `frontend/src/components/ui.tsx`).

Investigation (code review + a Playwright reproduction against the mock backend, both mobile and desktop viewports) found no mobile-specific rendering path — `MyAvatar` and `Av` are single shared implementations, byte-identical between breakpoints, and render correctly when the photo loads successfully. The actual defect is independent of viewport: both components decide "show photo" purely from *whether a photo URL string is present* (`user.photo`/`photo` truthy), not from whether the underlying image actually finished loading. The URL is rendered as a CSS `background-image`, which has no error event — so when the request behind that URL fails (expired presigned S3 URL, a slow/flaky mobile network dropping the request, an object-store host unreachable from the client), the `Box` renders with `background-image: url(...)` pointing at a failed resource and **shows nothing at all**: no photo, no colored-initials fallback. Mobile connections are more prone to exactly this kind of flaky/slow image load than desktop broadband, which is consistent with the bug being noticed there first.

## What Changes

- `Av` (`frontend/src/components/ui.tsx`) and `MyAvatar` (`frontend/src/layouts/AppShell.tsx`) detect a failed photo load (via a preloaded `Image()`) and fall back to the colored-initials placeholder instead of rendering an empty circle.
- `MyAvatar` is reimplemented on top of the shared `Av` component instead of duplicating its CSS background-image logic, so the fix (and any future avatar-rendering change) lives in one place.

## Capabilities

### Modified Capabilities
- `profile-photos`: avatar rendering now falls back to initials when a photo URL is present but fails to load, not just when no photo URL exists at all.

## Impact

- Frontend only: `frontend/src/components/ui.tsx` (`Av`), `frontend/src/layouts/AppShell.tsx` (`MyAvatar`), and their existing tests.
- No API/spec change, no backend change.
- CI: frontend lint/typecheck/test/build + bundle budget.
