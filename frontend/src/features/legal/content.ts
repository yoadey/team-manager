// =============================================================================
// Legal-notice / privacy-policy content.
//
// This is deliberately NOT part of the i18n message catalogs (i18n/de.ts,
// i18n/en.ts) -- those only support flat, short strings via t(), while this is
// multi-paragraph prose. Structured data (not raw HTML/Markdown) so LegalSheet
// can render it with plain semantic elements -- no dangerouslySetInnerHTML, no
// Markdown-parsing dependency.
//
// The generic legal boilerplate below (liability, dispute resolution, GDPR
// purposes/rights/retention) is the same for every deployment and stays
// hardcoded here. The operator-specific fields (name, address, contact,
// register entry, VAT ID, which processors are in use) are NOT hardcoded --
// they're read from `config.operator` (frontend/src/config.ts), which is
// populated at container start from OPERATOR_* environment variables (see
// frontend/docker/docker-entrypoint-runtime-config.sh and
// docs/operations.md's "Legal setup before going public"), the same
// build-once/configure-at-deploy pattern already used for API_BASE_URL/
// SENTRY_DSN/VAPID_PUBLIC_KEY. This lets one built image serve any operator
// without a rebuild.
//
// Two different behaviors for an unset operator field, by design (see
// openspec/changes/operator-data-runtime-config/design.md Decision 2):
//   - Always-required fields (name, street, postal code + city, phone,
//     email): unset renders an explicit `[BETREIBER: ...]`/`[OPERATOR: ...]`
//     placeholder marker, so an unconfigured deployment is obviously
//     incomplete rather than silently non-compliant.
//   - Optional fields (legal form, represented-by, register entry, VAT ID,
//     and each processor-disclosure line): unset omits the corresponding
//     line/section/list item entirely, since these are only applicable to
//     some operators/deployments in the first place.
// =============================================================================

import type { Locale } from '@/i18n';
import { config } from '@/config';
import type { OperatorConfigField } from '@/config';

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

type Operator = Record<OperatorConfigField, string | undefined>;

/** Always-required field: the placeholder marker if unset, else the real value. */
function required(operator: Operator, field: OperatorConfigField, placeholderDe: string, placeholderEn: string, locale: Locale): string {
  return operator[field] ?? (locale === 'de' ? placeholderDe : placeholderEn);
}

function addressLine(operator: Operator, locale: Locale): string {
  if (operator.postalCode && operator.city) return `${operator.postalCode} ${operator.city}`;
  return locale === 'de' ? '[BETREIBER: Postleitzahl und Ort]' : '[OPERATOR: Postal code and city]';
}

function buildImpressum(operator: Operator, locale: Locale): LegalPageText {
  const de = locale === 'de';

  const sections: LegalSection[] = [
    {
      heading: de ? 'Angaben gemäß § 5 DDG' : 'Information pursuant to § 5 DDG (Germany)',
      paragraphs: [
        required(operator, 'name', '[BETREIBER: Name des Vereins/der Organisation]', '[OPERATOR: Name of the club/organization]', locale),
        ...(operator.legalForm ? [operator.legalForm] : []),
        required(operator, 'street', '[BETREIBER: Straße und Hausnummer]', '[OPERATOR: Street and house number]', locale),
        addressLine(operator, locale),
      ],
    },
  ];

  if (operator.representedBy) {
    sections.push({
      heading: de ? 'Vertreten durch' : 'Represented by',
      paragraphs: [operator.representedBy],
    });
  }

  sections.push({
    heading: de ? 'Kontakt' : 'Contact',
    paragraphs: [
      `${de ? 'Telefon' : 'Phone'}: ${required(operator, 'phone', '[BETREIBER: Telefonnummer]', '[OPERATOR: phone number]', locale)}`,
      `${de ? 'E-Mail' : 'Email'}: ${required(operator, 'email', '[BETREIBER: Kontakt-E-Mail-Adresse]', '[OPERATOR: contact email address]', locale)}`,
    ],
  });

  if (operator.registerCourt && operator.registerNumber) {
    sections.push({
      heading: de ? 'Registereintrag' : 'Register entry',
      paragraphs: [de ? `${operator.registerCourt}, ${operator.registerNumber}` : `${operator.registerCourt}, ${operator.registerNumber}`],
    });
  }

  if (operator.vatId) {
    sections.push({
      heading: de ? 'Umsatzsteuer-Identifikationsnummer' : 'VAT identification number',
      paragraphs: [operator.vatId],
    });
  }

  sections.push(
    {
      heading: de ? 'Haftung für Inhalte' : 'Liability for content',
      paragraphs: [
        de
          ? 'Als Diensteanbieter sind wir gemäß § 7 Abs. 1 DDG für eigene Inhalte auf diesen Seiten nach den allgemeinen Gesetzen verantwortlich. Nach §§ 8 bis 10 DDG sind wir als Diensteanbieter jedoch nicht verpflichtet, übermittelte oder gespeicherte fremde Informationen zu überwachen oder nach Umständen zu forschen, die auf eine rechtswidrige Tätigkeit hinweisen.'
          : 'As a service provider, we are responsible for our own content on these pages under general law pursuant to § 7(1) DDG. Under §§ 8 to 10 DDG, however, we are not obligated to monitor transmitted or stored third-party information or to investigate circumstances that indicate unlawful activity.',
      ],
    },
    {
      heading: de ? 'Streitschlichtung' : 'Dispute resolution',
      paragraphs: [
        de
          ? 'Die Europäische Kommission stellt eine Plattform zur Online-Streitbeilegung (OS) bereit: https://ec.europa.eu/consumers/odr/. Wir sind nicht verpflichtet und nicht bereit, an Streitbeilegungsverfahren vor einer Verbraucherschlichtungsstelle teilzunehmen.'
          : 'The European Commission provides a platform for online dispute resolution (ODR): https://ec.europa.eu/consumers/odr/. We are not obligated and not willing to participate in dispute resolution proceedings before a consumer arbitration board.',
      ],
    },
  );

  return { title: de ? 'Impressum' : 'Legal notice', sections };
}

