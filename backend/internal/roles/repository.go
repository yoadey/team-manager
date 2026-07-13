package roles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ErrSystemRole is returned when attempting to delete a system role, or to
// change the name/permissions of a system role (color is cosmetic-only and
// remains editable).
var ErrSystemRole = errors.New("cannot modify system role")

// ErrLastSettingsAdmin is returned when DeleteRole would remove, or
// UpdateRole would revoke, the last role assignment in the team that grants
// settings:write. A custom (non-system) role can grant settings:write just
// like the built-in Admin role, and membership_roles cascades on role
// deletion (ON DELETE CASCADE), so deleting such a role — or simply editing
// its permissions to no longer include settings:write — can silently strip
// every member holding it of admin access in one step — the same
// unrecoverable "locked out of settings" state that members.SetRoles/
// RemoveMember already guard against for individual membership changes.
var ErrLastSettingsAdmin = errors.New("cannot remove the last role granting settings management permission")

// ErrInsufficientPermissionToGrant is returned by UpdateRole when the patch
// would give the role, on any module, a higher permission than the greater
// of (a) the caller's own current effective permission for that module in
// this team, or (b) the level the role itself already granted before this
// edit. Mirrors members.enforceNoPermissionEscalation's ceiling, applied to
// editing a role definition rather than assigning one -- see
// enforceNoRoleEscalation for why this is needed even though SetRoles
// already closed the equivalent path for role *assignment*: middleware
// gates both role definition (POST/PATCH .../roles) and role assignment
// (PUT .../members/{id}/roles) on nothing more than settings:write, so
// without this check a settings:write-only caller could create (or reuse)
// a role granting only what they already hold, assign it to themselves --
// passing SetRoles's ceiling trivially, since it grants nothing beyond what
// they already have -- and then PATCH that same role's permissions upward
// afterward, silently escalating their own effective permissions with no
// assignment step left to catch it.
var ErrInsufficientPermissionToGrant = errors.New("cannot grant a permission level you do not hold yourself")

// Repository handles role-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListRoles returns all roles for the given team.
func (r *Repository) ListRoles(ctx context.Context, teamID string) ([]teams.RoleRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, team_id, name, system, color, permissions
		FROM roles
		WHERE team_id = $1
		ORDER BY system DESC, name
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("roles.Repository.ListRoles: %w", err)
	}
	defer rows.Close()

	var out []teams.RoleRow
	for rows.Next() {
		rr, err := scanRole(rows)
		if err != nil {
			return nil, fmt.Errorf("roles.Repository.ListRoles scan: %w", err)
		}
		out = append(out, *rr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("roles.Repository.ListRoles rows: %w", err)
	}
	return out, nil
}

// CreateRole inserts a new role for the team.
func (r *Repository) CreateRole(ctx context.Context, teamID, name string, color *string, permissions teams.PermissionsJSON) (*teams.RoleRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	permJSON, err := json.Marshal(permissions)
	if err != nil {
		return nil, fmt.Errorf("roles.Repository.CreateRole: marshal permissions: %w", err)
	}

	rr := &teams.RoleRow{}
	var permBytes []byte
	err = r.pool.QueryRow(ctx, `
		INSERT INTO roles (team_id, name, color, permissions)
		VALUES ($1, $2, $3, $4)
		RETURNING id, team_id, name, system, color, permissions
	`, teamID, name, color, permJSON).Scan(
		&rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("roles.Repository.CreateRole: %w", err)
	}
	if err := json.Unmarshal(permBytes, &rr.Permissions); err != nil {
		return nil, fmt.Errorf("roles.Repository.CreateRole: unmarshal: %w", err)
	}
	return rr, nil
}

// buildRoleUpdateSets constructs a SET clause and args slice for patch.
func buildRoleUpdateSets(patch RolePatch) (setSQL string, args []any, err error) {
	var sets []string
	n := 1

	if patch.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", n))
		args = append(args, *patch.Name)
		n++
	}
	if patch.Color != nil {
		sets = append(sets, fmt.Sprintf("color = $%d", n))
		args = append(args, *patch.Color)
		n++
	}
	if patch.Permissions != nil {
		permJSON, marshalErr := json.Marshal(*patch.Permissions)
		if marshalErr != nil {
			return "", nil, fmt.Errorf("roles.buildRoleUpdateSets: marshal permissions: %w", marshalErr)
		}
		sets = append(sets, fmt.Sprintf("permissions = $%d", n))
		args = append(args, permJSON)
	}

	return strings.Join(sets, ", "), args, nil
}

