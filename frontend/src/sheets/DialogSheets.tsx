import { useState } from 'react';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import type { SheetProps } from './types';
import { buildTokens, statusMeta, NEUTRAL } from '@/styles/tokens';
import { Sym, Chip, PrimaryButton, inputSx } from '@/components/ui';
import { t } from '@/i18n';

export function ConfirmSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tok = buildTokens(state.primaryColor);
  const c = sheet.cfg || {};
  const danger = !!c.danger;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
      <Box
        key="ic"
        sx={{
          width: '56px',
          height: '56px',
          borderRadius: '16px',
          background: danger ? NEUTRAL.errorBg : tok.primaryContainer,
          color: danger ? NEUTRAL.error : tok.primary,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontFamily: "'Material Symbols Outlined'",
          fontSize: '30px',
        }}
      >
        {danger ? 'warning' : 'help'}
      </Box>
      <Box key="m" sx={{ fontSize: '14px', color: NEUTRAL.onSurfaceVariant, lineHeight: 1.55 }}>
        {c.message || t('common.areYouSure')}
      </Box>
      <Box key="b" sx={{ display: 'flex', gap: '10px' }}>
        <ButtonBase
          key="x"
          onClick={() => app.cancelConfirm()}
          sx={{
            flex: 1,
            p: '13px',
            borderRadius: '13px',
            border: `1px solid ${NEUTRAL.inputBorder}`,
            background: NEUTRAL.card,
            color: NEUTRAL.onSurfaceVariant,
            fontWeight: 600,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          {t('common.cancel')}
        </ButtonBase>
        <ButtonBase
          key="ok"
          onClick={() => app.runConfirm()}
          sx={{
            flex: 1,
            p: '13px',
            borderRadius: '13px',
            border: 'none',
            background: danger ? NEUTRAL.error : tok.primary,
            color: NEUTRAL.card,
            fontWeight: 700,
            fontSize: '14px',
            cursor: 'pointer',
          }}
        >
          {c.confirmLabel || t('common.confirm')}
        </ButtonBase>
      </Box>
    </Box>
  );
}

export function SeriesActionSheet({ app, sheet }: SheetProps) {
  const act = sheet.action!;
  // 'cancel' is also the fallback for an unrecognized action, so it's bound
  // to its own real object literal rather than looked up by indexing (which
  // would be possibly-undefined under noUncheckedIndexedAccess even for a
  // key -- 'cancel' -- that the map literal below always defines).
  const cancelCfg = { d: t('events.seriesCancelDesc'), ic: 'event_busy', col: NEUTRAL.warn };
  const cfg: Record<string, { d: string; ic: string; col: string }> = {
    cancel: cancelCfg,
    delete: { d: t('events.seriesDeleteDesc'), ic: 'delete', col: NEUTRAL.error },
    reactivate: { d: t('events.seriesReactivateDesc'), ic: 'event_available', col: NEUTRAL.success },
  };
  const L = cfg[act] || cancelCfg;

  const opt = (scope: 'single' | 'series', title: string, sub: string, icon: string) => (
    <ButtonBase
      key={scope}
      onClick={() => app.runEventAction(sheet.action!, sheet.event!, scope)}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: '13px',
        width: '100%',
        p: '15px',
        borderRadius: '15px',
        cursor: 'pointer',
        border: `1px solid ${NEUTRAL.line3}`,
        background: NEUTRAL.card,
        textAlign: 'left',
        justifyContent: 'flex-start',
      }}
    >
      <Box
        component="span"
        key="i"
        sx={{
          width: '40px',
          height: '40px',
          borderRadius: '11px',
          background: L.col + '1A',
          color: L.col,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontFamily: "'Material Symbols Outlined'",
          fontSize: '21px',
          flex: '0 0 auto',
        }}
      >
        {icon}
      </Box>
      <Box key="m" sx={{ flex: 1, minWidth: 0 }}>
        <Box key="t" sx={{ fontSize: '15px', fontWeight: 700 }}>
          {title}
        </Box>
        <Box key="s" sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '2px' }}>
          {sub}
        </Box>
      </Box>
      <Sym name="chevron_right" size={20} color={NEUTRAL.faint} />
    </ButtonBase>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '11px' }}>
      <Box key="h" sx={{ display: 'flex', gap: '11px', alignItems: 'flex-start', mb: '2px' }}>
        <Sym name={L.ic} size={24} color={L.col} />
        <Box key="d" sx={{ flex: 1, fontSize: '13px', color: NEUTRAL.onSurfaceVariant, lineHeight: 1.5 }}>
          {L.d}
        </Box>
      </Box>
      {opt('single', t('events.seriesScopeSingle'), t('events.seriesScopeSingleSub'), 'event')}
      {opt('series', t('events.seriesScopeSeries'), t('events.seriesScopeSeriesSub'), 'repeat')}
    </Box>
  );
}

export function CommentSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const [text, setText] = useState((sheet.formInitial as string) || '');
  const sm = statusMeta(sheet.status!);
  const isMe = sheet.userId === state.user!.id;
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
      <Box
        key="who"
        sx={{ display: 'flex', alignItems: 'center', gap: '9px', fontSize: '13px', color: NEUTRAL.secondary }}
      >
        <Chip key="c" label={sm.label} color={sm.color} bg={sm.bg} icon={sm.icon} />
        <Box component="span" key="n">
          {isMe ? t('attendance.yourComment') : t('attendance.commentFor', { name: sheet.name ?? '' })}
        </Box>
      </Box>
      <textarea
        key="t"
        name="commentText"
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder={t('attendance.commentPlaceholder')}
        maxLength={500}
        style={{ ...inputSx, minHeight: '100px', resize: 'vertical' }}
      />
      {sheet.status === 'no' ? (
        <Box key="h" sx={{ display: 'flex', gap: '8px', fontSize: '12px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>
          <Sym name="visibility" size={16} color={NEUTRAL.faint} />
          {t('attendance.cancelCommentsHint')}
        </Box>
      ) : null}
      <PrimaryButton
        label={t('attendance.saveComment')}
        onClick={() => app.submitComment(text)}
        busy={state.savingComment}
      />
    </Box>
  );
}
