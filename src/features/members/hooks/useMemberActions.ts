import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { Member } from '../types';
import type { AppState } from '@/context/AppContext';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type MemberDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  refreshMembers: () => Promise<void>;
  refreshTeams: () => Promise<void>;
  askConfirm: (cfg: { title: string; message: string; confirmLabel?: string; danger?: boolean; onConfirm: () => void | Promise<void> }) => void;
  toastMsg: (m: string) => void;
};

export function useMemberActions({ api, S, setState, refreshMembers, refreshTeams, askConfirm, toastMsg }: MemberDeps) {
  const openMemberDetail = useCallback(async (membershipId: string) => {
    const m = S().members.find((x) => x.membershipId === membershipId);
    setState({ sheet: { type: 'memberDetail', membershipId, member: m, stats: null } });
    const stats = await api.stats.attendanceFor(S().activeTeamId!, m!.userId);
    setState((s) => (s.sheet && s.sheet.type === 'memberDetail') ? { sheet: { ...s.sheet, stats } } : {});
  }, [api, S, setState]);

  const openMemberForm = useCallback((member: Member) => {
    const f = { membershipId: member.membershipId, name: member.name, email: member.email, phone: member.phone, birthday: member.birthday || '', address: member.address || '', roleIds: member.roles.map((r) => r.id), group: member.group, photo: member.photo };
    setState((st) => ({ sheet: { type: 'memberForm', mode: 'edit', self: member.userId === st.user!.id, back: (st.sheet && st.sheet.type === 'memberDetail') ? st.sheet : null }, form: f }));
  }, [setState]);

  const toggleFormRole = useCallback((roleId: string) => setState((s) => {
    const cur = s.form.roleIds || [];
    const next = cur.includes(roleId) ? cur.filter((x: string) => x !== roleId) : cur.concat(roleId);
    return { form: { ...s.form, roleIds: next.length ? next : cur } };
  }), [setState]);

  const saveMember = useCallback(async () => {
    const f = S().form;
    if (!f.name) { toastMsg('Bitte einen Namen angeben'); return; }
    const sh = S().sheet!;
    const back = sh.back; const self = sh.self;
    setState({ busy: 'save' });
    await api.members.update(f.membershipId, { name: f.name, email: f.email, phone: f.phone, birthday: f.birthday, address: f.address, roleIds: f.roleIds, group: f.group, photo: f.photo });
    await refreshMembers();
    if (self) { const u = await api.auth.currentUser(); await refreshTeams(); setState({ user: u }); }
    setState({ busy: null, sheet: null });
    if (back && back.type === 'memberDetail') openMemberDetail(f.membershipId);
    toastMsg('Profil gespeichert');
  }, [api, S, setState, refreshMembers, refreshTeams, openMemberDetail, toastMsg]);

  const removeMember = useCallback((membershipId: string) => {
    const m = S().members.find((x) => x.membershipId === membershipId);
    askConfirm({
      title: 'Mitglied entfernen?',
      message: '„' + (m ? m.name : 'Das Mitglied') + '" wird aus dem Team entfernt und verliert den Zugriff. Diese Aktion kann nicht rückgängig gemacht werden.',
      confirmLabel: 'Entfernen', danger: true,
      onConfirm: async () => { await api.members.remove(membershipId); await refreshMembers(); setState({ sheet: null }); toastMsg('Mitglied entfernt'); },
    });
  }, [api, S, askConfirm, refreshMembers, setState, toastMsg]);

  return { openMemberDetail, openMemberForm, toggleFormRole, saveMember, removeMember };
}
