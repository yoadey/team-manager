import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { getLocale, setLocale as applyLocale, subscribeLocale, SUPPORTED_LOCALES, type Locale } from './index';

interface LocaleContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  supported: Locale[];
}

const LocaleContext = createContext<LocaleContextValue | null>(null);

/**
 * Bridges the framework-agnostic i18n module into React: re-renders consumers
 * when the active locale changes so all `Intl`/`t()` output updates.
 */
export function LocaleProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(getLocale());

  useEffect(() => subscribeLocale(() => setLocaleState(getLocale())), []);

  // Keep the document language in sync so assistive technology and the browser
  // pick the correct pronunciation/hyphenation when the user switches locale.
  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const value: LocaleContextValue = {
    locale,
    setLocale: applyLocale,
    supported: SUPPORTED_LOCALES,
  };
  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale(): LocaleContextValue {
  const ctx = useContext(LocaleContext);
  if (!ctx) throw new Error('useLocale must be used within LocaleProvider');
  return ctx;
}
