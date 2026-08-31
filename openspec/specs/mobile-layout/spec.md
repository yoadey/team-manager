# mobile-layout Specification

## Purpose
Defines layout adjustments applied specifically at mobile viewport widths: bottom-anchored navigation and actions must stay fully visible and tappable around the mobile browser's toolbar and device safe area, and the calendar must use tighter cell spacing and smaller corner radii to give day entries more room, while desktop breakpoints retain their existing, roomier layout.

## Requirements

### Requirement: Bottom controls clear the mobile browser chrome
On mobile viewports, bottom-anchored navigation and actions MUST remain fully visible and tappable, accounting for the browser toolbar and device safe area.

#### Scenario: Bottom navigation on a mobile browser
- **WHEN** the app is viewed in a mobile browser whose toolbar overlaps the viewport bottom
- **THEN** the bottom navigation and its symbols are not occluded and remain tappable

### Requirement: Denser calendar on mobile
On mobile viewports, the calendar MUST minimize spacing between day cells and use small corner radii so day entries have more room.

#### Scenario: Calendar on a small screen
- **WHEN** the calendar renders at a mobile breakpoint
- **THEN** there is no gap between adjacent day cells and cell corners use a small radius

#### Scenario: Calendar on desktop
- **WHEN** the calendar renders above the mobile breakpoint
- **THEN** the existing (roomier) spacing and radius are retained
