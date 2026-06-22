import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import type { SheetProps } from '@/sheets/types';
import { buildTokens, NEUTRAL } from '@/styles/tokens';
import { Sym, PrimaryButton } from '@/components/ui';
import { t } from '@/i18n';

export function CalExportSheet({ app, sheet }: SheetProps) {
  const { state } = app;
  const tk = buildTokens(state.primaryColor);
  const team = app.activeTeam()!;
  const cnt = (state.events || []).filter((e) => e.status !== 'cancelled').length;
  const url = 'https://teamverwaltung.app/cal/' + ((team && team.id) || 'team') + '.ics';

  const hint = (icon: string, title: string, text: string) => (
    <Box key={title} sx={{ display: 'flex', gap: '12px', alignItems: 'flex-start', p: '12px 0' }}>
      <Sym name={icon} size={20} color={NEUTRAL.secondary} />
      <Box key="m" sx={{ flex: 1 }}>
        <Box key="t" sx={{ fontSize: '13px', fontWeight: 700, mb: '2px' }}>
          {title}
        </Box>
        <Box key="d" sx={{ fontSize: '12px', color: NEUTRAL.secondary, lineHeight: 1.5 }}>
          {text}
        </Box>
      </Box>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Box key="hero" sx={{ textAlign: 'center', p: '4px 2px 6px' }}>
        <Box
          key="i"
          sx={{
            width: '62px',
            height: '62px',
            borderRadius: '18px',
            background: tk.primaryContainer,
            color: tk.primary,
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontFamily: "'Material Symbols Outlined'",
            fontSize: '32px',
          }}
        >
          event_upcoming
        </Box>
        <Box key="s" sx={{ fontSize: '14px', color: NEUTRAL.secondary, mt: '12px', lineHeight: 1.5 }}>
          {t('events.calExportHero', { n: cnt })}
        </Box>
      </Box>

      <PrimaryButton label={t('events.calDownload')} onClick={() => app.downloadIcs()} />

      <Box
        key="sub"
        sx={{ border: `1px solid ${NEUTRAL.line}`, borderRadius: '16px', p: '14px', background: NEUTRAL.card }}
      >
        <Box
          key="h"
          sx={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', fontWeight: 700, mb: '4px' }}
        >
          <Sym name="sync" size={18} color={tk.primary} />
          {t('events.calSubscribe')}
        </Box>
        <Box key="d" sx={{ fontSize: '12px', color: NEUTRAL.secondary, lineHeight: 1.5, mb: '10px' }}>
          {t('events.calSubscribeDesc')}
        </Box>
        <Box
          key="box"
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            background: NEUTRAL.sidebar,
            border: `1px solid ${NEUTRAL.line3}`,
            borderRadius: '12px',
            p: '10px 12px',
          }}
        >
          <Box
            key="l"
            component="span"
            sx={{
              flex: 1,
              fontSize: '12px',
              fontFamily: 'monospace',
              color: NEUTRAL.onSurface,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {url}
          </Box>
          <ButtonBase
            key="c"
            onClick={() => app.copyCalUrl()}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              background: tk.primary,
              color: tk.onPrimary,
              border: 'none',
              borderRadius: '9px',
              p: '8px 12px',
              fontSize: '12px',
              fontWeight: 600,
              cursor: 'pointer',
              flex: '0 0 auto',
            }}
          >
            <Sym name="content_copy" size={15} color={tk.onPrimary} />
            {sheet.copied ? t('events.calCopied') : t('events.calCopy')}
          </ButtonBase>
        </Box>
      </Box>

      <Box key="hints" sx={{ borderTop: `1px solid ${NEUTRAL.line2}`, pt: '6px' }}>
        {hint('calendar_month', t('events.calGoogle'), t('events.calGoogleDesc'))}
        {hint('phone_iphone', t('events.calApple'), t('events.calAppleDesc'))}
        {hint('download', t('events.calOneTime'), t('events.calOneTimeDesc'))}
      </Box>

      <Box
        key="note"
        sx={{
          display: 'flex',
          gap: '10px',
          background: '#FFF7E6',
          border: '1px solid #F0DBA8',
          borderRadius: '13px',
          p: '12px 14px',
          fontSize: '12px',
          color: '#6B5413',
          lineHeight: 1.5,
        }}
      >
        <Sym name="info" size={18} color="#8A6100" />
        {t('events.calPrototypeNote')}
      </Box>
    </Box>
  );
}
