package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/jobs"
)

// eventsReminderAdapter bridges *events.Repository to jobs' eventsReminderLister
// interface. It exists because internal/jobs cannot import internal/events
// directly (internal/events already imports internal/jobs, to enqueue
// notification jobs via jobs.Client -- see events.NewService) -- cmd/server
// is free to import both, so the timezone-aware start-instant computation
// (events.EventStartInstant) happens here instead.
type eventsReminderAdapter struct {
	repo *events.Repository
}

func (a eventsReminderAdapter) ListUpcomingForReminders(ctx context.Context, from, to time.Time) ([]jobs.ReminderCandidate, error) {
	rows, err := a.repo.ListUpcomingForReminders(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("eventsReminderAdapter.ListUpcomingForReminders: %w", err)
	}

	eventIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		eventIDs[i] = r.Id
	}
	// One candidate per (event, targeted team) -- for a cross-team event
	// this fans reminders out to every targeted team's members, not just the
	// owning team's, mirroring the same event_teams-driven expansion
	// ListEvents/ListAttendance already do for reads. Safe against a
	// multi-team member being reminded twice: event_reminders_sent's PK is
	// (event_id, user_id), not (event_id, team_id, user_id), so the first
	// team's send marks it and the second team's attempt for the same user
	// is a no-op via ON CONFLICT DO NOTHING (see remindMember). A
	// single-team event's ListEventTeamsBatch entry is just its own owning
	// team, so this reduces to the previous one-candidate-per-event
	// behavior there.
	eventTeams, err := a.repo.ListEventTeamsBatch(ctx, eventIDs)
	if err != nil {
		return nil, fmt.Errorf("eventsReminderAdapter.ListUpcomingForReminders: %w", err)
	}

	var out []jobs.ReminderCandidate
	for _, r := range rows {
		targets := eventTeams[r.Id]
		if len(targets) == 0 {
			// Defensive: every event has at least its owning team's row in
			// event_teams (see that table's migration comment) -- this
			// shouldn't happen, but fall back rather than silently dropping
			// the event's reminders entirely.
			targets = []events.EventTeamRow{{TeamID: r.TeamId}}
		}
		start := events.EventStartInstant(r.Date, r.StartTime, r.MeetTime)
		for _, team := range targets {
			out = append(out, jobs.ReminderCandidate{
				EventID: r.Id,
				TeamID:  team.TeamID,
				Title:   r.Title,
				Start:   start,
			})
		}
	}
	return out, nil
}
