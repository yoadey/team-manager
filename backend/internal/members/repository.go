package members

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yoadey/team-manager/backend/internal/teams"
)

// pgUniqueViolation is the Postgres SQLSTATE for a violated UNIQUE constraint.
const pgUniqueViolation = "23505"

// querier is satisfied by both *pgxpool.Pool and pgx.Tx, letting the read
// helpers below run either as a standalone query or inside a caller's
// transaction.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// ErrRoleNotInTeam is returned when one or more role IDs passed to SetRoles do
// not belong to the membership's team.
var ErrRoleNotInTeam = errors.New("role does not belong to team")

// dedupeStrings returns ids with duplicates removed, preserving the order of
// first occurrence. SetRoles validates role ownership via
// `COUNT(*) FROM roles WHERE id = ANY($1)` (which counts matching rows once
// per distinct role, not input array elements) and then INSERT one
// membership_roles row per ID -- that table's composite primary key
// (membership_id, role_id) rejects a duplicate pair within the same call, so
// a caller legitimately repeating the same valid role ID must be
// deduplicated up front, not just tolerated by the count check.
func dedupeStrings(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// ErrEmailTaken is returned when UpdateMember would change a user's email to
// one already used by a different user account (users.email UNIQUE
// violation).
var ErrEmailTaken = errors.New("email is already used by another account")

// ErrLastSettingsAdmin is returned when SetRoles or RemoveMember would leave
// a team with no member holding settings:write — that would be an
// unrecoverable state via the API (no one left able to manage roles,
// invites, or settings).
var ErrLastSettingsAdmin = errors.New("cannot remove the last member with settings management permission")

// ErrInsufficientPermissionToGrant is returned when SetRoles would give the
// target membership a higher effective permission on some module than the
// caller is allowed to grant -- see enforceNoPermissionEscalation.
var ErrInsufficientPermissionToGrant = errors.New("cannot grant a permission level you do not hold yourself")

// ErrCannotChangeOthersEmail is returned by UpdateMember when the caller
// tries to change another member's email without settings:write. email is
// the account's login-lookup key (auth.Service.Login resolves by email), not
// a cosmetic profile field, and PATCH .../members/{id} is otherwise gated on
// nothing more than members:write -- unlike role assignment
// (isMemberRolesPath, authz.go), which the middleware deliberately upgrades
// to require settings:write specifically because it's privilege-relevant.
// Without this check, a members:write-only "roster manager" (explicitly
// denied settings:write, and blocked from touching roles by
// enforceNoPermissionEscalation) could silently overwrite a settings:write
// admin's login email -- account-wide, across every team that admin belongs
// to -- with no notification and no audit trail of the old value.
var ErrCannotChangeOthersEmail = errors.New("changing another member's email requires settings permission")

// ErrCannotRemoveSettingsAdmin is returned by RemoveMember when the caller
// tries to remove a member who currently holds settings:write while the
// caller does not hold settings:write themselves. DELETE .../members/{id} is
// gated on nothing more than members:write -- unlike role assignment
// (isMemberRolesPath, authz.go), which the middleware deliberately upgrades
// to require settings:write specifically because it's privilege-relevant.
// Without this check, a members:write-only "roster manager" (explicitly
// denied settings:write, and blocked from touching roles by
// enforceNoPermissionEscalation, or another admin's email by
// ErrCannotChangeOthersEmail) could still forcibly remove an admin's
// membership outright -- a strictly more severe action than either of those
// -- as long as at least one other settings:write holder remained (so
// ErrLastSettingsAdmin alone never triggered).
var ErrCannotRemoveSettingsAdmin = errors.New("removing a member with settings management permission requires settings permission yourself")

// Repository handles member-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListCursor is the keyset position for member pagination
// (ORDER BY name ASC, membership id ASC).
//
// Known, accepted limitation: name is mutable (UpdateMember, and GDPR
// erasure rewriting it to a fixed placeholder). If a row's name changes to
// fall on the other side of an in-progress pagination's cursor while a
// caller is mid-page, that row can be skipped or, less likely, duplicated
// across pages -- the same tradeoff any keyset pagination scheme accepts
// when sorting by an editable column. The window is self-healing (a fresh
// list call is always fully correct) and low-impact (an admin viewing the
// team roster, not a security or data-integrity issue), so this is
// deliberately not being architected around -- doing so would mean either
// sorting by an immutable column (changing the roster's user-visible
// alphabetical order) or a materially larger pagination redesign.
type ListCursor struct {
	Name string    `json:"n"`
	ID   uuid.UUID `json:"i"`
}

// ListMembers returns up to limit members of a team (with their roles), ordered
// by name then membership id, starting after cur (nil = first page). Keyset
// pagination — no OFFSET.
func (r *Repository) ListMembers(ctx context.Context, teamID string, limit int, cur *ListCursor) ([]MemberRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// First, get all memberships + user data for the team.
	args := []any{teamID, limit}
	predicate := ""
	if cur != nil {
		predicate = "AND (u.name, m.id) > ($3, $4)"
		args = append(args, cur.Name, cur.ID)
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT m.id, u.id, u.name, u.email, u.phone,
		       u.birthday, u.address, u.avatar_color,
		       (u.photo_object_key IS NOT NULL AND length(u.photo_object_key) > 0),
		       m."group", m.joined_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.team_id = $1 %s
		ORDER BY u.name, m.id
		LIMIT $2
	`, predicate), args...)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.ListMembers: %w", err)
	}
	defer rows.Close()

	var members []MemberRow
	for rows.Next() {
		mr, err := scanMemberRow(rows)
		if err != nil {
			return nil, fmt.Errorf("members.Repository.ListMembers scan: %w", err)
		}
		members = append(members, *mr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Batch-load all roles for the page in a single query instead of N per-member queries.
	if len(members) > 0 {
		ids := make([]string, len(members))
		for i := range members {
			ids[i] = members[i].MembershipID.String()
		}
		rolesByID, err := r.batchGetRoles(ctx, ids)
		if err != nil {
			return nil, err
		}
		for i := range members {
			members[i].Roles = rolesByID[members[i].MembershipID.String()]
		}
	}

	return members, nil
}

// GetMemberPhotoKey returns the object store key for the given membership's
// user photo, scoped to teamID, or pgx.ErrNoRows if the membership doesn't
// belong to that team (or the member has no photo set).
func (r *Repository) GetMemberPhotoKey(ctx context.Context, teamID, membershipID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var key *string
	err := r.pool.QueryRow(ctx, `
		SELECT u.photo_object_key
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.id = $1 AND m.team_id = $2
	`, membershipID, teamID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("members.Repository.GetMemberPhotoKey: %w", err)
	}
	if key == nil || *key == "" {
		return "", pgx.ErrNoRows
	}
	return *key, nil
}

// UpdateMember applies a partial update to the user fields and optionally the
// group, scoped to a membership that belongs to teamID.
func (r *Repository) UpdateMember(ctx context.Context, membershipID, teamID, callerUserID string, patch MemberPatch) (*MemberRow, error) { //nolint:gocognit,cyclop // complexity inherent in dynamic SQL builder
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.UpdateMember: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize against a concurrent RemoveMember/SetRoles for the same team
	// (same lock key both already use) so the membership-existence check
	// below can't be invalidated between here and the UPDATE users
	// statement, which is scoped only by the userID resolved from it -- not
	// by membershipID/teamID -- and so has no way to notice on its own that
	// the caller's authority over that user has since been revoked. Without
	// this, a concurrent RemoveMember could delete the membership after this
	// check but before the write commits: the write still succeeds (email/
	// name/phone/etc. permanently changed on the departed user's global
	// account) while the caller sees a 404 from the reload below, actively
	// suggesting nothing happened.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
		return nil, fmt.Errorf("members.Repository.UpdateMember: advisory lock: %w", err)
	}

	// Get user_id from membership, scoped to the team, inside the same
	// transaction and after the lock so this can't race a concurrent
	// membership removal.
	var userID string
	err = tx.QueryRow(ctx, `SELECT user_id FROM memberships WHERE id = $1 AND team_id = $2`, membershipID, teamID).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("members.Repository.UpdateMember: get user_id: %w", err)
	}

	// Changing another member's email requires settings:write -- see
	// ErrCannotChangeOthersEmail. A member editing their OWN email needs
	// nothing beyond being authenticated as themselves.
	if patch.Email != nil && callerUserID != userID {
		callerPerms, permErr := getEffectivePermissionsByUserQ(ctx, tx, teamID, callerUserID)
		if permErr != nil {
			return nil, fmt.Errorf("members.Repository.UpdateMember: caller permissions: %w", permErr)
		}
		if callerPerms.Settings != "write" {
			return nil, ErrCannotChangeOthersEmail
		}
	}

	// Update user fields.
	userCols := []string{}
	userArgs := []any{}
	n := 1
	if patch.Name != nil {
		userCols = append(userCols, fmt.Sprintf("name = $%d", n))
		userArgs = append(userArgs, *patch.Name)
		n++
	}
	if patch.Email != nil {
		userCols = append(userCols, fmt.Sprintf("email = $%d", n))
		userArgs = append(userArgs, *patch.Email)
		n++
	}
	if patch.Phone != nil {
		userCols = append(userCols, fmt.Sprintf("phone = $%d", n))
		userArgs = append(userArgs, *patch.Phone)
		n++
	}
	if patch.Address != nil {
		userCols = append(userCols, fmt.Sprintf("address = $%d", n))
		userArgs = append(userArgs, *patch.Address)
		n++
	}
	if patch.Birthday != nil {
		userCols = append(userCols, fmt.Sprintf("birthday = $%d", n))
		userArgs = append(userArgs, *patch.Birthday)
		n++
	}
	if len(userCols) > 0 {
		setSQL := ""
		for i, c := range userCols {
			if i > 0 {
				setSQL += ", "
			}
			setSQL += c
		}
		userArgs = append(userArgs, userID)
		_, err = tx.Exec(ctx, fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", setSQL, n), userArgs...)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
				return nil, ErrEmailTaken
			}
			return nil, fmt.Errorf("members.Repository.UpdateMember: update user: %w", err)
		}
	}

	// Update membership group.
	if patch.Group != nil {
		_, err = tx.Exec(ctx, `UPDATE memberships SET "group" = $1 WHERE id = $2 AND team_id = $3`, *patch.Group, membershipID, teamID)
		if err != nil {
			return nil, fmt.Errorf("members.Repository.UpdateMember: update membership: %w", err)
		}
	}

	// Read the final row back inside the same transaction, still holding the
	// advisory lock, so this can't race a concurrent RemoveMember the way a
	// reload after commit (once the lock is released) could -- that window
	// let a legitimate write get reported back to the caller as
	// pgx.ErrNoRows, since the reload alone would find the membership gone.
	mr, err := getMemberByMembershipIDQ(ctx, tx, membershipID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("members.Repository.UpdateMember: commit: %w", err)
	}

	return mr, nil
}

// SetRoles replaces the role assignments for the given membership. The
// membership must belong to teamID, and every role in roleIDs must also
// belong to teamID — otherwise pgx.ErrNoRows / ErrRoleNotInTeam is returned.
func (r *Repository) SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string, callerUserID string) (*MemberRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	roleIDs = dedupeStrings(roleIDs)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.SetRoles: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize against concurrent role changes in the same team so the
	// "last settings admin" check below, and the membership/role-existence
	// validation, can't race with another admin's simultaneous role
	// deletion/demotion (checking on r.pool before this transaction began, as
	// this used to, left a window where a concurrent DeleteRole could commit
	// between the check and the INSERT below, hitting its FK constraint and
	// surfacing as an unhandled 500 instead of the clean ErrRoleNotInTeam
	// this check exists to produce).
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
		return nil, fmt.Errorf("members.Repository.SetRoles: advisory lock: %w", err)
	}

	if err := validateSetRolesInputs(ctx, tx, membershipID, teamID, roleIDs); err != nil {
		return nil, err
	}

	if err := enforceNoPermissionEscalation(ctx, tx, teamID, callerUserID, membershipID, roleIDs); err != nil {
		return nil, err
	}

	if err := enforceSettingsAdminGuard(ctx, tx, teamID, membershipID, roleIDs); err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `DELETE FROM membership_roles WHERE membership_id = $1`, membershipID)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.SetRoles: delete: %w", err)
	}

	for _, roleID := range roleIDs {
		_, err = tx.Exec(ctx, `
			INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)
		`, membershipID, roleID)
		if err != nil {
			return nil, fmt.Errorf("members.Repository.SetRoles: insert role %s: %w", roleID, err)
		}
	}

	// Read the final row back inside the same transaction, before it's
	// committed and the advisory lock released -- see the identical comment
	// in UpdateMember for why a post-commit reload on r.pool isn't safe here.
	mr, err := getMemberByMembershipIDQ(ctx, tx, membershipID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("members.Repository.SetRoles: commit: %w", err)
	}

	return mr, nil
}

// validateSetRolesInputs checks that membershipID belongs to teamID and every
// role in roleIDs belongs to teamID, returning pgx.ErrNoRows / ErrRoleNotInTeam
// respectively when not. Takes tx (not the pool) so the check runs inside the
// caller's transaction, after it holds the team's advisory lock — otherwise a
// role could be deleted between this check and the caller's later write.
func validateSetRolesInputs(ctx context.Context, tx pgx.Tx, membershipID, teamID string, roleIDs []string) error {
	var exists bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM memberships WHERE id = $1 AND team_id = $2)`,
		membershipID, teamID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("members.Repository.SetRoles: check membership: %w", err)
	}
	if !exists {
		return pgx.ErrNoRows
	}

	if len(roleIDs) > 0 {
		var count int
		err = tx.QueryRow(ctx,
			`SELECT COUNT(*)::int FROM roles WHERE id = ANY($1) AND team_id = $2`,
			roleIDs, teamID,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("members.Repository.SetRoles: check roles: %w", err)
		}
		if count != len(roleIDs) {
			return ErrRoleNotInTeam
		}
	}
	return nil
}

