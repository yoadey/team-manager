package polls

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/pagination"
)

// ErrSingleChoiceMultipleOptions is returned when a vote submits more than one
// option for a poll that does not allow multiple selections.
var ErrSingleChoiceMultipleOptions = errors.New("cannot select multiple options on a single-choice poll")

// ErrTooManyPolls is returned once a team hits maxPollsPerTeam. Unlike
// finances' per-team caps, ListByTeam here is already properly
// keyset-paginated (O(limit), not O(table size)), so this isn't closing an
// availability bug -- it's a cheap defense-in-depth cap against unbounded
// storage growth from a scripted or careless polls:write caller.
var ErrTooManyPolls = fmt.Errorf("team has reached the maximum of %d polls", maxPollsPerTeam)

const maxPollsPerTeam = 50_000

// pollRepo is the interface the Service relies on.
type pollRepo interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*PollRow, error)
	CountByTeam(ctx context.Context, teamID uuid.UUID) (int, error)
	FindByID(ctx context.Context, id, teamID uuid.UUID) (*PollRow, error)
	Create(ctx context.Context, teamID, creatorID uuid.UUID, question string, multiple, anonymous bool, options []string) (uuid.UUID, error)
	Delete(ctx context.Context, id, teamID uuid.UUID) error
	ListOptions(ctx context.Context, pollID uuid.UUID) ([]*PollOptionRow, error)
	ListVotes(ctx context.Context, pollID uuid.UUID) ([]*PollVoteRow, error)
	ListOptionsByPollIDs(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*PollOptionRow, error)
	ListVotesByPollIDs(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*PollVoteRow, error)
	ReplaceVotes(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID, multiple bool) error
	WithReadTx(ctx context.Context, fn func(PollListReader) error) error
}

// jobEnqueuer is satisfied by *jobs.Client.
type jobEnqueuer interface {
	EnqueueNotification(ctx context.Context, args jobs.NotificationArgs) error
}

// Service implements polls business logic.
type Service struct {
	repo   pollRepo
	jobs   jobEnqueuer
	pager  *pagination.Paginator
	logger *slog.Logger
}

// NewService creates a new Service. pager may be nil, in which case a default
// (unsigned) Paginator is used.
func NewService(repo pollRepo, enq jobEnqueuer, pager *pagination.Paginator, logger *slog.Logger) *Service {
	if pager == nil {
		pager = pagination.New(nil)
	}
	return &Service{repo: repo, jobs: enq, pager: pager, logger: logger}
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

	var pollRows []*PollRow
	var optionsByPoll map[uuid.UUID][]*PollOptionRow
	var votesByPoll map[uuid.UUID][]*PollVoteRow
	var next *string
	// Run all three reads inside one read-only transaction so the poll list
	// and its options/votes observe a single consistent snapshot, instead of
	// possibly drifting under a concurrent Delete (which would otherwise
	// leave a "ghost" poll in the response -- a real question/options row
	// with empty options/votes for a poll that no longer exists).
	err := s.repo.WithReadTx(ctx, func(r PollListReader) error {
		var err error
		pollRows, err = r.ListByTeam(ctx, teamID, limit+1, cur)
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}

		if len(pollRows) > limit {
			pollRows = pollRows[:limit]
			last := pollRows[len(pollRows)-1]
			token, err := s.pager.Encode(ListCursor{CreatedAt: last.CreatedAt, ID: last.Id})
			if err != nil {
				return fmt.Errorf("cursor: %w", err)
			}
			next = &token
		}

		pollIDs := make([]uuid.UUID, len(pollRows))
		for i, pr := range pollRows {
			pollIDs[i] = pr.Id
		}
		optionsByPoll, err = r.ListOptionsByPollIDs(ctx, pollIDs)
		if err != nil {
			return fmt.Errorf("options: %w", err)
		}
		votesByPoll, err = r.ListVotesByPollIDs(ctx, pollIDs)
		if err != nil {
			return fmt.Errorf("votes: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("polls.Service.ListByTeam: %w", err)
	}

	result := make([]gen.Poll, 0, len(pollRows))
	for _, pr := range pollRows {
		result = append(result, assemblePoll(pr, optionsByPoll[pr.Id], votesByPoll[pr.Id], currentUserID))
	}
	return result, next, nil
}

// Create adds a new poll and returns it fully assembled.
func (s *Service) Create(ctx context.Context, teamID, creatorID uuid.UUID, body *gen.CreatePollRequest) (gen.Poll, error) {
	count, err := s.repo.CountByTeam(ctx, teamID)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.Create: %w", err)
	}
	if count >= maxPollsPerTeam {
		return gen.Poll{}, ErrTooManyPolls
	}

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
		if err := s.jobs.EnqueueNotification(ctx, jobs.NotificationArgs{
			TeamID:  teamID,
			Type:    "poll",
			ActorID: creatorID,
			Title:   &question,
		}); err != nil {
			s.logger.Warn("polls: failed to enqueue notification", slog.String("pollId", pollID.String()), slog.String("error", err.Error()))
		}
	}
	return s.buildPoll(ctx, pr, creatorID)
}