// guardRoleUpdate rejects renaming/re-permissioning a system role
// (ErrSystemRole); rejects a permissions patch that would grant, on any
// module, more than the caller's own ceiling allows (ErrInsufficientPermissionToGrant,
// see enforceNoRoleEscalation); and rejects revoking settings:write from a
// role's permissions when no other role held by any member would still
// grant it (ErrLastSettingsAdmin) — the same invariant DeleteRole enforces
// for deleting a role outright, applied here to editing one.
func (r *Repository) guardRoleUpdate(ctx context.Context, tx pgx.Tx, roleID, teamID, callerUserID string, patch RolePatch) error {
	// Same advisory lock key as DeleteRole/members.SetRoles/RemoveMember,
	// serializing this against concurrent role/assignment changes guarding
	// the same invariant.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
		return fmt.Errorf("roles.Repository.UpdateRole: advisory lock: %w", err)
	}

	var isSystem bool
	var currentPermBytes []byte
	if err := tx.QueryRow(ctx, `SELECT system, permissions FROM roles WHERE id = $1 AND team_id = $2`, roleID, teamID).
		Scan(&isSystem, &currentPermBytes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return fmt.Errorf("roles.Repository.UpdateRole: check system: %w", err)
	}
	if isSystem {
		return ErrSystemRole
	}

	if patch.Permissions == nil {
		return nil
	}

	var currentPerms teams.PermissionsJSON
	if err := json.Unmarshal(currentPermBytes, &currentPerms); err != nil {
		return fmt.Errorf("roles.Repository.UpdateRole: unmarshal current permissions: %w", err)
	}
	if err := enforceNoRoleEscalation(ctx, tx, teamID, callerUserID, currentPerms, *patch.Permissions); err != nil {
		return err
	}

	if patch.Permissions.Settings == "write" {
		return nil
	}

	// The users join (with deleted_at IS NULL) excludes GDPR-erased accounts:
	// EraseUser only anonymizes users, leaving membership_roles intact, so an
	// erased user's settings:write role would otherwise still count as a
	// usable admin even though the account can never authenticate again.
	var othersHaveSettingsWrite bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM memberships m
			JOIN membership_roles mr ON mr.membership_id = m.id
			JOIN roles r ON r.id = mr.role_id
			JOIN users u ON u.id = m.user_id
			WHERE m.team_id = $1 AND r.id != $2 AND r.team_id = m.team_id
			  AND r.permissions->>'settings' = 'write'
			  AND u.deleted_at IS NULL
		)`, teamID, roleID,
	).Scan(&othersHaveSettingsWrite); err != nil {
		return fmt.Errorf("roles.Repository.UpdateRole: check other settings admins: %w", err)
	}
	if othersHaveSettingsWrite {
		return nil
	}

	if currentPerms.Settings == "write" {
		return ErrLastSettingsAdmin
	}
	return nil
}

// enforceNoRoleEscalation returns ErrInsufficientPermissionToGrant if
// newPerms would grant, on any module, a higher permission than the greater
// of (a) callerUserID's own current effective permission for that module in
// teamID, or (b) the level currentPerms (the role's permissions before this
// edit) already granted. (b) mirrors members.enforceNoPermissionEscalation's
// identical allowance: reorganizing/demoting what a role already grants is
// never treated as "granting" something new, even if the result still
// exceeds the caller's own permissions, since nothing actually increased.
func enforceNoRoleEscalation(ctx context.Context, tx pgx.Tx, teamID, callerUserID string, currentPerms, newPerms teams.PermissionsJSON) error {
	callerPerms, err := getEffectivePermissionsByUserQ(ctx, tx, teamID, callerUserID)
	if err != nil {
		return fmt.Errorf("roles.Repository.UpdateRole: caller permissions: %w", err)
	}

	ceilings := []string{
		foldMax(callerPerms.Events, currentPerms.Events),
		foldMax(callerPerms.Members, currentPerms.Members),
		foldMax(callerPerms.Finances, currentPerms.Finances),
		foldMax(callerPerms.News, currentPerms.News),
		foldMax(callerPerms.Polls, currentPerms.Polls),
		foldMax(callerPerms.Settings, currentPerms.Settings),
	}
	granted := []string{newPerms.Events, newPerms.Members, newPerms.Finances, newPerms.News, newPerms.Polls, newPerms.Settings}
	for i, level := range granted {
		if permLevelRank(level) > permLevelRank(ceilings[i]) {
			return ErrInsufficientPermissionToGrant
		}
	}
	return nil
}

// getEffectivePermissionsByUserQ returns the per-module maximum permission
// across all of userID's currently-assigned roles in teamID. Mirrors
// members.getEffectivePermissionsByUserQ, duplicated here (rather than
// exported cross-package) since it's a small, self-contained query and the
// two packages otherwise have no dependency on each other.
func getEffectivePermissionsByUserQ(ctx context.Context, tx pgx.Tx, teamID, userID string) (teams.PermissionsJSON, error) {
	eff := teams.PermissionsJSON{Events: "none", Members: "none", Finances: "none", News: "none", Polls: "none", Settings: "none"}
	rows, err := tx.Query(ctx, `
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

