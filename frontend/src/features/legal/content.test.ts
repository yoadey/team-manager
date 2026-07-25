import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';

// getLegalContent reads config.operator, which config.ts computes once at
// module load from window.__RUNTIME_CONFIG__ -- same reset pattern as
// config.test.ts.
beforeEach(() => {
  vi.resetModules();
  delete (window as { __RUNTIME_CONFIG__?: unknown }).__RUNTIME_CONFIG__;
});

afterEach(() => {
  delete (window as { __RUNTIME_CONFIG__?: unknown }).__RUNTIME_CONFIG__;
});

describe('getLegalContent(impressum)', () => {
  it('renders placeholder markers for every always-required field when unset', async () => {
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('de', 'impressum');
    const angaben = page.sections.find((s) => s.heading === 'Angaben gemäß § 5 DDG')!;
    expect(angaben.paragraphs).toContain('[BETREIBER: Name des Vereins/der Organisation]');
    expect(angaben.paragraphs).toContain('[BETREIBER: Straße und Hausnummer]');
    expect(angaben.paragraphs).toContain('[BETREIBER: Postleitzahl und Ort]');

    const kontakt = page.sections.find((s) => s.heading === 'Kontakt')!;
    expect(kontakt.paragraphs).toEqual(['Telefon: [BETREIBER: Telefonnummer]', 'E-Mail: [BETREIBER: Kontakt-E-Mail-Adresse]']);
  });

  it('omits the optional sections when the corresponding fields are unset', async () => {
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('de', 'impressum');
    expect(page.sections.find((s) => s.heading === 'Vertreten durch')).toBeUndefined();
    expect(page.sections.find((s) => s.heading === 'Registereintrag')).toBeUndefined();
    expect(page.sections.find((s) => s.heading === 'Umsatzsteuer-Identifikationsnummer')).toBeUndefined();
  });

  it('renders configured values instead of placeholders', async () => {
    window.__RUNTIME_CONFIG__ = {
      OPERATOR_NAME: 'Stefan May',
      OPERATOR_STREET: 'Robensstraße 56',
      OPERATOR_POSTAL_CODE: '52070',
      OPERATOR_CITY: 'Aachen',
      OPERATOR_PHONE: '+49 241 000000',
      OPERATOR_EMAIL: 'info@yoadey.de',
    };
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('de', 'impressum');
    const angaben = page.sections.find((s) => s.heading === 'Angaben gemäß § 5 DDG')!;
    expect(angaben.paragraphs).toEqual(['Stefan May', 'Robensstraße 56', '52070 Aachen']);
    const kontakt = page.sections.find((s) => s.heading === 'Kontakt')!;
    expect(kontakt.paragraphs).toEqual(['Telefon: +49 241 000000', 'E-Mail: info@yoadey.de']);
  });

  it('includes the optional sections once their fields are configured', async () => {
    window.__RUNTIME_CONFIG__ = {
      OPERATOR_LEGAL_FORM: 'Eingetragener Verein (e. V.)',
      OPERATOR_REPRESENTED_BY: 'Max Mustermann',
      OPERATOR_REGISTER_COURT: 'Amtsgericht Aachen',
      OPERATOR_REGISTER_NUMBER: 'VR 1234',
      OPERATOR_VAT_ID: 'DE123456789',
    };
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('de', 'impressum');
    expect(page.sections.find((s) => s.heading === 'Vertreten durch')?.paragraphs).toEqual(['Max Mustermann']);
    expect(page.sections.find((s) => s.heading === 'Registereintrag')?.paragraphs).toEqual(['Amtsgericht Aachen, VR 1234']);
    expect(page.sections.find((s) => s.heading === 'Umsatzsteuer-Identifikationsnummer')?.paragraphs).toEqual(['DE123456789']);
    const angaben = page.sections.find((s) => s.heading === 'Angaben gemäß § 5 DDG')!;
    expect(angaben.paragraphs).toContain('Eingetragener Verein (e. V.)');
  });
});

describe('getLegalContent(datenschutz)', () => {
  it('omits the processor list entirely when no OPERATOR_*_PROVIDER var is set', async () => {
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('de', 'datenschutz');
    const empfaenger = page.sections.find((s) => s.heading === 'Empfänger und Auftragsverarbeiter')!;
    expect(empfaenger.list).toBeUndefined();
  });

  it('renders exactly the configured processor lines', async () => {
    window.__RUNTIME_CONFIG__ = { OPERATOR_S3_PROVIDER: 'selbst gehostet auf eigener Infrastruktur' };
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('de', 'datenschutz');
    const empfaenger = page.sections.find((s) => s.heading === 'Empfänger und Auftragsverarbeiter')!;
    expect(empfaenger.list).toEqual(['Hosting und Objektspeicher für Foto-/Logo-Uploads: selbst gehostet auf eigener Infrastruktur']);
  });

  it('describes Sentry as not in use when OPERATOR_SENTRY_PROVIDER is unset', async () => {
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('de', 'datenschutz');
    const cookies = page.sections.find((s) => s.heading === 'Cookies und lokale Speicherung')!;
    expect(cookies.paragraphs.join(' ')).toContain('keine Fehlerüberwachung (Sentry)');
  });

  it('describes the configured Sentry determination when OPERATOR_SENTRY_PROVIDER is set', async () => {
    window.__RUNTIME_CONFIG__ = { OPERATOR_SENTRY_PROVIDER: 'sentry.io (EU region)' };
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('de', 'datenschutz');
    const cookies = page.sections.find((s) => s.heading === 'Cookies und lokale Speicherung')!;
    expect(cookies.paragraphs.join(' ')).toContain('§ 25 TDDDG');
  });

  it('falls back to OPERATOR_EMAIL for the data-protection contact when OPERATOR_DATA_PROTECTION_EMAIL is unset', async () => {
    window.__RUNTIME_CONFIG__ = { OPERATOR_EMAIL: 'info@yoadey.de' };
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('de', 'datenschutz');
    const verantwortlicher = page.sections.find((s) => s.heading === 'Verantwortlicher')!;
    expect(verantwortlicher.paragraphs[1]).toContain('info@yoadey.de');
  });

  it('renders the English placeholder markers for the controller identity when unset', async () => {
    const { getLegalContent } = await import('./content');
    const page = getLegalContent('en', 'datenschutz');
    const controller = page.sections.find((s) => s.heading === 'Controller')!;
    expect(controller.paragraphs[0]).toContain('[OPERATOR: name, address and contact, same as the legal notice]');
  });
});