function controllerIdentity(operator: Operator, locale: Locale): string {
  if (operator.name && operator.street && operator.postalCode && operator.city) {
    return `${operator.name}, ${operator.street}, ${operator.postalCode} ${operator.city}`;
  }
  return locale === 'de'
    ? '[BETREIBER: Name, Anschrift und Kontakt wie im Impressum]'
    : '[OPERATOR: name, address and contact, same as the legal notice]';
}

function dataProtectionContact(operator: Operator, locale: Locale): string {
  return (
    operator.dataProtectionEmail ??
    operator.email ??
    (locale === 'de' ? '[BETREIBER: Datenschutz-Kontakt-E-Mail-Adresse eintragen]' : '[OPERATOR: data-protection contact email address]')
  );
}

function controllerSection(operator: Operator, locale: Locale): LegalSection {
  const de = locale === 'de';
  const identity = controllerIdentity(operator, locale);
  const contact = dataProtectionContact(operator, locale);
  return {
    heading: de ? 'Verantwortlicher' : 'Controller',
    paragraphs: [
      de
        ? `Verantwortlich für die Datenverarbeitung im Sinne der Datenschutz-Grundverordnung (DSGVO) ist: ${identity}.`
        : `The controller responsible for data processing under the General Data Protection Regulation (GDPR) is: ${identity}.`,
      de ? `Bei Fragen zum Datenschutz wende dich an: ${contact}.` : `For questions about data protection, contact: ${contact}.`,
    ],
  };
}

const PROCESSOR_LINES: { field: OperatorConfigField; de: string; en: string }[] = [
  {
    field: 's3Provider',
    de: 'Hosting und Objektspeicher für Foto-/Logo-Uploads',
    en: 'Hosting and object storage for photo/logo uploads',
  },
  { field: 'smtpProvider', de: 'Versand der Registrierungs-Bestätigungsmail', en: 'Sending the registration confirmation email' },
  { field: 'sentryProvider', de: 'Technische Fehlerüberwachung (Sentry)', en: 'Technical error monitoring (Sentry)' },
  { field: 'otelProvider', de: 'Technisches Tracing/Monitoring (OpenTelemetry)', en: 'Technical tracing/monitoring (OpenTelemetry)' },
];

function recipientsSection(operator: Operator, locale: Locale): LegalSection {
  const de = locale === 'de';
  const list = PROCESSOR_LINES.filter((line) => operator[line.field]).map(
    (line) => `${de ? line.de : line.en}: ${operator[line.field]}`,
  );
  return {
    heading: de ? 'Empfänger und Auftragsverarbeiter' : 'Recipients and processors',
    paragraphs: [
      de
        ? 'Deine Daten werden nicht verkauft oder zu Werbezwecken an Dritte weitergegeben. Folgende externe Dienstleister können im Auftrag des Betreibers eingebunden sein, jeweils nur soweit vom Betreiber aktiviert:'
        : "Your data is not sold or passed on to third parties for advertising purposes. The following external providers may be involved on the operator's behalf, only to the extent enabled by the operator:",
    ],
    ...(list.length > 0 ? { list } : {}),
  };
}

