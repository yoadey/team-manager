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

interface RegisterProps {
  /** Returns to the login screen's provider list / password form. */
  onBack: () => void;
}

/**
 * Self-service registration form, shown as an alternate view inside Login.tsx
 * (mirroring how invite acceptance folds into the login flow rather than
 * getting its own top-level Phase). On success it swaps to an inline "check
 * your email" confirmation with a resend link -- registration never logs the
 * user in directly, since the account isn't verified yet.
 */
export function Register({ onBack }: RegisterProps) {
  const { doRegister, doResendVerification, openLegal } = useApp();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  // GDPR Art. 8 requires parental consent for services offered directly to a
  // child under 16 in Germany; self-registration has no admin in the loop to
  // apply judgment (unlike the invite flow), so this self-asserted checkbox is
  // the gate -- see openspec/changes/webapp-legal-compliance/design.md
  // Decision 3 for why a checkbox rather than real age verification.
  const [ageConfirmed, setAgeConfirmed] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [submittedEmail, setSubmittedEmail] = useState<string | null>(null);
  const [resendBusy, setResendBusy] = useState(false);
  const [resendSent, setResendSent] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    // Belt-and-suspenders alongside the disabled submit button below --
    // pressing Enter in a field submits the form directly, bypassing the
    // button's disabled attribute.
    if (busy || !ageConfirmed) return;
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
    const ok = await doRegister(email, password);
    setBusy(false);
    // The generic 202 response never confirms whether an account already
    // existed for this email (see backend enumeration-safety design) -- the
    // same "check your email" confirmation is shown regardless.
    if (ok) setSubmittedEmail(email);
  }

  async function handleResend() {
    if (!submittedEmail || resendBusy) return;
    setResendBusy(true);
    const ok = await doResendVerification(submittedEmail);
    setResendBusy(false);
    if (ok) setResendSent(true);
  }

  if (submittedEmail) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
        <Box sx={{ fontSize: '16px', fontWeight: 700, color: NEUTRAL.onSurface }}>{t('auth.registerCheckEmailTitle')}</Box>
        <Box sx={{ fontSize: '13px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>
          {t('auth.registerCheckEmailBody', { email: submittedEmail })}
        </Box>
        {resendSent && (
          <Box role="status" sx={{ fontSize: '13px', color: NEUTRAL.secondary }}>
            {t('auth.resendVerificationSent')}
          </Box>
        )}
        <ButtonBase
          onClick={handleResend}
          disabled={resendBusy}
          sx={{ fontSize: '13px', color: '#1565C0', py: '4px', justifyContent: 'center' }}
        >
          {resendBusy ? <Spinner size={16} /> : t('auth.resendVerification')}
        </ButtonBase>
        <ButtonBase onClick={onBack} sx={{ fontSize: '13px', color: NEUTRAL.secondary, py: '4px', justifyContent: 'center' }}>
          ← {t('auth.back')}
        </ButtonBase>
      </Box>
    );
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
      <Box component="label" htmlFor="register-email" sx={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <Box component="span" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, px: '2px' }}>
          {t('auth.emailLabel')}
        </Box>
        <Box
          id="register-email"
          component="input"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEmail(e.target.value)}
          required
          sx={inputSx}
        />
      </Box>
      <Box component="label" htmlFor="register-password" sx={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <Box component="span" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, px: '2px' }}>
          {t('auth.passwordLabel')}
        </Box>
        <Box
          id="register-password"
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
      <Box component="label" htmlFor="register-confirm-password" sx={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <Box component="span" sx={{ fontSize: '12px', fontWeight: 600, color: NEUTRAL.secondary, px: '2px' }}>
          {t('auth.confirmPasswordLabel')}
        </Box>
        <Box
          id="register-confirm-password"
          component="input"
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setConfirmPassword(e.target.value)}
          required
          sx={inputSx}
        />
      </Box>
      <Box
        component="label"
        sx={{ display: 'flex', alignItems: 'flex-start', gap: '9px', fontSize: '12px', color: NEUTRAL.secondary, lineHeight: 1.5 }}
      >
        <Box
          component="input"
          type="checkbox"
          checked={ageConfirmed}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setAgeConfirmed(e.target.checked)}
          sx={{ width: '18px', height: '18px', flex: '0 0 auto', mt: '1px', accentColor: '#1565C0' }}
        />
        <Box component="span">
          {t('auth.registerAgeConfirm')} {t('auth.registerPrivacyIntro')}{' '}
          <ButtonBase
            component="span"
            onClick={() => openLegal('datenschutz')}
            sx={{ fontSize: '12px', color: '#1565C0', textDecoration: 'underline', display: 'inline', p: 0 }}
          >
            {t('sheet.legalDatenschutz')}
          </ButtonBase>
          .
        </Box>
      </Box>

      <ButtonBase
        component="button"
        type="submit"
        disabled={busy || !ageConfirmed}
        sx={{
          width: '100%',
          p: '12px 16px',
          borderRadius: '16px',
          background: '#1565C0',
          color: '#fff',
          fontWeight: 600,
          fontSize: '15px',
          justifyContent: 'center',
          opacity: busy || !ageConfirmed ? 0.7 : 1,
        }}
      >
        {busy ? <Spinner size={18} /> : t('auth.registerSubmit')}
      </ButtonBase>
      <ButtonBase onClick={onBack} sx={{ fontSize: '13px', color: NEUTRAL.secondary, py: '4px', justifyContent: 'center' }}>
        ← {t('auth.back')}
      </ButtonBase>
    </Box>
  );
}