// Vote records a user's vote and returns the updated poll.
func (s *Service) Vote(ctx context.Context, pollID, teamID, userID uuid.UUID, optionIDs []uuid.UUID) (gen.Poll, error) {
	pr, err := s.repo.FindByID(ctx, pollID, teamID)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.Vote FindByID: %w", err)
	}
	// Dedupe before the single-choice check: a client submitting the same
	// option twice (the OpenAPI schema doesn't declare uniqueItems) is still
	// only choosing one option and must not trip ErrSingleChoiceMultipleOptions.
	optionIDs = dedupeUUIDs(optionIDs)
	if !pr.Multiple && len(optionIDs) > 1 {
		return gen.Poll{}, ErrSingleChoiceMultipleOptions
	}
	if err := s.repo.ReplaceVotes(ctx, pollID, userID, optionIDs, pr.Multiple); err != nil {
		if errors.Is(err, ErrOptionNotInPoll) {
			return gen.Poll{}, ErrOptionNotInPoll
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.Poll{}, pgx.ErrNoRows
		}
		return gen.Poll{}, fmt.Errorf("polls.Service.Vote ReplaceVotes: %w", err)
	}
	// Re-fetch rather than reuse the pre-write pr: a concurrent DeletePoll
	// could cascade this poll away between ReplaceVotes committing and this
	// point. buildPoll's ListOptions/ListVotes would then silently return
	// empty results (a poll_id matching zero rows is a valid, error-free
	// query result, not pgx.ErrNoRows), assembling a nonsensical empty 200 OK
	// from the stale pre-delete pr instead of the accurate 404 -- and the
	// vote the caller just successfully cast would already be gone too,
	// cascaded away along with the poll.
	fresh, err := s.repo.FindByID(ctx, pollID, teamID)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.Vote refetch: %w", err)
	}
	return s.buildPoll(ctx, fresh, userID)
}

// Delete removes a poll by ID, scoped to teamID.
func (s *Service) Delete(ctx context.Context, id, teamID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, teamID); err != nil {
		return fmt.Errorf("polls.Service.Delete: %w", err)
	}
	return nil
}

// buildPoll fetches a single poll's options and votes and assembles a
// gen.Poll. Used for single-poll operations (Create, Vote); ListByTeam
// bulk-fetches options/votes for a whole page and calls assemblePoll
// directly to avoid an N+1 query pattern.
func (s *Service) buildPoll(ctx context.Context, pr *PollRow, currentUserID uuid.UUID) (gen.Poll, error) {
	options, err := s.repo.ListOptions(ctx, pr.Id)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.buildPoll ListOptions: %w", err)
	}
	votes, err := s.repo.ListVotes(ctx, pr.Id)
	if err != nil {
		return gen.Poll{}, fmt.Errorf("polls.Service.buildPoll ListVotes: %w", err)
	}
	return assemblePoll(pr, options, votes, currentUserID), nil
}

// assemblePoll builds a gen.Poll from a poll row plus its already-fetched
// options and votes (no I/O).
func assemblePoll(pr *PollRow, options []*PollOptionRow, votes []*PollVoteRow, currentUserID uuid.UUID) gen.Poll {
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
				hasPhoto := v.HasPhoto
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
	return poll
}
