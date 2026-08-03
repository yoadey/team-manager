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
	MembersCreated, MembersExisting      int
	EventsCreated, EventsSkipped         int
	AttendanceWritten, AttendanceSkipped int
	AbsencesCreated, AbsencesSkipped     int
	SkipReasons                          []string
}

func (s *Summary) skip(format string, args ...any) {
	s.SkipReasons = append(s.SkipReasons, fmt.Sprintf(format, args...))
}

// Run executes the full import against an already-authenticated SpielerPlus
// client and an already-open Store (which itself decides whether to persist
// writes based on Store.DryRun - see db.Store).
func Run(ctx context.Context, sp *spielerplus.Client, store *db.Store, opts Options) (*Summary, error) {
	summary := &Summary{}

	memberIDs, err := importMembers(ctx, sp, store, opts, summary)
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

	return summary, nil
}

// importMembers resolves every SpielerPlus member's role against the
// configured mapping *before* writing anything, so an unmapped role fails
// the whole run without a half-imported roster (spec: "Unmapped SpielerPlus
// roles fail loudly").
func importMembers(ctx context.Context, sp *spielerplus.Client, store *db.Store, opts Options, summary *Summary) (map[string]string, error) {
	members, err := sp.FetchMembers()
	if err != nil {
		return nil, fmt.Errorf("importrun: fetch members: %w", err)
	}

	roleIDs := make(map[string]string, len(members))
	for _, m := range members {
		tvRoleName, err := opts.RoleMapping.Resolve(m.Role)
		if err != nil {
			return nil, fmt.Errorf("importrun: member %s (%s): %w", m.Name, m.Email, err)
		}
		if _, ok := roleIDs[tvRoleName]; !ok {
			roleID, err := store.RoleIDByName(ctx, opts.TeamID, tvRoleName)
			if err != nil {
				return nil, fmt.Errorf("importrun: member %s (%s): %w", m.Name, m.Email, err)
			}
			roleIDs[tvRoleName] = roleID
		}
	}

	memberIDs := make(map[string]string, len(members))
	for _, m := range members {
		tvRoleName, _ := opts.RoleMapping.Resolve(m.Role) // already validated above
		roleID := roleIDs[tvRoleName]

		userID, created, err := store.EnsureUser(ctx, m.Email, m.Name)
		if err != nil {
			return nil, fmt.Errorf("importrun: create user for %s: %w", m.Email, err)
		}
		if _, err := store.EnsureMembership(ctx, opts.TeamID, userID, roleID); err != nil {
			return nil, fmt.Errorf("importrun: create membership for %s: %w", m.Email, err)
		}
		memberIDs[m.ID] = userID
		if created {
			summary.MembersCreated++
		} else {
			summary.MembersExisting++
		}
		log.Printf("member %s (%s): teamverwaltung user %s (new=%v)", m.Name, m.Email, userID, created)
	}
	return memberIDs, nil
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
