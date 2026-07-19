export interface Transaction {
  id: string;
  teamId: string;
  type: 'income' | 'expense';
  title: string;
  amount: number;
  date: string;
  category: string;
}

export interface Penalty {
  id: string;
  teamId: string;
  label: string;
  amount: number;
}

export interface PenaltyAssignment {
  id: string;
  teamId: string;
  userId: string;
  /** null when the source penalty catalog entry was deleted; the snapshot
   *  label/amount below remain the authoritative record of the assignment. */
  penaltyId: string | null;
  paid: boolean;
  date: string;
  name?: string;
  avatarColor?: string;
  photo?: string | null;
  label?: string;
  amount?: number;
}

export interface OpenPenalty {
  userId: string;
  name: string;
  avatarColor: string;
  photo: string | null;
  amount: number;
}

export interface Contribution {
  id: string;
  teamId: string;
  userId: string;
  month: string;
  label: string;
  amount: number;
  status: 'paid' | 'open';
  name?: string;
  avatarColor?: string;
  photo?: string | null;
}

/** UI ViewModel assembled from several finance DTO collections. */
export interface FinanceOverview {
  balance: number;
  income: number;
  expense: number;
  transactions: Transaction[];
  penalties: Penalty[];
  assignments: PenaltyAssignment[];
  openPenalties: OpenPenalty[];
  openPenaltySum: number;
  contributions: Contribution[];
  contribOpen: number;
}

// --- Editing buffer shapes for the finance sheets (amounts held as strings) ---

/** Transaction create/edit sheet. */
export interface TxFormValues extends Record<string, unknown> {
  id?: string;
  type: 'income' | 'expense';
  title: string;
  amount: string;
  category: string;
}

/** Penalty-catalog create/edit sheet. */
export interface PenaltyFormValues extends Record<string, unknown> {
  id?: string;
  label: string;
  amount: string;
}

/** Assign-a-penalty-to-a-member sheet. */
export interface PenaltyAssignFormValues extends Record<string, unknown> {
  userId: string;
  penaltyId: string | null;
}

/** Monthly-contribution edit sheet. */
export interface ContribFormValues extends Record<string, unknown> {
  id: string;
  label: string;
  amount: string;
}
