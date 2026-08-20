import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { buildTokens } from '@/styles/tokens';
import { LinkedPaymentPicker } from './LinkedPaymentPicker';
import type { Contribution, PenaltyAssignment } from '../types';

const tk = buildTokens('#4285F4');

function makeContribution(overrides: Partial<Contribution> = {}): Contribution {
  return {
    id: 'c1',
    teamId: 't1',
    userId: 'u1',
    label: 'Mitgliedsbeitrag Januar',
    amount: 25,
    amountCents: 2500,
    paidAmount: 0,
    paidAmountCents: 0,
    status: 'open',
    archived: false,
    name: 'Anna Müller',
    ...overrides,
  };
}

function makeAssignment(overrides: Partial<PenaltyAssignment> = {}): PenaltyAssignment {
  return {
    id: 'a1',
    teamId: 't1',
    userId: 'u1',
    penaltyId: 'p1',
    paid: false,
    paidAmount: 0,
    date: '2025-06-01',
    label: 'Zu spät zum Training',
    amount: 5,
    name: 'Ben Schmidt',
    ...overrides,
  };
}

function makeHandlers() {
  return {
    onSelectContribution: vi.fn(),
    onSelectPenalty: vi.fn(),
    onClear: vi.fn(),
  };
}

describe('LinkedPaymentPicker', () => {
  it('renders nothing when there are no open contributions or assignments', () => {
    const handlers = makeHandlers();
    const { container } = render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[]}
        assignments={[]}
        contributionId={undefined}
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    expect(container.firstChild).toBeNull();
  });

  it('shows the "Verknüpfen mit" heading with both buttons directly, no collapsed toggle', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[makeContribution()]}
        assignments={[makeAssignment()]}
        contributionId={undefined}
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    expect(screen.getByText('Verknüpfen mit')).toBeTruthy();
    expect(screen.getByText('Beiträge')).toBeTruthy();
    expect(screen.getByText('Strafen')).toBeTruthy();
    expect(screen.queryByPlaceholderText('Mitglied oder Beitrag suchen…')).toBeNull();
  });

  it('opens the fee matrix dialog directly from the "Beiträge" button, filterable by search text', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[
          makeContribution({ id: 'c1', name: 'Anna Müller', label: 'Mitgliedsbeitrag Januar' }),
          makeContribution({ id: 'c2', userId: 'u2', name: 'Ben Schmidt', label: 'Turniergebühr' }),
        ]}
        assignments={[]}
        contributionId={undefined}
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    fireEvent.click(screen.getByText('Beiträge'));
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.getByText('Ben Schmidt')).toBeTruthy();

    fireEvent.change(screen.getByPlaceholderText('Mitglied oder Beitrag suchen…'), {
      target: { value: 'Turnier' },
    });
    expect(screen.queryByText('Anna Müller')).toBeNull();
    expect(screen.getByText('Ben Schmidt')).toBeTruthy();
  });

  it('calls onSelectContribution when a matrix cell is clicked', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[makeContribution({ id: 'c1', name: 'Anna Müller' })]}
        assignments={[]}
        contributionId={undefined}
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    fireEvent.click(screen.getByText('Beiträge'));
    fireEvent.click(screen.getByRole('checkbox'));
    expect(handlers.onSelectContribution).toHaveBeenCalledWith('c1');
  });

  it('opens the penalty popup directly from the "Strafen" button and calls onSelectPenalty when a row is clicked', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[]}
        assignments={[makeAssignment({ id: 'a1', name: 'Ben Schmidt' })]}
        contributionId={undefined}
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    fireEvent.click(screen.getByText('Strafen'));
    fireEvent.click(screen.getByText('Ben Schmidt'));
    expect(handlers.onSelectPenalty).toHaveBeenCalledWith('a1');
  });

  it('shows a summary with member/label/amount once a contribution is selected', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[makeContribution({ id: 'c1', name: 'Anna Müller', label: 'Mitgliedsbeitrag Januar', amount: 25, paidAmount: 10 })]}
        assignments={[]}
        contributionId="c1"
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    expect(screen.getByText(/Anna Müller.*Mitgliedsbeitrag Januar/)).toBeTruthy();
    // Shows the outstanding amount (25 - 10 paid), not the full fee amount.
    expect(screen.getByText('15,00 €')).toBeTruthy();
    // No un-selected linking buttons once something is already linked.
    expect(screen.queryByText('Beiträge')).toBeNull();
    expect(screen.queryByText('Strafen')).toBeNull();
  });

  it('shows the outstanding balance, not the full amount, once a partially-paid penalty assignment is selected', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[]}
        assignments={[makeAssignment({ id: 'a1', name: 'Ben Schmidt', amount: 50, paidAmount: 30, paid: false })]}
        contributionId={undefined}
        penaltyAssignmentId="a1"
        {...handlers}
      />,
    );
    expect(screen.getByText('20,00 €')).toBeTruthy();
    expect(screen.queryByText('50,00 €')).toBeNull();
  });

  it('calls onClear when the remove button on the summary is clicked', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[makeContribution({ id: 'c1' })]}
        assignments={[]}
        contributionId="c1"
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    fireEvent.click(screen.getByLabelText('Verknüpfung entfernen'));
    expect(handlers.onClear).toHaveBeenCalled();
  });

  it('reopens the matrix dialog when "Ändern" is clicked on a fee summary', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[makeContribution({ id: 'c1', name: 'Anna Müller' })]}
        assignments={[]}
        contributionId="c1"
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    fireEvent.click(screen.getByText('Ändern'));
    expect(screen.getAllByText('Anna Müller').length).toBeGreaterThan(0);
    expect(screen.getByRole('checkbox')).toBeTruthy();
  });

  it('reopens the penalty popup when "Ändern" is clicked on a penalty summary', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[]}
        assignments={[makeAssignment({ id: 'a1', name: 'Ben Schmidt' })]}
        contributionId={undefined}
        penaltyAssignmentId="a1"
        {...handlers}
      />,
    );
    fireEvent.click(screen.getByText('Ändern'));
    expect(screen.getByRole('option')).toBeTruthy();
  });
});
