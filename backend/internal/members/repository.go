package members

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

// Repository handles member-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListMembers returns all members of a team with their roles.
func (r *Repository) ListMembers(ctx context.Context, teamID string, limit, offset int) ([]MemberRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// First, get all memberships + user data for the team.
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, u.id, u.name, u.email, u.phone,
		       u.birthday, u.address, u.avatar_color,
		       COALESCE(u.photo_data, ''::bytea),
		       m."group", m.joined_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.team_id = $1
		ORDER BY u.name
		LIMIT $2 OFFSET $3
	`, teamID, limit, offset)
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

	// For each membership, load roles.
	for i := range members {
		roles, err := r.getRolesForMembership(ctx, members[i].MembershipID.String())
		if err != nil {
			return nil, err
		}
		members[i].Roles = roles
	}

	return members, nil
}

// AddMember inserts a user (if not exists by email), creates membership and assigns roles.
func (r *Repository) AddMember(ctx context.Context, teamID string, params AddMemberParams) (*MemberRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.AddMember: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Upsert user: find by email or insert.
	var userID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM users WHERE email = $1
	`, params.Email).Scan(&userID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("members.Repository.AddMember: find user: %w", err)
		}
		// User not found, create.
		err = tx.QueryRow(ctx, `
			INSERT INTO users (name, email, phone, avatar_color)
			VALUES ($1, $2, $3, '#6366f1')
			RETURNING id
		`, params.Name, params.Email, params.Phone).Scan(&userID)
		if err != nil {
			return nil, fmt.Errorf("members.Repository.AddMember: create user: %w", err)
		}
	}

	// Insert membership.
	var membershipID string
	err = tx.QueryRow(ctx, `
		INSERT INTO memberships (team_id, user_id, "group")
		VALUES ($1, $2, $3)
		RETURNING id
	`, teamID, userID, params.Group).Scan(&membershipID)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.AddMember: create membership: %w", err)
	}

	// Assign roles.
	for _, roleID := range params.RoleIDs {
		_, err = tx.Exec(ctx, `
			INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)
		`, membershipID, roleID)
		if err != nil {
			return nil, fmt.Errorf("members.Repository.AddMember: assign role %s: %w", roleID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("members.Repository.AddMember: commit: %w", err)
	}

	return r.getMemberByMembershipID(ctx, membershipID)
}

// UpdateMember applies a partial update to the user fields and optionally the group.
func (r *Repository) UpdateMember(ctx context.Context, membershipID string, patch MemberPatch) (*MemberRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Get user_id from membership first.
	var userID string
	err := r.pool.QueryRow(ctx, `SELECT user_id FROM memberships WHERE id = $1`, membershipID).Scan(&userID)
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
			return nil, fmt.Errorf("members.Repository.UpdateMember: update user: %w", err)
		}
	}

	// Update membership group.
	if patch.Group != nil {
		_, err = tx.Exec(ctx, `UPDATE memberships SET "group" = $1 WHERE id = $2`, *patch.Group, membershipID)
		if err != nil {
			return nil, fmt.Errorf("members.Repository.UpdateMember: update membership: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("members.Repository.UpdateMember: commit: %w", err)
	}

	return r.getMemberByMembershipID(ctx, membershipID)
}

// SetRoles replaces the role assignments for the given membership.
func (r *Repository) SetRoles(ctx context.Context, membershipID string, roleIDs []string) (*MemberRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("members.Repository.SetRoles: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

// RemoveMember deletes a membership (cascades membership_roles).
func (r *Repository) RemoveMember(ctx context.Context, membershipID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, `DELETE FROM memberships WHERE id = $1`, membershipID)
	if err != nil {
		return fmt.Errorf("members.Repository.RemoveMember: %w", err)
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
		WHERE m.team_id = $1 AND m.user_id = $2
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
		       COALESCE(u.photo_data, ''::bytea),
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
		&mr.PhotoData,
		&mr.Group, &mr.JoinedAt,
	)
	if err != nil {
		return nil, err
	}
	return mr, nil
}

// ensure uuid is used
var _ = uuid.UUID{}
