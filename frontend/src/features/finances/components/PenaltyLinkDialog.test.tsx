import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { buildTokens } from '@/styles/tokens';
import { PenaltyLinkDialog } from './PenaltyLinkDialog';
import type { PenaltyAssignment } from '../types';

const tk = buildTokens('#4285F4');

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

describe('PenaltyLinkDialog', () => {
  it('renders nothing visible when closed', () => {
    render(
      <PenaltyLinkDialog tk={tk} open={false} onClose={vi.fn()} assignments={[makeAssignment()]} selectedId={undefined} onSelect={vi.fn()} />,
    );
    expect(screen.queryByText('Ben Schmidt')).toBeNull();
  });

  it('lists open penalty assignments when open', () => {
    render(
      <PenaltyLinkDialog tk={tk} open={true} onClose={vi.fn()} assignments={[makeAssignment()]} selectedId={undefined} onSelect={vi.fn()} />,
    );
    expect(screen.getByText('Ben Schmidt')).toBeTruthy();
    expect(screen.getByText('Zu spät zum Training')).toBeTruthy();
  });

  it('shows an empty state when there are no assignments', () => {
    render(
      <PenaltyLinkDialog tk={tk} open={true} onClose={vi.fn()} assignments={[]} selectedId={undefined} onSelect={vi.fn()} />,
    );
    expect(screen.getByText('Keine offenen Strafen')).toBeTruthy();
  });

  it('filters by search text', () => {
    render(
      <PenaltyLinkDialog
        tk={tk}
        open={true}
        onClose={vi.fn()}
        assignments={[makeAssignment({ id: 'a1', name: 'Ben Schmidt' }), makeAssignment({ id: 'a2', name: 'Anna Müller' })]}
        selectedId={undefined}
        onSelect={vi.fn()}
      />,
    );
    fireEvent.change(screen.getByPlaceholderText('Mitglied oder Strafe suchen…'), { target: { value: 'Anna' } });
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.queryByText('Ben Schmidt')).toBeNull();
  });

  it('selecting an assignment calls onSelect with its id and closes', () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();
    render(
      <PenaltyLinkDialog tk={tk} open={true} onClose={onClose} assignments={[makeAssignment({ id: 'a1' })]} selectedId={undefined} onSelect={onSelect} />,
    );
    fireEvent.click(screen.getByText('Ben Schmidt'));
    expect(onSelect).toHaveBeenCalledWith('a1');
    expect(onClose).toHaveBeenCalled();
  });

  it('shows the still-owed amount, not the full amount, for a partially-paid assignment', () => {
    render(
      <PenaltyLinkDialog
        tk={tk}
        open={true}
        onClose={vi.fn()}
        assignments={[makeAssignment({ amount: 50, paidAmount: 30 })]}
        selectedId={undefined}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText('20,00 €')).toBeTruthy();
    expect(screen.queryByText('50,00 €')).toBeNull();
  });

  it('marks the currently selected assignment', () => {
    render(
      <PenaltyLinkDialog tk={tk} open={true} onClose={vi.fn()} assignments={[makeAssignment({ id: 'a1' })]} selectedId="a1" onSelect={vi.fn()} />,
    );
    expect(screen.getByRole('option')).toHaveAttribute('aria-selected', 'true');
  });

  it('closing the dialog calls onClose', () => {
    const onClose = vi.fn();
    render(
      <PenaltyLinkDialog tk={tk} open={true} onClose={onClose} assignments={[makeAssignment()]} selectedId={undefined} onSelect={vi.fn()} />,
    );
    fireEvent.click(screen.getByLabelText('Schließen'));
    expect(onClose).toHaveBeenCalled();
  });
});
