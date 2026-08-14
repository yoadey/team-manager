import { useEffect, useState } from 'react';
import Box from '@mui/material/Box';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Field, PrimaryButton, Sym, TextInput, Av } from '@/components/ui';
import { t } from '@/i18n';
import { useMembersQuery } from '@/features/members';
import type { SettingsPanelProps } from '../settingsCategories';

const TITLE_MAX_LEN = 40;

export function ProfilePanel({ app }: SettingsPanelProps) {
  const { state: S } = app;
  const tk = buildTokens(S.primaryColor);
  const { data: members } = useMembersQuery(app.api, S.activeTeamId);
  const me = members?.find((m) => m.userId === S.user!.id) ?? null;

  const [title, setTitle] = useState('');
  const [touched, setTouched] = useState(false);
  const [saving, setSaving] = useState(false);

  // Sync from the loaded member row once, but never after the user has
  // started editing -- otherwise a background refetch (e.g. after this same
  // save invalidates the members query) would clobber in-progress input.
  useEffect(() => {
    if (me && !touched) setTitle(me.title);
  }, [me, touched]);

  const tooLong = title.length > TITLE_MAX_LEN;
  const dirty = touched && me !== null && title.trim() !== me.title;

  const save = async () => {
    if (!me || tooLong) return;
    setSaving(true);
    try {
      await app.setMyTitle(me.membershipId, title.trim());
      setTouched(false);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
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
      {me ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', maxWidth: '360px' }}>
          <Field
            label={t('members.fieldTitle')}
            helperText={t('members.fieldTitleHint')}
            error={tooLong}
            errorText={tooLong ? t('members.fieldTitleError') : undefined}
          >
            <TextInput
              name="title"
              value={title}
              onChange={(e) => {
                setTouched(true);
                setTitle(e.target.value);
              }}
              placeholder={t('members.fieldTitlePlaceholder')}
              maxLength={TITLE_MAX_LEN}
            />
          </Field>
          <Box>
            <PrimaryButton label={t('common.save')} onClick={save} disabled={!dirty || tooLong} busy={saving} />
          </Box>
        </Box>
      ) : null}
    </Box>
  );
}
