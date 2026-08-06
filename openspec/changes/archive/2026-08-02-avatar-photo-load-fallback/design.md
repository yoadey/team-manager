## Context

`Av` (`frontend/src/components/ui.tsx`) is the shared avatar renderer used everywhere a person is shown (members, attendance, comments, poll voters, finance rows, stats, notifications, and the profile sheet). `MyAvatar` (`frontend/src/layouts/AppShell.tsx`) duplicates the same pattern just for the signed-in user's own avatar in the shell chrome. Both render a photo as a CSS `background-image` chosen purely by "is there a URL string", with no way to detect or react to the underlying image request failing.

## Goals / Non-Goals

**Goals:**
- When a photo URL exists but the image fails to load, fall back to the same colored-initials placeholder already used for "no photo".
- Fix the logic once, in `Av`, and have `MyAvatar` reuse it instead of keeping a second copy.

**Non-Goals:**
- Retrying failed loads, showing a loading spinner, or caching load state across renders/components.
- Team logo/photo (`TeamIcon` in `AppShell.tsx`) — not reported, out of scope for this change.
- Diagnosing *why* a given deployment's photo requests fail (network/S3/CORS) — this only fixes the UI's behavior once a load does fail.

## Decisions

- Detect failure with a side-effect `Image()` preload: on mount/`photo` change, create `new Image()`, set `.onerror` to flip local `failed` state to `true`, set `.src = photo`. Reset `failed` to `false` whenever `photo` changes. This is the standard way to get a load/error signal out of something rendered as a CSS background rather than an `<img>` element (chosen originally, and kept, for `object-fit: cover`-style sizing via `background-size: cover`).
- Render initials whenever `!photo || failed`, exactly mirroring the existing "no photo" branch — no new visual variant.
- `MyAvatar` becomes a thin wrapper: `<Av name={user.name} photo={user.photo} color={user.avatarColor} size={38} font={13} />`, dropping its separate `Box` markup. `ButtonBase` call sites (`AppShell.tsx:459`, `:539`) are unaffected since they only wrap `<MyAvatar user={...} />`.

## Risks / Trade-offs

- One extra `Image()` request is issued for every `Av` render with a `photo` — this duplicates the browser's own `background-image` fetch (browsers dedupe identical-URL requests via the HTTP cache, so in practice this is a cache hit, not a second network round-trip, once the CSS load has started).
- `Av` is used in lists (attendance, members) where many instances mount at once; each gets its own tiny `Image()` preload. Existing `photoUrl()` cache-busting (`?v=timestamp`, recomputed only per API response, not per render) means these stay stable across re-renders, so no request storm on unrelated state changes.
