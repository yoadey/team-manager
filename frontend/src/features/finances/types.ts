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
  /** The penalty assignment (fine) this transaction (fully or partially) pays, or null/absent if it isn't a fine payment. Mutually exclusive with contributionId. */
  penaltyAssignmentId?: string | null;
  /** Optional free-text note, settable on create/update. Never shown in the transaction list -- only in the edit form. */
  note?: string | null;
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
  /** Derived from paidAmount vs. amount (paidAmount >= amount) -- never independently settable, see Transaction.penaltyAssignmentId. */
  paid: boolean;
  /** Sum of every income transaction linked to this assignment, in euros. */
  paidAmount: number;
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
  /** Optional free-text description beyond the short name. */
  description?: string | null;
  amount: number;
  /** Same value as `amount`, in integer cents pre-conversion (see api/map.ts's centsToEuros). Sum groups of rows via this field, not `amount`, and convert to euros once -- summing already-converted euro floats accumulates rounding error. */
  amountCents: number;
  /** Optional due date (YYYY-MM-DD). */
  dueDate?: string | null;
  /** Sum of every income transaction linked to this contribution, in euros. May exceed `amount` when overpaid -- not capped. */
  paidAmount: number;
  /** Same value as `paidAmount`, in integer cents pre-conversion -- see `amountCents`. */
  paidAmountCents: number;
  status: 'open' | 'partial' | 'paid';
  /** When true, excluded from the default display, the matrix, and the linking picker -- never affects linked transactions. */
  archived: boolean;
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
