package main

import (
	"context"
	"fmt"
	"time"

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
	out := make([]jobs.ReminderCandidate, len(rows))
	for i, r := range rows {
		out[i] = jobs.ReminderCandidate{
			EventID: r.Id,
			TeamID:  r.TeamId,
			Title:   r.Title,
			Start:   events.EventStartInstant(r.Date, r.StartTime, r.MeetTime),
		}
	}
	return out, nil
}
