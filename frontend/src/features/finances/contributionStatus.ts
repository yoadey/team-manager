/**
 * Presentation-only refinement of a contribution's paid/partial/open status:
 * "overpaid" isn't a separate wire-level status (paidAmount >= amount is
 * still `paid` at the API layer, see backend Contribution.status), but the
 * list row, the matrix cell, and the linking-picker cell all need to show
 * the same distinct treatment for it, so the derivation lives here once.
 */
export type ContributionAmountStatus = 'open' | 'partial' | 'paid' | 'overpaid';

export interface ContributionAmountInfo {
  status: ContributionAmountStatus;
  /** The amount to actually display -- the real paidAmount, never capped at `amount`. */
  displayAmount: number;
  /** paidAmount - amount when positive (overpaid), otherwise 0. */
  excess: number;
}

export function contributionAmountStatus(amount: number, paidAmount: number): ContributionAmountInfo {
  // Mirrors finances.contributionStatus (backend) bucket-for-bucket --
  // paidAmount <= 0 is open, paidAmount >= amount is (over)paid, otherwise
  // partial -- with "overpaid" layered on top as the paidAmount > amount
  // slice of the paid bucket.
  if (paidAmount <= 0) return { status: 'open', displayAmount: paidAmount, excess: 0 };
  if (paidAmount > amount) return { status: 'overpaid', displayAmount: paidAmount, excess: paidAmount - amount };
  if (paidAmount >= amount) return { status: 'paid', displayAmount: paidAmount, excess: 0 };
  return { status: 'partial', displayAmount: paidAmount, excess: 0 };
}
