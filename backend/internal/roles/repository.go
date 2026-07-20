package roles

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yoadey/team-manager/backend/internal/db/gen"
	"github.com/yoadey/team-manager/backend/internal/db/sqlbuilder"
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
	q    *dbgen.Queries
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: dbgen.New(pool)}
}

// textToPtr converts a nullable pgtype.Text (as returned by generated
// queries for the nullable roles.color column) into the *string the
// teams.RoleRow domain model uses.
func textToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// ptrToText converts a *string patch/create field into the pgtype.Text a
// generated query expects for the nullable roles.color column.
func ptrToText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// ListRoles returns all roles for the given team.
func (r *Repository) ListRoles(ctx context.Context, teamID string) ([]teams.RoleRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.q.ListRolesByTeam(ctx, dbgen.ListRolesByTeamParams{
		TeamID: uuid.MustParse(teamID), Limit: maxRolesPerTeam,
	})
	if err != nil {
		return nil, fmt.Errorf("roles.Repository.ListRoles: %w", err)
	}
	out := make([]teams.RoleRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, teams.RoleRow{
			Id: row.ID, TeamID: row.TeamID, Name: row.Name, System: row.System,
			Color: textToPtr(row.Color), Permissions: row.Permissions,
		})
	}
	return out, nil
}

// CountRoles returns the number of roles the team has, used to enforce
// maxRolesPerTeam before an insert.
func (r *Repository) CountRoles(ctx context.Context, teamID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	count, err := r.q.CountRoles(ctx, uuid.MustParse(teamID))
	if err != nil {
		return 0, fmt.Errorf("roles.Repository.CountRoles: %w", err)
	}
	return int(count), nil
}

// CreateRole inserts a new role for the team.
func (r *Repository) CreateRole(ctx context.Context, teamID, name string, color *string, permissions teams.PermissionsJSON) (*teams.RoleRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row, err := r.q.CreateRole(ctx, dbgen.CreateRoleParams{
		TeamID: uuid.MustParse(teamID), Name: name, Color: ptrToText(color), Permissions: permissions,
	})
	if err != nil {
		return nil, fmt.Errorf("roles.Repository.CreateRole: %w", err)
	}
	return &teams.RoleRow{
		Id: row.ID, TeamID: row.TeamID, Name: row.Name, System: row.System,
		Color: textToPtr(row.Color), Permissions: row.Permissions,
	}, nil
}

// buildRoleUpdateSets constructs a SET clause and args slice for patch via
// sqlbuilder (see its package doc for why a hand-rolled builder with a
// placeholder fallback isn't used here).
func buildRoleUpdateSets(patch RolePatch, startIdx int) (setSQL string, args []any, nextIdx int, ok bool) {
	b := sqlbuilder.New()
	if patch.Name != nil {
		b.Add("name", *patch.Name)
	}
	if patch.Color != nil {
		b.Add("color", *patch.Color)
	}
	if patch.Permissions != nil {
		b.Add("permissions", *patch.Permissions)
	}
	return b.Build(startIdx)
}

