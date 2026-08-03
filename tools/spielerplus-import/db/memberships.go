package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// RoleIDByName resolves an existing role name (created ahead of time through
// the normal Teamverwaltung UI) to its id within teamID. The role-mapping
// config maps SpielerPlus role names to Teamverwaltung role *names*; this
// turns that into the real foreign key.
func (s *Store) RoleIDByName(ctx context.Context, teamID, name string) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `SELECT id FROM roles WHERE team_id = $1 AND name = $2`, teamID, name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("db: role %q not found in team %s (create it in Teamverwaltung first, or fix the role mapping file): %w", name, teamID, err)
	}
	return id, nil
}

// EnsureMembership creates a membership for userID in teamID with roleID if
// one doesn't already exist. Idempotent: an existing membership is left
// untouched (its roles are not modified), matching "existing account is left
// alone" for users that were already migrated.
func (s *Store) EnsureMembership(ctx context.Context, teamID, userID, roleID string) (created bool, err error) {
	var membershipID string
	err = s.Pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, userID).Scan(&membershipID)
	if err == nil {
		return false, nil
	}

	if s.DryRun {
		return true, nil
	}

	membershipID = uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO memberships (id, team_id, user_id, joined_at, updated_at)
		VALUES ($1, $2, $3, now(), now())
		ON CONFLICT (team_id, user_id) DO NOTHING
	`, membershipID, teamID, userID)
	if err != nil {
		return false, fmt.Errorf("db: insert membership (team %s, user %s): %w", teamID, userID, err)
	}

	// Re-select in case of a concurrent-insert race, same pattern as EnsureUser.
	err = s.Pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, userID).Scan(&membershipID)
	if err != nil {
		return false, fmt.Errorf("db: select membership (team %s, user %s) after insert: %w", teamID, userID, err)
	}

	_, err = s.Pool.Exec(ctx, `
		INSERT INTO membership_roles (membership_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, membershipID, roleID)
	if err != nil {
		return false, fmt.Errorf("db: link role %s to membership %s: %w", roleID, membershipID, err)
	}
	return true, nil
}
