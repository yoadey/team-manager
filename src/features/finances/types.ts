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
  penaltyId: string;
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
