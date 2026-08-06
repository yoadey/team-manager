// Package importrun orchestrates a full SpielerPlus -> Teamverwaltung
// import: members/roles, then events, then attendance, then absences.
package importrun

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/yoadey/team-manager/tools/spielerplus-import/db"
	"github.com/yoadey/team-manager/tools/spielerplus-import/mapping"
	"github.com/yoadey/team-manager/tools/spielerplus-import/spielerplus"
)

// Options configures a run.
type Options struct {
	TeamID      string
	RoleMapping mapping.RoleMapping
	State       *mapping.State
	// Now anchors relative date resolution (SpielerPlus's year-less event
	// dates, the cutoff for expanding recurring absences). Tests pass a
	// fixed value; real runs use time.Now().
	Now time.Time
}

// Summary reports what a run did (or, in dry-run mode, would do).
type Summary struct {
	MembersCreated, MembersExisting          int
	EventsCreated, EventsSkipped             int
	AttendanceWritten, AttendanceSkipped     int
	AbsencesCreated, AbsencesSkipped         int
	TransactionsCreated, TransactionsSkipped int
	DuesCreated, DuesSkipped                 int
	PenaltiesCreated, PenaltiesSkipped       int
	SkipReasons                              []string
}

func (s *Summary) skip(format string, args ...any) {
	s.SkipReasons = append(s.SkipReasons, fmt.Sprintf(format, args...))
}

// Run executes the full import against an already-authenticated SpielerPlus
// client and an already-open Store (which itself decides whether to persist
// writes based on Store.DryRun - see db.Store).
func Run(ctx context.Context, sp *spielerplus.Client, store *db.Store, opts Options) (*Summary, error) {
	summary := &Summary{}

	memberIDs, memberNames, err := importMembers(ctx, sp, store, opts, summary)
	if err != nil {
		return summary, err
	}

	events, err := importEvents(ctx, sp, store, opts, summary)
	if err != nil {
		return summary, err
	}

	importAttendance(ctx, sp, store, events, memberIDs, summary)

	if err := importAbsences(ctx, sp, store, opts, memberIDs, summary); err != nil {
		return summary, err
	}

	// Finances are supplementary data, not central to the migration the
	// way members/events/attendance/absences are - a fetch/parse failure
	// here is logged and skipped rather than aborting the whole run.
	if err := importTransactions(ctx, sp, store, opts, summary); err != nil {
		summary.skip("transactions: %v", err)
	}
	importDues(ctx, sp, store, opts, memberIDs, summary)
	if err := importPenalties(ctx, sp, store, opts, memberNames, summary); err != nil {
		summary.skip("penalties: %v", err)
	}

	return summary, nil
}

// importMembers resolves every SpielerPlus member's role against the
// configured mapping *before* writing anything, so an unmapped role fails
// the whole run without a half-imported roster (spec: "Unmapped SpielerPlus
// roles fail loudly").
//
// Returns two lookups: SpielerPlus member id -> Teamverwaltung user id (the
// reliable join used everywhere else), and member display name ->
// Teamverwaltung user id - used only for matching assigned punishments,
// which SpielerPlus identifies by name alone (see spielerplus.PenaltyAssignment).
func importMembers(ctx context.Context, sp *spielerplus.Client, store *db.Store, opts Options, summary *Summary) (memberIDs, memberNames map[string]string, err error) {
	members, err := sp.FetchMembers()
	if err != nil {
		return nil, nil, fmt.Errorf("importrun: fetch members: %w", err)
	}

	roleIDs := make(map[string]string, len(members))
	for _, m := range members {
		tvRoleName, err := opts.RoleMapping.Resolve(m.Role)
		if err != nil {
			return nil, nil, fmt.Errorf("importrun: member %s (%s): %w", m.Name, m.Email, err)
		}
		if _, ok := roleIDs[tvRoleName]; !ok {
			roleID, err := store.RoleIDByName(ctx, opts.TeamID, tvRoleName)
			if err != nil {
				return nil, nil, fmt.Errorf("importrun: member %s (%s): %w", m.Name, m.Email, err)
			}
			roleIDs[tvRoleName] = roleID
		}
	}

	memberIDs = make(map[string]string, len(members))
	memberNames = make(map[string]string, len(members))
	for _, m := range members {
		tvRoleName, _ := opts.RoleMapping.Resolve(m.Role) // already validated above
		roleID := roleIDs[tvRoleName]

		userID, created, err := store.EnsureUser(ctx, m.Email, m.Name, m.Birthday)
		if err != nil {
			return nil, nil, fmt.Errorf("importrun: create user for %s: %w", m.Email, err)
		}
		if _, err := store.EnsureMembership(ctx, opts.TeamID, userID, roleID); err != nil {
			return nil, nil, fmt.Errorf("importrun: create membership for %s: %w", m.Email, err)
		}
		memberIDs[m.ID] = userID
		memberNames[m.Name] = userID
		if created {
			summary.MembersCreated++
		} else {
			summary.MembersExisting++
		}
		log.Printf("member %s (%s): teamverwaltung user %s (new=%v)", m.Name, m.Email, userID, created)
	}
	return memberIDs, memberNames, nil
}

