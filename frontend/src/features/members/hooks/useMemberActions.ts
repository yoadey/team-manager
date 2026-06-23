import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { Member, MemberFormValues } from '../types';
import type { AppState } from '@/context/AppContext';
import { formValues } from '@/utils/forms';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type MemberDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  refreshMembers: () => Promise<void>;
  refreshTeams: () => Promise<void>;
  askConfirm: (cfg: {
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void | Promise<void>;
  }) => void;
  toastMsg: (m: string) => void;
};

export function useMemberActions({ api, S, setState, refreshMembers, refreshTeams, askConfirm, toastMsg }: MemberDeps) {
  const openMemberDetail = useCallback(
    async (membershipId: string) => {
      const m = S().members.find((x) => x.membershipId === membershipId);
      setState({ sheet: { type: 'memberDetail', membershipId, member: m, stats: null } });
      try {
        const stats = await api.stats.attendanceFor(S().activeTeamId!, m!.userId);
        setState((s) =>
          s.sheet?.type === 'memberDetail' && s.sheet.membershipId === membershipId
            ? { sheet: { ...s.sheet, stats } }
            : {},
        );
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.load');
      }
    },
    [api, S, setState, toastMsg],
  );

  const openMemberForm = useCallback(
    (member: Member) => {
      const f: MemberFormValues = {
        membershipId: member.membershipId,
        name: member.name,
        email: member.email,
        phone: member.phone,
        birthday: member.birthday || '',
        address: member.address || '',
        roleIds: member.roles.map((r) => r.id),
        group: member.group,
        photo: member.photo,
      };
      setState((st) => ({
        sheet: {
          type: 'memberForm',
          mode: 'edit',
          self: member.userId === st.user!.id,
          back: st.sheet && st.sheet.type === 'memberDetail' ? st.sheet : null,
        },
        form: f,
      }));
    },
    [setState],
  );

  const toggleFormRole = useCallback(
    (roleId: string) =>
      setState((s) => {
        const cur = formValues<MemberFormValues>(s).roleIds ?? [];
        const next = cur.includes(roleId) ? cur.filter((x) => x !== roleId) : cur.concat(roleId);
        return { form: { ...s.form, roleIds: next.length ? next : cur } };
      }),
    [setState],
  );

  const saveMember = useCallback(async () => {
    const f = S().form as MemberFormValues;
    if (!f.name) {
      toastMsg(t('members.fieldNameError'));
      return;
    }
    const sh = S().sheet!;
    const back = sh.back;
    const self = sh.self;
    setState({ busy: 'save' });
    try {
      await api.members.update(f.membershipId, {
        name: f.name,
        email: f.email,
        phone: f.phone,
        birthday: f.birthday,
        address: f.address,
        roleIds: f.roleIds,
        group: f.group,
        photo: f.photo,
      });
      await refreshMembers();
      if (self) {
        const u = await api.auth.currentUser();
        await refreshTeams();
        setState({ user: u });
      }
      setState({ busy: null, sheet: null });
      if (back && back.type === 'memberDetail') openMemberDetail(f.membershipId);
      toastMsg(t('members.toastProfileSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, refreshMembers, refreshTeams, openMemberDetail, toastMsg]);

  const removeMember = useCallback(
    (membershipId: string) => {
      const m = S().members.find((x) => x.membershipId === membershipId);
      askConfirm({
        title: t('members.removeTitle'),
        message: t('members.removeMsg', { name: m ? m.name : '?' }),
        confirmLabel: t('members.removeConfirm'),
        danger: true,
        onConfirm: async () => {
          try {
            await api.members.remove(membershipId);
            await refreshMembers();
            setState({ sheet: null });
            toastMsg(t('members.toastMemberRemoved'));
          } catch (err) {
            reportActionError({ setState, toastMsg }, err, 'error.delete');
          }
        },
      });
    },
    [api, S, askConfirm, refreshMembers, setState, toastMsg],
  );

  return { openMemberDetail, openMemberForm, toggleFormRole, saveMember, removeMember };
}
