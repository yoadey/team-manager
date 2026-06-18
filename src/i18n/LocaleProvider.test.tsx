import { describe, it, expect } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { LocaleProvider, useLocale } from './LocaleProvider';
import { setLocale } from './index';

function LocaleConsumer() {
  const { locale, supported } = useLocale();
  return (
    <div>
      <div data-testid="locale">{locale}</div>
      <div data-testid="supported">{supported.join(',')}</div>
    </div>
  );
}

describe('LocaleProvider', () => {
  it('provides current locale to consumers', () => {
    render(
      <LocaleProvider>
        <LocaleConsumer />
      </LocaleProvider>,
    );
    expect(screen.getByTestId('locale').textContent).toMatch(/^(de|en)$/);
  });

  it('provides supported locales', () => {
    render(
      <LocaleProvider>
        <LocaleConsumer />
      </LocaleProvider>,
    );
    const supported = screen.getByTestId('supported').textContent!;
    expect(supported).toContain('de');
    expect(supported).toContain('en');
  });

  it('updates when locale changes', async () => {
    render(
      <LocaleProvider>
        <LocaleConsumer />
      </LocaleProvider>,
    );
    await act(async () => {
      setLocale('en');
    });
    expect(screen.getByTestId('locale').textContent).toBe('en');
    // Reset back to de
    await act(async () => {
      setLocale('de');
    });
  });

  it('throws when useLocale is used outside provider', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    expect(() => {
      render(<LocaleConsumer />);
    }).toThrow('useLocale must be used within LocaleProvider');
    spy.mockRestore();
  });
});
