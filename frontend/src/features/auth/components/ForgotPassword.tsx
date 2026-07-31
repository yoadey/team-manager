import React, { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { NEUTRAL } from '@/styles/tokens';
import { Spinner } from '@/components/ui';
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

interface ForgotPasswordProps {
  /** Returns to the login screen's password form. */
  onBack: () => void;
}

/**
 * Password-reset request form, shown as an alternate view inside Login.tsx
 * (mirrors Register.tsx's placement). On success it swaps to an inline
 * "check your email" confirmation -- the generic 202 response never
 * confirms whether an account or password exists for the submitted email,
 * so the same confirmation is shown regardless.
 */
export function ForgotPassword({ onBack }: ForgotPasswordProps) {
  const { doForgotPassword } = useApp();

  const [email, setEmail] = useState('');
  const [busy, setBusy] = useState(false);
  const [submittedEmail, setSubmittedEmail] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (busy) return;
    setBusy(true);
    const ok = await doForgotPassword(email);
    setBusy(false);
    if (ok) setSubmittedEmail(email);
  }

  if (submittedEmail) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
        <Box sx={{ fontSize: '16px', fontWeight: 700, color: NEUTRAL.onSurface }}>{t('auth.forgotPasswordCheckEmailTitle')}</Box>
        <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>
          {t('auth.forgotPasswordCheckEmailBody', { email: submittedEmail })}
        </Box>
        <ButtonBase onClick={onBack} sx={{ fontSize: '13px', color: NEUTRAL.secondary, py: '4px', justifyContent: 'center' }}>
          ← {t('auth.back')}
        </ButtonBase>
      </Box>
    );
  }

  return (
    <Box component="form" onSubmit={handleSubmit} sx={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>{t('auth.forgotPasswordIntro')}</Box>
      <Box component="label" htmlFor="forgot-password-email" sx={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <Box component="span" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, px: '2px' }}>
          {t('auth.emailLabel')}
        </Box>
        <Box
          id="forgot-password-email"
          component="input"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEmail(e.target.value)}
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
        {busy ? <Spinner size={18} /> : t('auth.forgotPasswordSubmit')}
      </ButtonBase>
      <ButtonBase onClick={onBack} sx={{ fontSize: '13px', color: NEUTRAL.secondary, py: '4px', justifyContent: 'center' }}>
        ← {t('auth.back')}
      </ButtonBase>
    </Box>
  );
}
