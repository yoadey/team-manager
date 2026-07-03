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
// (ErrSystemRole), and rejects revoking settings:write from a role's
// permissions when no other role held by any member would still grant it
// (ErrLastSettingsAdmin) — the same invariant DeleteRole enforces for
// deleting a role outright, applied here to editing one.
func (r *Repository) guardRoleUpdate(ctx context.Context, tx pgx.Tx, roleID, teamID string, patch RolePatch) error {
	// Same advisory lock key as DeleteRole/members.SetRoles/RemoveMember,
	// serializing this against concurrent role/assignment changes guarding
	// the same invariant.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
		return fmt.Errorf("roles.Repository.UpdateRole: advisory lock: %w", err)
	}

	var isSystem bool
	if err := tx.QueryRow(ctx, `SELECT system FROM roles WHERE id = $1 AND team_id = $2`, roleID, teamID).Scan(&isSystem); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return fmt.Errorf("roles.Repository.UpdateRole: check system: %w", err)
	}
	if isSystem {
		return ErrSystemRole
	}

	if patch.Permissions == nil || patch.Permissions.Settings == "write" {
		return nil
	}

	var othersHaveSettingsWrite bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM memberships m
			JOIN membership_roles mr ON mr.membership_id = m.id
			JOIN roles r ON r.id = mr.role_id
			WHERE m.team_id = $1 AND r.id != $2 AND r.permissions->>'settings' = 'write'
		)`, teamID, roleID,
	).Scan(&othersHaveSettingsWrite); err != nil {
		return fmt.Errorf("roles.Repository.UpdateRole: check other settings admins: %w", err)
	}
	if othersHaveSettingsWrite {
		return nil
	}

	var currentlyHasSettingsWrite bool
	if err := tx.QueryRow(ctx,
		`SELECT permissions->>'settings' = 'write' FROM roles WHERE id = $1 AND team_id = $2`,
		roleID, teamID,
	).Scan(&currentlyHasSettingsWrite); err != nil {
		return fmt.Errorf("roles.Repository.UpdateRole: check own settings write: %w", err)
	}
	if currentlyHasSettingsWrite {
		return ErrLastSettingsAdmin
	}
	return nil
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
func (r *Repository) UpdateRole(ctx context.Context, roleID, teamID string, patch RolePatch) (*teams.RoleRow, error) {
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
		if err := r.guardRoleUpdate(ctx, tx, roleID, teamID, patch); err != nil {
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

	var othersHaveSettingsWrite bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM memberships m
			JOIN membership_roles mr ON mr.membership_id = m.id
			JOIN roles r ON r.id = mr.role_id
			WHERE m.team_id = $1 AND r.id != $2 AND r.permissions->>'settings' = 'write'
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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: commit: %w", err)
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
	return count == len(roleIDs), nil
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
