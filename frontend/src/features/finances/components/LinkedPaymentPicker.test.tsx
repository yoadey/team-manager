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
    paidAmount: 0,
    status: 'open',
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

  it('starts collapsed behind a toggle button', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[makeContribution()]}
        assignments={[]}
        contributionId={undefined}
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    expect(screen.getByText('Mit Beitrag oder Strafe verknüpfen (optional)')).toBeTruthy();
    expect(screen.queryByPlaceholderText('Mitglied oder Bezeichnung suchen…')).toBeNull();
  });

  it('expands to show both kind tabs with counts', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[makeContribution(), makeContribution({ id: 'c2' })]}
        assignments={[makeAssignment()]}
        contributionId={undefined}
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    fireEvent.click(screen.getByText('Mit Beitrag oder Strafe verknüpfen (optional)'));
    expect(screen.getByText('Beiträge (2)')).toBeTruthy();
    expect(screen.getByText('Strafen (1)')).toBeTruthy();
  });

  it('filters the contribution list by search text across member name and label', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[
          makeContribution({ id: 'c1', name: 'Anna Müller', label: 'Mitgliedsbeitrag Januar' }),
          makeContribution({ id: 'c2', name: 'Ben Schmidt', label: 'Turniergebühr' }),
        ]}
        assignments={[]}
        contributionId={undefined}
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    fireEvent.click(screen.getByText('Mit Beitrag oder Strafe verknüpfen (optional)'));
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.getByText('Ben Schmidt')).toBeTruthy();

    fireEvent.change(screen.getByPlaceholderText('Mitglied oder Bezeichnung suchen…'), {
      target: { value: 'Turnier' },
    });
    expect(screen.queryByText('Anna Müller')).toBeNull();
    expect(screen.getByText('Ben Schmidt')).toBeTruthy();
  });

  it('shows an empty state when the search matches nothing', () => {
    const handlers = makeHandlers();
    render(
      <LinkedPaymentPicker
        tk={tk}
        contributions={[makeContribution()]}
        assignments={[]}
        contributionId={undefined}
        penaltyAssignmentId={undefined}
        {...handlers}
      />,
    );
    fireEvent.click(screen.getByText('Mit Beitrag oder Strafe verknüpfen (optional)'));
    fireEvent.change(screen.getByPlaceholderText('Mitglied oder Bezeichnung suchen…'), {
      target: { value: 'nonexistent' },
    });
    expect(screen.getByText('Keine Treffer')).toBeTruthy();
  });

  it('calls onSelectContribution when a contribution row is clicked', () => {
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
    fireEvent.click(screen.getByText('Mit Beitrag oder Strafe verknüpfen (optional)'));
    fireEvent.click(screen.getByText('Anna Müller'));
    expect(handlers.onSelectContribution).toHaveBeenCalledWith('c1');
  });

  it('switches to the penalty tab and calls onSelectPenalty when a row is clicked', () => {
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
    fireEvent.click(screen.getByText('Mit Beitrag oder Strafe verknüpfen (optional)'));
    fireEvent.click(screen.getByText(/Strafen \(1\)/));
    fireEvent.click(screen.getByText('Ben Schmidt'));
    expect(handlers.onSelectPenalty).toHaveBeenCalledWith('a1');
  });

  it('shows a collapsed summary with member/label/amount once a contribution is selected', () => {
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

  it('reopens the picker when "Ändern" is clicked on the summary', () => {
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
    fireEvent.click(screen.getByText('Ändern'));
    expect(screen.getByPlaceholderText('Mitglied oder Bezeichnung suchen…')).toBeTruthy();
  });
});
