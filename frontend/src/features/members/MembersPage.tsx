import { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Av, Chip, EmptyState, Sym, inputSx, SkeletonList } from '@/components/ui';
import { t } from '@/i18n';
import { useMembersQuery } from './hooks/useMemberQueries';

export function MembersPage() {
  const app = useApp();
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const [search, setSearch] = useState('');
  const { data: members } = useMembersQuery(app.api, state.activeTeamId);

  if (!members) return <SkeletonList rows={6} rowHeight={64} />;

  const query = search.trim().toLowerCase();
  const list = query
    ? members.filter(
        (m) => m.name.toLowerCase().includes(query) || m.roles.some((r) => r.name.toLowerCase().includes(query)),
      )
    : members;

  const rows = list.map((m) => {
    const isMe = m.userId === state.user!.id;
    const extra = m.roles.length - 1;
    return (
      <ButtonBase
        key={m.membershipId}
        data-testid="member-row"
        onClick={() => app.openMemberDetail(m.membershipId)}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '13px',
          width: '100%',
          textAlign: 'left',
          justifyContent: 'flex-start',
          background: NEUTRAL.card,
          border: `1px solid ${NEUTRAL.line}`,
          borderRadius: '16px',
          p: '12px 14px',
        }}
      >
        <Av name={m.name} photo={m.photo} color={m.avatarColor} size={44} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: '7px' }}>
            <Box
              component="span"
              sx={{
                fontSize: '15px',
                fontWeight: 600,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {m.name}
            </Box>
            {isMe ? <Chip label={t('members.meLabel')} color={tk.primary} bg={tk.primaryContainer} /> : null}
          </Box>
          <Box
            sx={{
              fontSize: '12px',
              color: NEUTRAL.secondary,
              mt: '2px',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {m.roles.map((r) => r.name).join(' · ')}
          </Box>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
          {m.primaryRole ? (
            <Chip
              label={m.primaryRole.name}
              color={m.primaryRole.color}
              bg={m.primaryRole.color + '1A'}
              icon="circle"
              fs={11}
            />
          ) : null}
          {extra > 0 ? (
            <Box
              component="span"
              sx={{
                fontSize: '11px',
                fontWeight: 700,
                color: NEUTRAL.secondary,
                background: NEUTRAL.line2,
                p: '4px 8px',
                borderRadius: '999px',
              }}
            >
              {'+' + extra}
            </Box>
          ) : null}
        </Box>
        <Sym name="chevron_right" size={22} color={NEUTRAL.faint} />
      </ButtonBase>
    );
  });

  return (
    <Box sx={{ maxWidth: '820px' }}>
      <Box sx={{ display: 'flex', gap: '8px', mb: '16px', flexWrap: 'wrap', alignItems: 'center' }}>
        <Box sx={{ fontSize: '13px', fontWeight: 600, color: NEUTRAL.secondary }}>
          {t('members.count', { n: list.length, count: list.length })}
        </Box>
        <Box sx={{ flex: 1 }} />
        <input
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t('members.searchPlaceholder')}
          aria-label={t('members.searchPlaceholder')}
          style={{ ...inputSx, width: '200px', padding: '8px 14px' }}
        />
        <ButtonBase
          onClick={() => app.openRoles()}
          sx={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '7px',
            p: '8px 14px',
            borderRadius: '999px',
            border: `1px solid ${NEUTRAL.inputBorder}`,
            background: NEUTRAL.card,
            fontSize: '13px',
            fontWeight: 600,
            color: NEUTRAL.onSurfaceVariant,
          }}
        >
          <Sym name="admin_panel_settings" size={18} color={NEUTRAL.onSurfaceVariant} />
          {t('members.rolesAndRights')}
        </ButtonBase>
      </Box>
      {list.length ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '9px' }}>{rows}</Box>
      ) : (
        <EmptyState icon="search_off" text={t('members.searchEmpty')} />
      )}
    </Box>
  );
}
