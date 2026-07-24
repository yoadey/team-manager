// =============================================================================
// Static legal-notice / privacy-policy content.
//
// This is deliberately NOT part of the i18n message catalogs (i18n/de.ts,
// i18n/en.ts) -- those only support flat, short strings via t(), while this is
// multi-paragraph prose with operator-specific placeholders. Structured data
// (not raw HTML/Markdown) so LegalSheet can render it with plain semantic
// elements -- no dangerouslySetInnerHTML, no Markdown-parsing dependency.
//
// Every `[BETREIBER: ...]` / `[OPERATOR: ...]` marker MUST be filled in by
// whoever deploys this app before it goes live -- see docs/operations.md
// ("Legal setup before going public") for exactly what each one requires and
// why. Shipping with unfilled markers is intentional: an obviously incomplete
// page is far better than one that looks complete but is legally empty.
// =============================================================================

import type { Locale } from '@/i18n';

export type LegalPageId = 'impressum' | 'datenschutz';

export interface LegalSection {
  heading: string;
  paragraphs: string[];
  list?: string[];
}

export interface LegalPageText {
  title: string;
  sections: LegalSection[];
}

const impressumDe: LegalPageText = {
  title: 'Impressum',
  sections: [
    {
      heading: 'Angaben gemäß § 5 DDG',
      paragraphs: [
        '[BETREIBER: Name des Vereins/der Organisation]',
        '[BETREIBER: Rechtsform, z. B. eingetragener Verein (e. V.)]',
        '[BETREIBER: Straße und Hausnummer]',
        '[BETREIBER: Postleitzahl und Ort]',
      ],
    },
    {
      heading: 'Vertreten durch',
      paragraphs: ['[BETREIBER: Namen der vertretungsberechtigten Vorstandsmitglieder]'],
    },
    {
      heading: 'Kontakt',
      paragraphs: ['Telefon: [BETREIBER: Telefonnummer]', 'E-Mail: [BETREIBER: Kontakt-E-Mail-Adresse]'],
    },
    {
      heading: 'Registereintrag',
      paragraphs: [
        '[BETREIBER: Falls im Vereinsregister eingetragen: Registergericht und Registernummer angeben. Falls nicht eingetragen, diesen Abschnitt entfernen.]',
      ],
    },
    {
      heading: 'Umsatzsteuer-Identifikationsnummer',
      paragraphs: ['[BETREIBER: USt-IdNr. gemäß § 27a UStG eintragen, falls vorhanden, sonst diesen Abschnitt entfernen.]'],
    },
    {
      heading: 'Verantwortlich für den Inhalt nach § 18 Abs. 2 MStV',
      paragraphs: [
        '[BETREIBER: Nur auszufüllen, falls redaktionelle Inhalte mit journalistisch-redaktionellem Charakter angeboten werden — für vereinsinterne Neuigkeiten in der Regel nicht einschlägig; im Zweifel rechtlich prüfen lassen.]',
      ],
    },
    {
      heading: 'Haftung für Inhalte',
      paragraphs: [
        'Als Diensteanbieter sind wir gemäß § 7 Abs. 1 DDG für eigene Inhalte auf diesen Seiten nach den allgemeinen Gesetzen verantwortlich. Nach §§ 8 bis 10 DDG sind wir als Diensteanbieter jedoch nicht verpflichtet, übermittelte oder gespeicherte fremde Informationen zu überwachen oder nach Umständen zu forschen, die auf eine rechtswidrige Tätigkeit hinweisen.',
      ],
    },
    {
      heading: 'Streitschlichtung',
      paragraphs: [
        '[BETREIBER: Prüfen, ob dieser Hinweis für euer Angebot einschlägig ist — er ist primär für Anbieter mit Verbraucherverträgen relevant.] Die Europäische Kommission stellt eine Plattform zur Online-Streitbeilegung (OS) bereit: https://ec.europa.eu/consumers/odr/. Wir sind nicht verpflichtet und nicht bereit, an Streitbeilegungsverfahren vor einer Verbraucherschlichtungsstelle teilzunehmen, sofern nicht anders angegeben.',
      ],
    },
  ],
};

