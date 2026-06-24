package teams

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yoadey/team-manager/backend/internal/gen"
)

// Repository handles all team-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// TeamPatch carries optional fields for an UPDATE teams query.
type TeamPatch struct {
	Name                    *string
	Short                   *string
	Icon                    *string
	IconBg                  *string
	IconFg                  *string
	Description             *string
	ReasonVisibilityRoleIDs []string
}

// ─── Team queries ─────────────────────────────────────────────────────────────

const selectTeamFields = `
	t.id, t.name, t.short, t.icon, t.icon_bg, t.icon_fg,
	COALESCE(t.photo_data, ''::bytea), t.photo_mime,
	COALESCE(t.logo_data, ''::bytea), t.logo_mime,
	t.description, t.reason_visibility_role_ids, t.created_at
`

func scanTeam(row interface{ Scan(dest ...any) error }) (*TeamRow, error) {
	tr := &TeamRow{}
	err := row.Scan(
		&tr.Id, &tr.Name, &tr.Short, &tr.Icon, &tr.IconBg, &tr.IconFg,
		&tr.PhotoData, &tr.PhotoMime,
		&tr.LogoData, &tr.LogoMime,
		&tr.Description, &tr.ReasonVisibilityRoleIDs, &tr.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// ListTeamsForUser returns all teams the given user is a member of.
func (r *Repository) ListTeamsForUser(ctx context.Context, userID string) ([]TeamRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`
		SELECT %s
		FROM teams t
		JOIN memberships m ON m.team_id = t.id
		WHERE m.user_id = $1
		ORDER BY t.name
	`, selectTeamFields)

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.ListTeamsForUser: %w", err)
	}
	defer rows.Close()

	var out []TeamRow
	for rows.Next() {
		tr, err := scanTeam(rows)
		if err != nil {
			return nil, fmt.Errorf("teams.Repository.ListTeamsForUser scan: %w", err)
		}
		out = append(out, *tr)
	}
	return out, rows.Err()
}

// GetTeam returns the team with the given ID or (nil, pgx.ErrNoRows) if not found.
func (r *Repository) GetTeam(ctx context.Context, teamID string) (*TeamRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`SELECT %s FROM teams t WHERE t.id = $1`, selectTeamFields)
	row := r.pool.QueryRow(ctx, q, teamID)
	tr, err := scanTeam(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("teams.Repository.GetTeam: %w", err)
	}
	return tr, nil
}

