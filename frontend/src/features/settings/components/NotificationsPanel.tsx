import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import { t } from '@/i18n';
import { usePushActions, usePushPreferencesActions } from '@/features/notifications';
import type { PushCategoryPreferences } from '@/features/notifications';
import type { SettingsPanelProps } from '../settingsCategories';
import type { AppContextValue } from '@/context/AppContext';

const PUSH_CATEGORIES: Array<{ key: keyof PushCategoryPreferences; labelKey: string }> = [
  { key: 'attendance', labelKey: 'push.categoryAttendance' },
  { key: 'events', labelKey: 'push.categoryEvents' },
  { key: 'news', labelKey: 'push.categoryNews' },
  { key: 'polls', labelKey: 'push.categoryPolls' },
  { key: 'absence', labelKey: 'push.categoryAbsence' },
];

/** Per-team push-category toggles, shown once the browser has an active push
 * subscription -- its own component to keep NotificationsPanel's complexity down. */
function PushCategoryPreferencesPanel({
  app,
  teamId,
  tk,
}: {
  app: AppContextValue;
  teamId: string;
  tk: ReturnType<typeof buildTokens>;
}) {
  const pushPrefs = usePushPreferencesActions(app.api, teamId, app.toastMsg, true);
  const teamName = app.activeTeam()?.name ?? '';
  return (
    <Box sx={{ mt: '18px' }}>
      <Box sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, mb: '4px' }}>
        {t('push.categoriesTitle')}
      </Box>
      <Box sx={{ fontSize: '11px', color: NEUTRAL.secondary, lineHeight: 1.4, mb: '8px' }}>
        {t('push.categoriesDescription', { team: teamName })}
      </Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
        {PUSH_CATEGORIES.map(({ key, labelKey }) => {
          const on = pushPrefs.prefs[key];
          return (
            <ButtonBase
              key={key}
              role="switch"
              aria-checked={on}
              disabled={pushPrefs.isLoading || pushPrefs.busy}
              onClick={() => pushPrefs.setCategory(key, !on)}
              sx={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: '10px',
                p: '10px 14px',
                borderRadius: '12px',
                border: `1px solid ${NEUTRAL.line}`,
                background: NEUTRAL.card,
                opacity: pushPrefs.busy ? 0.6 : 1,
                textAlign: 'left',
                fontSize: '13px',
                fontWeight: 600,
                color: NEUTRAL.onSurfaceVariant,
              }}
            >
              {t(labelKey)}
              <Sym name={on ? 'toggle_on' : 'toggle_off'} size={26} color={on ? tk.primary : NEUTRAL.secondary} />
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  );
}

export function NotificationsPanel({ app }: SettingsPanelProps) {
  const { state: S } = app;
  const tk = buildTokens(S.primaryColor);
  const push = usePushActions(app.api, app.toastMsg);
  return (
    <Box>
      {push.support === 'supported' && (
        <Box sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, mb: '8px' }}>{t('push.title')}</Box>
      )}
      {push.support === 'supported' && (
        <ButtonBase
          onClick={() => (push.subscribed ? push.disablePush() : push.enablePush())}
          disabled={push.busy || push.subscribed === null}
          aria-pressed={push.subscribed === true}
          sx={{
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            p: '12px 14px',
            borderRadius: '14px',
            border: push.subscribed ? `2px solid ${tk.primary}` : `1px solid ${NEUTRAL.line}`,
            background: push.subscribed ? tk.primaryContainer : NEUTRAL.card,
            color: push.subscribed ? tk.primary : NEUTRAL.onSurfaceVariant,
            opacity: push.busy ? 0.6 : 1,
            textAlign: 'left',
          }}
        >
          <Sym
            name={push.subscribed ? 'notifications_active' : 'notifications_off'}
            size={20}
            color={push.subscribed ? tk.primary : NEUTRAL.secondary}
          />
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box sx={{ fontSize: '13px', fontWeight: 600 }}>{t('push.title')}</Box>
            <Box sx={{ fontSize: '11px', color: NEUTRAL.secondary, lineHeight: 1.4 }}>{t('push.description')}</Box>
          </Box>
        </ButtonBase>
      )}

      {push.subscribed === true && S.activeTeamId && (
        <PushCategoryPreferencesPanel app={app} teamId={S.activeTeamId} tk={tk} />
      )}
    </Box>
  );
}
