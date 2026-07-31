import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import { t, type Locale } from '@/i18n';
import { useLocale } from '@/i18n/LocaleProvider';
import type { SettingsPanelProps } from '../settingsCategories';

/** Each language is shown in its own name (endonym), independent of UI locale. */
const LANGUAGE_LABELS: Record<Locale, string> = { de: 'Deutsch', en: 'English' };

export function AppearancePanel({ app }: SettingsPanelProps) {
  const { state } = app;
  const { locale, setLocale, supported } = useLocale();
  const tk = buildTokens(state.primaryColor);
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
      <Box>
        <Box sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, mb: '8px' }}>
          {t('team.colorScheme')}
        </Box>
        <Box sx={{ display: 'flex', gap: '6px' }}>
          {(['system', 'light', 'dark'] as const).map((scheme) => {
            const active = state.colorScheme === scheme;
            const label = t(`team.colorScheme${scheme.charAt(0).toUpperCase() + scheme.slice(1)}`);
            const icon = scheme === 'system' ? 'brightness_auto' : scheme === 'light' ? 'light_mode' : 'dark_mode';
            return (
              <ButtonBase
                key={scheme}
                onClick={() => app.setColorScheme(scheme)}
                aria-pressed={active}
                sx={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: '4px',
                  p: '10px 6px',
                  borderRadius: '12px',
                  border: active ? `2px solid ${tk.primary}` : `1px solid ${NEUTRAL.line}`,
                  background: active ? tk.primaryContainer : NEUTRAL.card,
                  color: active ? tk.primary : NEUTRAL.onSurfaceVariant,
                  fontSize: '11px',
                  fontWeight: 600,
                }}
              >
                <Sym name={icon} size={20} color={active ? tk.primary : NEUTRAL.secondary} />
                {label}
              </ButtonBase>
            );
          })}
        </Box>
      </Box>

      <Box>
        <Box sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, mb: '8px' }}>{t('team.language')}</Box>
        <Box sx={{ display: 'flex', gap: '6px' }}>
          {supported.map((lng) => {
            const active = locale === lng;
            return (
              <ButtonBase
                key={lng}
                onClick={() => setLocale(lng)}
                aria-pressed={active}
                sx={{
                  flex: 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  p: '10px 6px',
                  borderRadius: '12px',
                  border: active ? `2px solid ${tk.primary}` : `1px solid ${NEUTRAL.line}`,
                  background: active ? tk.primaryContainer : NEUTRAL.card,
                  color: active ? tk.primary : NEUTRAL.onSurfaceVariant,
                  fontSize: '13px',
                  fontWeight: 600,
                }}
              >
                {LANGUAGE_LABELS[lng]}
              </ButtonBase>
            );
          })}
        </Box>
      </Box>
    </Box>
  );
}
