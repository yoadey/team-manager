# Enterprise-Readiness Review — Teamverwaltung Frontend

_Stand: 2026-06-22 · Scope: Frontend. Der gemockte Backend-Layer
(`src/services/serviceLayer.ts`) ist bewusst ausgeklammert und wird in einem
späteren Schritt durch eine echte API ersetzt._

## Zusammenfassung

Die Code-Basis ist bereits überdurchschnittlich reif. Vorhanden sind u. a.:
strikter TypeScript-Build, ESLint (flat config) inkl. `jsx-a11y` und
`no-restricted-syntax` gegen `dangerouslySetInnerHTML`, Prettier, Vitest mit
hohen Coverage-Floors (80 % statements/lines, 75 % functions, 65 % branches),
Komponenten- und A11y-Tests (`vitest-axe`), Playwright-E2E, Lighthouse-CI,
Bundle-Size-Budgets, SBOM-Generierung, `npm audit` im CI, Dependabot, eine
PWA (Service Worker + Manifest + Offline-Seite) sowie Sentry-Monitoring mit
PII-Stripping und Release-Tracking.

Bewertung: **produktionsreifes Fundament**. Die verbleibenden Lücken sind klar
umrissen und liegen primär in Dark-Mode-Korrektheit, i18n-Vollständigkeit,
punktueller A11y und einigen architektonischen Themen (Routing/State).

> Hinweis: Mehrere Punkte aus der ersten automatisierten Analyse waren
> Fehlbefunde (z. B. „TypeScript 6 / Vite 8 / Vitest 4 sind pre-release/veraltet"
> — falsch, das sind die genutzten aktuellen Versionen; „kein Dependabot",
> „offline.html fehlt", „NewsPage-Buttons ohne aria-label" — alle bereits
> vorhanden). Sie sind unten nicht als Maßnahmen geführt.

## Befunde (priorisiert, in scope)

### 1 — Dark Mode faktisch gebrochen · HOCH

`NEUTRAL` ist als CSS-Custom-Properties implementiert und schaltet bei
`data-color-scheme="dark"` automatisch um (`src/styles/tokens.ts`,
`src/styles/theme.ts`). Ein Color-Scheme-Umschalter existiert
(`state.colorScheme: 'system' | 'light' | 'dark'`). Viele Komponenten umgehen
die Tokens jedoch mit **literalen Hex-Farben** (`#fff`, `#44474E`, `#C8CAD2`,
`#E0E2EA`, …), die im Dark Mode nicht mitschalten → helle Karten/Texte auf
dunklem Grund.

- Betroffen: ~30 Dateien in `src/components`, `src/sheets`, `src/features`,
  `src/layouts`, `src/pages`. Hotspot: `EventDetailSheet.tsx`.
- Fix: literale **Neutral**-Farben durch `NEUTRAL.*`-Tokens ersetzen; fehlende
  semantische Töne (Warn-Akzent) als Token ergänzen (erledigt: `NEUTRAL.warn` /
  `NEUTRAL.warnBg`). Reine Akzent-Chips (Event-Typ/Status) tragen eigene
  Hintergründe und sind in beiden Schemata kontrastsicher → niedrigere Priorität.

### 2 — i18n-Leaks (hardcodierte deutsche Strings) · HOCH · ERLEDIGT

Hardcodierte Strings blockierten den mehrsprachigen Betrieb.

- `src/sheets/DialogSheets.tsx`: „Abbrechen", „Bestätigen", „Bist du sicher?",
  Serien-Aktionstexte, Scope-Optionen, Kommentar-Sheet-Texte → auf `t()` migriert.
- `src/components/cards.tsx`: „Abgesagt", „Treff …", deutsche `aria-label` der
  Summary-Zähler → auf `t()` migriert.
- Neue Keys in `de.ts` **und** `en.ts` ergänzt (Parität wird über den
  `Messages`-Typ beim Typecheck erzwungen).

### 3 — A11y-Feinheiten · MITTEL

Grundlage ist gut (Fokus-Management in Sheets, Skip-Link, `role="alert"`,
`aria-live`, modale Semantik, Tab-/Radio-Rollen). Verbleibend:

- Custom-Toggles/Switches auf semantische Rollen prüfen
  (`role="switch"`/`role="checkbox"` + `aria-checked` + Tastaturaktivierung),
  z. B. in `EventFormSheet.tsx`.
- `aria-label` der Summary-Zähler nun lokalisiert (Teil von Befund 2).

### 4 — State-basiertes Routing ohne Deep-Links · MITTEL

`pushRoute` synct nur das Top-Level-Segment (`/events`). Filter
(`eventScope`, `eventsView`, `eventsOnlyPending`, `notifFilter`, `finTab`) und
geöffnete Detail-Sheets liegen nur im State → keine bookmark-/teilbaren URLs;
Browser-Zurück verlässt das ganze Feature statt das Sheet zu schließen.

- Empfehlung: kein `react-router` (Projekt-Philosophie), sondern den bestehenden
  `history`-Sync in `src/context/AppContext.tsx` erweitern: bookmark-relevante
  Filter als Query-Params, Detail-Sheets als Pfadsegmente (`/events/:id`),
  `popstate` schließt Sheets statt das Feature zu verlassen.

### 5 — Monolithischer State-Context · MITTEL

`AppState` bündelt 34 Felder in einem Objekt. State/Actions sind bereits in
`AppStateContext`/`AppActionsContext` getrennt, aber jede State-Änderung
re-rendert alle State-Consumer (teils durch memoisierte Cards abgefedert).

- Empfehlung: selektor-basierter Zugriff (`useAppSelector(selector)` via
  `useSyncExternalStore`) oder Aufteilung in wenige Domänen-Provider; `useApp()`
  als Kompatibilitäts-Shim erhalten.

## Optional / nicht im Umsetzungsumfang

- CI: Sentry-Sourcemap-Upload (Symbolikation in Produktion) und CodeQL/Semgrep-SAST.
- `en.ts`-Übersetzungsqualität fachlich gegenlesen (Struktur-Parität ist erfüllt).
- Offline-Seite an Dark Mode angleichen (statische Seite, geringe Priorität).

## Backend-abhängig (später, bewusst ausgeklammert)

Token-Persistenz/-Refresh, echter OIDC/PKCE-Flow, serverseitige
Permission-Enforcement und -Invalidierung, Audit-Trail, Verschlüsselung at rest,
Session-Recovery nach Reload, automatische Retries bei `NetworkError`.

## Stärken (zur Einordnung)

Strikter TS-Build · umfassende CI (lint/typecheck/test/audit/build/e2e/lighthouse)
· Bundle-Budgets · Security-Header (dev) · Sentry mit PII-Stripping ·
A11y-Tests · hohe Coverage-Floors · SBOM · Dependabot · PWA.