// enforceNoPermissionEscalation returns ErrInsufficientPermissionToGrant if
// replacing membershipID's roles with roleIDs would give it, on any module,
// a higher effective permission than the greater of (a) the caller's own
// current effective permission for that module, or (b) the module's
// effective permission the target already held before this call.
//
// (b) matters because SetRoles fully replaces a membership's role set: a
// caller reorganizing/consolidating a member's EXISTING roles, or demoting
// them, never counts as "granting" anything new even if the target's
// resulting permission still exceeds the caller's own -- only an actual
// INCREASE beyond what the target already had is a grant. Without this
// distinction, a settings:write-only caller could never touch the role
// assignment of a member who (legitimately, via someone else) already holds
// e.g. finances:write, even just to remove an unrelated role from them.
//
// This is the fix for a privilege-escalation path: middleware gates both
// role definition (POST/PATCH .../roles) and role assignment (PUT
// .../members/{id}/roles) on nothing more than settings:write -- with no
// check here, a member holding only settings:write could create a role
// granting arbitrary module permissions and assign it to themselves (or to
// a second, colluding membership), ending up with de facto full admin
// despite never having been granted anything beyond settings:write.
func enforceNoPermissionEscalation(ctx context.Context, tx pgx.Tx, teamID, callerUserID, membershipID string, roleIDs []string) error {
	callerPerms, err := getEffectivePermissionsByUserQ(ctx, tx, teamID, callerUserID)
	if err != nil {
		return fmt.Errorf("members.Repository.SetRoles: caller permissions: %w", err)
	}
	targetPermsBefore, err := getMembershipEffectivePermissionsQ(ctx, tx, membershipID)
	if err != nil {
		return fmt.Errorf("members.Repository.SetRoles: target permissions: %w", err)
	}
	newPerms, err := roleSetEffectivePermissions(ctx, tx, teamID, roleIDs)
	if err != nil {
		return fmt.Errorf("members.Repository.SetRoles: %w", err)
	}

	ceilings := []string{
		foldMax(callerPerms.Events, targetPermsBefore.Events),
		foldMax(callerPerms.Members, targetPermsBefore.Members),
		foldMax(callerPerms.Finances, targetPermsBefore.Finances),
		foldMax(callerPerms.News, targetPermsBefore.News),
		foldMax(callerPerms.Polls, targetPermsBefore.Polls),
		foldMax(callerPerms.Settings, targetPermsBefore.Settings),
	}
	granted := []string{newPerms.Events, newPerms.Members, newPerms.Finances, newPerms.News, newPerms.Polls, newPerms.Settings}
	for i, level := range granted {
		if permLevelRank(level) > permLevelRank(ceilings[i]) {
			return ErrInsufficientPermissionToGrant
		}
	}
	return nil
}

