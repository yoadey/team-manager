# team-manager

Verwaltung von Trainingsterminen und Spielern für einen Verein.

Produktive React-Umsetzung des Design-Prototyps „Teamverwaltung" (mandantenfähige
Team-/Vereins-App für den Formations-/Sportbetrieb). Das Design des hochauflösenden
Prototyps wurde **1:1 übernommen**, aber vollständig auf **React + TypeScript + MUI**
aufgebaut. In dieser ersten Stufe läuft die App **rein im Frontend** – das Backend
wird über den `serviceLayer` gemockt (In-Memory + `localStorage`, künstliche Latenz).

## Tech-Stack

- **React 18 + TypeScript + Vite**
- **MUI (Material UI v6)** als Komponenten- und Theming-Basis (Material Design 3)
- **Roboto** (`@fontsource/roboto`) + **Material Symbols Outlined** (`material-symbols`)
- Zentraler App-State über React Context (`src/store/AppContext.tsx`)
- Mock-Backend / API-Vertrag: `src/services/serviceLayer.ts`

## Loslegen

```bash
npm install
npm run dev        # Dev-Server (http://localhost:5173)
npm run build      # Typecheck + Produktions-Build
npm run preview    # Build lokal ansehen
npm run typecheck  # nur TypeScript prüfen
```

> Beim Login genügt ein Klick auf einen beliebigen Identity-Provider – der
> Mock-Service meldet automatisch den Demo-Nutzer „Lena Bergmann" an.

## Projektstruktur

```
src/
├── main.tsx                # Entry, Fonts & Icons
├── App.tsx                 # ThemeProvider (MUI) + Provider-Wrapper
├── index.css
├── services/
│   ├── serviceLayer.ts     # Mock-Backend = API-Vertrag (auth, teams, members,
│   │                       #   roles, events, attendance, absences, news,
│   │                       #   finances, stats, polls, notifications)
│   └── types.ts            # Domänen-Typen (gemeinsamer Vertrag)
├── theme/
│   ├── tokens.ts           # Design-Tokens, Farb-/Status-/Typ-Metadaten, Formatter
│   └── theme.ts            # MUI-Theme aus den 5 M3-Presets
├── store/
│   └── AppContext.tsx      # Zentraler Zustand + alle Aktionen
├── components/
│   ├── Root.tsx            # Phase-Switch (loading / login / app)
│   ├── Login.tsx           # OIDC-Provider-Login
│   ├── Shell.tsx           # Responsive Hülle (Sidebar / Bottom-Nav, Header)
│   ├── SheetHost.tsx       # Modale Sheets (Bottom-Sheet mobil, Modal Desktop)
│   ├── Toast.tsx
│   ├── cards.tsx           # EventCard, NewsCard
│   └── ui.tsx              # Atome (Sym, Av, Chip, Field, Buttons, …)
├── screens/                # home, events, members, finances, stats, news, polls, team
└── sheets/                 # Detail-/Formular-/Dialog-Sheets (Termin, Mitglied,
                            #   Rollen, Team-Einstellungen, Finanzen, Umfrage, …)
```

## Architektur & Konzepte

- **Responsives Layout** statt harter Desktop/Mobil-Trennung: unterhalb von **760 px**
  schaltet die Hülle automatisch auf das kompakte Layout (Top-Bar + Bottom-Tab-Bar + FAB).
  Der simulierte Smartphone-Rahmen des Prototyps entfällt bewusst.
- **Inline-Seiten („Page-Sheets")**: Termin-/Mitglied-Detail, Formulare, Rollen und
  Team-Einstellungen öffnen inline im Content-Bereich (Navigation & Header bleiben sichtbar,
  Zurück-Pfeil im Header) – kein Overlay. Restliche Sheets erscheinen als Modal (Desktop)
  bzw. Bottom-Sheet (mobil).
- **Rechte-Modell**: Module `events · members · finances · news · polls · settings`,
  Level `none < read < write`; Mehrfach-Rollen werden gemerged (höchstes Level gewinnt).
  UI-Elemente erscheinen abhängig von `can(modul, 'read'|'write')`.
- **Anwesenheit**: Status `yes | maybe | no | pending | not_nominated`, effektiver Status
  berücksichtigt geplante Abwesenheiten und Antwortmodus (`opt_in` / `opt_out`).
- **Serientermine**: gemeinsame `seriesId`; Bearbeiten/Absagen/Löschen wahlweise einzeln
  oder als ganze Serie.
- **Theming**: 5 Material-3-Presets (Blau Default, Violett, Türkis, Rot, Grün) über das
  MUI-Theme. Primärfarbe ist im App-State (`primaryColor`) hinterlegt.

## Vom Mock zum echten Backend

`serviceLayer.ts` definiert den vollständigen **API-Vertrag** (gleiche Namespaces &
Signaturen wie das spätere Go/PostgreSQL-Backend). Für die Produktivanbindung werden
lediglich die Methodenrümpfe gegen HTTP-Calls getauscht – Signaturen und Typen
(`src/services/types.ts`) bleiben unverändert.

## Funktionsumfang (gemäß Lastenheft)

Termine & Anwesenheit (Liste/Kalender/Abwesenheiten, Serien, Nominierung, Kommentare,
iCal-/ICS-Export) · Mitglieder & Mehrfach-Rollen · Rollen & Rechte · Home-Dashboard mit
verlinkten Kennzahlen · Anwesenheitsstatistik mit Zeitraumfilter · Finanzen (Umsätze,
Strafen + Strafenkatalog, monatliche Beiträge) · Neuigkeiten · Umfragen ·
Benachrichtigungs-Center · Team-Wechsel, Einladungslinks & Team-Einstellungen · OIDC-Login.

### Noch offen / nächste Schritte
- Echtes Backend (Go/PostgreSQL) anbinden – `serviceLayer` als Vertrag nutzen.
- Echte OIDC-Anbindung (Authorization Code Flow + PKCE).
- PWA / Web-Push (Service Worker), serverseitiger Kalender-Abo-Feed.
- i18n-Strings auslagern (initial Deutsch).
