export type { Transaction, Penalty, PenaltyAssignment, OpenPenalty, Contribution, FinanceOverview } from './types';

// Sheet components + financeSheetMap declared first so the map is initialised
// before the heavier page/hook re-exports below can trigger a circular re-entry
// (mirrors the events barrel; avoids a dev-mode TDZ on `financeSheetMap`).
import { TxFormSheet } from './components/TxFormSheet';
import { PenaltyCatalogSheet } from './components/PenaltyCatalogSheet';
import { PenaltyFormSheet } from './components/PenaltyFormSheet';
import { PenaltyAssignSheet } from './components/PenaltyAssignSheet';
import { ContribFormSheet } from './components/ContribFormSheet';

export const financeSheetMap = {
  txForm: TxFormSheet,
  penaltyCatalog: PenaltyCatalogSheet,
  penaltyForm: PenaltyFormSheet,
  penaltyAssign: PenaltyAssignSheet,
  contribForm: ContribFormSheet,
} as const;

export { TxFormSheet, PenaltyCatalogSheet, PenaltyFormSheet, PenaltyAssignSheet, ContribFormSheet };

export { FinancesPage } from './FinancesPage';
export { FinancesTransactions } from './components/FinancesTransactions';
export { FinancesPenalties } from './components/FinancesPenalties';
export { FinancesContributions } from './components/FinancesContributions';
export { useFinanceActions } from './hooks/useFinanceActions';
