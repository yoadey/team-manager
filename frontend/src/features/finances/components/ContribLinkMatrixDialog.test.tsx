import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { buildTokens } from '@/styles/tokens';
import { ContribLinkMatrixDialog } from './ContribLinkMatrixDialog';
import type { Contribution } from '../types';

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

describe('ContribLinkMatrixDialog', () => {
  it('renders nothing visible when closed', () => {
    render(
      <ContribLinkMatrixDialog
        tk={tk}
        open={false}
        onClose={vi.fn()}
        contributions={[makeContribution()]}
        selectedId={undefined}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.queryByText('Anna Müller')).toBeNull();
  });

  it('renders a member x fee-group grid when open', () => {
    render(
      <ContribLinkMatrixDialog
        tk={tk}
        open={true}
        onClose={vi.fn()}
        contributions={[makeContribution()]}
        selectedId={undefined}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.getByText('Mitgliedsbeitrag Januar')).toBeTruthy();
  });

  it('selecting a cell calls onSelect with that contribution id and closes', () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();
    render(
      <ContribLinkMatrixDialog
        tk={tk}
        open={true}
        onClose={onClose}
        contributions={[makeContribution({ id: 'c1' })]}
        selectedId={undefined}
        onSelect={onSelect}
      />,
    );
    fireEvent.click(screen.getByRole('checkbox'));
    expect(onSelect).toHaveBeenCalledWith('c1');
    expect(onClose).toHaveBeenCalled();
  });

  it('marks the currently selected contribution as checked', () => {
    render(
      <ContribLinkMatrixDialog
        tk={tk}
        open={true}
        onClose={vi.fn()}
        contributions={[makeContribution({ id: 'c1' })]}
        selectedId="c1"
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByRole('checkbox')).toHaveAttribute('aria-checked', 'true');
  });

  it('shows the still-owed amount, not the full fee amount, for a partially-paid contribution', () => {
    render(
      <ContribLinkMatrixDialog
        tk={tk}
        open={true}
        onClose={vi.fn()}
        contributions={[makeContribution({ amount: 50, paidAmount: 30 })]}
        selectedId={undefined}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText('20,00 €')).toBeTruthy();
    expect(screen.queryByText('50,00 €')).toBeNull();
  });

  it('shows a dash for a member with no contribution in a given fee group', () => {
    const contribs = [
      makeContribution({ id: 'c1', userId: 'u1', name: 'Anna', label: 'Monatsbeitrag' }),
      makeContribution({ id: 'c2', userId: 'u2', name: 'Bob', label: 'Turniergebühr' }),
    ];
    render(
      <ContribLinkMatrixDialog tk={tk} open={true} onClose={vi.fn()} contributions={contribs} selectedId={undefined} onSelect={vi.fn()} />,
    );
    // Only 2 selectable cells exist (Anna x Monatsbeitrag, Bob x Turniergebühr) --
    // Anna x Turniergebühr and Bob x Monatsbeitrag render as non-interactive dashes.
    expect(screen.getAllByRole('checkbox')).toHaveLength(2);
  });

  it('renders without a member photo/avatar', () => {
    render(
      <ContribLinkMatrixDialog
        tk={tk}
        open={true}
        onClose={vi.fn()}
        contributions={[makeContribution({ name: 'Anna Müller' })]}
        selectedId={undefined}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    // Av's no-photo fallback renders the member's initials -- confirming
    // that isn't present rules out the avatar being rendered at all.
    expect(screen.queryByText('AM')).toBeNull();
  });

  it('closing the dialog calls onClose', () => {
    const onClose = vi.fn();
    render(
      <ContribLinkMatrixDialog
        tk={tk}
        open={true}
        onClose={onClose}
        contributions={[makeContribution()]}
        selectedId={undefined}
        onSelect={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByLabelText('Schließen'));
    expect(onClose).toHaveBeenCalled();
  });
});
