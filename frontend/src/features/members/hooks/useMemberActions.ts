import { useCallback, useRef } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { Member, MemberFormValues } from '../types';
import type { AppState } from '@/context/AppContext';
import { formValues, clearBusyIfOwned } from '@/utils/forms';
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
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useMemberActions({
  api,
  S,
  setState,
  refreshMembers,
  refreshTeams,
  askConfirm,
  toastMsg,
  logout,
}: MemberDeps) {
  // Guards against a STALE stats response overwriting a NEWER one for the
  // SAME membershipId -- e.g. rapid double-clicks on the same member row (no
  // busy/disabled state blocks a second click here), or mashing browser
  // back/forward between /members/A -> /members -> /members/A before the
  // first fetch for A resolves (the popstate handler calls openMemberDetail
  // again on each landing). The existing sheet.membershipId check already
  // handles the member-CHANGED case; it can't tell two in-flight fetches for
  // the SAME member apart, so if the network responds out of request order
  // the older, stale fetch would silently overwrite the newer stats. Mirrors
  // reloadDetailSeq's identical reasoning in useEventActions.ts.
  const openMemberDetailSeq = useRef(0);
  const openMemberDetail = useCallback(
    async (membershipId: string) => {
      const seq = ++openMemberDetailSeq.current;
      const m = (S().members ?? []).find((x) => x.membershipId === membershipId);
      setState({ sheet: { type: 'memberDetail', membershipId, member: m, stats: null } });
      // m can genuinely be missing (a stale bookmarked/back-forward URL for
      // a member who has since been removed) -- MemberDetailSheet renders a
      // graceful empty state for that case; there's no stats to load.
      if (!m) return;
      try {
        const stats = await api.stats.attendanceFor(S().activeTeamId!, m.userId);
        setState((s) =>
          s.sheet?.type === 'memberDetail' &&
          s.sheet.membershipId === membershipId &&
          openMemberDetailSeq.current === seq
            ? { sheet: { ...s.sheet, stats } }
            : {},
        );
      } catch (err) {
        if (openMemberDetailSeq.current !== seq) return;
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.load');
      }
    },
    [api, S, setState, toastMsg, logout],
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
        formErrors: {},
      }));
    },
    [setState],
  );

  const toggleFormRole = useCallback(
    (roleId: string) => {
      const cur = formValues<MemberFormValues>(S()).roleIds ?? [];
      const next = cur.includes(roleId) ? cur.filter((x) => x !== roleId) : cur.concat(roleId);
      if (!next.length) {
        toastMsg(t('team.roleAtLeastOne'), undefined, 'error');
        return;
      }
      setState((s) => ({ form: { ...s.form, roleIds: next } }));
    },
    [S, setState, toastMsg],
  );

  const saveMember = useCallback(async () => {
    const f = S().form as MemberFormValues;
    const nameResult = validateRequiredText(f.name, t('members.fieldNameError'));
    if (!nameResult.ok) {
      toastMsg(nameResult.message!, undefined, 'error');
      return;
    }
    const emailResult = validateEmail(f.email, t('validation.emailInvalid'));
    if (!emailResult.ok) {
      toastMsg(emailResult.message!, undefined, 'error');
      return;
    }
    const phoneResult = validatePhone(f.phone, t('validation.phoneInvalid'));
    if (!phoneResult.ok) {
      toastMsg(phoneResult.message!, undefined, 'error');
      return;
    }
    const birthdayResult = validateBirthday(f.birthday, t('validation.birthdayInvalid'));
    if (!birthdayResult.ok) {
      toastMsg(birthdayResult.message!, undefined, 'error');
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
    const teamId = S().activeTeamId!;
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
        teamId,
      );
      if (rolesChanged) {
        await api.members.setRoles(f.membershipId, nextRoleIds, teamId);
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
      clearBusyIfOwned(S, setState, 'save');
      // Only touch the sheet if the user is still on the team this save was
      // for -- otherwise closing/reopening it would clobber whatever sheet
      // they've since opened for the team they switched to, and
      // openMemberDetail would look up f.membershipId in the NEW team's
      // (already-refreshed) member list, finding nothing and rendering a
      // broken detail sheet. Also skip it if the user has since closed this
      // form and opened a different one (same team) while the save was in
      // flight -- otherwise a slow save for one member would silently close
      // and replace whatever the user is now looking at with this member's
      // detail view.
      if (S().activeTeamId === teamId && S().sheet === sh) {
        setState({ sheet: null });
        if (back && back.type === 'memberDetail') openMemberDetail(f.membershipId);
      }
      toastMsg(t('members.toastProfileSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout, S, busyOwner: 'save' }, err, 'error.save');
    }
  }, [api, S, setState, refreshMembers, refreshTeams, openMemberDetail, toastMsg, logout]);

  const removeMember = useCallback(
    (membershipId: string) => {
      const m = (S().members ?? []).find((x) => x.membershipId === membershipId);
      askConfirm({
        title: t('members.removeTitle'),
        message: t('members.removeMsg', { name: m ? m.name : '?' }),
        confirmLabel: t('members.removeConfirm'),
        danger: true,
        onConfirm: async () => {
          const sh = S().sheet;
          const teamId = S().activeTeamId!;
          try {
            await api.members.remove(membershipId, teamId);
            await refreshMembers();
            // Don't close a sheet the user has since opened for a different
            // team after switching away mid-request, or one they've since
            // opened for a different member (same team) while this delete
            // was in flight.
            if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: null });
            toastMsg(t('members.toastMemberRemoved'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      });
    },
    [api, S, askConfirm, refreshMembers, setState, toastMsg, logout],
  );

  return { openMemberDetail, openMemberForm, toggleFormRole, saveMember, removeMember };
}