// UpdateRole applies a partial update to a role that belongs to teamID.
// Renaming or re-permissioning a system role (Admin/Member, created at team
// setup) would let any settings:write holder silently rewrite what those
// built-in roles grant — the same escalation DeleteRole already blocks, so
// those changes are rejected with ErrSystemRole. Color is cosmetic-only and
// stays editable even on system roles.
//
// Revoking settings:write from a role's permissions is guarded the same way
// DeleteRole guards deleting a role outright: if no other role held by any
// member would still grant settings:write, the change is rejected with
// ErrLastSettingsAdmin — otherwise editing (rather than deleting) the last
// settings-admin role would lock the team out of settings/role management
// just the same.
//
// A permissions patch is also rejected with ErrInsufficientPermissionToGrant
// if it would grant, on any module, more than callerUserID's own ceiling
// allows -- see enforceNoRoleEscalation.
func (r *Repository) UpdateRole(ctx context.Context, roleID, teamID, callerUserID string, patch RolePatch) (*teams.RoleRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	setSQL, args, err := buildRoleUpdateSets(patch)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return r.getRoleByID(ctx, roleID, teamID)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles.Repository.UpdateRole: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if patch.Name != nil || patch.Permissions != nil {
		if err := r.guardRoleUpdate(ctx, tx, roleID, teamID, callerUserID, patch); err != nil {
			return nil, err
		}
	}

	n := len(args) + 1
	args = append(args, roleID, teamID)

	rr := &teams.RoleRow{}
	var permBytes []byte
	err = tx.QueryRow(ctx, fmt.Sprintf(`
		UPDATE roles SET %s WHERE id = $%d AND team_id = $%d
		RETURNING id, team_id, name, system, color, permissions
	`, setSQL, n, n+1), args...).Scan(
		&rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permBytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("roles.Repository.UpdateRole: %w", err)
	}
	if err := json.Unmarshal(permBytes, &rr.Permissions); err != nil {
		return nil, fmt.Errorf("roles.Repository.UpdateRole: unmarshal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("roles.Repository.UpdateRole: commit: %w", err)
	}
	return rr, nil
}

// DeleteRole deletes a non-system role that belongs to teamID. Returns an
// error if the role is system=true, ErrLastSettingsAdmin if deleting it would
// leave the team with no role assignment granting settings:write, or
// pgx.ErrNoRows if no role with roleID exists within teamID.
func (r *Repository) DeleteRole(ctx context.Context, roleID, teamID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Uses the same advisory lock key as members.SetRoles/RemoveMember
	// (hashtextextended(teamID, 0)) so role deletion is serialized against
	// concurrent role (re)assignment changes guarding the same invariant.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: advisory lock: %w", err)
	}

	var isSystem bool
	err = tx.QueryRow(ctx, `SELECT system FROM roles WHERE id = $1 AND team_id = $2`, roleID, teamID).Scan(&isSystem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return fmt.Errorf("roles.Repository.DeleteRole: check system: %w", err)
	}
	if isSystem {
		return ErrSystemRole
	}

	// The users join (with deleted_at IS NULL) excludes GDPR-erased accounts;
	// see the identical comment in guardRoleUpdate above.
	var othersHaveSettingsWrite bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM memberships m
			JOIN membership_roles mr ON mr.membership_id = m.id
			JOIN roles r ON r.id = mr.role_id
			JOIN users u ON u.id = m.user_id
			WHERE m.team_id = $1 AND r.id != $2 AND r.team_id = m.team_id
			  AND r.permissions->>'settings' = 'write'
			  AND u.deleted_at IS NULL
		)`, teamID, roleID,
	).Scan(&othersHaveSettingsWrite)
	if err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: check other settings admins: %w", err)
	}
	if !othersHaveSettingsWrite {
		var deletedRoleHasSettingsWrite bool
		err = tx.QueryRow(ctx,
			`SELECT permissions->>'settings' = 'write' FROM roles WHERE id = $1 AND team_id = $2`,
			roleID, teamID,
		).Scan(&deletedRoleHasSettingsWrite)
		if err != nil {
			return fmt.Errorf("roles.Repository.DeleteRole: check own settings write: %w", err)
		}
		if deletedRoleHasSettingsWrite {
			return ErrLastSettingsAdmin
		}
	}

	tag, err := tx.Exec(ctx, `DELETE FROM roles WHERE id = $1 AND team_id = $2`, roleID, teamID)
	if err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	if err := scrubDanglingRoleReferences(ctx, tx, roleID, teamID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: commit: %w", err)
	}
	return nil
}

// scrubDanglingRoleReferences removes roleID from every plain UUID[] column
// that references roles with no FK: teams.reason_visibility_role_ids and
// events/event_series.nominated_role_ids. Without this, deleting a role
// referenced there leaves a permanently dangling ID with no error or
// indication. Runs inside the caller's transaction so it's atomic with the
// role deletion itself.
func scrubDanglingRoleReferences(ctx context.Context, tx pgx.Tx, roleID, teamID string) error {
	if _, err := tx.Exec(ctx,
		`UPDATE teams SET reason_visibility_role_ids = array_remove(reason_visibility_role_ids, $1) WHERE id = $2`,
		roleID, teamID,
	); err != nil {
		return fmt.Errorf("roles.Repository: scrub reason_visibility_role_ids: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE events SET nominated_role_ids = array_remove(nominated_role_ids, $1) WHERE team_id = $2`,
		roleID, teamID,
	); err != nil {
		return fmt.Errorf("roles.Repository: scrub events.nominated_role_ids: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE event_series SET nominated_role_ids = array_remove(nominated_role_ids, $1) WHERE team_id = $2`,
		roleID, teamID,
	); err != nil {
		return fmt.Errorf("roles.Repository: scrub event_series.nominated_role_ids: %w", err)
	}
	return nil
}