// guardRoleUpdate rejects renaming/re-permissioning a system role
// (ErrSystemRole); rejects a permissions patch that would grant, on any
// module, more than the caller's own ceiling allows (ErrInsufficientPermissionToGrant,
// see enforceNoRoleEscalation); and rejects revoking settings:write from a
// role's permissions when no other role held by any member would still
// grant it (ErrLastSettingsAdmin) — the same invariant DeleteRole enforces
// for deleting a role outright, applied here to editing one.
func (r *Repository) guardRoleUpdate(ctx context.Context, tx pgx.Tx, roleID, teamID uuid.UUID, callerUserID string, patch RolePatch) error {
	qtx := r.q.WithTx(tx)

	// Same advisory lock key as DeleteRole/members.SetRoles/RemoveMember,
	// serializing this against concurrent role/assignment changes guarding
	// the same invariant.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
		return fmt.Errorf("roles.Repository.UpdateRole: advisory lock: %w", err)
	}

	current, err := qtx.GetRoleSystemAndPermissions(ctx, dbgen.GetRoleSystemAndPermissionsParams{ID: roleID, TeamID: teamID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return fmt.Errorf("roles.Repository.UpdateRole: check system: %w", err)
	}
	if current.System {
		return ErrSystemRole
	}

	if patch.Permissions == nil {
		return nil
	}

	if err := enforceNoRoleEscalation(ctx, qtx, teamID, callerUserID, current.Permissions, *patch.Permissions); err != nil {
		return err
	}

	if patch.Permissions.Settings == "write" {
		return nil
	}

	othersHaveSettingsWrite, err := qtx.CheckOtherRolesHaveSettingsWrite(ctx, dbgen.CheckOtherRolesHaveSettingsWriteParams{TeamID: teamID, ID: roleID})
	if err != nil {
		return fmt.Errorf("roles.Repository.UpdateRole: check other settings admins: %w", err)
	}
	if othersHaveSettingsWrite {
		return nil
	}

	if current.Permissions.Settings == "write" {
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
func enforceNoRoleEscalation(ctx context.Context, qtx *dbgen.Queries, teamID uuid.UUID, callerUserID string, currentPerms, newPerms teams.PermissionsJSON) error {
	callerPerms, err := getEffectivePermissionsByUserQ(ctx, qtx, teamID, uuid.MustParse(callerUserID))
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
func getEffectivePermissionsByUserQ(ctx context.Context, qtx *dbgen.Queries, teamID, userID uuid.UUID) (teams.PermissionsJSON, error) {
	eff := teams.PermissionsJSON{Events: "none", Members: "none", Finances: "none", News: "none", Polls: "none", Settings: "none"}
	perms, err := qtx.GetEffectivePermissionsForUser(ctx, dbgen.GetEffectivePermissionsForUserParams{TeamID: teamID, UserID: userID})
	if err != nil {
		return eff, fmt.Errorf("getEffectivePermissionsByUserQ: %w", err)
	}
	for _, p := range perms {
		eff = foldPermissions(eff, p)
	}
	return eff, nil
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

	roleUUID := uuid.MustParse(roleID)
	teamUUID := uuid.MustParse(teamID)

	setSQL, args, nextIdx, ok := buildRoleUpdateSets(patch, 1)
	if !ok {
		row, err := r.q.GetRoleByID(ctx, dbgen.GetRoleByIDParams{ID: roleUUID, TeamID: teamUUID})
		if err != nil {
			return nil, err
		}
		return &teams.RoleRow{
			Id: row.ID, TeamID: row.TeamID, Name: row.Name, System: row.System,
			Color: textToPtr(row.Color), Permissions: row.Permissions,
		}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles.Repository.UpdateRole: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if patch.Name != nil || patch.Permissions != nil {
		if err := r.guardRoleUpdate(ctx, tx, roleUUID, teamUUID, callerUserID, patch); err != nil {
			return nil, err
		}
	}

	args = append(args, roleUUID, teamUUID)
	rr := &teams.RoleRow{}
	var color pgtype.Text
	err = tx.QueryRow(ctx, fmt.Sprintf(`
		UPDATE roles SET %s WHERE id = $%d AND team_id = $%d
		RETURNING id, team_id, name, system, color, permissions
	`, setSQL, nextIdx, nextIdx+1), args...).Scan(
		&rr.Id, &rr.TeamID, &rr.Name, &rr.System, &color, &rr.Permissions,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("roles.Repository.UpdateRole: %w", err)
	}
	rr.Color = textToPtr(color)

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

	roleUUID := uuid.MustParse(roleID)
	teamUUID := uuid.MustParse(teamID)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := r.q.WithTx(tx)

	// Uses the same advisory lock key as members.SetRoles/RemoveMember
	// (hashtextextended(teamID, 0)) so role deletion is serialized against
	// concurrent role (re)assignment changes guarding the same invariant.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamUUID); err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: advisory lock: %w", err)
	}

	isSystem, err := qtx.GetRoleSystem(ctx, dbgen.GetRoleSystemParams{ID: roleUUID, TeamID: teamUUID})
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
	// see the identical comment on CheckOtherRolesHaveSettingsWrite's query.
	othersHaveSettingsWrite, err := qtx.CheckOtherRolesHaveSettingsWrite(ctx, dbgen.CheckOtherRolesHaveSettingsWriteParams{TeamID: teamUUID, ID: roleUUID})
	if err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: check other settings admins: %w", err)
	}
	if !othersHaveSettingsWrite {
		deletedRoleHasSettingsWrite, err := qtx.GetRoleHasSettingsWrite(ctx, dbgen.GetRoleHasSettingsWriteParams{ID: roleUUID, TeamID: teamUUID})
		if err != nil {
			return fmt.Errorf("roles.Repository.DeleteRole: check own settings write: %w", err)
		}
		if deletedRoleHasSettingsWrite {
			return ErrLastSettingsAdmin
		}
	}

	n, err := qtx.DeleteRole(ctx, dbgen.DeleteRoleParams{ID: roleUUID, TeamID: teamUUID})
	if err != nil {
		return fmt.Errorf("roles.Repository.DeleteRole: %w", err)
	}
	if n == 0 {
		return pgx.ErrNoRows
	}

	if err := scrubDanglingRoleReferences(ctx, qtx, roleUUID, teamUUID); err != nil {
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
func scrubDanglingRoleReferences(ctx context.Context, qtx *dbgen.Queries, roleID, teamID uuid.UUID) error {
	if err := qtx.ScrubTeamReasonVisibilityRoleID(ctx, dbgen.ScrubTeamReasonVisibilityRoleIDParams{RoleID: roleID, ID: teamID}); err != nil {
		return fmt.Errorf("roles.Repository: scrub reason_visibility_role_ids: %w", err)
	}
	if err := qtx.ScrubEventsNominatedRoleID(ctx, dbgen.ScrubEventsNominatedRoleIDParams{RoleID: roleID, TeamID: teamID}); err != nil {
		return fmt.Errorf("roles.Repository: scrub events.nominated_role_ids: %w", err)
	}
	if err := qtx.ScrubEventSeriesNominatedRoleID(ctx, dbgen.ScrubEventSeriesNominatedRoleIDParams{RoleID: roleID, TeamID: teamID}); err != nil {
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

	count, err := r.q.CountRolesForTeam(ctx, dbgen.CountRolesForTeamParams{TeamID: uuid.MustParse(teamID), RoleIds: roleIDs})
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
	return int(count) == len(seen), nil
}
