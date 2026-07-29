package calendarshare

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrCannotShareWithSelf is returned by Grant when ownerTeamID and
// viewerTeamID are the same team.
var ErrCannotShareWithSelf = errors.New("calendarshare: cannot share a calendar with its own team")

// ErrNoGrant is returned by ListEvents when ownerTeamID currently has no
// calendar-share grant to viewerTeamID. The handler maps it to a plain 404,
// matching this codebase's convention (see calendarfeed.ErrFeedUnavailable,
// middleware.RequireMembership) of not distinguishing "never existed" from
// "access was revoked" in cross-team-boundary responses.
var ErrNoGrant = errors.New("calendarshare: no grant from owner team to viewer team")

// shareRepo is the interface Service relies on.
type shareRepo interface {
	Grant(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (*ShareRow, error)
	Revoke(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) error
	ListGrantedByOwner(ctx context.Context, ownerTeamID uuid.UUID) ([]ShareRow, error)
	ListGrantedToViewer(ctx context.Context, viewerTeamID uuid.UUID) ([]ShareRow, error)
	HasGrant(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (bool, error)
	ListRedactedEvents(ctx context.Context, ownerTeamID uuid.UUID, from, to *time.Time) ([]RedactedEventRow, error)
}

// Service implements calendar-share business logic: granting/revoking
// visibility and reading the redacted schedule of a team that granted it.
type Service struct {
	repo shareRepo
}

// NewService creates a new Service.
func NewService(repo shareRepo) *Service {
	return &Service{repo: repo}
}

// Grant creates (or idempotently re-confirms) a calendar-share grant from
// ownerTeamID to viewerTeamID.
func (s *Service) Grant(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (*ShareRow, error) {
	if ownerTeamID == viewerTeamID {
		return nil, ErrCannotShareWithSelf
	}
	row, err := s.repo.Grant(ctx, ownerTeamID, viewerTeamID)
	if err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, fmt.Errorf("calendarshare.Service.Grant: %w", err)
	}
	return row, nil
}

// Revoke removes ownerTeamID's grant to viewerTeamID, if any.
func (s *Service) Revoke(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) error {
	if err := s.repo.Revoke(ctx, ownerTeamID, viewerTeamID); err != nil {
		return fmt.Errorf("calendarshare.Service.Revoke: %w", err)
	}
	return nil
}

// ListGrants returns every team ownerTeamID currently grants calendar
// visibility to.
func (s *Service) ListGrants(ctx context.Context, ownerTeamID uuid.UUID) ([]ShareRow, error) {
	rows, err := s.repo.ListGrantedByOwner(ctx, ownerTeamID)
	if err != nil {
		return nil, fmt.Errorf("calendarshare.Service.ListGrants: %w", err)
	}
	return rows, nil
}

// ListSources returns every team that currently grants viewerTeamID
// calendar visibility.
func (s *Service) ListSources(ctx context.Context, viewerTeamID uuid.UUID) ([]ShareRow, error) {
	rows, err := s.repo.ListGrantedToViewer(ctx, viewerTeamID)
	if err != nil {
		return nil, fmt.Errorf("calendarshare.Service.ListSources: %w", err)
	}
	return rows, nil
}

// ListEvents re-checks that ownerTeamID currently grants viewerTeamID
// calendar visibility (so a just-revoked grant takes effect immediately)
// and, if so, returns ownerTeamID's redacted schedule.
func (s *Service) ListEvents(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID, from, to *time.Time) ([]RedactedEventRow, error) {
	ok, err := s.repo.HasGrant(ctx, ownerTeamID, viewerTeamID)
	if err != nil {
		return nil, fmt.Errorf("calendarshare.Service.ListEvents: check grant: %w", err)
	}
	if !ok {
		return nil, ErrNoGrant
	}
	rows, err := s.repo.ListRedactedEvents(ctx, ownerTeamID, from, to)
	if err != nil {
		return nil, fmt.Errorf("calendarshare.Service.ListEvents: %w", err)
	}
	return rows, nil
}