// roleSetEffectivePermissions returns the per-module maximum permission
// across every role in roleIDs (which must already be validated as
// belonging to teamID by validateSetRolesInputs).
func roleSetEffectivePermissions(ctx context.Context, tx pgx.Tx, teamID string, roleIDs []string) (teams.PermissionsJSON, error) {
	eff := teams.PermissionsJSON{Events: "none", Members: "none", Finances: "none", News: "none", Polls: "none", Settings: "none"}
	if len(roleIDs) == 0 {
		return eff, nil
	}
	rows, err := tx.Query(ctx, `SELECT permissions FROM roles WHERE id = ANY($1) AND team_id = $2`, roleIDs, teamID)
	if err != nil {
		return eff, fmt.Errorf("roleSetEffectivePermissions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var permJSON []byte
		if err := rows.Scan(&permJSON); err != nil {
			return eff, fmt.Errorf("roleSetEffectivePermissions scan: %w", err)
		}
		var p teams.PermissionsJSON
		if err := json.Unmarshal(permJSON, &p); err != nil {
			return eff, fmt.Errorf("roleSetEffectivePermissions unmarshal: %w", err)
		}
		eff = foldPermissions(eff, p)
	}
	return eff, rows.Err()
}

// getMembershipEffectivePermissionsQ returns the per-module maximum
// permission across membershipID's CURRENTLY assigned roles. The
// r.team_id = m.team_id join predicate is defense in depth against a
// membership_roles row ever pointing at a role from a different team (see
// teams.Repository.GetRolesForMembership's identical rationale) -- notably
// this feeds enforceNoPermissionEscalation's escalation-ceiling calculation,
// so a cross-team role slipping through here would inflate that ceiling
// instead of failing safe.
func getMembershipEffectivePermissionsQ(ctx context.Context, q querier, membershipID string) (teams.PermissionsJSON, error) {
	eff := teams.PermissionsJSON{Events: "none", Members: "none", Finances: "none", News: "none", Polls: "none", Settings: "none"}
	rows, err := q.Query(ctx, `
		SELECT r.permissions
		FROM roles r
		JOIN membership_roles mr ON mr.role_id = r.id
		JOIN memberships m ON m.id = mr.membership_id
		WHERE mr.membership_id = $1 AND r.team_id = m.team_id
	`, membershipID)
	if err != nil {
		return eff, fmt.Errorf("getMembershipEffectivePermissionsQ: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var permJSON []byte
		if err := rows.Scan(&permJSON); err != nil {
			return eff, fmt.Errorf("getMembershipEffectivePermissionsQ scan: %w", err)
		}
		var p teams.PermissionsJSON
		if err := json.Unmarshal(permJSON, &p); err != nil {
			return eff, fmt.Errorf("getMembershipEffectivePermissionsQ unmarshal: %w", err)
		}
		eff = foldPermissions(eff, p)
	}
	return eff, rows.Err()
}

// foldPermissions folds b into a, taking the per-module maximum.
func foldPermissions(a, b teams.PermissionsJSON) teams.PermissionsJSON {
	return teams.PermissionsJSON{
		Events:   foldMax(a.Events, b.Events),
		Members:  foldMax(a.Members, b.Members),
		Finances: foldMax(a.Finances, b.Finances),
		News:     foldMax(a.News, b.News),
		Polls:    foldMax(a.Polls, b.Polls),
		Settings: foldMax(a.Settings, b.Settings),
	}
}

// enforceSettingsAdminGuard returns ErrLastSettingsAdmin if replacing
// membershipID's roles with roleIDs would leave the team with no
// settings:write holder.
func enforceSettingsAdminGuard(ctx context.Context, tx pgx.Tx, teamID, membershipID string, roleIDs []string) error {
	willHaveSettingsWrite, err := roleSetHasSettingsWrite(ctx, tx, teamID, roleIDs)
	if err != nil {
		return fmt.Errorf("members.Repository.SetRoles: %w", err)
	}
	if willHaveSettingsWrite {
		return nil
	}
	othersHaveSettingsWrite, err := teamHasOtherSettingsWriteMember(ctx, tx, teamID, membershipID)
	if err != nil {
		return fmt.Errorf("members.Repository.SetRoles: %w", err)
	}
	if !othersHaveSettingsWrite {
		return ErrLastSettingsAdmin
	}
	return nil
}

// roleSetHasSettingsWrite reports whether any role in roleIDs grants
// settings:write.
func roleSetHasSettingsWrite(ctx context.Context, tx pgx.Tx, teamID string, roleIDs []string) (bool, error) {
	if len(roleIDs) == 0 {
		return false, nil
	}
	var has bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM roles WHERE id = ANY($1) AND team_id = $2 AND permissions->>'settings' = 'write')`,
		roleIDs, teamID,
	).Scan(&has)
	if err != nil {
		return false, fmt.Errorf("check settings write: %w", err)
	}
	return has, nil
}

// teamHasOtherSettingsWriteMember reports whether any membership in teamID,
// other than excludeMembershipID, holds settings:write via any assigned
// role AND belongs to a still-authenticatable (not GDPR-erased) account --
// an erased user's membership_roles row survives EraseUser (only users is
// anonymized), so without the deleted_at check this would still count a
// permanently unloginable account as a usable settings admin.
func teamHasOtherSettingsWriteMember(ctx context.Context, tx pgx.Tx, teamID, excludeMembershipID string) (bool, error) {
	var has bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM memberships m
			JOIN membership_roles mr ON mr.membership_id = m.id
			JOIN roles r ON r.id = mr.role_id
			JOIN users u ON u.id = m.user_id
			WHERE m.team_id = $1 AND m.id != $2 AND r.team_id = m.team_id
			  AND r.permissions->>'settings' = 'write'
			  AND u.deleted_at IS NULL
		)`, teamID, excludeMembershipID,
	).Scan(&has)
	if err != nil {
		return false, fmt.Errorf("check other settings admins: %w", err)
	}
	return has, nil
}

