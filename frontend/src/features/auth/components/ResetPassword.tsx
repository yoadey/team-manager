import React, { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { NEUTRAL } from '@/styles/tokens';
import { Spinner, Sym } from '@/components/ui';
import { t } from '@/i18n';

const inputSx = {
  width: '100%',
  p: '10px 14px',
  borderRadius: '12px',
  border: `1.5px solid ${NEUTRAL.inputBorder}`,
  background: NEUTRAL.surface,
  color: NEUTRAL.onSurface,
  fontSize: '14px',
  outline: 'none',
  fontFamily: 'inherit',
};

interface ResetPasswordProps {
  /** Raw reset token parsed from the /reset-password/<token> URL. */
  token: string;
}

/**
 * Password-reset form reached via an emailed /reset-password/<token> link
 * (see context/urlState.ts's parseResetPasswordToken and the bootstrap
 * effect in AppContext.tsx that surfaces this view). On success
 * doResetPassword establishes a session the same way a password login does,
 * so there is no local "back" affordance -- a successful submit leaves the
 * login screen entirely.
 */
export function ResetPassword({ token }: ResetPasswordProps) {
  const { doResetPassword } = useApp();

  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [localError, setLocalError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (busy) return;
    setLocalError(null);
    if (password.length < 8) {
      setLocalError(t('auth.passwordTooShort'));
      return;
    }
    if (password !== confirmPassword) {
      setLocalError(t('auth.passwordMismatch'));
      return;
    }
    setBusy(true);
    const ok = await doResetPassword(token, password);
    if (!ok) setBusy(false);
  }

  return (
    <Box component="form" onSubmit={handleSubmit} sx={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      {localError && (
        <Box
          role="alert"
          sx={{
            p: '12px 14px',
            borderRadius: '14px',
            background: NEUTRAL.errorBg,
            color: NEUTRAL.error,
            fontSize: '13px',
            lineHeight: 1.5,
            display: 'flex',
            gap: '10px',
            alignItems: 'flex-start',
          }}
        >
          <Sym name="error" size={18} color={NEUTRAL.error} sx={{ lineHeight: 1.2 }} />
          {localError}
        </Box>
      )}
      <Box sx={{ fontSize: '16px', fontWeight: 700, color: NEUTRAL.onSurface }}>{t('auth.resetPasswordTitle')}</Box>
      <Box component="label" htmlFor="reset-password-password" sx={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <Box component="span" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, px: '2px' }}>
          {t('auth.newPasswordLabel')}
        </Box>
        <Box
          id="reset-password-password"
          component="input"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPassword(e.target.value)}
          required
          minLength={8}
          sx={inputSx}
        />
      </Box>
      <Box component="label" htmlFor="reset-password-confirm" sx={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <Box component="span" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, px: '2px' }}>
          {t('auth.confirmPasswordLabel')}
        </Box>
        <Box
          id="reset-password-confirm"
          component="input"
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setConfirmPassword(e.target.value)}
          required
          sx={inputSx}
        />
      </Box>
      <ButtonBase
        component="button"
        type="submit"
        disabled={busy}
        sx={{
          width: '100%',
          p: '12px 16px',
          borderRadius: '16px',
          background: '#1565C0',
          color: '#fff',
          fontWeight: 600,
          fontSize: '15px',
          justifyContent: 'center',
          opacity: busy ? 0.7 : 1,
        }}
      >
        {busy ? <Spinner size={18} /> : t('auth.resetPasswordSubmit')}
      </ButtonBase>
    </Box>
  );
}