function sentryCookieText(operator: Operator, locale: Locale): string {
  const de = locale === 'de';
  if (!operator.sentryProvider) {
    return de
      ? 'Es wird keine Fehlerüberwachung (Sentry) oder vergleichbares Tracking eingesetzt; darüber hinaus werden keine weiteren Cookies oder nicht-notwendigen lokalen Speicher verwendet.'
      : 'No error-monitoring (Sentry) or comparable tracking is used; beyond the session cookie above, no further cookies or non-essential local storage are used.';
  }
  return de
    ? 'Die optionale technische Fehlerüberwachung (Sentry) ist bei diesem Betreiber so eingerichtet, dass sie keine Cookies oder nicht-notwendigen lokalen Speicher verwendet; es ist daher keine gesonderte Einwilligung nach § 25 TDDDG erforderlich. Dies gilt nur, solange keine Sentry-Funktionen wie Session Replay ergänzt werden — im Zweifel erneut prüfen.'
    : "The optional error-monitoring integration (Sentry), in the configuration this operator uses, is set up so that it writes no cookies or non-essential local storage; separate consent under § 25 TDDDG is therefore not required. This holds only as long as no additional Sentry features such as Session Replay are added — re-verify if that changes.";
}

function cookiesSection(operator: Operator, locale: Locale): LegalSection {
  const de = locale === 'de';
  return {
    heading: de ? 'Cookies und lokale Speicherung' : 'Cookies and local storage',
    paragraphs: [
      de
        ? 'Ein technisch notwendiges Sitzungs-Cookie hält deine Anmeldung aufrecht; es ist für die Nutzung der App erforderlich und bedarf keiner gesonderten Einwilligung.'
        : 'A technically necessary session cookie keeps you signed in; it is required to use the app and does not require separate consent.',
      sentryCookieText(operator, locale),
    ],
  };
}

function dataWeProcessSection(locale: Locale): LegalSection {
  const de = locale === 'de';
  return {
    heading: de ? 'Welche Daten wir verarbeiten' : 'What data we process',
    paragraphs: de
      ? [
          'Profildaten: Name, E-Mail-Adresse, Telefonnummer, Geburtsdatum, Adresse und Profilfoto (jeweils soweit angegeben).',
          'Team- und Vereinsdaten: Anwesenheiten und Kommentare zu Terminen, gemeldete Abwesenheiten, verfasste Neuigkeiten, Umfrage-Teilnahmen sowie finanzbezogene Einträge (Beiträge, Strafen, Buchungen) im Rahmen der Vereinsverwaltung.',
          'Technische Daten: IP-Adressen und Zeitstempel in Server-Protokollen (u. a. zur Ratenbegrenzung gegen Missbrauch) sowie ein technisch notwendiges Sitzungs-Cookie zur Anmeldung.',
        ]
      : [
          'Profile data: name, email address, phone number, birthday, address and profile photo (where provided).',
          'Team and club data: attendance and comments on events, reported absences, authored news items, poll participation, and finance-related entries (contributions, penalties, transactions) as part of club administration.',
          'Technical data: IP addresses and timestamps in server logs (among other things for rate limiting against abuse), and a technically necessary session cookie for sign-in.',
        ],
  };
}

function purposesSection(locale: Locale): LegalSection {
  const de = locale === 'de';
  return {
    heading: de ? 'Zwecke und Rechtsgrundlagen' : 'Purposes and legal basis',
    paragraphs: de
      ? [
          'Die Verarbeitung erfolgt zur Durchführung der Mitgliedschafts- und Vereinsverwaltung, insbesondere der Termin-, Anwesenheits-, Finanz- und Kommunikationsfunktionen (Art. 6 Abs. 1 lit. b DSGVO — Vertragserfüllung bzw. vorvertragliche Maßnahmen bei der Registrierung).',
          'Sicherheitsbezogene Verarbeitungen (z. B. Anmeldeprotokolle, Missbrauchsschutz durch Ratenbegrenzung) erfolgen auf Grundlage berechtigter Interessen (Art. 6 Abs. 1 lit. f DSGVO) an einem sicheren Betrieb der Anwendung.',
        ]
      : [
          'Processing takes place to carry out membership and club administration, in particular the event, attendance, finance and communication features (Art. 6(1)(b) GDPR — performance of a contract, or pre-contractual steps taken at registration).',
          'Security-related processing (e.g. login logs, abuse protection via rate limiting) is based on legitimate interests (Art. 6(1)(f) GDPR) in operating the application securely.',
        ],
  };
}

