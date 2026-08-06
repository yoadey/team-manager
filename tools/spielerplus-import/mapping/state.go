package mapping

import (
	"encoding/json"
	"fmt"
	"os"
)

// State is the local idempotency mapping from SpielerPlus IDs to
// Teamverwaltung UUIDs, for entity kinds with no natural unique key to
// dedupe on (events, absences, transactions, the penalty catalog, penalty
// assignments). Users dedupe on email, attendance dedupes on the DB's own
// UNIQUE(event_id, user_id), and dues/contributions dedupe on
// UNIQUE(team_id, user_id, month) - see design.md.
type State struct {
	path               string
	Events             map[string]string `json:"events"`              // spielerplus event id -> teamverwaltung events.id
	Absences           map[string]string `json:"absences"`            // spielerplus absence occurrence id -> teamverwaltung absences.id
	Transactions       map[string]string `json:"transactions"`        // spielerplus cashbox transaction id -> teamverwaltung transactions.id
	PenaltyCatalog     map[string]string `json:"penalty_catalog"`     // spielerplus punishment-catalog entry id -> teamverwaltung penalties.id
	PenaltyAssignments map[string]string `json:"penalty_assignments"` // spielerplus punishment id -> teamverwaltung penalty_assignments.id
}

// LoadState reads the state file at path, returning an empty State if it
// doesn't exist yet (first run).
func LoadState(path string) (*State, error) {
	s := &State{
		path:               path,
		Events:             map[string]string{},
		Absences:           map[string]string{},
		Transactions:       map[string]string{},
		PenaltyCatalog:     map[string]string{},
		PenaltyAssignments: map[string]string{},
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("mapping: read state file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("mapping: parse state file %s: %w", path, err)
	}
	s.path = path
	if s.Events == nil {
		s.Events = map[string]string{}
	}
	if s.Absences == nil {
		s.Absences = map[string]string{}
	}
	if s.Transactions == nil {
		s.Transactions = map[string]string{}
	}
	if s.PenaltyCatalog == nil {
		s.PenaltyCatalog = map[string]string{}
	}
	if s.PenaltyAssignments == nil {
		s.PenaltyAssignments = map[string]string{}
	}
	return s, nil
}

// Save persists the state file. Call after every successful write so a
// crash mid-run loses at most the in-flight record, not the whole run.
func (s *State) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("mapping: marshal state: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("mapping: write state file %s: %w", s.path, err)
	}
	return nil
}