const datenschutzDe: LegalPageText = {
  title: 'Datenschutzerklärung',
  sections: [
    {
      heading: 'Verantwortlicher',
      paragraphs: [
        'Verantwortlich für die Datenverarbeitung im Sinne der Datenschutz-Grundverordnung (DSGVO) ist: [BETREIBER: Name, Anschrift und Kontakt wie im Impressum].',
        'Bei Fragen zum Datenschutz wende dich an: [BETREIBER: Datenschutz-Kontakt-E-Mail-Adresse eintragen; falls eine/ein Datenschutzbeauftragte/r bestellt ist, ebenfalls hier angeben].',
      ],
    },
    {
      heading: 'Welche Daten wir verarbeiten',
      paragraphs: [
        'Profildaten: Name, E-Mail-Adresse, Telefonnummer, Geburtsdatum, Adresse und Profilfoto (jeweils soweit angegeben).',
        'Team- und Vereinsdaten: Anwesenheiten und Kommentare zu Terminen, gemeldete Abwesenheiten, verfasste Neuigkeiten, Umfrage-Teilnahmen sowie finanzbezogene Einträge (Beiträge, Strafen, Buchungen) im Rahmen der Vereinsverwaltung.',
        'Technische Daten: IP-Adressen und Zeitstempel in Server-Protokollen (u. a. zur Ratenbegrenzung gegen Missbrauch) sowie ein technisch notwendiges Sitzungs-Cookie zur Anmeldung.',
      ],
    },
    {
      heading: 'Zwecke und Rechtsgrundlagen',
      paragraphs: [
        'Die Verarbeitung erfolgt zur Durchführung der Mitgliedschafts- und Vereinsverwaltung, insbesondere der Termin-, Anwesenheits-, Finanz- und Kommunikationsfunktionen (Art. 6 Abs. 1 lit. b DSGVO — Vertragserfüllung bzw. vorvertragliche Maßnahmen bei der Registrierung).',
        'Sicherheitsbezogene Verarbeitungen (z. B. Anmeldeprotokolle, Missbrauchsschutz durch Ratenbegrenzung) erfolgen auf Grundlage berechtigter Interessen (Art. 6 Abs. 1 lit. f DSGVO) an einem sicheren Betrieb der Anwendung.',
      ],
    },
    {
      heading: 'Empfänger und Auftragsverarbeiter',
      paragraphs: [
        'Deine Daten werden nicht verkauft oder zu Werbezwecken an Dritte weitergegeben. Folgende externe Dienstleister können im Auftrag des Betreibers eingebunden sein, jeweils nur soweit vom Betreiber aktiviert:',
      ],
      list: [
        'Hosting und Objektspeicher für Foto-/Logo-Uploads: [BETREIBER: Anbieter und Serverstandort eintragen, falls S3-Objektspeicher aktiviert ist]',
        'Versand der Registrierungs-Bestätigungsmail: [BETREIBER: SMTP-Anbieter eintragen, falls E-Mail-Versand aktiviert ist]',
        'Technische Fehlerüberwachung (Sentry): [BETREIBER: eintragen, falls aktiviert — siehe Abschnitt „Cookies und lokale Speicherung“]',
        'Technisches Tracing/Monitoring (OpenTelemetry): [BETREIBER: Anbieter/Zielsystem eintragen, falls aktiviert]',
      ],
    },
    {
      heading: 'Cookies und lokale Speicherung',
      paragraphs: [
        'Ein technisch notwendiges Sitzungs-Cookie hält deine Anmeldung aufrecht; es ist für die Nutzung der App erforderlich und bedarf keiner gesonderten Einwilligung.',
        'Die optionale technische Fehlerüberwachung (Sentry) ist in der von diesem Betreiber verwendeten Konfiguration so eingerichtet, dass sie keine Cookies oder nicht-notwendigen lokalen Speicher verwendet; es ist daher keine gesonderte Einwilligung nach § 25 TDDDG erforderlich. [BETREIBER: Diese Aussage gilt nur, solange keine Sentry-Funktionen wie Session Replay ergänzt werden — im Zweifel erneut prüfen.]',
      ],
    },
    {
      heading: 'Speicherdauer',
      paragraphs: [
        'Benachrichtigungen werden nach 90 Tagen, abgelaufene Sitzungen nach 30 Tagen und Protokolldaten (Audit-Log) nach 365 Tagen automatisch gelöscht (Standardwerte, vom Betreiber konfigurierbar). Nicht verifizierte Selbstregistrierungen werden nach 7 Tagen automatisch entfernt.',
        'Finanzbezogene Daten können aus handels- und steuerrechtlichen Gründen länger aufbewahrt werden.',
      ],
    },
    {
      heading: 'Deine Rechte',
      paragraphs: [
        'Du hast das Recht auf Auskunft (Art. 15 DSGVO — direkt in der App unter „Meine Daten exportieren“), Berichtigung (Art. 16), Löschung (Art. 17 — direkt in der App unter „Konto löschen“), Einschränkung der Verarbeitung (Art. 18), Datenübertragbarkeit (Art. 20) sowie Widerspruch gegen die Verarbeitung (Art. 21 DSGVO).',
        'Du hast außerdem das Recht, dich bei einer Datenschutz-Aufsichtsbehörde zu beschweren, insbesondere in dem Mitgliedstaat deines Aufenthaltsorts, deines Arbeitsplatzes oder des mutmaßlichen Verstoßes.',
      ],
    },
    {
      heading: 'Minderjährige Mitglieder',
      paragraphs: [
        'Die Selbstregistrierung setzt voraus, dass die registrierende Person mindestens 16 Jahre alt ist. Jüngere Vereinsmitglieder werden von einem Team-Administrator per Einladungslink angelegt, nicht per Selbstregistrierung.',
      ],
    },
    {
      heading: 'Änderungen dieser Erklärung',
      paragraphs: [
        'Diese Datenschutzerklärung kann angepasst werden, um sie an eine geänderte Rechtslage oder einen geänderten Funktionsumfang anzupassen. Stand: [BETREIBER: Datum eintragen].',
      ],
    },
  ],
};

