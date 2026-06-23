package polls

import (
	"context"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
)

// pollRepo is the interface the Service relies on.
type pollRepo interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*PollRow, error)
	FindByID(ctx context.Context, id uuid.UUID) (*PollRow, error)
	Create(ctx context.Context, teamID, creatorID uuid.UUID, question string, multiple, anonymous bool, options []string) (uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
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
	repo pollRepo
	jobs jobEnqueuer
}

// NewService creates a new Service.
func NewService(repo pollRepo, enq jobEnqueuer) *Service {
	return &Service{repo: repo, jobs: enq}
}

// ListByTeam returns all polls for the given team with full vote data.
func (s *Service) ListByTeam(ctx context.Context, teamID, currentUserID uuid.UUID) ([]gen.Poll, error) {
	pollRows, err := s.repo.ListByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	result := make([]gen.Poll, 0, len(pollRows))
	for _, pr := range pollRows {
		p, err := s.buildPoll(ctx, pr, currentUserID)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
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
		return gen.Poll{}, err
	}
	pr, err := s.repo.FindByID(ctx, pollID)
	if err != nil {
		return gen.Poll{}, err
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
func (s *Service) Vote(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID) (gen.Poll, error) {
	pr, err := s.repo.FindByID(ctx, pollID)
	if err != nil {
		return gen.Poll{}, err
	}
	if err := s.repo.ReplaceVotes(ctx, pollID, userID, optionIDs, pr.Multiple); err != nil {
		return gen.Poll{}, err
	}
	return s.buildPoll(ctx, pr, userID)
}

// Delete removes a poll by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// buildPoll assembles a gen.Poll from a row, its options, and votes.
func (s *Service) buildPoll(ctx context.Context, pr *PollRow, currentUserID uuid.UUID) (gen.Poll, error) {
	options, err := s.repo.ListOptions(ctx, pr.Id)
	if err != nil {
		return gen.Poll{}, err
	}
	votes, err := s.repo.ListVotes(ctx, pr.Id)
	if err != nil {
		return gen.Poll{}, err
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
			myVoteIDs = append(myVoteIDs, openapi_types.UUID(v.OptionId))
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
			Id:    openapi_types.UUID(opt.Id),
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
		Id:         openapi_types.UUID(pr.Id),
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