function retentionSection(locale: Locale): LegalSection {
  const de = locale === 'de';
  return {
    heading: de ? 'Speicherdauer' : 'Retention periods',
    paragraphs: de
      ? [
          'Benachrichtigungen werden nach 90 Tagen, abgelaufene Sitzungen nach 30 Tagen und Protokolldaten (Audit-Log) nach 365 Tagen automatisch gelöscht (Standardwerte, vom Betreiber konfigurierbar). Nicht verifizierte Selbstregistrierungen werden nach 7 Tagen automatisch entfernt.',
          'Finanzbezogene Daten können aus handels- und steuerrechtlichen Gründen länger aufbewahrt werden.',
        ]
      : [
          'Notifications are deleted after 90 days, expired sessions after 30 days, and audit-log entries after 365 days (default values, configurable by the operator). Unverified self-registered accounts are automatically removed after 7 days.',
          'Finance-related data may be retained longer for commercial and tax-law reasons.',
        ],
  };
}

function rightsSection(locale: Locale): LegalSection {
  const de = locale === 'de';
  return {
    heading: de ? 'Deine Rechte' : 'Your rights',
    paragraphs: de
      ? [
          'Du hast das Recht auf Auskunft (Art. 15 DSGVO — direkt in der App unter „Meine Daten exportieren“), Berichtigung (Art. 16), Löschung (Art. 17 — direkt in der App unter „Konto löschen“), Einschränkung der Verarbeitung (Art. 18), Datenübertragbarkeit (Art. 20) sowie Widerspruch gegen die Verarbeitung (Art. 21 DSGVO).',
          'Du hast außerdem das Recht, dich bei einer Datenschutz-Aufsichtsbehörde zu beschweren, insbesondere in dem Mitgliedstaat deines Aufenthaltsorts, deines Arbeitsplatzes oder des mutmaßlichen Verstoßes.',
        ]
      : [
          'You have the right to access (Art. 15 GDPR — available in the app under "Export my data"), rectification (Art. 16), erasure (Art. 17 — available in the app under "Delete account"), restriction of processing (Art. 18), data portability (Art. 20), and objection to processing (Art. 21 GDPR).',
          'You also have the right to lodge a complaint with a data protection supervisory authority, in particular in the member state of your residence, place of work, or the place of the alleged infringement.',
        ],
  };
}

function minorsSection(locale: Locale): LegalSection {
  const de = locale === 'de';
  return {
    heading: de ? 'Minderjährige Mitglieder' : 'Members under 18',
    paragraphs: [
      de
        ? 'Die Selbstregistrierung setzt voraus, dass die registrierende Person mindestens 16 Jahre alt ist. Jüngere Vereinsmitglieder werden von einem Team-Administrator per Einladungslink angelegt, nicht per Selbstregistrierung.'
        : 'Self-registration requires the registering person to be at least 16 years old. Younger club members are added by a team administrator via an invite link, not through self-registration.',
    ],
  };
}

// Document-revision date for the boilerplate text in the sections above, not
// operator data -- bump this whenever their wording changes.
function changesSection(locale: Locale): LegalSection {
  const de = locale === 'de';
  return {
    heading: de ? 'Änderungen dieser Erklärung' : 'Changes to this policy',
    paragraphs: [
      de
        ? 'Diese Datenschutzerklärung kann angepasst werden, um sie an eine geänderte Rechtslage oder einen geänderten Funktionsumfang anzupassen. Stand: 25. Juli 2026.'
        : "This privacy policy may be updated to reflect changes in the law or in the app's functionality. Last updated: 25 July 2026.",
    ],
  };
}

function buildDatenschutz(operator: Operator, locale: Locale): LegalPageText {
  return {
    title: locale === 'de' ? 'Datenschutzerklärung' : 'Privacy policy',
    sections: [
      controllerSection(operator, locale),
      dataWeProcessSection(locale),
      purposesSection(locale),
      recipientsSection(operator, locale),
      cookiesSection(operator, locale),
      retentionSection(locale),
      rightsSection(locale),
      minorsSection(locale),
      changesSection(locale),
    ],
  };
}

export function getLegalContent(locale: Locale, page: LegalPageId): LegalPageText {
  const operator = config.operator;
  return page === 'impressum' ? buildImpressum(operator, locale) : buildDatenschutz(operator, locale);
}