const impressumEn: LegalPageText = {
  title: 'Legal notice',
  sections: [
    {
      heading: 'Information pursuant to § 5 DDG (Germany)',
      paragraphs: [
        '[OPERATOR: Name of the club/organization]',
        '[OPERATOR: Legal form, e.g. registered association]',
        '[OPERATOR: Street and house number]',
        '[OPERATOR: Postal code and city]',
      ],
    },
    {
      heading: 'Represented by',
      paragraphs: ['[OPERATOR: Names of the authorized board members]'],
    },
    {
      heading: 'Contact',
      paragraphs: ['Phone: [OPERATOR: phone number]', 'Email: [OPERATOR: contact email address]'],
    },
    {
      heading: 'Register entry',
      paragraphs: [
        '[OPERATOR: If registered in the association register, add the register court and registration number. Remove this section if not registered.]',
      ],
    },
    {
      heading: 'VAT identification number',
      paragraphs: ['[OPERATOR: Add the VAT ID if one exists, otherwise remove this section.]'],
    },
    {
      heading: 'Responsible for content pursuant to § 18(2) MStV',
      paragraphs: [
        '[OPERATOR: Only fill in if editorial content of a journalistic-editorial nature is offered — usually not applicable for internal club news; seek legal advice if unsure.]',
      ],
    },
    {
      heading: 'Liability for content',
      paragraphs: [
        'As a service provider, we are responsible for our own content on these pages under general law pursuant to § 7(1) DDG. Under §§ 8 to 10 DDG, however, we are not obligated to monitor transmitted or stored third-party information or to investigate circumstances that indicate unlawful activity.',
      ],
    },
    {
      heading: 'Dispute resolution',
      paragraphs: [
        '[OPERATOR: Check whether this notice applies to your offering — it is primarily relevant for providers with consumer contracts.] The European Commission provides a platform for online dispute resolution (ODR): https://ec.europa.eu/consumers/odr/. We are not obligated and not willing to participate in dispute resolution proceedings before a consumer arbitration board unless stated otherwise.',
      ],
    },
  ],
};

