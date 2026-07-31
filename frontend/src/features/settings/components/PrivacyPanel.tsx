import { useState, type ChangeEvent } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { NEUTRAL } from '@/styles/tokens';
import { Sym } from '@/components/ui';
import { t } from '@/i18n';
import { captureException } from '@/monitoring';
import { reportActionError, AuthError } from '@/utils/errors';
import type { SettingsPanelProps } from '../settingsCategories';

export function PrivacyPanel({ app }: SettingsPanelProps) {
  const { state: S } = app;

  // Account erasure (GDPR Art. 17): a destructive, irreversible action gated by
  // retyping the account email — no password, so the same flow also covers a
  // future OIDC-only account (no OIDC integration exists yet).
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [confirmEmail, setConfirmEmail] = useState('');
  const [deleting, setDeleting] = useState(false);
  const [deleteErr, setDeleteErr] = useState(false);
  const accountEmail = S.user?.email ?? '';
  const canConfirmDelete = confirmEmail.trim().toLowerCase() === accountEmail.toLowerCase() && accountEmail !== '';

  return (
    <Box>
      <ButtonBase
        onClick={async () => {
          try {
            await app.exportMyData();
          } catch (err) {
            reportActionError({ setState: app.setState, toastMsg: app.toastMsg, onAuthError: app.logout }, err, 'team.exportDataError');
          }
        }}
        sx={{
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '8px',
          p: '13px',
          mb: '10px',
          borderRadius: '14px',
          border: `1px solid ${NEUTRAL.line}`,
          background: NEUTRAL.card,
          color: NEUTRAL.onSurfaceVariant,
          fontWeight: 600,
          fontSize: '14px',
          cursor: 'pointer',
        }}
      >
        <Sym name="download" size={20} color={NEUTRAL.secondary} />
        {t('team.exportData')}
      </ButtonBase>
      {!deleteOpen ? (
        <ButtonBase
          onClick={() => {
            setDeleteOpen(true);
            setConfirmEmail('');
            setDeleteErr(false);
          }}
          sx={{
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            p: '13px',
            borderRadius: '14px',
            border: '1px solid #F0C4C0',
            background: NEUTRAL.errorBg,
            color: NEUTRAL.error,
            fontWeight: 600,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          <Sym name="delete_forever" size={20} color={NEUTRAL.error} />
          {t('team.deleteAccount')}
        </ButtonBase>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
          <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>
            {t('team.deleteAccountWarning')}
          </Box>
          <Box component="label" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary }}>
            {t('team.deleteAccountConfirmLabel')}
          </Box>
          <Box
            component="input"
            type="email"
            autoComplete="off"
            value={confirmEmail}
            placeholder={accountEmail}
            onChange={(e: ChangeEvent<HTMLInputElement>) => {
              setConfirmEmail(e.target.value);
              setDeleteErr(false);
            }}
            sx={{
              p: '11px 12px',
              borderRadius: '12px',
              border: `1px solid ${deleteErr ? NEUTRAL.error : NEUTRAL.line}`,
              background: NEUTRAL.card,
              fontSize: '14px',
              width: '100%',
              boxSizing: 'border-box',
            }}
          />
          {deleteErr ? <Box sx={{ fontSize: '12px', color: NEUTRAL.error }}>{t('team.deleteAccountError')}</Box> : null}
          <Box sx={{ display: 'flex', gap: '8px' }}>
            <ButtonBase
              onClick={() => {
                setDeleteOpen(false);
                setConfirmEmail('');
                setDeleteErr(false);
              }}
              disabled={deleting}
              sx={{
                flex: 1,
                p: '12px',
                borderRadius: '14px',
                border: `1px solid ${NEUTRAL.line}`,
                background: NEUTRAL.card,
                color: NEUTRAL.secondary,
                fontWeight: 600,
                fontSize: '14px',
              }}
            >
              {t('common.cancel')}
            </ButtonBase>
            <ButtonBase
              disabled={!canConfirmDelete || deleting}
              onClick={async () => {
                setDeleting(true);
                try {
                  await app.deleteAccount(confirmEmail.trim());
                  // On success the app resets to the login screen and this panel unmounts.
                } catch (err) {
                  if (err instanceof AuthError) {
                    reportActionError({ setState: app.setState, toastMsg: app.toastMsg, onAuthError: app.logout }, err);
                  } else {
                    captureException(err);
                    setDeleteErr(true);
                  }
                  setDeleting(false);
                }
              }}
              sx={{
                flex: 1,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: '6px',
                p: '12px',
                borderRadius: '14px',
                border: 'none',
                background: NEUTRAL.error,
                color: '#fff',
                fontWeight: 700,
                fontSize: '14px',
                opacity: !canConfirmDelete || deleting ? 0.5 : 1,
              }}
            >
              <Sym name="delete_forever" size={18} color="#fff" />
              {t('team.deleteAccountConfirmButton')}
            </ButtonBase>
          </Box>
        </Box>
      )}
    </Box>
  );
}
