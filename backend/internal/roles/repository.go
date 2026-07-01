package roles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ErrSystemRole is returned when attempting to delete a system role.
var ErrSystemRole = errors.New("cannot delete system role")

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

// UpdateRole applies a partial update to a role that belongs to teamID.
func (r *Repository) UpdateRole(ctx context.Context, roleID, teamID string, patch RolePatch) (*teams.RoleRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setClauses := []string{}
	args := []any{}
	n := 1

	if patch.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", n))
		args = append(args, *patch.Name)
		n++
	}
	if patch.Color != nil {
		setClauses = append(setClauses, fmt.Sprintf("color = $%d", n))
		args = append(args, *patch.Color)
		n++
	}
	if patch.Permissions != nil {
		permJSON, err := json.Marshal(*patch.Permissions)
		if err != nil {
			return nil, fmt.Errorf("roles.Repository.UpdateRole: marshal permissions: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("permissions = $%d", n))
		args = append(args, permJSON)
		n++
	}

	if len(setClauses) == 0 {
		return r.getRoleByID(ctx, roleID, teamID)
	}

	setSQL := ""
	for i, c := range setClauses {
		if i > 0 {
			setSQL += ", "
		}
		setSQL += c
	}
	args = append(args, roleID, teamID)

	rr := &teams.RoleRow{}
	var permBytes []byte
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
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
	return rr, nil
}

// DeleteRole deletes a non-system role that belongs to teamID. Returns an
// error if the role is system=true, or pgx.ErrNoRows if no role with roleID
// exists within teamID.
func (r *Repository) DeleteRole(ctx context.Context, roleID, teamID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Check system flag first, scoped to the team.
	var isSystem bool
	err := r.pool.QueryRow(ctx, `SELECT system FROM roles WHERE id = $1 AND team_id = $2`, roleID, teamID).Scan(&isSystem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return fmt.Errorf("roles.Repository.DeleteRole: check system: %w", err)
	}
	if isSystem {
		return ErrSystemRole
	}

	tag, err := r.pool.Exec(ctx, `DELETE FROM roles WHERE id = $1 AND team_id = $2`, roleID, teamID)
	if err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
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
