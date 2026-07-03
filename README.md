# team-manager

Verwaltung von Trainingsterminen und Spielern für einen Verein.

Produktive React-Umsetzung des Design-Prototyps „Teamverwaltung“ (mandantenfähige
Team-/Vereins-App für den Formations-/Sportbetrieb). Das Design des hochauflösenden
Prototyps wurde **1:1 übernommen**, aber vollständig auf **React + TypeScript + MUI**
aufgebaut. Das Repo enthält ein produktivfähiges **Go/PostgreSQL-Backend** (siehe
`backend/`); `serviceLayer.ts` schaltet automatisch zwischen dem Mock-Backend
(In-Memory + `localStorage`, künstliche Latenz) und echten HTTP-Aufrufen an dieses
Backend um, je nachdem ob `VITE_API_BASE_URL` gesetzt ist.

> Die ausführliche Entwickler-/Architekturdokumentation steht in [`CLAUDE.md`](./CLAUDE.md).

## Tech-Stack

- **React 19 + TypeScript 5 (strict)** + **Vite**
- **MUI (Material UI v6)** + **Emotion** als Komponenten- und Theming-Basis (Material Design 3)
- **Roboto** (`@fontsource/roboto`) + **Material Symbols Outlined** (`material-symbols`)
- **Vitest** (+ Testing Library, `vitest-axe`) für Unit-/Komponententests, **Playwright** für E2E
- **Sentry** für Fehler-Monitoring (optional, via `VITE_SENTRY_DSN`)
- Zentraler App-State über React Context (`src/context/AppContext.tsx`)
- Mock-Backend / API-Vertrag: `src/services/serviceLayer.ts`

## Loslegen

```bash
npm install
npm run dev        # Dev-Server (http://localhost:5173)
npm run build      # Typecheck + Produktions-Build
npm run preview    # Build lokal ansehen
npm run typecheck  # nur TypeScript prüfen
npm run lint       # ESLint
npm test           # Tests einmalig ausführen
```

> Beim Login genügt ein Klick auf einen beliebigen Identity-Provider – der
> Mock-Service meldet automatisch den Demo-Nutzer „Lena Bergmann“ an.

## Projektstruktur

```
src/
├── main.tsx                # Entry: Fonts, Icons, Monitoring, ErrorBoundary, LocaleProvider
├── App.tsx                 # ThemeProvider (MUI) + AppProvider
├── config.ts               # Validierte Runtime-Konfiguration aus VITE_*-Variablen
├── monitoring.ts           # Sentry-Initialisierung + globale Error-Handler
├── components/             # Geteilte UI-Atome (ui.tsx, cards.tsx, ErrorBoundary, Toast, …)
├── context/
│   ├── AppContext.tsx      # Zentraler Zustand (State/Actions-Context-Split) + Helfer
│   └── useFeatureActions.ts# Feature-spezifische Aktionen
├── features/               # Feature-Module: events, members, finances, news, polls,
│   └── <feature>/          #   team, auth, notifications (Page, components/, hooks/, types.ts)
├── i18n/                   # Eigene i18n-Schicht: t()-Kataloge (de/en) + LocaleProvider
├── layouts/                # AppShell (Navigation), useCompact, pageMeta
├── pages/                  # RouteScreen (lazy-load + Per-Route-ErrorBoundary), Home, Stats
├── services/               # Mock-Backend (= API-Vertrag) + Mapper
├── sheets/                 # Sheet-Dispatcher (modale Dialoge / Bottom-Sheets)
├── styles/                 # MUI-Theme-Builder + Design-Tokens (light/dark CSS-Variablen)
├── types/                  # Geteilte Domänen-Typen
└── utils/                  # date.ts, validation.ts, errors.ts, permissions.ts
```

## Architektur & Konzepte

- **State-basiertes Routing** (`state.route`) mit URL-Synchronisierung via
  `history.pushState`/`popstate` – inkl. Browser-Zurück und Deep-Links.
- **Responsives Layout** statt harter Desktop/Mobil-Trennung: unterhalb von **760 px**
  schaltet die Hülle automatisch auf das kompakte Layout (Top-Bar + Bottom-Tab-Bar + FAB).
- **Inline-Seiten („Page-Sheets“)**: Termin-/Mitglied-Detail, Formulare, Rollen und
  Team-Einstellungen öffnen inline im Content-Bereich; übrige Sheets erscheinen als Modal
  (Desktop) bzw. Bottom-Sheet (mobil).
- **Rechte-Modell (RBAC)**: Module `events · members · finances · news · polls · settings`,
  Level `none < read < write`; Mehrfach-Rollen werden gemerged (höchstes Level gewinnt).
  Geprüft via `can(modul, 'read'|'write')` – sowohl auf Routen- als auch UI-Ebene.
- **Theming & Dark-Mode**: 5 Material-3-Presets über das MUI-Theme; neutrale Flächen über
  CSS-Custom-Properties (`data-color-scheme="dark"`), umschaltbar im Profil-Sheet.
- **i18n**: Deutsch (Default) und Englisch vollständig; Sprache im Profil-Sheet umschaltbar
  und in `localStorage` persistiert (`<html lang>` wird synchronisiert).
- **Fehler-Handling**: getypte Fehlerklassen + `reportActionError`, App-Level- und
  Per-Route-ErrorBoundary, globale `unhandledrejection`/`error`-Handler → Sentry.
- **Sicherheit**: CSP + Security-Header (Dev-Server & `index.html`), Idle-Session-Timeout,
  PII-Scrubbing vor dem Senden an Sentry.

## Qualität & CI

`.github/workflows/ci.yml` führt bei jedem PR aus: Lint → Typecheck → Test (Coverage) →
Build (inkl. Bundle-Size-Budget + SBOM) → Playwright-E2E → Lighthouse CI. Zusätzlich läuft
`npm audit` (High/Critical in Prod-Deps blockierend). Dependabot aktualisiert Dependencies
und GitHub-Actions wöchentlich. Siehe [`CONTRIBUTING.md`](./CONTRIBUTING.md).

## Vom Mock zum echten Backend

`serviceLayer.ts` definiert den vollständigen **API-Vertrag** (gleiche Namespaces &
Signaturen wie das Go/PostgreSQL-Backend in `backend/`). Die Produktivanbindung
(`serviceLayerReal.ts`, ein generierter TypeScript-Client aus dem OpenAPI-Spec) ist
bereits implementiert und wird automatisch verwendet, sobald `VITE_API_BASE_URL`
gesetzt ist – ohne weitere Änderungen am restlichen Frontend-Code, da die exportierte
`api`-Form unverändert bleibt.

## Funktionsumfang (gemäß Lastenheft)

Termine & Anwesenheit (Liste/Kalender/Abwesenheiten, Serien, Nominierung, Kommentare,
iCal-/ICS-Export) · Mitglieder & Mehrfach-Rollen · Rollen & Rechte · Home-Dashboard mit
verlinkten Kennzahlen · Anwesenheitsstatistik mit Zeitraumfilter · Finanzen (Umsätze,
Strafen + Strafenkatalog, monatliche Beiträge) · Neuigkeiten · Umfragen ·
Benachrichtigungs-Center · Team-Wechsel, Einladungslinks & Team-Einstellungen · OIDC-Login.

### Noch offen / nächste Schritte

- Echte OIDC-Anbindung (Authorization Code Flow + PKCE) statt Mock-Login.
- Web-Push-Benachrichtigungen, serverseitiger Kalender-Abo-Feed.
- Vollständige Typisierung der Formularzustände (`AppState.form`) je Sheet.
