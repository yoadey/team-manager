import Box from '@mui/material/Box';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Sym, Av } from '@/components/ui';
import { t } from '@/i18n';
import type { SettingsPanelProps } from '../settingsCategories';

export function ProfilePanel({ app }: SettingsPanelProps) {
  const { state: S } = app;
  const tk = buildTokens(S.primaryColor);
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: '14px', p: '4px 2px 4px' }}>
      <Box sx={{ position: 'relative' }}>
        <Av name={S.user!.name} photo={S.user!.photo} color={S.user!.avatarColor} size={60} font={21} />
        <Box
          component="label"
          aria-label={t('team.changeProfilePhoto')}
          sx={{
            position: 'absolute',
            right: '-4px',
            bottom: '-4px',
            width: '28px',
            height: '28px',
            borderRadius: '50%',
            background: tk.primary,
            color: tk.onPrimary,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: 'pointer',
            boxShadow: '0 2px 6px rgba(0,0,0,.3)',
          }}
        >
          <Sym name="photo_camera" size={16} color={tk.onPrimary} />
          <input
            type="file"
            accept="image/*"
            onChange={(e) => app.onFile(e, (d) => app.uploadMyPhoto(d))}
            style={{ display: 'none' }}
          />
        </Box>
      </Box>
      <Box sx={{ minWidth: 0 }}>
        <Box sx={{ fontSize: '17px', fontWeight: 700 }}>{S.user!.name}</Box>
        <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, display: 'flex', alignItems: 'center', gap: '6px' }}>
          <Sym name="mail" size={15} />
          {S.user!.email}
        </Box>
      </Box>
    </Box>
  );
}