// importedEvent carries what importAttendance needs beyond the bare
// Teamverwaltung id: fetching attendance requires SpielerPlus's own event
// type identifier (see spielerplus.Client.FetchAttendance).
type importedEvent struct {
	tvID string
	typ  spielerplus.EventType
}

func importEvents(ctx context.Context, sp *spielerplus.Client, store *db.Store, opts Options, summary *Summary) (map[string]importedEvent, error) {
	events, err := sp.FetchEvents(opts.Now)
	if err != nil {
		return nil, fmt.Errorf("importrun: fetch events: %w", err)
	}

	imported := make(map[string]importedEvent, len(events))
	for _, ev := range events {
		if tvID, already := opts.State.Events[ev.ID]; already {
			imported[ev.ID] = importedEvent{tvID: tvID, typ: ev.Type}
			summary.EventsSkipped++
			continue
		}

		tvID, err := store.InsertEvent(ctx, opts.TeamID, ev)
		if err != nil {
			return nil, fmt.Errorf("importrun: insert event %s (%s): %w", ev.ID, ev.Title, err)
		}
		imported[ev.ID] = importedEvent{tvID: tvID, typ: ev.Type}
		summary.EventsCreated++

		if !store.DryRun {
			opts.State.Events[ev.ID] = tvID
			if err := opts.State.Save(); err != nil {
				return nil, fmt.Errorf("importrun: save state after event %s: %w", ev.ID, err)
			}
		}
	}
	return imported, nil
}

func importAttendance(ctx context.Context, sp *spielerplus.Client, store *db.Store, events map[string]importedEvent, memberIDs map[string]string, summary *Summary) {
	for spEventID, ev := range events {
		records, err := sp.FetchAttendance(spEventID, ev.typ)
		if err != nil {
			summary.skip("attendance for event %s: fetch failed: %v", spEventID, err)
			continue
		}
		for _, rec := range records {
			tvUserID, ok := memberIDs[rec.MemberID]
			if !ok {
				summary.AttendanceSkipped++
				summary.skip("attendance for event %s: unknown member %s (not on imported roster)", spEventID, rec.MemberID)
				continue
			}
			if err := store.UpsertAttendance(ctx, ev.tvID, tvUserID, rec.Status, rec.Reason); err != nil {
				summary.AttendanceSkipped++
				summary.skip("attendance for event %s, member %s: %v", spEventID, rec.MemberID, err)
				continue
			}
			summary.AttendanceWritten++
		}
	}
}

// dateRange is a half-open-free [from, to] pair (both inclusive), used to
// track absences accepted earlier in the *same* run.
type dateRange struct{ from, to time.Time }

func (r dateRange) overlaps(other dateRange) bool {
	return !r.to.Before(other.from) && !other.to.Before(r.from)
}