// RolesExistForTeam returns true when every ID in roleIDs is a role belonging
// to teamID. An empty roleIDs slice always returns true.
func (r *Repository) RolesExistForTeam(ctx context.Context, teamID string, roleIDs []uuid.UUID) (bool, error) {
	if len(roleIDs) == 0 {
		return true, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM roles WHERE team_id = $1 AND id = ANY($2)`,
		teamID, roleIDs,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("roles.Repository.RolesExistForTeam: %w", err)
	}
	// COUNT(*) counts matching rows (one per distinct role), not input array
	// elements -- compare against the distinct id count so a request that
	// legitimately repeats the same valid role ID isn't wrongly rejected.
	seen := make(map[uuid.UUID]struct{}, len(roleIDs))
	for _, id := range roleIDs {
		seen[id] = struct{}{}
	}
	return count == len(seen), nil
}

// ─── internal helpers ─────────────────────────────────────────────────────────

func (r *Repository) getRoleByID(ctx context.Context, roleID, teamID string) (*teams.RoleRow, error) {
	rr := &teams.RoleRow{}
	var permBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, team_id, name, system, color, permissions
		FROM roles WHERE id = $1 AND team_id = $2
	`, roleID, teamID).Scan(&rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permBytes)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(permBytes, &rr.Permissions); err != nil {
		return nil, fmt.Errorf("roles.Repository.getRoleByID: unmarshal: %w", err)
	}
	return rr, nil
}

func scanRole(row interface{ Scan(dest ...any) error }) (*teams.RoleRow, error) {
	rr := &teams.RoleRow{}
	var permBytes []byte
	err := row.Scan(&rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permBytes)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	if err := json.Unmarshal(permBytes, &rr.Permissions); err != nil {
		return nil, fmt.Errorf("unmarshal permissions: %w", err)
	}
	return rr, nil
}
