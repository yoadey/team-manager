import { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { useCompact } from '@/layouts/useCompact';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import { t } from '@/i18n';
import { SETTINGS_CATEGORIES, type SettingsCategoryKey } from './settingsCategories';

const logoutButtonSx = {
  width: '100%',
  display: 'flex',
  alignItems: 'center',
  gap: '10px',
  p: '12px 14px',
  borderRadius: '14px',
  border: '1px solid #F0C4C0',
  background: NEUTRAL.errorBg,
  color: NEUTRAL.error,
  fontWeight: 600,
  fontSize: '14px',
  cursor: 'pointer',
  justifyContent: 'flex-start',
  textAlign: 'left' as const,
};

export function SettingsPage() {
  const app = useApp();
  const compact = useCompact();
  const tk = buildTokens(app.state.primaryColor);
  const [selected, setSelected] = useState<SettingsCategoryKey | null>(compact ? null : SETTINGS_CATEGORIES[0]!.key);

  const categoryList = (onSelect: (key: SettingsCategoryKey) => void, active: SettingsCategoryKey | null) => (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
      {SETTINGS_CATEGORIES.map((cat) => {
        const isActive = cat.key === active;
        return (
          <ButtonBase
            key={cat.key}
            onClick={() => onSelect(cat.key)}
            aria-current={isActive ? 'page' : undefined}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              width: '100%',
              p: '12px 14px',
              borderRadius: '13px',
              justifyContent: 'flex-start',
              background: isActive && !compact ? tk.secondaryContainer : 'transparent',
              color: isActive && !compact ? tk.onSecondaryContainer : NEUTRAL.onSurfaceVariant,
              fontWeight: isActive && !compact ? 700 : 500,
            }}
          >
            <Sym name={cat.icon} size={22} />
            <Box component="span" sx={{ fontSize: '14px', fontWeight: 'inherit', flex: 1, textAlign: 'left' }}>
              {t(cat.labelKey)}
            </Box>
            {compact ? <Sym name="chevron_right" size={20} color={NEUTRAL.faint} /> : null}
          </ButtonBase>
        );
      })}
    </Box>
  );

  const logoutButton = (
    <ButtonBase onClick={() => app.logout()} sx={logoutButtonSx}>
      <Sym name="logout" size={20} color={NEUTRAL.error} />
      {t('team.logout')}
    </ButtonBase>
  );

  if (compact) {
    if (!selected) {
      return (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
          {categoryList(setSelected, null)}
          {logoutButton}
        </Box>
      );
    }
    const category = SETTINGS_CATEGORIES.find((c) => c.key === selected)!;
    const Panel = category.Component;
    return (
      <Box>
        <ButtonBase
          onClick={() => setSelected(null)}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            mb: '16px',
            color: NEUTRAL.onSurfaceVariant,
            fontWeight: 600,
            fontSize: '14px',
          }}
        >
          <Sym name="arrow_back" size={20} />
          {t('nav.settings')}
        </ButtonBase>
        <Panel app={app} />
      </Box>
    );
  }

  const activeCategory = SETTINGS_CATEGORIES.find((c) => c.key === selected) ?? SETTINGS_CATEGORIES[0]!;
  const ActivePanel = activeCategory.Component;
  return (
    <Box sx={{ display: 'flex', gap: '32px', alignItems: 'flex-start' }}>
      <Box sx={{ flex: '0 0 240px', display: 'flex', flexDirection: 'column', gap: '18px' }}>
        {categoryList(setSelected, selected)}
        {logoutButton}
      </Box>
      <Box sx={{ flex: 1, minWidth: 0, maxWidth: '640px' }}>
        <ActivePanel app={app} />
      </Box>
    </Box>
  );
}
