← [Zurück zur Übersicht](./README.md)

# Statistik

Der Bereich *Statistik* wertet die Anwesenheit bei vergangenen Terminen
aus. Ihr könnt oben den Zeitraum wählen: „Gesamt", einen festen
Monatsbereich oder einen eigenen Zeitraum (von/bis). Wer den Zeitraum
häufiger braucht, kann ihn über „Neu" unter einem eigenen Namen speichern
(z. B. „Saison 2026/27") und beim nächsten Besuch per Klick wieder
auswählen — das erfordert „Schreiben" im Modul *Statistik*
(siehe [Rollen & Rechte](./rollen-und-rechte.md)); den Bereich ansehen und
bereits gespeicherte Zeiträume auswählen reicht „Lesen".

Der Bereich *Statistik* ist ein eigenes Modul in den Rollen & Rechten —
standardmäßig sehen alle Mitglieder ihn, ein Team kann ihn aber auf
bestimmte Rollen (z. B. Trainer:innen) einschränken.

## Team-Anwesenheit

Die durchschnittliche Zusagequote über alle betrachteten Termine, sowohl
als Gesamtwert als auch **Quote pro Person**. Nicht nominierte Termine
zählen dabei nicht mit — nur Termine, zu denen eine Person tatsächlich
zurückmelden konnte, fließen in ihre Quote ein.

## Matrix

Der Reiter **Matrix** zeigt je Mitglied (Zeile) und Termin (Spalte), ob
zugesagt, abgesagt oder nicht geantwortet wurde. Direkt hinter dem Namen
steht in der Spalte „Ja" die Anzahl der Anwesenheiten im gewählten
Zeitraum. Auf dem Desktop bleiben Name und Anzahl beim seitlichen Scrollen
durch viele Termine stehen; auf dem Handy bleibt nur der Name stehen.
Über „Termintypen" lassen sich Trainings, Wettkämpfe/Auftritte und
Team-Events ein- und ausblenden.

## Termine

Der Reiter **Termine** listet alle Termine des gewählten Zeitraums und
zeigt je Termin, wie viele der erwarteten Mitglieder anwesend waren
(„x/y anwesend"). Darunter steht die Anwesenheit **je Gruppe** — eine
Gruppe ist eine Rolle eures Teams (z. B. „Trainer", „Mitglied",
„Spieler"):

- **Häkchen** — alle erwarteten Mitglieder der Gruppe waren anwesend.
- **Prozentzahl** — der Anteil der Anwesenden in der Gruppe. Liegt er unter
  dem Schwellenwert der Gruppe, wird er **rot** und mit einem Warnsymbol
  markiert.
- **Strich (–)** — aus dieser Gruppe wurde bei dem Termin niemand erwartet
  (z. B. weil die Gruppe nicht nominiert war).

**Wer nicht geantwortet hat, zählt hier als nicht anwesend.** Drei Zusagen
bei zwölf fehlenden Rückmeldungen sind also 3 von 15 (20 %) und kein
Häkchen. Das gilt nur für diese Ansicht: In der **Quote pro Person** zählt
ein Termin ohne Rückmeldung weiterhin nicht mit. Nicht erwartet — und damit
nirgends mitgezählt — werden nicht nominierte Mitglieder (auch wenn der
Termin nur bestimmte Rollen nominiert), Mitglieder mit einer als „nicht
statistikrelevant" markierten Abwesenheit und Mitglieder, die von der
Statistik ausgenommen sind (deren Zu- oder Absage zählt aber mit, wenn sie
eine abgeben). Bei Terminen
im Modus „Aktiv absagen" (alle gelten als zugesagt) zählt eine fehlende
Rückmeldung wie überall als Zusage, bei einer eingetragenen Abwesenheit als
Absage.

Wer mehrere Rollen hat, zählt in jeder seiner Gruppen mit — die Gruppen
ergeben zusammengezählt deshalb nicht die Gesamtzahl des Termins.

### Schwellenwert je Gruppe einstellen

Ab wann eine Gruppe rot markiert wird, legt ihr in den
**Team-Einstellungen** (Bereich *Team* → „Team-Einstellungen") unter „Statistik:
Mindest-Anwesenheit je Gruppe" fest — für jede Rolle einzeln, wahlweise als

- **Prozent** — rot, wenn weniger als z. B. 75 % der Gruppe anwesend waren,
  oder
- **Anzahl** — rot, wenn weniger als z. B. 8 Personen der Gruppe anwesend
  waren (praktisch für „wir brauchen mindestens 8 Spieler").

Ohne eigene Einstellung gilt 50 %. Ist eine Gruppe vollständig anwesend,
erscheint immer das Häkchen — auch wenn die eingestellte Mindestanzahl
größer ist als die Gruppe selbst. Mit dem Wert 0 wird eine Gruppe nie rot
markiert. Zum Ändern braucht ihr „Schreiben" im Modul *Einstellungen*
(siehe [Rollen & Rechte](./rollen-und-rechte.md)).

Diese Kennzahlen sind rein lesend — sie basieren automatisch auf den
Rückmeldungen aus dem Bereich [Termine](./termine.md), es gibt hier nichts
manuell einzutragen.