func importAbsences(ctx context.Context, sp *spielerplus.Client, store *db.Store, opts Options, memberIDs map[string]string, summary *Summary) error {
	absences, err := sp.FetchAbsences(opts.Now)
	if err != nil {
		return fmt.Errorf("importrun: fetch absences: %w", err)
	}

	// InsertAbsence's overlap check only sees rows already committed to the
	// database. In a real run that's enough (each absence is written
	// immediately, so the next one sees it) - but in --dry-run mode nothing
	// is ever actually written, so two overlapping absences *within this
	// same run* would both be reported as "would create" even though a real
	// run would only accept the first and skip the second. Track
	// this-run acceptances locally so dry-run reporting matches what a real
	// run would do.
	acceptedThisRun := map[string][]dateRange{} // tvUserID -> ranges

	for _, a := range absences {
		if _, already := opts.State.Absences[a.ID]; already {
			summary.AbsencesSkipped++
			continue
		}
		tvUserID, ok := memberIDs[a.MemberID]
		if !ok {
			summary.AbsencesSkipped++
			summary.skip("absence %s: unknown member %s (not on imported roster)", a.ID, a.MemberID)
			continue
		}

		candidate := dateRange{from: a.From, to: a.To}
		if overlapsAny(acceptedThisRun[tvUserID], candidate) {
			summary.AbsencesSkipped++
			summary.skip("absence %s (member %s): overlaps another absence imported earlier in this run", a.ID, a.MemberID)
			continue
		}

		tvID, err := store.InsertAbsence(ctx, opts.TeamID, tvUserID, a.From, a.To, a.Reason)
		if err != nil {
			if errors.Is(err, db.ErrAbsenceSkipped) {
				summary.AbsencesSkipped++
				summary.skip("absence %s (member %s): %v", a.ID, a.MemberID, err)
				continue
			}
			return fmt.Errorf("importrun: insert absence %s: %w", a.ID, err)
		}
		summary.AbsencesCreated++
		acceptedThisRun[tvUserID] = append(acceptedThisRun[tvUserID], candidate)

		if !store.DryRun {
			opts.State.Absences[a.ID] = tvID
			if err := opts.State.Save(); err != nil {
				return fmt.Errorf("importrun: save state after absence %s: %w", a.ID, err)
			}
		}
	}
	return nil
}

func overlapsAny(ranges []dateRange, candidate dateRange) bool {
	for _, r := range ranges {
		if r.overlaps(candidate) {
			return true
		}
	}
	return false
}

// importTransactions imports the team's Kasse (cashbox) ledger, using the
// state file for idempotency (transactions has no natural unique key).
func importTransactions(ctx context.Context, sp *spielerplus.Client, store *db.Store, opts Options, summary *Summary) error {
	transactions, err := sp.FetchTransactions()
	if err != nil {
		return fmt.Errorf("importrun: fetch transactions: %w", err)
	}

	for _, tx := range transactions {
		if _, already := opts.State.Transactions[tx.ID]; already {
			summary.TransactionsSkipped++
			continue
		}

		tvID, err := store.InsertTransaction(ctx, opts.TeamID, tx)
		if err != nil {
			summary.TransactionsSkipped++
			summary.skip("transaction %s (%s): %v", tx.ID, tx.Title, err)
			continue
		}
		summary.TransactionsCreated++

		if !store.DryRun {
			opts.State.Transactions[tx.ID] = tvID
			if err := opts.State.Save(); err != nil {
				return fmt.Errorf("importrun: save state after transaction %s: %w", tx.ID, err)
			}
		}
	}
	return nil
}