// CreateTeam inserts a new team, a membership for the creator, and two default roles
// (Admin with all-write, Member with events/news/polls read).
func (r *Repository) CreateTeam(ctx context.Context, name string, creatorUserID string) (*TeamRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.CreateTeam: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert team.
	var tr TeamRow
	err = tx.QueryRow(ctx, `
		INSERT INTO teams (name)
		VALUES ($1)
		RETURNING id, name, short, icon, icon_bg, icon_fg,
		          COALESCE(photo_data, ''::bytea), photo_mime,
		          COALESCE(logo_data, ''::bytea), logo_mime,
		          description, reason_visibility_role_ids, created_at
	`, name).Scan(
		&tr.Id, &tr.Name, &tr.Short, &tr.Icon, &tr.IconBg, &tr.IconFg,
		&tr.PhotoData, &tr.PhotoMime,
		&tr.LogoData, &tr.LogoMime,
		&tr.Description, &tr.ReasonVisibilityRoleIDs, &tr.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.CreateTeam insert team: %w", err)
	}

	// Insert membership.
	var membershipID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO memberships (team_id, user_id)
		VALUES ($1, $2)
		RETURNING id
	`, tr.Id, creatorUserID).Scan(&membershipID)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.CreateTeam insert membership: %w", err)
	}

	// Insert Admin role (all write).
	adminPerms, _ := json.Marshal(PermissionsJSON{
		Events: "write", Members: "write", Finances: "write",
		News: "write", Polls: "write", Settings: "write",
	})
	var adminRoleID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO roles (team_id, name, system, permissions)
		VALUES ($1, 'Admin', true, $2)
		RETURNING id
	`, tr.Id, adminPerms).Scan(&adminRoleID)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.CreateTeam insert admin role: %w", err)
	}

	// Insert Member role (events/news/polls read).
	memberPerms, _ := json.Marshal(PermissionsJSON{
		Events: "read", Members: "none", Finances: "none",
		News: "read", Polls: "read", Settings: "none",
	})
	var memberRoleID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO roles (team_id, name, system, permissions)
		VALUES ($1, 'Member', true, $2)
		RETURNING id
	`, tr.Id, memberPerms).Scan(&memberRoleID)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.CreateTeam insert member role: %w", err)
	}

	// Assign Admin role to creator.
	_, err = tx.Exec(ctx, `
		INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)
	`, membershipID, adminRoleID)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.CreateTeam assign admin role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("teams.Repository.CreateTeam commit: %w", err)
	}
	return &tr, nil
}

// UpdateTeam applies a partial update to the teams row and returns the updated row.
func (r *Repository) UpdateTeam(ctx context.Context, teamID string, patch TeamPatch) (*TeamRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Build a dynamic SET clause.
	setClauses := []string{}
	args := []any{}
	argN := 1

	if patch.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argN))
		args = append(args, *patch.Name)
		argN++
	}
	if patch.Short != nil {
		setClauses = append(setClauses, fmt.Sprintf("short = $%d", argN))
		args = append(args, *patch.Short)
		argN++
	}
	if patch.Icon != nil {
		setClauses = append(setClauses, fmt.Sprintf("icon = $%d", argN))
		args = append(args, *patch.Icon)
		argN++
	}
	if patch.IconBg != nil {
		setClauses = append(setClauses, fmt.Sprintf("icon_bg = $%d", argN))
		args = append(args, *patch.IconBg)
		argN++
	}
	if patch.IconFg != nil {
		setClauses = append(setClauses, fmt.Sprintf("icon_fg = $%d", argN))
		args = append(args, *patch.IconFg)
		argN++
	}
	if patch.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argN))
		args = append(args, *patch.Description)
		argN++
	}
	if patch.ReasonVisibilityRoleIDs != nil {
		uids := make([]uuid.UUID, len(patch.ReasonVisibilityRoleIDs))
		for i, s := range patch.ReasonVisibilityRoleIDs {
			u, err := uuid.Parse(s)
			if err != nil {
				return nil, fmt.Errorf("teams.Repository.UpdateTeam: invalid role id %q: %w", s, err)
			}
			uids[i] = u
		}
		setClauses = append(setClauses, fmt.Sprintf("reason_visibility_role_ids = $%d", argN))
		args = append(args, uids)
		argN++
	}

	if len(setClauses) == 0 {
		// Nothing to update — just return the current row.
		return r.GetTeam(ctx, teamID)
	}

	// Build SQL: SET col=$1, col=$2 ... WHERE id=$N
	setSQL := ""
	for i, c := range setClauses {
		if i > 0 {
			setSQL += ", "
		}
		setSQL += c
	}
	args = append(args, teamID)
	q := fmt.Sprintf(`
		UPDATE teams SET %s WHERE id = $%d
		RETURNING id, name, short, icon, icon_bg, icon_fg,
		          COALESCE(photo_data, ''::bytea), photo_mime,
		          COALESCE(logo_data, ''::bytea), logo_mime,
		          description, reason_visibility_role_ids, created_at
	`, setSQL, argN)

	var tr TeamRow
	err := r.pool.QueryRow(ctx, q, args...).Scan(
		&tr.Id, &tr.Name, &tr.Short, &tr.Icon, &tr.IconBg, &tr.IconFg,
		&tr.PhotoData, &tr.PhotoMime,
		&tr.LogoData, &tr.LogoMime,
		&tr.Description, &tr.ReasonVisibilityRoleIDs, &tr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("teams.Repository.UpdateTeam: %w", err)
	}
	return &tr, nil
}

// GetMemberCount returns the number of members in the given team.
func (r *Repository) GetMemberCount(ctx context.Context, teamID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM memberships WHERE team_id = $1`, teamID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("teams.Repository.GetMemberCount: %w", err)
	}
	return count, nil
}

