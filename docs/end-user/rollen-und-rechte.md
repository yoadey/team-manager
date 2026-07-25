← [Zurück zur Übersicht](./README.md)

# Rollen & Rechte

Jedes Mitglied hat eine oder mehrere **Rollen**. Jede Rolle legt fest, was
ihre Träger:innen in den einzelnen Bereichen der App dürfen. Das nennt sich
Rechte-Modell (RBAC).

## Die Bereiche (Module)

| Modul | Was er umfasst |
|---|---|
| Termine | Kalender, Zu-/Absagen, Kommentare |
| Mitglieder | Mitgliederliste, Profile |
| Finanzen | Umsätze, Strafen, Beiträge |
| News | Neuigkeiten |
| Umfragen | Umfragen & Abstimmungen |
| Einstellungen | Team-Einstellungen, Rollenverwaltung, Einladungen |

## Die drei Berechtigungsstufen

| Stufe | Bedeutung |
|---|---|
| **—** (kein Zugriff) | Der Bereich ist für diese Rolle weder sichtbar noch bearbeitbar. |
| **Lesen** | Der Bereich kann angesehen, aber nicht verändert werden. |
| **Schreiben** | Der Bereich kann angesehen **und** bearbeitet werden (z. B. Termine anlegen, Buchungen erfassen). |

„Lesen" schließt „—" ein, „Schreiben" schließt „Lesen" ein — jede höhere
Stufe darf alles, was die niedrigeren auch dürfen.

## Mehrere Rollen gleichzeitig

Ein Mitglied kann mehr als eine Rolle gleichzeitig haben. In dem Fall gilt
pro Modul immer die **höchste** Berechtigungsstufe aus allen zugewiesenen
Rollen — eine Rolle mit „Lesen" bei Finanzen und eine zweite Rolle mit
„Schreiben" bei Finanzen ergeben zusammen „Schreiben".

## Standard- und eigene Rollen

Bei der Team-Erstellung werden automatisch **Standard-Rollen** angelegt.
Admin-berechtigte Mitglieder können zusätzlich **eigene Rollen** mit einem
frei wählbaren Namen und individuellen Rechten je Modul anlegen — unter
*Team → Rollen & Rechte*. Jedes Team braucht dabei mindestens eine Person
mit „Schreiben" im Modul Einstellungen; sonst könnte niemand mehr Rollen
oder Mitglieder verwalten.

## Ausnahme: eigene Rückmeldungen

Für ein paar Aktionen reicht die reine Team-Mitgliedschaft aus, unabhängig
von der zugewiesenen Rolle — z. B. die eigene Zu-/Absage zu einem Termin
oder die Teilnahme an einer Umfrage. Ihr müsst dafür kein „Schreiben" im
jeweiligen Modul haben; das betrifft ausschließlich eure eigenen Einträge,
nicht die anderer Mitglieder.

## Wo sehe ich meine eigenen Rechte?

Unter *Mein Profil* seht ihr eure zugewiesenen Rollen. Bereiche, für die
ihr keinerlei Zugriff habt, tauchen in der Navigation gar nicht erst auf.
