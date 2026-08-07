← [Zurück zur Übersicht](./README.md)

# Finanzen

Der Bereich *Finanzen* zeigt oben den aktuellen Kassenstand sowie die
Summe der Einnahmen und Ausgaben, darunter drei Tabs:

## Umsätze

Einzelne Buchungen (Einnahme oder Ausgabe) mit Bezeichnung, Betrag, Datum
und Kategorie. Bestehende Kategorien lassen sich auswählen, neue einfach
eintippen — sie werden automatisch übernommen; über Kategorie-Chips lässt
sich die Liste auf eine einzelne Kategorie filtern. Beim Erfassen einer
Einnahme kann sie direkt mit einem offenen Beitrag oder einer offenen
Strafe verknüpft werden — je ein Button unter „Verknüpfen mit" öffnet ein
Auswahl-Popup. Buchungen erfassen, bearbeiten oder löschen erfordert
„Schreiben" im Modul *Finanzen*; ohne Schreibrecht ist die Übersicht nur
lesbar.

## Strafen

Zwei Schritte:

1. **Strafenkatalog** — die Liste möglicher Strafen (Bezeichnung + Betrag),
   z. B. „Zu spät zum Training". Wird zentral gepflegt (mit „Schreiben").
2. **Strafe erfassen** — eine Strafe aus dem Katalog einer Person
   zuweisen. Erfasste Strafen sind entweder **offen** oder **bezahlt**; die
   Summe der offenen Strafen wird oben angezeigt.

Das Entfernen einer Strafe aus dem Katalog löscht nicht die bereits
erfassten (zugewiesenen) Strafen — die bleiben erhalten.

## Beiträge

Mitgliedsbeiträge (z. B. der Monatsbeitrag) je Mitglied, ebenfalls mit dem
Status **offen** oder **bezahlt**. Standardmäßig als Matrix (Mitglied x
Beitragsperiode) dargestellt; eine Listenansicht mit Zusammenfassung
(wie viele Beiträge bzw. welcher Betrag insgesamt bereits bezahlt wurde)
ist über den Ansicht-Umschalter erreichbar.

Ein Klick auf eine Zelle der Matrix oder eine Zeile der Liste öffnet immer
die reine Detailansicht des Beitrags für dieses Mitglied: Name,
bezahlter/geforderter Betrag und die bereits verknüpften Buchungen. Über
den Button „Beitrag erfassen" darin lässt sich eine neue Zahlung dafür
buchen. Bezeichnung, Betrag, Beschreibung und Fälligkeit lassen sich hier
nicht ändern — das geschieht ausschließlich über die Beitragsperiode als
Ganzes: Der Button „Bearbeiten" bei der Beitragsperiode (neben
„Fälligkeitsperiode archivieren") ändert diese Angaben für alle
Mitglieder der Periode auf einmal, damit die Beiträge einer Periode nicht
unbemerkt auseinanderlaufen können.

## Wer sieht was?

Wie bei allen Modulen gilt: mit „Lesen" im Modul *Finanzen* könnt ihr alle
drei Tabs einsehen, mit „Schreiben" zusätzlich bearbeiten. Ohne Zugriff auf
das Modul erscheint *Finanzen* in der Navigation gar nicht. Siehe
[Rollen & Rechte](./rollen-und-rechte.md) für Details.
