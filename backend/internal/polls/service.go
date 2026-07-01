package polls

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/pagination"
)

// pollRepo is the interface the Service relies on.
type pollRepo interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*PollRow, error)
	FindByID(ctx context.Context, id, teamID uuid.UUID) (*PollRow, error)
	Create(ctx context.Context, teamID, creatorID uuid.UUID, question string, multiple, anonymous bool, options []string) (uuid.UUID, error)
	Delete(ctx context.Context, id, teamID uuid.UUID) error
	ListOptions(ctx context.Context, pollID uuid.UUID) ([]*PollOptionRow, error)
	ListVotes(ctx context.Context, pollID uuid.UUID) ([]*PollVoteRow, error)
	ReplaceVotes(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID, multiple bool) error
}

// jobEnqueuer is satisfied by *jobs.Client.
type jobEnqueuer interface {
	EnqueueNotification(ctx context.Context, args jobs.NotificationArgs) error
}

// Service implements polls business logic.
type Service struct {
	repo  pollRepo
	jobs  jobEnqueuer
	pager *pagination.Paginator
}

// NewService creates a new Service. pager may be nil, in which case a default
// (unsigned) Paginator is used.
func NewService(repo pollRepo, enq jobEnqueuer, pager *pagination.Paginator) *Service {
	if pager == nil {
		pager = pagination.New(nil)
	}
	return &Service{repo: repo, jobs: enq, pager: pager}
}

// ListByTeam returns a keyset page of polls (with full vote data) plus the
// cursor for the next page (nil on the last page). cursor is the opaque token
// from a prior page ("" = first page).
func (s *Service) ListByTeam(ctx context.Context, teamID, currentUserID uuid.UUID, limit int, cursor string) ([]gen.Poll, *string, error) {
	var cur *ListCursor
	var decoded ListCursor
	if ok, err := s.pager.Decode(cursor, &decoded); err != nil {
		return nil, nil, fmt.Errorf("polls.Service.ListByTeam: %w", err)
	} else if ok {
		cur = &decoded
	}

	pollRows, err := s.repo.ListByTeam(ctx, teamID, limit+1, cur)
	if err != nil {
		return nil, nil, fmt.Errorf("polls.Service.ListByTeam: %w", err)
	}

	var next *string
	if len(pollRows) > limit {
		pollRows = pollRows[:limit]
		last := pollRows[len(pollRows)-1]
		token, err := s.pager.Encode(ListCursor{CreatedAt: last.CreatedAt, ID: last.Id})
		if err != nil {
			return nil, nil, fmt.Errorf("polls.Service.ListByTeam: %w", err)
		}
		next = &token
	}

	result := make([]gen.Poll, 0, len(pollRows))
	for _, pr := range pollRows {
		p, err := s.buildPoll(ctx, pr, currentUserID)
		if err != nil {
			return nil, nil, err
		}
		result = append(result, p)
	}
	return result, next, nil
}

// Create adds a new poll and returns it fully assembled.
func (s *Service) Create(ctx context.Context, teamID, creatorID uuid.UUID, body *gen.CreatePollRequest) (gen.Poll, error) {
	multiple := false
	if body.Multiple != nil {
		multiple = *body.Multiple
	}
	anonymous := false
	if body.Anonymous != nil {
		anonymous = *body.Anonymous
	}
	pollID, err := s.repo.Create(ctx, teamID, creatorID, body.Question, multiple, anonymous, body.Options)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.Create: %w", err)
	}
	pr, err := s.repo.FindByID(ctx, pollID, teamID)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.Create FindByID: %w", err)
	}
	// Enqueue notification (best-effort; ignore error so it doesn't fail the request).
	if s.jobs != nil {
		question := body.Question
		_ = s.jobs.EnqueueNotification(ctx, jobs.NotificationArgs{
			TeamID:  teamID,
			Type:    "poll",
			ActorID: creatorID,
			Title:   &question,
		})
	}
	return s.buildPoll(ctx, pr, creatorID)
}

// Vote records a user's vote and returns the updated poll.
func (s *Service) Vote(ctx context.Context, pollID, teamID, userID uuid.UUID, optionIDs []uuid.UUID) (gen.Poll, error) {
	pr, err := s.repo.FindByID(ctx, pollID, teamID)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.Vote FindByID: %w", err)
	}
	if err := s.repo.ReplaceVotes(ctx, pollID, userID, optionIDs, pr.Multiple); err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.Vote ReplaceVotes: %w", err)
	}
	return s.buildPoll(ctx, pr, userID)
}

// Delete removes a poll by ID, scoped to teamID.
func (s *Service) Delete(ctx context.Context, id, teamID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, teamID); err != nil {
		return fmt.Errorf("polls.Service.Delete: %w", err)
	}
	return nil
}

// buildPoll assembles a gen.Poll from a row, its options, and votes.
func (s *Service) buildPoll(ctx context.Context, pr *PollRow, currentUserID uuid.UUID) (gen.Poll, error) {
	options, err := s.repo.ListOptions(ctx, pr.Id)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.buildPoll ListOptions: %w", err)
	}
	votes, err := s.repo.ListVotes(ctx, pr.Id)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.buildPoll ListVotes: %w", err)
	}

	// Count votes per option and track my votes.
	voteCounts := make(map[uuid.UUID]int)
	votersByOption := make(map[uuid.UUID][]*PollVoteRow)
	var myVoteIDs []openapi_types.UUID
	totalVoters := make(map[uuid.UUID]struct{}) // unique voters

	for _, v := range votes {
		voteCounts[v.OptionId]++
		votersByOption[v.OptionId] = append(votersByOption[v.OptionId], v)
		totalVoters[v.UserId] = struct{}{}
		if v.UserId == currentUserID {
			myVoteIDs = append(myVoteIDs, v.OptionId)
		}
	}
	total := len(totalVoters)

	genOptions := make([]gen.PollOption, 0, len(options))
	for _, opt := range options {
		count := voteCounts[opt.Id]
		var pct float32
		if total > 0 {
			pct = float32(count) / float32(total) * 100
		}
		po := gen.PollOption{
			Id:    opt.Id,
			Text:  opt.Text,
			Count: count,
			Pct:   pct,
		}
		if !pr.Anonymous {
			voters := votersByOption[opt.Id]
			voterList := make([]struct {
				Color    *string `json:"color,omitempty"`
				HasPhoto *bool   `json:"hasPhoto,omitempty"`
				Name     *string `json:"name,omitempty"`
			}, 0, len(voters))
			for _, v := range voters {
				hasPhoto := len(v.PhotoData) > 0
				voterList = append(voterList, struct {
					Color    *string `json:"color,omitempty"`
					HasPhoto *bool   `json:"hasPhoto,omitempty"`
					Name     *string `json:"name,omitempty"`
				}{
					Color:    v.UserColor,
					HasPhoto: &hasPhoto,
					Name:     v.UserName,
				})
			}
			po.Voters = &voterList
		}
		genOptions = append(genOptions, po)
	}

	poll := gen.Poll{
		Id:         pr.Id,
		Question:   pr.Question,
		Multiple:   pr.Multiple,
		Anonymous:  pr.Anonymous,
		CreatedAt:  pr.CreatedAt,
		Options:    genOptions,
		TotalVotes: total,
		MyVote:     &myVoteIDs,
	}
	return poll, nil
}
