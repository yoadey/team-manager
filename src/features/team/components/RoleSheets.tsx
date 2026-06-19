import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { buildTokens } from '@/styles/tokens';
import { Chip, Field, labelSx, PrimaryButton, Sym, TextInput } from '@/components/ui';
import { MODULE_LABELS } from '@/services/serviceLayer';
import type { ModuleKey, PermLevel } from '@/types';
import type { SheetProps } from '@/sheets/types';
import { t } from '@/i18n';

export function RolesSheet({ app, sheet }: SheetProps) {
  void sheet;
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  void team;

  const lvl = (v: string) =>
    v === 'write'
      ? { l: t('team.permWrite'), bg: '#D7F0D8', c: '#235C26' }
      : v === 'read'
        ? { l: t('team.permRead'), bg: '#D7E3FF', c: '#00315C' }
        : { l: t('team.permNone'), bg: '#ECEDF3', c: '#9A9DA6' };

  const cards = state.roles.map((r) => (
    <Box key={r.id} sx={{ border: '1px solid #E6E7EE', borderRadius: '16px', p: '14px', background: '#fff' }}>
      <Box key="h" sx={{ display: 'flex', alignItems: 'center', gap: '10px', mb: '10px' }}>
        <Box
          key="d"
          component="span"
          sx={{ width: '12px', height: '12px', borderRadius: '50%', background: r.color, flex: '0 0 auto' }}
        />
        <Box key="n" component="span" sx={{ fontSize: '15px', fontWeight: 700, flex: 1 }}>
          {r.name}
        </Box>
        <Chip
          key="b"
          label={r.system ? t('team.roleStandard') : t('team.roleCustom')}
          color={r.system ? '#6A6D76' : tk.primary}
          bg={r.system ? '#ECEDF3' : tk.primaryContainer}
        />
      </Box>
      <Box key="p" sx={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
        {(Object.keys(MODULE_LABELS) as ModuleKey[]).map((mod) => {
          const L = lvl(r.permissions[mod]);
          return (
            <Box
              key={mod}
              component="span"
              sx={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: '4px',
                fontSize: '11px',
                color: L.c,
                background: L.bg,
                p: '4px 9px',
                borderRadius: '8px',
                fontWeight: 500,
              }}
            >
              {MODULE_LABELS[mod] + ': '}
              <b key="b">{L.l}</b>
            </Box>
          );
        })}
      </Box>
    </Box>
  ));

  const add = app.can('settings', 'write') ? (
    <ButtonBase
      key="add"
      onClick={() => app.openCreateRole()}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        p: '14px',
        borderRadius: '16px',
        border: '1.5px dashed #C8CAD2',
        background: 'transparent',
        cursor: 'pointer',
        color: tk.primary,
        fontWeight: 600,
        fontSize: '14px',
      }}
    >
      <Sym name="add_circle" size={24} color={tk.primary} />
      {t('team.addRole')}
    </ButtonBase>
  ) : null;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      {cards}
      {add}
    </Box>
  );
}

export function RoleFormSheet({ app, sheet }: SheetProps) {
  void sheet;
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  void team;
  const F = app.state.form;

  const rows = (Object.keys(MODULE_LABELS) as ModuleKey[]).map((mod) => (
    <Box
      key={mod}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '10px',
        p: '8px 12px',
        background: '#F4F4FA',
        borderRadius: '12px',
      }}
    >
      <Box key="l" component="span" sx={{ flex: 1, fontSize: '14px', fontWeight: 500 }}>
        {MODULE_LABELS[mod]}
      </Box>
      <Box
        key="b"
        sx={{ display: 'flex', background: '#fff', borderRadius: '9px', p: '3px', border: '1px solid #E0E2EA' }}
      >
        {(
          [
            ['none', t('team.permNone')],
            ['read', t('team.permRead')],
            ['write', t('team.permWrite')],
          ] as [PermLevel, string][]
        ).map(([v, l]) => {
          const sel = (F.perms || {})[mod] === v;
          return (
            <ButtonBase
              key={v}
              onClick={() => app.setRolePerm(mod, v)}
              sx={{
                p: '6px 11px',
                borderRadius: '7px',
                border: 'none',
                cursor: 'pointer',
                fontSize: '12px',
                fontWeight: 600,
                background: sel ? tk.primary : 'transparent',
                color: sel ? tk.onPrimary : '#6A6D76',
              }}
            >
              {l}
            </ButtonBase>
          );
        })}
      </Box>
    </Box>
  ));

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Field label={t('team.roleNameField')}>
        <TextInput name="name" placeholder={t('team.roleNamePlaceholder')} />
      </Field>
      <Box key="p">
        <Box key="l" sx={labelSx}>
          {t('team.permPerModule')}
        </Box>
        <Box key="r" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {rows}
        </Box>
      </Box>
      <PrimaryButton label={t('team.saveRole')} onClick={() => app.saveRole()} busy={app.state.busy === 'save'} />
    </Box>
  );
}
