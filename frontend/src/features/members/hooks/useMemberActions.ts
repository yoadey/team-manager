import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { Member, MemberFormValues } from '../types';
import type { AppState } from '@/context/AppContext';
import { formValues } from '@/utils/forms';
import { reportActionError } from '@/utils/errors';
import { validateEmail, validatePhone, validateBirthday, validateRequiredText } from '@/utils/validation';
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
      const m = (S().members ?? []).find((x) => x.membershipId === membershipId);
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
    const nameResult = validateRequiredText(f.name, t('members.fieldNameError'));
    if (!nameResult.ok) {
      toastMsg(nameResult.message!);
      return;
    }
    const emailResult = validateEmail(f.email, t('validation.emailInvalid'));
    if (!emailResult.ok) {
      toastMsg(emailResult.message!);
      return;
    }
    const phoneResult = validatePhone(f.phone, t('validation.phoneInvalid'));
    if (!phoneResult.ok) {
      toastMsg(phoneResult.message!);
      return;
    }
    const birthdayResult = validateBirthday(f.birthday, t('validation.birthdayInvalid'));
    if (!birthdayResult.ok) {
      toastMsg(birthdayResult.message!);
      return;
    }
    const sh = S().sheet!;
    const back = sh.back;
    const self = sh.self;
    // Role assignment is a separate write path (members.setRoles ->
    // PUT .../roles, gated on settings:write) from the profile-field patch
    // (members.update -> PATCH .../{membershipId}, gated on members:write) —
    // the backend's UpdateMember handler never applies a roleIds field
    // embedded in the PATCH body, so it must be sent via setRoles() whenever
    // it actually changed, not folded into the profile update.
    const original = (S().members ?? []).find((x) => x.membershipId === f.membershipId);
    const originalRoleIds = original ? original.roles.map((r) => r.id) : [];
    const nextRoleIds = f.roleIds ?? [];
    const rolesChanged =
      originalRoleIds.length !== nextRoleIds.length ||
      [...originalRoleIds].sort().some((id, i) => id !== [...nextRoleIds].sort()[i]);
    // Photo has its own dedicated endpoint (auth.setPhoto, self-only — there
    // is no backend endpoint to set another member's photo at all), not a
    // members.update() field; the sheet only lets you change your own photo
    // (MemberSheets.tsx hides the control when editing someone else), so this
    // only ever fires for self.
    const photoChanged = self && !!f.photo && f.photo !== original?.photo;
    setState({ busy: 'save' });
    try {
      await api.members.update(
        f.membershipId,
        {
          name: nameResult.value!,
          email: f.email,
          phone: f.phone,
          birthday: f.birthday,
          address: f.address,
          group: f.group,
        },
        S().activeTeamId!,
      );
      if (rolesChanged) {
        await api.members.setRoles(f.membershipId, nextRoleIds, S().activeTeamId!);
      }
      if (photoChanged) {
        await api.auth.setPhoto(f.photo!);
      }
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
      const m = (S().members ?? []).find((x) => x.membershipId === membershipId);
      askConfirm({
        title: t('members.removeTitle'),
        message: t('members.removeMsg', { name: m ? m.name : '?' }),
        confirmLabel: t('members.removeConfirm'),
        danger: true,
        onConfirm: async () => {
          try {
            await api.members.remove(membershipId, S().activeTeamId!);
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