// RemoveMember deletes a membership (cascades membership_roles) that belongs
// to teamID. Returns pgx.ErrNoRows if no membership with id exists within
// teamID, ErrLastSettingsAdmin if the membership is the team's last
// settings:write holder, or ErrCannotRemoveSettingsAdmin if the membership
// holds settings:write and callerUserID does not.
func (r *Repository) RemoveMember(ctx context.Context, membershipID, teamID, callerUserID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("members.Repository.RemoveMember: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
		return fmt.Errorf("members.Repository.RemoveMember: advisory lock: %w", err)
	}

	var isSettingsWriter bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM membership_roles mr
			JOIN roles r ON r.id = mr.role_id
			WHERE mr.membership_id = $1 AND r.team_id = $2 AND r.permissions->>'settings' = 'write'
		)`, membershipID, teamID,
	).Scan(&isSettingsWriter)
	if err != nil {
		return fmt.Errorf("members.Repository.RemoveMember: check settings write: %w", err)
	}
	if isSettingsWriter {
		callerPerms, err := getEffectivePermissionsByUserQ(ctx, tx, teamID, callerUserID)
		if err != nil {
			return fmt.Errorf("members.Repository.RemoveMember: caller permissions: %w", err)
		}
		if callerPerms.Settings != "write" {
			return ErrCannotRemoveSettingsAdmin
		}
		othersHaveSettingsWrite, err := teamHasOtherSettingsWriteMember(ctx, tx, teamID, membershipID)
		if err != nil {
			return fmt.Errorf("members.Repository.RemoveMember: %w", err)
		}
		if !othersHaveSettingsWrite {
			return ErrLastSettingsAdmin
		}
	}

	tag, err := tx.Exec(ctx, `DELETE FROM memberships WHERE id = $1 AND team_id = $2`, membershipID, teamID)
	if err != nil {
		return fmt.Errorf("members.Repository.RemoveMember: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("members.Repository.RemoveMember: commit: %w", err)
	}
	return nil
}

// IsMember returns true when the user is an active member of the team.
func (r *Repository) IsMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var exists bool
	err := r.pool.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM memberships WHERE team_id = $1 AND user_id = $2)`,
		teamID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("members.Repository.IsMember: %w", err)
	}
	return exists, nil
}

