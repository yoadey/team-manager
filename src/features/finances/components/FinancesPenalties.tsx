import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import { useApp } from '@/context/AppContext';
import { buildTokens, fmtDate, fmtMoney, NEUTRAL } from '@/styles/tokens';
import { Av, EmptyState, SectionTitle, Sym } from '@/components/ui';
import type { FinanceOverview } from '../types';
import { t } from '@/i18n';

type App = ReturnType<typeof useApp>;
type Tk = ReturnType<typeof buildTokens>;

interface Props {
  app: App;
  t: Tk;
  f: FinanceOverview;
  canFin: boolean;
}

export function FinancesPenalties({ app, t: tk, f, canFin }: Props) {
  const header = (
    <Box key="hd" sx={{ display: 'flex', alignItems: 'center', gap: '10px', mb: '16px', flexWrap: 'wrap' }}>
      <Box sx={{ flex: 1, minWidth: '120px' }}>
        <Box sx={{ fontSize: '22px', fontWeight: 800, color: NEUTRAL.error }}>{fmtMoney(f.openPenaltySum)}</Box>
        <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '2px' }}>{t('finances.penaltyOpenSum')}</Box>
      </Box>
      <ButtonBase
        onClick={() => app.openPenaltyCatalog()}
        sx={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: '7px',
          p: '10px 14px',
          borderRadius: '12px',
          border: '1px solid #D0D2DA',
          background: '#fff',
          cursor: 'pointer',
          fontSize: '13px',
          fontWeight: 600,
          color: NEUTRAL.onSurfaceVariant,
        }}
      >
        <Sym name="menu_book" size={18} color={NEUTRAL.secondary} />
        {t('finances.penaltyCatalogBtn')}
      </ButtonBase>
      {canFin ? (
        <ButtonBase
          onClick={() => app.openPenaltyAssign()}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            border: 'none',
            background: tk.primary,
            color: tk.onPrimary,
            borderRadius: '12px',
            p: '10px 14px',
            fontSize: '13px',
            fontWeight: 700,
            cursor: 'pointer',
          }}
        >
          <Sym name="gavel" size={17} color={tk.onPrimary} />
          {t('finances.penaltyAssignBtn')}
        </ButtonBase>
      ) : null}
    </Box>
  );
  const list = (
    <Box key="op" sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      {f.assignments.length ? (
        f.assignments
          .slice()
          .reverse()
          .map((a) => (
            <Box
              key={a.id}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: '11px',
                background: '#fff',
                border: `1px solid ${NEUTRAL.line}`,
                borderRadius: '14px',
                p: '10px 13px',
              }}
            >
              <Av name={a.name} photo={a.photo} color={a.avatarColor} size={36} />
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Box sx={{ fontSize: '14px', fontWeight: 600 }}>{a.name}</Box>
                <Box sx={{ fontSize: '12px', color: NEUTRAL.faint }}>{a.label + ' · ' + fmtDate(a.date)}</Box>
              </Box>
              <Box
                component="span"
                sx={{
                  fontSize: '14px',
                  fontWeight: 700,
                  color: a.paid ? NEUTRAL.faint : NEUTRAL.error,
                  textDecoration: a.paid ? 'line-through' : 'none',
                }}
              >
                {fmtMoney(a.amount || 0)}
              </Box>
              {canFin ? (
                <ButtonBase
                  onClick={() => app.togglePenalty(a.id)}
                  sx={{
                    border: 'none',
                    cursor: 'pointer',
                    borderRadius: '9px',
                    p: '7px 11px',
                    fontSize: '12px',
                    fontWeight: 600,
                    background: a.paid ? '#ECEDF3' : NEUTRAL.successBg,
                    color: a.paid ? NEUTRAL.secondary : '#235C26',
                  }}
                >
                  {a.paid ? t('finances.penaltyOpen') : t('finances.penaltyPaid')}
                </ButtonBase>
              ) : null}
              {canFin ? (
                <ButtonBase
                  onClick={() => app.deleteAssignment(a.id)}
                  aria-label={t('common.delete')}
                  sx={{
                    width: '30px',
                    height: '30px',
                    borderRadius: '8px',
                    border: 'none',
                    cursor: 'pointer',
                    background: '#FFF4F3',
                    color: NEUTRAL.error,
                    flex: '0 0 auto',
                  }}
                >
                  <Sym name="delete" size={18} color={NEUTRAL.error} />
                </ButtonBase>
              ) : null}
            </Box>
          ))
      ) : (
        <EmptyState icon="savings" text={t('finances.penaltyEmpty')} />
      )}
    </Box>
  );
  return (
    <Box key="str">
      {header}
      <SectionTitle>{t('finances.penaltiesTitle')}</SectionTitle>
      {list}
    </Box>
  );
}
