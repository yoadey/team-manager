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

// ErrRoleNotInTeam is returned when one or more role IDs passed to UpdateTeam
// (via ReasonVisibilityRoleIDs) do not belong to the team being updated.
var ErrRoleNotInTeam = errors.New("role does not belong to team")

// ErrInviteNotFound is returned by AcceptInvite when no non-expired invite
// matches the given code.
var ErrInviteNotFound = errors.New("invite not found or expired")

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
	(t.photo_object_key IS NOT NULL),
	(t.logo_object_key IS NOT NULL),
	t.description, t.reason_visibility_role_ids, t.created_at
`

func scanTeam(row interface{ Scan(dest ...any) error }) (*TeamRow, error) {
	tr := &TeamRow{}
	err := row.Scan(
		&tr.Id, &tr.Name, &tr.Short, &tr.Icon, &tr.IconBg, &tr.IconFg,
		&tr.HasPhoto,
		&tr.HasLogo,
		&tr.Description, &tr.ReasonVisibilityRoleIDs, &tr.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return tr, nil
}

// GetTeamPhotoKey returns the object-store key for teamID's photo, or
// pgx.ErrNoRows if the team has no photo set (or does not exist). Kept
// separate from GetTeam (which only exposes a HasPhoto boolean) so a lookup
// that actually needs the key doesn't have to duplicate the query.
func (r *Repository) GetTeamPhotoKey(ctx context.Context, teamID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var key *string
	err := r.pool.QueryRow(ctx, `SELECT photo_object_key FROM teams WHERE id = $1`, teamID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("teams.Repository.GetTeamPhotoKey: %w", err)
	}
	if key == nil || *key == "" {
		return "", pgx.ErrNoRows
	}
	return *key, nil
}

// GetTeamLogoKey returns the object-store key for teamID's logo, or
// pgx.ErrNoRows if the team has no logo set (or does not exist).
func (r *Repository) GetTeamLogoKey(ctx context.Context, teamID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var key *string
	err := r.pool.QueryRow(ctx, `SELECT logo_object_key FROM teams WHERE id = $1`, teamID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("teams.Repository.GetTeamLogoKey: %w", err)
	}
	if key == nil || *key == "" {
		return "", pgx.ErrNoRows
	}
	return *key, nil
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
// (Admin with all-write, Member with events/members/news/polls/settings read).
// icon/iconBg/iconFg are optional (nil leaves the column at its DB default,
// NULL) -- UpdateTeam already lets a caller set these after the fact, but the
// frontend's create-team form collects them upfront and expects CreateTeam
// itself to persist what was submitted, not silently discard it.
func (r *Repository) CreateTeam(ctx context.Context, name, creatorUserID string, icon, iconBg, iconFg *string) (*TeamRow, error) {
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
		INSERT INTO teams (name, icon, icon_bg, icon_fg)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, short, icon, icon_bg, icon_fg,
		          (photo_object_key IS NOT NULL),
		          (logo_object_key IS NOT NULL),
		          description, reason_visibility_role_ids, created_at
	`, name, icon, iconBg, iconFg).Scan(
		&tr.Id, &tr.Name, &tr.Short, &tr.Icon, &tr.IconBg, &tr.IconFg,
		&tr.HasPhoto,
		&tr.HasLogo,
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

	// Insert Member role (events/news/polls read). Members and Settings are
	// also "read" (not "none") -- RequirePermission gates GET requests too,
	// and a module set to "none" hides reads entirely (see authz.go), so
	// "none" here would 403 every ordinary member's own dashboard load:
	// AppContext.afterLoginLoad unconditionally fetches both the member
	// roster (GET .../members) and the role catalog (GET .../roles, gated by
	// "settings") for every team member on every login/team switch, not just
	// for admins. Finances stays "none" -- financial data is legitimately
	// admin-only by default.
	memberPerms, _ := json.Marshal(PermissionsJSON{
		Events: "read", Members: "read", Finances: "none",
		News: "read", Polls: "read", Settings: "read",
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

// parseAndValidateTeamRoleIDs parses ids as UUIDs and verifies every one
// belongs to teamID, returning ErrRoleNotInTeam otherwise. Takes tx (not the
// pool) so the check runs inside the caller's transaction, after it holds the
// team's advisory lock -- otherwise a role could be deleted between this
// check and the caller's later write, leaving a dangling reference.
func parseAndValidateTeamRoleIDs(ctx context.Context, tx pgx.Tx, teamID string, ids []string) ([]uuid.UUID, error) {
	uids := make([]uuid.UUID, len(ids))
	for i, s := range ids {
		u, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("teams.Repository: invalid role id %q: %w", s, err)
		}
		uids[i] = u
	}
	if len(uids) == 0 {
		return uids, nil
	}
	var count int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM roles WHERE id = ANY($1) AND team_id = $2`,
		uids, teamID,
	).Scan(&count); err != nil {
		return nil, fmt.Errorf("teams.Repository: check roles: %w", err)
	}
	// COUNT(*) counts matching rows (one per distinct role), not input array
	// elements -- compare against the distinct id count so a request that
	// legitimately repeats the same valid role ID isn't wrongly rejected.
	seen := make(map[uuid.UUID]struct{}, len(uids))
	for _, u := range uids {
		seen[u] = struct{}{}
	}
	if count != len(seen) {
		return nil, ErrRoleNotInTeam
	}
	return uids, nil
}

// buildSimpleTeamPatchClauses builds the SET clauses/args for TeamPatch
// fields that need no extra validation, returning the next free placeholder
// index. ReasonVisibilityRoleIDs is handled separately by the caller since it
// needs a transaction to validate against.
func buildSimpleTeamPatchClauses(patch TeamPatch) (setClauses []string, args []any, argN int) {
	argN = 1
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
	return setClauses, args, argN
}

// UpdateTeam applies a partial update to the teams row and returns the updated row.
func (r *Repository) UpdateTeam(ctx context.Context, teamID string, patch TeamPatch) (*TeamRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.UpdateTeam: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if len(patch.ReasonVisibilityRoleIDs) > 0 {
		// Uses the same advisory lock key as members.SetRoles/roles.DeleteRole
		// (hashtextextended(teamID, 0)) so validating these role IDs can't
		// race with a concurrent DeleteRole -- otherwise a role could be
		// deleted (and scrubbed from this same array) between the check
		// below and this UPDATE's commit, re-introducing a dangling
		// reference right after DeleteRole just removed it. Skipped when the
		// caller is clearing the list (a non-nil but empty slice) -- an empty
		// array has nothing to validate against a concurrent role deletion,
		// mirroring events.validateNominatedRolesInTx's same short-circuit.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
			return nil, fmt.Errorf("teams.Repository.UpdateTeam: advisory lock: %w", err)
		}
	}

	// Build a dynamic SET clause.
	setClauses, args, argN := buildSimpleTeamPatchClauses(patch)

	if patch.ReasonVisibilityRoleIDs != nil {
		uids, err := parseAndValidateTeamRoleIDs(ctx, tx, teamID, patch.ReasonVisibilityRoleIDs)
		if err != nil {
			return nil, err
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
		          (photo_object_key IS NOT NULL),
		          (logo_object_key IS NOT NULL),
		          description, reason_visibility_role_ids, created_at
	`, setSQL, argN)

	var tr TeamRow
	err = tx.QueryRow(ctx, q, args...).Scan(
		&tr.Id, &tr.Name, &tr.Short, &tr.Icon, &tr.IconBg, &tr.IconFg,
		&tr.HasPhoto,
		&tr.HasLogo,
		&tr.Description, &tr.ReasonVisibilityRoleIDs, &tr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("teams.Repository.UpdateTeam: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("teams.Repository.UpdateTeam: commit: %w", err)
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

// GetRolesForMembership returns all roles assigned to the given membership
// that belong to teamID (defense in depth against a membership_roles row
// ever pointing at a role from a different team).
func (r *Repository) GetRolesForMembership(ctx context.Context, membershipID, teamID string) ([]RoleRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.team_id, r.name, r.system, r.color, r.permissions
		FROM roles r
		JOIN membership_roles mr ON mr.role_id = r.id
		WHERE mr.membership_id = $1 AND r.team_id = $2
	`, membershipID, teamID)
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

// GetMemberCounts returns the member count for each of the given team IDs, in
// one query rather than one round trip per team (used by ListTeamsForUser to
// avoid an N+1 pattern when a user belongs to many teams).
func (r *Repository) GetMemberCounts(ctx context.Context, teamIDs []string) (map[string]int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT team_id, COUNT(*)
		FROM memberships
		WHERE team_id = ANY($1)
		GROUP BY team_id
	`, teamIDs)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.GetMemberCounts: %w", err)
	}
	defer rows.Close()

	out := make(map[string]int, len(teamIDs))
	for rows.Next() {
		var teamID string
		var count int
		if err := rows.Scan(&teamID, &count); err != nil {
			return nil, fmt.Errorf("teams.Repository.GetMemberCounts scan: %w", err)
		}
		out[teamID] = count
	}
	return out, rows.Err()
}

// GetMembershipsForUser returns the given user's membership row for each of
// the given team IDs, keyed by team ID (see GetMemberCounts for why this is
// batched rather than called once per team).
func (r *Repository) GetMembershipsForUser(ctx context.Context, teamIDs []string, userID string) (map[string]MembershipRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, team_id, user_id, "group", joined_at
		FROM memberships
		WHERE team_id = ANY($1) AND user_id = $2
	`, teamIDs, userID)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.GetMembershipsForUser: %w", err)
	}
	defer rows.Close()

	out := make(map[string]MembershipRow, len(teamIDs))
	for rows.Next() {
		var m MembershipRow
		if err := rows.Scan(&m.Id, &m.TeamID, &m.UserID, &m.Group, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("teams.Repository.GetMembershipsForUser scan: %w", err)
		}
		out[m.TeamID.String()] = m
	}
	return out, rows.Err()
}

// GetRolesForMemberships returns the roles assigned to each of the given
// membership IDs, keyed by membership ID (see GetMemberCounts for why this is
// batched rather than called once per membership).
func (r *Repository) GetRolesForMemberships(ctx context.Context, membershipIDs []string) (map[string][]RoleRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT mr.membership_id, r.id, r.team_id, r.name, r.system, r.color, r.permissions
		FROM roles r
		JOIN membership_roles mr ON mr.role_id = r.id
		JOIN memberships m ON m.id = mr.membership_id
		WHERE mr.membership_id = ANY($1) AND r.team_id = m.team_id
	`, membershipIDs)
	if err != nil {
		return nil, fmt.Errorf("teams.Repository.GetRolesForMemberships: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]RoleRow, len(membershipIDs))
	for rows.Next() {
		var membershipID string
		rr := RoleRow{}
		var permJSON []byte
		if err := rows.Scan(&membershipID, &rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permJSON); err != nil {
			return nil, fmt.Errorf("teams.Repository.GetRolesForMemberships scan: %w", err)
		}
		if err := json.Unmarshal(permJSON, &rr.Permissions); err != nil {
			return nil, fmt.Errorf("teams.Repository.GetRolesForMemberships unmarshal permissions: %w", err)
		}
		out[membershipID] = append(out[membershipID], rr)
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

// AcceptInvite redeems a non-expired invite code, adding userID as a member of
// its team. Idempotent: redeeming a code for a team the user already belongs
// to is a no-op that still returns that team, and only a brand-new membership
// gets the default system "Member" role -- an admin who has since stripped a
// re-joining member's roles must not have that silently undone by the member
// re-clicking their old invite link.
// AcceptInvite's second return value reports whether the caller was already
// a member of the team before this call (a no-op join-wise), so the caller
// can distinguish that from an actual new join (e.g. to avoid showing a
// misleading "joined" toast on a repeat visit to an old invite link).
func (r *Repository) AcceptInvite(ctx context.Context, code, userID string) (*TeamRow, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("teams.Repository.AcceptInvite: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var teamID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT team_id FROM invites WHERE code = $1 AND expires_at > now()
	`, code).Scan(&teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrInviteNotFound
		}
		return nil, false, fmt.Errorf("teams.Repository.AcceptInvite: lookup invite: %w", err)
	}

	// Same advisory lock key as every other team-mutating path (roles
	// deletion, CreateEvent/UpdateEvent nominations, UpdateTeam), so a
	// concurrent role deletion can't race the default-role assignment below.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID.String()); err != nil {
		return nil, false, fmt.Errorf("teams.Repository.AcceptInvite: advisory lock: %w", err)
	}

	var membershipID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, userID).
		Scan(&membershipID)
	isNewMembership := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !isNewMembership {
		return nil, false, fmt.Errorf("teams.Repository.AcceptInvite: check membership: %w", err)
	}

	if isNewMembership {
		err = tx.QueryRow(ctx, `
			INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id
		`, teamID, userID).Scan(&membershipID)
		if err != nil {
			return nil, false, fmt.Errorf("teams.Repository.AcceptInvite: create membership: %w", err)
		}

		var memberRoleID uuid.UUID
		err = tx.QueryRow(ctx, `
			SELECT id FROM roles WHERE team_id = $1 AND system = true AND name = 'Member'
		`, teamID).Scan(&memberRoleID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, false, fmt.Errorf("teams.Repository.AcceptInvite: find member role: %w", err)
		}
		if err == nil {
			if _, err = tx.Exec(ctx, `
				INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)
			`, membershipID, memberRoleID); err != nil {
				return nil, false, fmt.Errorf("teams.Repository.AcceptInvite: assign member role: %w", err)
			}
		}
	}

	var tr TeamRow
	err = tx.QueryRow(ctx, `
		SELECT id, name, short, icon, icon_bg, icon_fg,
		       (photo_object_key IS NOT NULL),
		       (logo_object_key IS NOT NULL),
		       description, reason_visibility_role_ids, created_at
		FROM teams WHERE id = $1
	`, teamID).Scan(
		&tr.Id, &tr.Name, &tr.Short, &tr.Icon, &tr.IconBg, &tr.IconFg,
		&tr.HasPhoto,
		&tr.HasLogo,
		&tr.Description, &tr.ReasonVisibilityRoleIDs, &tr.CreatedAt,
	)
	if err != nil {
		return nil, false, fmt.Errorf("teams.Repository.AcceptInvite: fetch team: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("teams.Repository.AcceptInvite: commit: %w", err)
	}
	return &tr, !isNewMembership, nil
}

// UpdateTeamPhoto stores the object-store key for the given team's photo.
func (r *Repository) UpdateTeamPhoto(ctx context.Context, teamID, objectKey string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(
		ctx,
		`UPDATE teams SET photo_object_key = $2 WHERE id = $1`,
		teamID, objectKey,
	)
	if err != nil {
		return fmt.Errorf("teams.Repository.UpdateTeamPhoto: %w", err)
	}
	return nil
}

// UpdateTeamLogo stores the object-store key for the given team's logo.
func (r *Repository) UpdateTeamLogo(ctx context.Context, teamID, objectKey string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(
		ctx,
		`UPDATE teams SET logo_object_key = $2 WHERE id = $1`,
		teamID, objectKey,
	)
	if err != nil {
		return fmt.Errorf("teams.Repository.UpdateTeamLogo: %w", err)
	}
	return nil
}

// DeleteTeamPhoto clears the stored photo key for the given team, reverting
// display to the icon fallback. Returns pgx.ErrNoRows if teamID doesn't exist.
func (r *Repository) DeleteTeamPhoto(ctx context.Context, teamID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `UPDATE teams SET photo_object_key = NULL WHERE id = $1`, teamID)
	if err != nil {
		return fmt.Errorf("teams.Repository.DeleteTeamPhoto: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteTeamLogo clears the stored logo key for the given team, reverting
// display to the icon fallback. Returns pgx.ErrNoRows if teamID doesn't exist.
func (r *Repository) DeleteTeamLogo(ctx context.Context, teamID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `UPDATE teams SET logo_object_key = NULL WHERE id = $1`, teamID)
	if err != nil {
		return fmt.Errorf("teams.Repository.DeleteTeamLogo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ─── Role query helpers ───────────────────────────────────────────────────────

func scanRole(row interface{ Scan(dest ...any) error }) (*RoleRow, error) {
	rr := &RoleRow{}
	var permJSON []byte
	err := row.Scan(&rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permJSON)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
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