// permLevelRank maps permission levels to a comparable rank (none < read < write).
func permLevelRank(level string) int {
	switch level {
	case "write":
		return 2
	case "read":
		return 1
	default:
		return 0
	}
}

// foldMax returns the higher-ranked of two permission levels.
func foldMax(cur, next string) string {
	if permLevelRank(next) > permLevelRank(cur) {
		return next
	}
	return cur
}

// GetPermissions returns the effective permissions for a user in a team,
// computed as the per-module maximum across all of the user's roles
// (none < read < write). A user with no roles gets all-"none".
func (r *Repository) GetPermissions(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return getEffectivePermissionsByUserQ(ctx, r.pool, teamID.String(), userID.String())
}

// getEffectivePermissionsByUserQ is the shared implementation behind
// GetPermissions. It takes a querier rather than always using r.pool so
// enforceNoPermissionEscalation can compute the caller's effective
// permissions inside its own transaction, behind the per-team advisory lock.
func getEffectivePermissionsByUserQ(ctx context.Context, q querier, teamID, userID string) (teams.PermissionsJSON, error) {
	eff := teams.PermissionsJSON{Events: "none", Members: "none", Finances: "none", News: "none", Polls: "none", Settings: "none"}
	rows, err := q.Query(ctx, `
		SELECT r.permissions
		FROM roles r
		JOIN membership_roles mr ON mr.role_id = r.id
		JOIN memberships m ON m.id = mr.membership_id
		WHERE m.team_id = $1 AND m.user_id = $2 AND r.team_id = $1
	`, teamID, userID)
	if err != nil {
		return eff, fmt.Errorf("getEffectivePermissionsByUserQ: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var permJSON []byte
		if err := rows.Scan(&permJSON); err != nil {
			return eff, fmt.Errorf("getEffectivePermissionsByUserQ scan: %w", err)
		}
		var p teams.PermissionsJSON
		if err := json.Unmarshal(permJSON, &p); err != nil {
			return eff, fmt.Errorf("getEffectivePermissionsByUserQ unmarshal: %w", err)
		}
		eff = foldPermissions(eff, p)
	}
	return eff, rows.Err()
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// getMemberByMembershipIDQ takes a querier rather than always using r.pool
// so callers that need to read back a row they just wrote can do so inside
// their own transaction -- e.g. UpdateMember and SetRoles read the final row
// before committing, while still holding the per-team advisory lock, so the
// read can't race a concurrent RemoveMember the way a post-commit reload on
// r.pool could.
func getMemberByMembershipIDQ(ctx context.Context, q querier, membershipID string) (*MemberRow, error) {
	row := q.QueryRow(ctx, `
		SELECT m.id, u.id, u.name, u.email, u.phone,
		       u.birthday, u.address, u.avatar_color,
		       (u.photo_object_key IS NOT NULL AND length(u.photo_object_key) > 0),
		       m."group", m.joined_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.id = $1
	`, membershipID)

	mr, err := scanMemberRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("members.Repository.getMemberByMembershipID: %w", err)
	}

	roles, err := getRolesForMembershipQ(ctx, q, membershipID)
	if err != nil {
		return nil, err
	}
	mr.Roles = roles

	return mr, nil
}

// batchGetRoles loads all roles for a set of membership IDs in a single query,
// returning a map keyed by membership ID. Callers with an empty id list should
// skip the call entirely — the function returns an empty map without a round-trip.
func (r *Repository) batchGetRoles(ctx context.Context, membershipIDs []string) (map[string][]teams.RoleRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT mr.membership_id, r.id, r.team_id, r.name, r.system, r.color, r.permissions
		FROM membership_roles mr
		JOIN roles r ON r.id = mr.role_id
		JOIN memberships m ON m.id = mr.membership_id
		WHERE mr.membership_id = ANY($1::uuid[]) AND r.team_id = m.team_id
		ORDER BY mr.membership_id, r.id
	`, membershipIDs)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.batchGetRoles: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]teams.RoleRow)
	for rows.Next() {
		var membershipID string
		rr := &teams.RoleRow{}
		var permJSON []byte
		if err := rows.Scan(&membershipID, &rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permJSON); err != nil {
			return nil, fmt.Errorf("members.Repository.batchGetRoles scan: %w", err)
		}
		if err := json.Unmarshal(permJSON, &rr.Permissions); err != nil {
			return nil, fmt.Errorf("members.Repository.batchGetRoles unmarshal: %w", err)
		}
		result[membershipID] = append(result[membershipID], *rr)
	}
	return result, rows.Err()
}

func getRolesForMembershipQ(ctx context.Context, q querier, membershipID string) ([]teams.RoleRow, error) {
	rows, err := q.Query(ctx, `
		SELECT r.id, r.team_id, r.name, r.system, r.color, r.permissions
		FROM roles r
		JOIN membership_roles mr ON mr.role_id = r.id
		JOIN memberships m ON m.id = mr.membership_id
		WHERE mr.membership_id = $1 AND r.team_id = m.team_id
		ORDER BY r.id
	`, membershipID)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.getRolesForMembership: %w", err)
	}
	defer rows.Close()

	var out []teams.RoleRow
	for rows.Next() {
		rr := &teams.RoleRow{}
		var permJSON []byte
		err := rows.Scan(&rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permJSON)
		if err != nil {
			return nil, fmt.Errorf("members.Repository.getRolesForMembership scan: %w", err)
		}
		if err := json.Unmarshal(permJSON, &rr.Permissions); err != nil {
			return nil, fmt.Errorf("members.Repository.getRolesForMembership unmarshal: %w", err)
		}
		out = append(out, *rr)
	}
	return out, rows.Err()
}

func scanMemberRow(row interface{ Scan(dest ...any) error }) (*MemberRow, error) {
	mr := &MemberRow{}
	err := row.Scan(
		&mr.MembershipID, &mr.UserID, &mr.Name, &mr.Email, &mr.Phone,
		&mr.Birthday, &mr.Address, &mr.AvatarColor,
		&mr.HasPhoto,
		&mr.Group, &mr.JoinedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return mr, nil
}

// ensure uuid is used.
var _ = uuid.UUID{}