// importDues imports the membership-dues matrix as contributions, keyed by
// the imported roster's SpielerPlus member id (a reliable id join, unlike
// penalty assignments - see importPenalties). Each member's due columns are
// spread across consecutive synthetic months starting at opts.Now, since
// contributions requires one row per (user, month) and SpielerPlus's due
// columns carry no real month of their own (a user-confirmed approximation,
// see design.md).
func importDues(ctx context.Context, sp *spielerplus.Client, store *db.Store, opts Options, memberIDs map[string]string, summary *Summary) {
	dues, err := sp.FetchDues()
	if err != nil {
		summary.skip("dues: fetch failed: %v", err)
		return
	}

	for _, d := range dues {
		tvUserID, ok := memberIDs[d.MemberID]
		if !ok {
			summary.DuesSkipped++
			summary.skip("due %s: unknown member %s (not on imported roster)", d.ID, d.MemberID)
			continue
		}

		month := opts.Now.AddDate(0, d.ColumnIndex, 0).Format("2006-01")
		if _, err := store.UpsertContribution(ctx, opts.TeamID, tvUserID, month, d.Label, d.AmountCents, d.Paid); err != nil {
			summary.DuesSkipped++
			summary.skip("due %s (member %s): %v", d.ID, d.MemberID, err)
			continue
		}
		summary.DuesCreated++
	}
}

// importPenalties imports the penalty catalog, then every assigned
// punishment. Catalog entries and assignments each use the state file for
// idempotency (neither has a natural unique key). An assignment is matched
// to the imported roster by member *name* - SpielerPlus's punishment pages
// show no member id/link at all (see spielerplus.PenaltyAssignment) -
// and to the catalog by matching its reason text against a catalog label;
// either match failing is not fatal: a name that doesn't match any imported
// member is skipped and logged, and a reason with no matching catalog entry
// still imports with its own amount/label snapshotted directly
// (penalty_assignments.penalty_id is nullable for exactly this case).
func importPenalties(ctx context.Context, sp *spielerplus.Client, store *db.Store, opts Options, memberNames map[string]string, summary *Summary) error {
	catalog, err := sp.FetchPenaltyCatalog()
	if err != nil {
		return fmt.Errorf("fetch penalty catalog: %w", err)
	}

	labelToPenaltyID := make(map[string]string, len(catalog))
	for _, entry := range catalog {
		if tvID, already := opts.State.PenaltyCatalog[entry.ID]; already {
			labelToPenaltyID[entry.Label] = tvID
			continue
		}

		tvID, err := store.InsertPenalty(ctx, opts.TeamID, entry.Label, entry.AmountCents)
		if err != nil {
			summary.skip("penalty catalog entry %s (%s): %v", entry.ID, entry.Label, err)
			continue
		}
		labelToPenaltyID[entry.Label] = tvID

		if !store.DryRun {
			opts.State.PenaltyCatalog[entry.ID] = tvID
			if err := opts.State.Save(); err != nil {
				return fmt.Errorf("save state after penalty catalog entry %s: %w", entry.ID, err)
			}
		}
	}

	assignments, err := sp.FetchPenaltyAssignments()
	if err != nil {
		return fmt.Errorf("fetch penalty assignments: %w", err)
	}

	for _, a := range assignments {
		if _, already := opts.State.PenaltyAssignments[a.ID]; already {
			summary.PenaltiesSkipped++
			continue
		}

		tvUserID, ok := memberNames[a.MemberName]
		if !ok {
			summary.PenaltiesSkipped++
			summary.skip("penalty assignment %s: unknown member %q (not matched to imported roster by name)", a.ID, a.MemberName)
			continue
		}

		tvID, err := store.InsertPenaltyAssignment(ctx, opts.TeamID, tvUserID, labelToPenaltyID[a.Reason], a.AmountCents, a.Reason, a.Paid, a.Date)
		if err != nil {
			summary.PenaltiesSkipped++
			summary.skip("penalty assignment %s (%s): %v", a.ID, a.MemberName, err)
			continue
		}
		summary.PenaltiesCreated++

		if !store.DryRun {
			opts.State.PenaltyAssignments[a.ID] = tvID
			if err := opts.State.Save(); err != nil {
				return fmt.Errorf("save state after penalty assignment %s: %w", a.ID, err)
			}
		}
	}
	return nil
}
