import { useEffect, useRef } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Av, Field, inputSx, labelSx, PrimaryButton, SectionTitle, Sym, TextArea, TextInput } from '@/components/ui';
import { shortName } from '@/layouts/useCompact';
import type { Invite, Role, StatsThresholdMode } from '@/types';
import type { SheetProps } from '@/sheets/types';
import { createTeamSchema, type CreateTeamFormValues } from './createTeamSchema';
import {
  changedThresholds,
  teamSettingsSchema,
  toThresholdFormRows,
  type TeamSettingsFormValues,
  type ThresholdFormRow,
} from './teamSettingsSchema';
import { useStatsGroupThresholdsQuery } from '../hooks/useStatsGroupThresholdsQuery';
import { t } from '@/i18n';

const TEAM_ICONS = ['🏆', '⭐', '💃', '🕺', '🎭', '🔥', '👑', '🎯', '💎', '🦅', '⚡', '🌟'];

// Sentinel used to locate the {teamName} placeholder inside the interpolated
// team.inviteDesc2 string without splitting on the team name itself (see the
// comment at its use site for why that's unsafe). Control character, so it
// can never occur in real translated text.
const TEAM_NAME_TOKEN = '\u0001';

export function CreateTeamSheet({ app, sheet }: SheetProps) {
  void sheet;
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  void team;

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<CreateTeamFormValues>({
    resolver: zodResolver(createTeamSchema),
    defaultValues: { name: '', icon: '⭐', photo: null },
    mode: 'onBlur',
  });

  const name = watch('name');
  const icon = watch('icon');
  const photo = watch('photo');
  const canSubmit = !!name?.trim();

  const onSubmit = async (values: CreateTeamFormValues) => {
    try {
      await app.createTeam(values);
    } catch {
      // Ignored
    }
  };

  const icons = TEAM_ICONS.map((em) => (
    <ButtonBase
      key={em}
      role="radio"
      aria-checked={icon === em}
      aria-label={em}
      onClick={() => setValue('icon', em, { shouldValidate: true })}
      sx={{
        width: '48px',
        height: '48px',
        borderRadius: '13px',
        border: '2px solid ' + (icon === em ? tk.primary : NEUTRAL.line3),
        background: icon === em ? tk.primaryContainer : NEUTRAL.card,
        cursor: 'pointer',
        fontSize: '22px',
      }}
    >
      {em}
    </ButtonBase>
  ));

  const photoRow = (
    <Box
      key="ph"
      component="label"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        p: '12px 14px',
        borderRadius: '13px',
        border: `1px dashed ${NEUTRAL.inputBorder}`,
        background: NEUTRAL.sidebar,
        cursor: 'pointer',
      }}
    >
      {photo ? (
        <Av key="a" name={t('team.photoAlt')} photo={photo} color={NEUTRAL.inputBorder} size={40} />
      ) : (
        <Sym name="add_photo_alternate" size={24} color={NEUTRAL.secondary} />
      )}
      <Box
        key="l"
        component="span"
        sx={{ flex: 1, fontSize: '13px', fontWeight: 600, color: NEUTRAL.onSurfaceVariant }}
      >
        {photo ? t('team.photoSelected') : t('team.photoUpload')}
      </Box>
      <input key="f" type="file" accept="image/*" onChange={(e) => app.onFile(e, (d) => setValue('photo', d))} hidden />
    </Box>
  );

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
    >
      <Box
        key="i"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          fontSize: '13px',
          color: NEUTRAL.secondary,
          lineHeight: 1.5,
          background: NEUTRAL.sidebar,
          p: '12px 14px',
          borderRadius: '13px',
        }}
      >
        <Sym name="shield_person" size={24} color={tk.primary} />
        {t('team.createTeamHint')}
      </Box>
      <Field label={t('team.teamNameField')} error={!!errors.name} errorText={errors.name?.message}>
        <TextInput placeholder={t('team.teamNamePlaceholder')} maxLength={60} {...register('name')} />
      </Field>
      <Box key="ic">
        <Box key="l" sx={labelSx}>
          {t('team.iconLabel')}
        </Box>
        <Box
          key="b"
          role="radiogroup"
          aria-label={t('team.iconLabel')}
          sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}
        >
          {icons}
        </Box>
      </Box>
      {photoRow}
      <PrimaryButton
        label={t('team.createBtn')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting}
        disabled={!canSubmit}
      />
    </Box>
  );
}

