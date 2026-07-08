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
		       (u.photo_data IS NOT NULL AND length(u.photo_data) > 0),
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

// UpdateMember applies a partial update to the user fields and optionally the
// group, scoped to a membership that belongs to teamID.
func (r *Repository) UpdateMember(ctx context.Context, membershipID, teamID string, patch MemberPatch) (*MemberRow, error) { //nolint:gocognit,cyclop // complexity inherent in dynamic SQL builder
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Get user_id from membership first, scoped to the team.
	var userID string
	err := r.pool.QueryRow(ctx, `SELECT user_id FROM memberships WHERE id = $1 AND team_id = $2`, membershipID, teamID).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("members.Repository.UpdateMember: get user_id: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.UpdateMember: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("members.Repository.UpdateMember: commit: %w", err)
	}

	return r.getMemberByMembershipID(ctx, membershipID)
}

// SetRoles replaces the role assignments for the given membership. The
// membership must belong to teamID, and every role in roleIDs must also
// belong to teamID — otherwise pgx.ErrNoRows / ErrRoleNotInTeam is returned.
func (r *Repository) SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string) (*MemberRow, error) {
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

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("members.Repository.SetRoles: commit: %w", err)
	}

	return r.getMemberByMembershipID(ctx, membershipID)
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
			WHERE m.team_id = $1 AND m.id != $2 AND r.permissions->>'settings' = 'write'
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
// teamID, or ErrLastSettingsAdmin if the membership is the team's last
// settings:write holder.
func (r *Repository) RemoveMember(ctx context.Context, membershipID, teamID string) error {
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
			WHERE mr.membership_id = $1 AND r.permissions->>'settings' = 'write'
		)`, membershipID,
	).Scan(&isSettingsWriter)
	if err != nil {
		return fmt.Errorf("members.Repository.RemoveMember: check settings write: %w", err)
	}
	if isSettingsWriter {
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
	rows, err := r.pool.Query(ctx, `
		SELECT r.permissions
		FROM roles r
		JOIN membership_roles mr ON mr.role_id = r.id
		JOIN memberships m ON m.id = mr.membership_id
		WHERE m.team_id = $1 AND m.user_id = $2 AND r.team_id = $1
	`, teamID, userID)
	if err != nil {
		return teams.PermissionsJSON{}, fmt.Errorf("members.Repository.GetPermissions: %w", err)
	}
	defer rows.Close()

	eff := teams.PermissionsJSON{
		Events: "none", Members: "none", Finances: "none",
		News: "none", Polls: "none", Settings: "none",
	}
	for rows.Next() {
		var permJSON []byte
		if err := rows.Scan(&permJSON); err != nil {
			return teams.PermissionsJSON{}, fmt.Errorf("members.Repository.GetPermissions scan: %w", err)
		}
		var p teams.PermissionsJSON
		if err := json.Unmarshal(permJSON, &p); err != nil {
			return teams.PermissionsJSON{}, fmt.Errorf("members.Repository.GetPermissions unmarshal: %w", err)
		}
		eff.Events = foldMax(eff.Events, p.Events)
		eff.Members = foldMax(eff.Members, p.Members)
		eff.Finances = foldMax(eff.Finances, p.Finances)
		eff.News = foldMax(eff.News, p.News)
		eff.Polls = foldMax(eff.Polls, p.Polls)
		eff.Settings = foldMax(eff.Settings, p.Settings)
	}
	return eff, rows.Err()
}

// ─── internal helpers ─────────────────────────────────────────────────────────

func (r *Repository) getMemberByMembershipID(ctx context.Context, membershipID string) (*MemberRow, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT m.id, u.id, u.name, u.email, u.phone,
		       u.birthday, u.address, u.avatar_color,
		       (u.photo_data IS NOT NULL AND length(u.photo_data) > 0),
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

	roles, err := r.getRolesForMembership(ctx, membershipID)
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
		WHERE mr.membership_id = ANY($1::uuid[])
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

func (r *Repository) getRolesForMembership(ctx context.Context, membershipID string) ([]teams.RoleRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.team_id, r.name, r.system, r.color, r.permissions
		FROM roles r
		JOIN membership_roles mr ON mr.role_id = r.id
		WHERE mr.membership_id = $1
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
