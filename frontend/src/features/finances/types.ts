export interface Transaction {
  id: string;
  teamId: string;
  type: 'income' | 'expense';
  title: string;
  amount: number;
  date: string;
  category: string;
  /** The membership fee this transaction (fully or partially) pays, or null/absent if it isn't a fee payment. */
  contributionId?: string | null;
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
  note?: string;
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
  /** Free-text fee name, e.g. "Mitgliedsbeitrag Januar 2026". */
  label: string;
  amount: number;
  /** Optional due date (YYYY-MM-DD). */
  dueDate?: string | null;
  /** Sum of every income transaction linked to this contribution, in euros. */
  paidAmount: number;
  status: 'open' | 'partial' | 'paid';
  /** Member display name (mapped from the backend's memberName). */
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

// Editing buffer shapes for the finance sheets (amounts held as strings) live
// alongside their zod schema in components/*FormSchema.ts (e.g.
// TxFormValues in components/txFormSchema.ts), not here.