// GetMembership returns the membership for the given team+user pair.
func (r *Repository) GetMembership(ctx context.Context, teamID, userID string) (*MembershipRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	m := &MembershipRow{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, team_id, user_id, "group", joined_at
		FROM memberships
		WHERE team_id = $1 AND user_id = $2
	`, teamID, userID).Scan(&m.Id, &m.TeamID, &m.UserID, &m.Group, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("teams.Repository.GetMembership: %w", err)
	}
	return m, nil
}

// GetRolesForMembership returns all roles assigned to the given membership.
func (r *Repository) GetRolesForMembership(ctx context.Context, membershipID string) ([]RoleRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.team_id, r.name, r.system, r.color, r.permissions
		FROM roles r
		JOIN membership_roles mr ON mr.role_id = r.id
		WHERE mr.membership_id = $1
	`, membershipID)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.GetRolesForMembership: %w", err)
	}
	defer rows.Close()

	var out []RoleRow
	for rows.Next() {
		rr, err := scanRole(rows)
		if err != nil {
			return nil, fmt.Errorf("teams.Repository.GetRolesForMembership scan: %w", err)
		}
		out = append(out, *rr)
	}
	return out, rows.Err()
}

// MergePermissions computes the highest permission level across all roles per module.
func MergePermissions(roles []RoleRow) gen.Permissions {
	order := map[string]int{"none": 0, "read": 1, "write": 2}

	best := PermissionsJSON{
		Events: "none", Members: "none", Finances: "none",
		News: "none", Polls: "none", Settings: "none",
	}

	merge := func(current, candidate string) string {
		if order[candidate] > order[current] {
			return candidate
		}
		return current
	}

	for _, r := range roles {
		best.Events = merge(best.Events, r.Permissions.Events)
		best.Members = merge(best.Members, r.Permissions.Members)
		best.Finances = merge(best.Finances, r.Permissions.Finances)
		best.News = merge(best.News, r.Permissions.News)
		best.Polls = merge(best.Polls, r.Permissions.Polls)
		best.Settings = merge(best.Settings, r.Permissions.Settings)
	}

	return gen.Permissions{
		Events:   gen.PermLevel(best.Events),
		Members:  gen.PermLevel(best.Members),
		Finances: gen.PermLevel(best.Finances),
		News:     gen.PermLevel(best.News),
		Polls:    gen.PermLevel(best.Polls),
		Settings: gen.PermLevel(best.Settings),
	}
}

// CreateInvite inserts an invite row with a random code and the given TTL.
func (r *Repository) CreateInvite(ctx context.Context, teamID string, ttl time.Duration) (*InviteRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	code, err := generateCode()
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.CreateInvite: generate code: %w", err)
	}
	expiresAt := time.Now().Add(ttl)

	inv := &InviteRow{}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO invites (team_id, code, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, team_id, code, expires_at, created_at
	`, teamID, code, expiresAt).Scan(
		&inv.Id, &inv.TeamID, &inv.Code, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.CreateInvite: %w", err)
	}
	return inv, nil
}

// UpdateTeamPhoto stores raw photo bytes and MIME type for the given team.
func (r *Repository) UpdateTeamPhoto(ctx context.Context, teamID string, data []byte, mime string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx,
		`UPDATE teams SET photo_data = $2, photo_mime = $3 WHERE id = $1`,
		teamID, data, mime,
	)
	if err != nil {
		return fmt.Errorf("teams.Repository.UpdateTeamPhoto: %w", err)
	}
	return nil
}

// ─── Role query helpers ───────────────────────────────────────────────────────

func scanRole(row interface{ Scan(dest ...any) error }) (*RoleRow, error) {
	rr := &RoleRow{}
	var permJSON []byte
	err := row.Scan(&rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permJSON)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(permJSON, &rr.Permissions); err != nil {
		return nil, fmt.Errorf("unmarshal permissions: %w", err)
	}
	return rr, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func generateCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