const datenschutzEn: LegalPageText = {
  title: 'Privacy policy',
  sections: [
    {
      heading: 'Controller',
      paragraphs: [
        'The controller responsible for data processing under the General Data Protection Regulation (GDPR) is: [OPERATOR: name, address and contact, same as the legal notice].',
        'For questions about data protection, contact: [OPERATOR: data-protection contact email address; if a data protection officer has been appointed, list them here too].',
      ],
    },
    {
      heading: 'What data we process',
      paragraphs: [
        'Profile data: name, email address, phone number, birthday, address and profile photo (where provided).',
        'Team and club data: attendance and comments on events, reported absences, authored news items, poll participation, and finance-related entries (contributions, penalties, transactions) as part of club administration.',
        'Technical data: IP addresses and timestamps in server logs (among other things for rate limiting against abuse), and a technically necessary session cookie for sign-in.',
      ],
    },
    {
      heading: 'Purposes and legal basis',
      paragraphs: [
        'Processing takes place to carry out membership and club administration, in particular the event, attendance, finance and communication features (Art. 6(1)(b) GDPR — performance of a contract, or pre-contractual steps taken at registration).',
        'Security-related processing (e.g. login logs, abuse protection via rate limiting) is based on legitimate interests (Art. 6(1)(f) GDPR) in operating the application securely.',
      ],
    },
    {
      heading: 'Recipients and processors',
      paragraphs: [
        'Your data is not sold or passed on to third parties for advertising purposes. The following external providers may be involved on the operator\'s behalf, only to the extent enabled by the operator:',
      ],
      list: [
        'Hosting and object storage for photo/logo uploads: [OPERATOR: add the provider and server location if S3 object storage is enabled]',
        'Sending the registration confirmation email: [OPERATOR: add the SMTP provider if email sending is enabled]',
        'Technical error monitoring (Sentry): [OPERATOR: fill in if enabled — see "Cookies and local storage" below]',
        'Technical tracing/monitoring (OpenTelemetry): [OPERATOR: add the provider/destination if enabled]',
      ],
    },
    {
      heading: 'Cookies and local storage',
      paragraphs: [
        'A technically necessary session cookie keeps you signed in; it is required to use the app and does not require separate consent.',
        "The optional error-monitoring integration (Sentry), in the configuration this operator uses, is set up so that it writes no cookies or non-essential local storage; separate consent under § 25 TDDDG is therefore not required. [OPERATOR: this holds only as long as no additional Sentry features such as Session Replay are added — re-verify if that changes.]",
      ],
    },
    {
      heading: 'Retention periods',
      paragraphs: [
        'Notifications are deleted after 90 days, expired sessions after 30 days, and audit-log entries after 365 days (default values, configurable by the operator). Unverified self-registered accounts are automatically removed after 7 days.',
        'Finance-related data may be retained longer for commercial and tax-law reasons.',
      ],
    },
    {
      heading: 'Your rights',
      paragraphs: [
        'You have the right to access (Art. 15 GDPR — available in the app under "Export my data"), rectification (Art. 16), erasure (Art. 17 — available in the app under "Delete account"), restriction of processing (Art. 18), data portability (Art. 20), and objection to processing (Art. 21 GDPR).',
        'You also have the right to lodge a complaint with a data protection supervisory authority, in particular in the member state of your residence, place of work, or the place of the alleged infringement.',
      ],
    },
    {
      heading: 'Members under 18',
      paragraphs: [
        'Self-registration requires the registering person to be at least 16 years old. Younger club members are added by a team administrator via an invite link, not through self-registration.',
      ],
    },
    {
      heading: 'Changes to this policy',
      paragraphs: [
        'This privacy policy may be updated to reflect changes in the law or in the app\'s functionality. Last updated: [OPERATOR: add date].',
      ],
    },
  ],
};

export const LEGAL_CONTENT: Record<Locale, Record<LegalPageId, LegalPageText>> = {
  de: { impressum: impressumDe, datenschutz: datenschutzDe },
  en: { impressum: impressumEn, datenschutz: datenschutzEn },
};
