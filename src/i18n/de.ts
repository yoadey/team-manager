// German message catalog. This is the reference catalog: every key that exists
// here should also exist in the other locale files. Strings can be migrated
// here from the components over time; the keys below cover cross-cutting
// concerns (errors, relative time, common actions, domain labels).

export const de = {
  common: {
    retry: 'Erneut versuchen',
    cancel: 'Abbrechen',
    confirm: 'Bestätigen',
    save: 'Speichern',
    delete: 'Löschen',
    loading: 'Lädt…',
  },
  error: {
    generic: 'Es ist ein Fehler aufgetreten',
    action: 'Aktion fehlgeschlagen',
    load: 'Daten konnten nicht geladen werden',
    save: 'Speichern fehlgeschlagen',
    delete: 'Löschen fehlgeschlagen',
    login: 'Anmeldung fehlgeschlagen',
    network: 'Verbindung zum Service fehlgeschlagen',
    unknown: 'Unbekannter Fehler',
  },
  relTime: {
    now: 'gerade eben',
    minutes: 'vor {n} Min',
    hours: 'vor {n} Std',
    day: 'vor 1 Tag',
    days: 'vor {n} Tagen',
  },
  eventType: {
    training: 'Training',
    auftritt: 'Auftritt / Turnier',
    event: 'Team-Event',
  },
  attendance: {
    yes: 'Zugesagt',
    maybe: 'Unsicher',
    no: 'Abgesagt',
    pending: 'Offen',
    not_nominated: 'Nicht nominiert',
  },
};