export function InviteSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const inv: Invite | null = sheet.invite ?? null;

  return (
    <Box>
      <Box key="hero" sx={{ textAlign: 'center', p: '6px 2px 18px' }}>
        <Box
          key="i"
          sx={{
            width: '64px',
            height: '64px',
            borderRadius: '18px',
            background: tk.primaryContainer,
            color: tk.primary,
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Sym name="link" size={34} color={tk.primary} />
        </Box>
        <Box key="s" sx={{ fontSize: '14px', color: NEUTRAL.secondary, mt: '12px', lineHeight: 1.5 }}>
          {/* Interpolate a sentinel token (never present in translated text) rather
              than the team name itself, then split on that -- splitting on the team
              name directly breaks when its first word (via shortName) collides with
              any other word already in the surrounding sentence, e.g. a team named
              "Neue Elf" would match both the template's own "Neue Mitglieder" and
              the intended placeholder, producing duplicated/misplaced bolding. */}
          {t('team.inviteDesc2', { teamName: TEAM_NAME_TOKEN })
            .split(TEAM_NAME_TOKEN)
            .reduce<React.ReactNode[]>((acc, part, i, arr) => {
              if (i < arr.length - 1) return [...acc, part, <b key={i}>{shortName(team.name)}</b>];
              return [...acc, part];
            }, [])}
        </Box>
      </Box>
      <Box
        key="box"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          background: NEUTRAL.sidebar,
          border: `1px solid ${NEUTRAL.line3}`,
          borderRadius: '13px',
          p: '12px 14px',
        }}
      >
        <Box
          key="l"
          component="span"
          sx={{
            flex: 1,
            fontSize: '13px',
            fontFamily: 'monospace',
            color: NEUTRAL.onSurface,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {inv ? inv.link : t('team.inviteGenerating')}
        </Box>
        {inv ? (
          <ButtonBase
            key="c"
            onClick={() => app.copyInvite()}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              background: tk.primary,
              color: tk.onPrimary,
              border: 'none',
              borderRadius: '9px',
              p: '8px 12px',
              fontSize: '13px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            <Sym name="content_copy" size={16} color={tk.onPrimary} />
            {sheet.copied ? t('team.inviteCopied') : t('team.inviteCopy')}
          </ButtonBase>
        ) : null}
      </Box>
      {inv ? (
        <Box key="code" sx={{ textAlign: 'center', mt: '14px', fontSize: '13px', color: NEUTRAL.secondary }}>
          {t('team.inviteCode')}{' '}
          <Box
            key="b"
            component="b"
            sx={{ fontFamily: 'monospace', fontSize: '15px', letterSpacing: '1px', color: NEUTRAL.onSurface }}
          >
            {inv.code}
          </Box>
        </Box>
      ) : null}
    </Box>
  );
}

// react-hook-form's per-row error shape for the `thresholds` array, narrowed
// to the one field that can fail (see thresholdRowSchema's superRefine).
type ThresholdRowErrors = ({ value?: { message?: string } } | undefined)[];

const THRESHOLD_MODES: StatsThresholdMode[] = ['percent', 'count'];

/**
 * Team settings section: one row per role with a percent/count toggle and
 * the threshold value. Below that value an incomplete group is marked red in
 * the statistics' "Termine" tab. Split out of TeamSettingsSheet to keep that
 * component's render function within the lint complexity limit.
 */
function ThresholdSection({
  roles,
  rows,
  errors,
  primary,
  onChange,
}: {
  roles: Role[];
  rows: ThresholdFormRow[];
  errors: ThresholdRowErrors;
  primary: string;
  onChange: (roleId: string, patch: Partial<ThresholdFormRow>) => void;
}) {
  const roleById = new Map(roles.map((r) => [r.id, r]));
  return (
    <Box>
      <SectionTitle>{t('team.settingsThresholdSection')}</SectionTitle>
      <Box sx={{ fontSize: '12px', color: NEUTRAL.faint, m: '-2px 0 10px', lineHeight: 1.5 }}>
        {t('team.settingsThresholdHint')}
      </Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
        {rows.map((row, idx) => {
          // A threshold whose role isn't in the loaded role list (e.g. the
          // role list hasn't caught up yet) has no name to show -- skip the
          // row; it stays in the form untouched and so is never re-saved.
          const role = roleById.get(row.roleId);
          if (!role) return null;
          const error = errors[idx]?.value?.message;
          const errorId = 'threshold-error-' + row.roleId;
          return (
            <Box key={row.roleId} data-testid="threshold-row">
              <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: '8px', flex: '1 1 140px', minWidth: 0 }}>
                  <Box
                    component="span"
                    sx={{ width: '9px', height: '9px', borderRadius: '50%', background: role.color, flex: '0 0 auto' }}
                  />
                  <Box
                    sx={{
                      fontSize: '14px',
                      fontWeight: 600,
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {role.name}
                  </Box>
                </Box>
                <Box
                  role="radiogroup"
                  aria-label={t('team.thresholdModeAria', { name: role.name })}
                  sx={{ display: 'flex', background: NEUTRAL.line2, borderRadius: '10px', p: '3px', flex: '0 0 auto' }}
                >
                  {THRESHOLD_MODES.map((mode) => {
                    const sel = row.mode === mode;
                    return (
                      <ButtonBase
                        key={mode}
                        role="radio"
                        aria-checked={sel}
                        onClick={() => onChange(row.roleId, { mode })}
                        sx={{
                          p: '6px 11px',
                          borderRadius: '8px',
                          fontSize: '12px',
                          fontWeight: 600,
                          background: sel ? NEUTRAL.card : 'transparent',
                          color: sel ? primary : NEUTRAL.secondary,
                          boxShadow: sel ? '0 1px 3px rgba(0,0,0,.12)' : 'none',
                        }}
                      >
                        {t(mode === 'percent' ? 'team.thresholdModePercent' : 'team.thresholdModeCount')}
                      </ButtonBase>
                    );
                  })}
                </Box>
                <input
                  type="number"
                  inputMode="numeric"
                  min={0}
                  max={row.mode === 'percent' ? 100 : 10000}
                  step={1}
                  value={row.value}
                  aria-label={t('team.thresholdValueAria', { name: role.name })}
                  aria-invalid={error ? true : undefined}
                  aria-describedby={error ? errorId : undefined}
                  onChange={(e) => onChange(row.roleId, { value: e.target.value })}
                  style={{
                    ...inputSx,
                    width: '84px',
                    padding: '8px 10px',
                    textAlign: 'right',
                    ...(error ? { borderColor: NEUTRAL.error } : {}),
                  }}
                />
              </Box>
              {error ? (
                <Box id={errorId} role="alert" sx={{ fontSize: '12px', color: NEUTRAL.error, mt: '4px', textAlign: 'right' }}>
                  {error}
                </Box>
              ) : null}
            </Box>
          );
        })}
      </Box>
    </Box>
  );
}

export function TeamSettingsSheet({ app, sheet }: SheetProps) {
  void sheet;
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const roles = app.state.roles;

  const {
    register,
    handleSubmit,
    setValue,
    getValues,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<TeamSettingsFormValues>({
    resolver: zodResolver(teamSettingsSchema),
    defaultValues: {
      name: team.name,
      description: team.description || '',
      reasonRoles: (team.reasonVisibilityRoles || []).slice(),
    },
    mode: 'onBlur',
  });

  const name = watch('name');
  const reasonRoles = watch('reasonRoles');
  const thresholds = watch('thresholds');
  const canSubmit = !!name?.trim();

  // Per-role attendance thresholds load separately from the team itself
  // (they're their own, settings-gated resource), so they're seeded into the
  // form once they arrive instead of via defaultValues. `seededFrom` is the
  // server snapshot the rows were last seeded from: a later refetch (after a
  // save, or a background one) re-seeds only while the user has no unsaved
  // threshold edits, so it can never clobber a half-typed value -- and the
  // save diff below is always taken against the latest server state.
  const { data: loadedThresholds } = useStatsGroupThresholdsQuery(app.api, state.activeTeamId);
  const seededFrom = useRef<typeof loadedThresholds>(undefined);
  useEffect(() => {
    if (!loadedThresholds) return;
    const untouched = changedThresholds(getValues('thresholds'), seededFrom.current).length === 0;
    if (untouched) setValue('thresholds', toThresholdFormRows(loadedThresholds));
    seededFrom.current = loadedThresholds;
  }, [loadedThresholds, getValues, setValue]);

  const setThreshold = (roleId: string, patch: Partial<ThresholdFormRow>) => {
    const next = (getValues('thresholds') ?? []).map((row) => (row.roleId === roleId ? { ...row, ...patch } : row));
    setValue('thresholds', next, { shouldValidate: true, shouldDirty: true });
  };

  const onSubmit = async (values: TeamSettingsFormValues) => {
    try {
      await app.saveTeamSettings(values, changedThresholds(values.thresholds, loadedThresholds));
    } catch {
      // Ignored
    }
  };

  const upLabel = (icon: string, label: string, cb: (d: string, teamId: string) => void) => (
    <Box
      key="u"
      component="label"
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '8px',
        p: '9px 14px',
        borderRadius: '12px',
        border: `1px solid ${NEUTRAL.inputBorder}`,
        background: NEUTRAL.card,
        cursor: 'pointer',
        fontSize: '13px',
        fontWeight: 600,
        color: NEUTRAL.onSurfaceVariant,
      }}
    >
      <Sym name={icon} size={18} />
      {label}
      <input
        key="f"
        type="file"
        accept="image/*"
        onChange={(e) => {
          // Snapshot activeTeamId here, synchronously, before onFile starts
          // its FileReader read -- reading it inside cb (which only runs
          // once the read completes) would capture whatever team is active
          // AT THAT LATER POINT, not the team this upload was actually for.
          const teamId = app.state.activeTeamId!;
          app.onFile(e, (d) => cb(d, teamId));
        }}
        hidden
      />
    </Box>
  );

  const logoPreview = (
    <Box
      key="lp"
      component="span"
      role={team.logo ? 'img' : undefined}
      aria-label={team.logo ? t('team.logoAlt') : undefined}
      sx={{
        width: '58px',
        height: '58px',
        borderRadius: '15px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: '28px',
        flex: '0 0 auto',
        overflow: 'hidden',
        ...(team.logo
          ? { backgroundImage: 'url(' + team.logo + ')', backgroundSize: 'cover', backgroundPosition: 'center' }
          : { background: team.iconBg, color: team.iconFg }),
      }}
    >
      {team.logo ? '' : team.icon}
    </Box>
  );

  const logoSec = (
    <Box key="logo">
      <SectionTitle>{t('team.settingsLogoSection')}</SectionTitle>
      <Box key="r" sx={{ display: 'flex', alignItems: 'center', gap: '14px', mb: '10px' }}>
        {logoPreview}
        {upLabel('upload', team.logo ? t('team.settingsLogoChange') : t('team.settingsLogoUpload'), (d, teamId) =>
          app.saveTeamLogo(d, teamId),
        )}
      </Box>
      <Box key="em" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {TEAM_ICONS.map((em) => (
          <ButtonBase
            key={em}
            role="radio"
            aria-checked={!team.logo && team.icon === em}
            aria-label={em}
            onClick={() => app.setTeamIcon(em)}
            sx={{
              width: '44px',
              height: '44px',
              borderRadius: '12px',
              border: '2px solid ' + (!team.logo && team.icon === em ? tk.primary : NEUTRAL.line3),
              background: !team.logo && team.icon === em ? tk.primaryContainer : NEUTRAL.card,
              cursor: 'pointer',
              fontSize: '20px',
            }}
          >
            {em}
          </ButtonBase>
        ))}
      </Box>
    </Box>
  );

  const photoSec = (
    <Box key="photo">
      <SectionTitle>{t('team.settingsPhotoSection')}</SectionTitle>
      <Box key="r" sx={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
        {team.photo ? (
          <Av key="a" name={team.name} photo={team.photo} color={NEUTRAL.inputBorder} size={58} />
        ) : (
          <Box
            key="i"
            component="span"
            sx={{
              width: '58px',
              height: '58px',
              borderRadius: '15px',
              background: NEUTRAL.line2,
              color: NEUTRAL.faint,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flex: '0 0 auto',
            }}
          >
            <Sym name="image" size={26} color={NEUTRAL.faint} />
          </Box>
        )}
        {upLabel(
          'photo_camera',
          team.photo ? t('team.settingsPhotoChange') : t('team.settingsPhotoUpload'),
          (d, teamId) => app.saveTeamPhoto(d, teamId),
        )}
        {team.photo ? (
          <ButtonBase
            key="rm"
            onClick={() => app.removeTeamPhoto()}
            aria-label={t('team.settingsPhotoRemove')}
            sx={{
              width: '36px',
              height: '36px',
              borderRadius: '50%',
              background: NEUTRAL.errorBg,
              color: NEUTRAL.error,
              cursor: 'pointer',
              flex: '0 0 auto',
            }}
          >
            <Sym name="delete" size={18} color={NEUTRAL.error} />
          </ButtonBase>
        ) : null}
      </Box>
      <Box key="h" sx={{ fontSize: '12px', color: NEUTRAL.faint, mt: '8px', lineHeight: 1.5 }}>
        {t('team.settingsPhotoHint')}
      </Box>
    </Box>
  );

  const visSec = (
    <Box key="vis">
      <SectionTitle>{t('team.settingsVisSection')}</SectionTitle>
      <Box key="h" sx={{ fontSize: '12px', color: NEUTRAL.faint, m: '-2px 0 10px', lineHeight: 1.5 }}>
        {t('team.settingsVisHint')}
      </Box>
      <Box key="b" sx={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {roles.map((r) => {
          const sel = (reasonRoles || []).includes(r.id);
          return (
            <ButtonBase
              key={r.id}
              role="checkbox"
              aria-checked={sel}
              aria-label={r.name}
              onClick={() => {
                const cur = reasonRoles || [];
                const next = sel ? cur.filter((x) => x !== r.id) : cur.concat(r.id);
                setValue('reasonRoles', next, { shouldValidate: true });
              }}
              sx={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: '7px',
                p: '8px 13px',
                borderRadius: '999px',
                cursor: 'pointer',
                fontSize: '13px',
                fontWeight: 600,
                border: '1.5px solid ' + (sel ? r.color : NEUTRAL.inputBorder),
                background: sel ? r.color + '1A' : NEUTRAL.card,
                color: sel ? r.color : NEUTRAL.onSurfaceVariant,
              }}
            >
              <Box
                key="d"
                component="span"
                sx={{ width: '9px', height: '9px', borderRadius: '50%', background: r.color }}
              />
              {r.name}
              {sel ? <Sym name="check" size={16} color={r.color} /> : null}
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  );

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      sx={{ display: 'flex', flexDirection: 'column', gap: '18px' }}
    >
      <Field label={t('team.settingsNameField')} error={!!errors.name} errorText={errors.name?.message}>
        <TextInput placeholder={t('team.settingsNamePlaceholder')} maxLength={60} {...register('name')} />
      </Field>
      <Field label={t('team.settingsDescField')} error={!!errors.description} errorText={errors.description?.message}>
        <TextArea
          placeholder={t('team.settingsDescPlaceholder')}
          minHeight={80}
          maxLength={10000}
          {...register('description')}
        />
      </Field>
      {logoSec}
      {photoSec}
      {visSec}
      {thresholds?.length ? (
        <ThresholdSection
          roles={roles}
          rows={thresholds}
          errors={(errors.thresholds ?? []) as ThresholdRowErrors}
          primary={tk.primary}
          onChange={setThreshold}
        />
      ) : null}
      <PrimaryButton
        label={t('team.settingsSave')}
        onClick={handleSubmit(onSubmit)}
        busy={isSubmitting}
        disabled={!canSubmit}
      />
    </Box>
  );
}
