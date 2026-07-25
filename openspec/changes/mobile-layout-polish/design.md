## Context

The shell renders a bottom navigation in `AppShell.tsx`. Mobile browsers (iOS Safari, Chrome Android) render their toolbar over the bottom of `100vh`, and notched devices reserve a home-indicator area — neither is accounted for, so bottom controls sit under the chrome. The calendar (`EventCalendar.tsx`) uses a grid with gaps and generous border-radius sized for desktop.

## Goals / Non-Goals

**Goals:**
- Bottom nav/actions fully tappable on mobile regardless of browser chrome.
- More usable day-cell area in the mobile calendar.

**Non-Goals:**
- Redesigning navigation or the calendar's interaction model.
- Desktop appearance changes (keep current spacing above the breakpoint).

## Decisions

- Height: switch the shell's full-height container from `100vh` to `100dvh` (with a `vh` fallback) so it tracks the dynamically-sized viewport.
- Safe area: add `padding-bottom: env(safe-area-inset-bottom)` (plus a small base) to the bottom nav; ensure `<meta name="viewport" content="…, viewport-fit=cover">` is set so the inset is non-zero. Apply matching offsets to bottom-anchored `SheetHost`/`Toast` if they currently sit flush.
- Calendar density: at the mobile breakpoint set the grid `gap` to `0` and reduce day-cell `border-radius`; keep desktop values unchanged via the existing responsive styling approach (MUI breakpoints / Emotion).

## Risks / Trade-offs

- `dvh` support: modern mobile browsers support it; keep a `vh` fallback for older engines.
- `viewport-fit=cover` makes `env(safe-area-inset-*)` meaningful but can let content run under rounded corners — apply insets deliberately where needed.
- Zero-gap cells need visible cell borders/dividers so the grid stays readable without the gap.
