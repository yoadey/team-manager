import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import { useInvalidateTeamQuery } from '@/query/useInvalidateTeamQuery';
import type { Contribution, Penalty, PenaltyAssignment, Transaction } from '../types';

/** Invalidates the team's finance overview, returning a promise that resolves once the invalidated query has refetched. */
function useInvalidateFinances(teamId: string | null) {
  return useInvalidateTeamQuery(teamId, queryKeys.finances);
}

export interface SaveTxInput {
  mode: 'create' | 'edit';
  // Explicit `| undefined` -- `mode` (not the presence of `id`) drives the
  // create/update branch, so the caller passes `undefined` in create mode
  // rather than omitting the key.
  id?: string | undefined;
  payload: { type: 'income' | 'expense'; title: string; amount: number; category: string };
}

export function useSaveTxMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateFinances(teamId);
  return useMutation({
    mutationFn: ({ mode, id, payload }: SaveTxInput): Promise<Transaction> =>
      mode === 'edit'
        ? api.finances.updateTransaction(id!, payload, teamId!)
        : api.finances.addTransaction(teamId!, payload),
    onSuccess: () => invalidate(),
  });
}

// Takes the team id per call rather than the hook-bound active team id,
// mirroring useDeleteEventMutation/useRemoveMemberMutation -- the confirm
// sheet that triggers this can still be open (and get confirmed) after the
// user has switched to a different active team.
export function useDeleteTxMutation(api: typeof defaultApi) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, teamId }: { id: string; teamId: string }) => api.finances.deleteTransaction(id, teamId),
    onSuccess: (_data, { teamId }) => qc.invalidateQueries({ queryKey: queryKeys.finances(teamId) }),
  });
}

export interface SavePenaltyInput {
  mode: 'create' | 'edit';
  // Same reasoning as SaveTxInput.id above.
  id?: string | undefined;
  payload: { label: string; amount: number };
}

export function useSavePenaltyMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateFinances(teamId);
  return useMutation({
    mutationFn: ({ mode, id, payload }: SavePenaltyInput): Promise<Penalty> =>
      mode === 'create'
        ? api.finances.createPenalty(teamId!, payload)
        : api.finances.updatePenalty(id!, payload, teamId!),
    onSuccess: () => invalidate(),
  });
}

// Per-call team id -- same confirm-gated-dialog rationale as useDeleteTxMutation above.
export function useDeletePenaltyMutation(api: typeof defaultApi) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, teamId }: { id: string; teamId: string }) => api.finances.deletePenalty(id, teamId),
    onSuccess: (_data, { teamId }) => qc.invalidateQueries({ queryKey: queryKeys.finances(teamId) }),
  });
}

export function useSavePenaltyAssignMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateFinances(teamId);
  return useMutation({
    mutationFn: ({ userId, penaltyId }: { userId: string; penaltyId: string }): Promise<PenaltyAssignment> =>
      api.finances.assignPenalty(teamId!, { userId, penaltyId }),
    onSuccess: () => invalidate(),
  });
}

// Per-call team id -- same confirm-gated-dialog rationale as useDeleteTxMutation above.
export function useDeleteAssignmentMutation(api: typeof defaultApi) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, teamId }: { id: string; teamId: string }) => api.finances.deleteAssignment(id, teamId),
    onSuccess: (_data, { teamId }) => qc.invalidateQueries({ queryKey: queryKeys.finances(teamId) }),
  });
}

export function useTogglePenaltyMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateFinances(teamId);
  return useMutation({
    mutationFn: (id: string) => api.finances.togglePenaltyPaid(id, teamId!),
    onSuccess: () => invalidate(),
  });
}

export interface SaveContribInput {
  id: string;
  payload: { label: string; amount: number };
}

export function useSaveContribMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateFinances(teamId);
  return useMutation({
    mutationFn: ({ id, payload }: SaveContribInput): Promise<Contribution> =>
      api.finances.updateContribution(id, payload, teamId!),
    onSuccess: () => invalidate(),
  });
}

export function useToggleContributionMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateFinances(teamId);
  return useMutation({
    mutationFn: (id: string) => api.finances.toggleContribution(id, teamId!),
    onSuccess: () => invalidate(),
  });
}
