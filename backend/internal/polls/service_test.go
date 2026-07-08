package polls_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/polls"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	listByTeam   func(ctx context.Context, teamID uuid.UUID, limit int, cur *polls.ListCursor) ([]*polls.PollRow, error)
	findByID     func(ctx context.Context, id, teamID uuid.UUID) (*polls.PollRow, error)
	create       func(ctx context.Context, teamID, creatorID uuid.UUID, question string, multiple, anonymous bool, options []string) (uuid.UUID, error)
	delete       func(ctx context.Context, id, teamID uuid.UUID) error
	listOptions  func(ctx context.Context, pollID uuid.UUID) ([]*polls.PollOptionRow, error)
	listVotes    func(ctx context.Context, pollID uuid.UUID) ([]*polls.PollVoteRow, error)
	replaceVotes func(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID, multiple bool) error

	// listOptionsByPollIDs/listVotesByPollIDs are optional; when unset, the
	// bulk methods below fall back to calling listOptions/listVotes per poll
	// ID so existing single-poll test setups keep working unchanged.
	listOptionsByPollIDs func(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*polls.PollOptionRow, error)
	listVotesByPollIDs   func(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*polls.PollVoteRow, error)
}

func (m *mockRepo) ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *polls.ListCursor) ([]*polls.PollRow, error) {
	return m.listByTeam(ctx, teamID, limit, cur)
}

func (m *mockRepo) FindByID(ctx context.Context, id, teamID uuid.UUID) (*polls.PollRow, error) {
	return m.findByID(ctx, id, teamID)
}

func (m *mockRepo) Create(ctx context.Context, teamID, creatorID uuid.UUID, question string, multiple, anonymous bool, options []string) (uuid.UUID, error) {
	return m.create(ctx, teamID, creatorID, question, multiple, anonymous, options)
}

func (m *mockRepo) Delete(ctx context.Context, id, teamID uuid.UUID) error {
	return m.delete(ctx, id, teamID)
}

func (m *mockRepo) ListOptions(ctx context.Context, pollID uuid.UUID) ([]*polls.PollOptionRow, error) {
	return m.listOptions(ctx, pollID)
}

func (m *mockRepo) ListVotes(ctx context.Context, pollID uuid.UUID) ([]*polls.PollVoteRow, error) {
	return m.listVotes(ctx, pollID)
}

func (m *mockRepo) ReplaceVotes(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID, multiple bool) error {
	return m.replaceVotes(ctx, pollID, userID, optionIDs, multiple)
}

func (m *mockRepo) ListOptionsByPollIDs(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*polls.PollOptionRow, error) {
	if m.listOptionsByPollIDs != nil {
		return m.listOptionsByPollIDs(ctx, pollIDs)
	}
	result := make(map[uuid.UUID][]*polls.PollOptionRow, len(pollIDs))
	for _, id := range pollIDs {
		opts, err := m.listOptions(ctx, id)
		if err != nil {
			return nil, err
		}
		result[id] = opts
	}
	return result, nil
}

func (m *mockRepo) ListVotesByPollIDs(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*polls.PollVoteRow, error) {
	if m.listVotesByPollIDs != nil {
		return m.listVotesByPollIDs(ctx, pollIDs)
	}
	result := make(map[uuid.UUID][]*polls.PollVoteRow, len(pollIDs))
	for _, id := range pollIDs {
		votes, err := m.listVotes(ctx, id)
		if err != nil {
			return nil, err
		}
		result[id] = votes
	}
	return result, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

var (
	teamID   = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	userID   = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	pollID   = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	optionID = uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
)

func makePollRow() *polls.PollRow {
	return &polls.PollRow{
		Id:        pollID,
		TeamId:    teamID,
		CreatorId: userID,
		Question:  "Best player?",
		Multiple:  false,
		Anonymous: false,
		CreatedAt: time.Now(),
	}
}

func makeOptionRow() *polls.PollOptionRow {
	return &polls.PollOptionRow{
		Id:        optionID,
		PollId:    pollID,
		Text:      "Alice",
		SortOrder: 0,
	}
}

func emptyVoteRepo() func(ctx context.Context, pollID uuid.UUID) ([]*polls.PollVoteRow, error) {
	return func(_ context.Context, _ uuid.UUID) ([]*polls.PollVoteRow, error) {
		return nil, nil
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestService_ListByTeam(t *testing.T) {
	t.Parallel()

	pr := makePollRow()
	opt := makeOptionRow()

	repo := &mockRepo{
		listByTeam: func(_ context.Context, tid uuid.UUID, _ int, _ *polls.ListCursor) ([]*polls.PollRow, error) {
			assert.Equal(t, teamID, tid)
			return []*polls.PollRow{pr}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]*polls.PollOptionRow, error) {
			return []*polls.PollOptionRow{opt}, nil
		},
		listVotes: emptyVoteRepo(),
	}

	svc := polls.NewService(repo, nil, nil, slog.Default())
	result, next, err := svc.ListByTeam(context.Background(), teamID, userID, 50, "")

	require.NoError(t, err)
	assert.Nil(t, next)
	require.Len(t, result, 1)
	assert.Equal(t, pollID, result[0].Id)
	assert.Equal(t, "Best player?", result[0].Question)
	require.Len(t, result[0].Options, 1)
	assert.Equal(t, "Alice", result[0].Options[0].Text)
}

// TestService_ListByTeam_BulkFetchesOptionsAndVotes guards against a
// regression to the N+1 query pattern: for a page of N polls, ListByTeam
// must fetch options and votes with one bulk call each (ListOptionsByPollIDs/
// ListVotesByPollIDs), not one call per poll.
func TestService_ListByTeam_BulkFetchesOptionsAndVotes(t *testing.T) {
	t.Parallel()

	pr1 := makePollRow()
	pr2 := &polls.PollRow{
		Id:        uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		TeamId:    teamID,
		CreatorId: userID,
		Question:  "Best moment?",
		Multiple:  false,
		Anonymous: false,
		CreatedAt: time.Now(),
	}
	opt := makeOptionRow()

	optionsCalls, votesCalls := 0, 0
	repo := &mockRepo{
		listByTeam: func(_ context.Context, _ uuid.UUID, _ int, _ *polls.ListCursor) ([]*polls.PollRow, error) {
			return []*polls.PollRow{pr1, pr2}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]*polls.PollOptionRow, error) {
			t.Fatal("ListByTeam must not call the per-poll ListOptions (N+1 pattern)")
			return nil, nil
		},
		listVotes: func(_ context.Context, _ uuid.UUID) ([]*polls.PollVoteRow, error) {
			t.Fatal("ListByTeam must not call the per-poll ListVotes (N+1 pattern)")
			return nil, nil
		},
		listOptionsByPollIDs: func(_ context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*polls.PollOptionRow, error) {
			optionsCalls++
			assert.Len(t, pollIDs, 2)
			return map[uuid.UUID][]*polls.PollOptionRow{pr1.Id: {opt}}, nil
		},
		listVotesByPollIDs: func(_ context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*polls.PollVoteRow, error) {
			votesCalls++
			assert.Len(t, pollIDs, 2)
			return map[uuid.UUID][]*polls.PollVoteRow{}, nil
		},
	}

	svc := polls.NewService(repo, nil, nil, slog.Default())
	result, _, err := svc.ListByTeam(context.Background(), teamID, userID, 50, "")

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, 1, optionsCalls, "options must be bulk-fetched exactly once regardless of poll count")
	assert.Equal(t, 1, votesCalls, "votes must be bulk-fetched exactly once regardless of poll count")
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	newID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	pr := &polls.PollRow{
		Id: newID, TeamId: teamID, CreatorId: userID,
		Question: "Vote now?", Multiple: false, Anonymous: false, CreatedAt: time.Now(),
	}

	repo := &mockRepo{
		create: func(_ context.Context, tid, cid uuid.UUID, q string, multi, anon bool, opts []string) (uuid.UUID, error) {
			assert.Equal(t, teamID, tid)
			assert.Equal(t, "Vote now?", q)
			assert.Len(t, opts, 2)
			return newID, nil
		},
		findByID: func(_ context.Context, _, _ uuid.UUID) (*polls.PollRow, error) {
			return pr, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]*polls.PollOptionRow, error) {
			return []*polls.PollOptionRow{
				{Id: optionID, PollId: newID, Text: "Yes", SortOrder: 0},
				{Id: uuid.New(), PollId: newID, Text: "No", SortOrder: 1},
			}, nil
		},
		listVotes: emptyVoteRepo(),
	}

	svc := polls.NewService(repo, nil, nil, slog.Default())
	body := &gen.CreatePollRequest{
		Question: "Vote now?",
		Options:  []string{"Yes", "No"},
	}
	result, err := svc.Create(context.Background(), teamID, userID, body)

	require.NoError(t, err)
	assert.Equal(t, "Vote now?", result.Question)
	assert.Len(t, result.Options, 2)
}

// mockJobEnqueuer satisfies jobEnqueuer for tests exercising the
// best-effort notification path.
type mockJobEnqueuer struct {
	err error
}

func (m *mockJobEnqueuer) EnqueueNotification(context.Context, jobs.NotificationArgs) error {
	return m.err
}

// TestService_Create_NotificationEnqueueFailure_StillSucceeds regression-tests
// that a failed best-effort notification enqueue doesn't fail the request
// (the write already succeeded) -- this was already true before, but a
// failure here used to be discarded with no trace at all; the logger
// parameter added alongside this test makes it observable instead.
func TestService_Create_NotificationEnqueueFailure_StillSucceeds(t *testing.T) {
	t.Parallel()

	newID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	pr := &polls.PollRow{
		Id: newID, TeamId: teamID, CreatorId: userID,
		Question: "Vote now?", Multiple: false, Anonymous: false, CreatedAt: time.Now(),
	}

	repo := &mockRepo{
		create: func(context.Context, uuid.UUID, uuid.UUID, string, bool, bool, []string) (uuid.UUID, error) {
			return newID, nil
		},
		findByID: func(_ context.Context, _, _ uuid.UUID) (*polls.PollRow, error) {
			return pr, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]*polls.PollOptionRow, error) {
			return []*polls.PollOptionRow{{Id: optionID, PollId: newID, Text: "Yes", SortOrder: 0}}, nil
		},
		listVotes: emptyVoteRepo(),
	}

	svc := polls.NewService(repo, &mockJobEnqueuer{err: errors.New("river unavailable")}, nil, slog.Default())
	body := &gen.CreatePollRequest{Question: "Vote now?", Options: []string{"Yes"}}
	result, err := svc.Create(context.Background(), teamID, userID, body)

	require.NoError(t, err)
	assert.Equal(t, "Vote now?", result.Question)
}

func TestService_Vote(t *testing.T) {
	t.Parallel()

	pr := makePollRow()
	opt := makeOptionRow()
	called := false

	repo := &mockRepo{
		findByID: func(_ context.Context, id, tid uuid.UUID) (*polls.PollRow, error) {
			assert.Equal(t, pollID, id)
			assert.Equal(t, teamID, tid)
			return pr, nil
		},
		replaceVotes: func(_ context.Context, pid, uid uuid.UUID, oids []uuid.UUID, multi bool) error {
			assert.Equal(t, pollID, pid)
			assert.Equal(t, userID, uid)
			called = true
			return nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]*polls.PollOptionRow, error) {
			return []*polls.PollOptionRow{opt}, nil
		},
		listVotes: func(_ context.Context, _ uuid.UUID) ([]*polls.PollVoteRow, error) {
			name := "Alice"
			color := "#000"
			return []*polls.PollVoteRow{
				{PollId: pollID, OptionId: optionID, UserId: userID, UserName: &name, UserColor: &color},
			}, nil
		},
	}

	svc := polls.NewService(repo, nil, nil, slog.Default())
	result, err := svc.Vote(context.Background(), pollID, teamID, userID, []uuid.UUID{optionID})

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, 1, result.TotalVotes)
	require.NotNil(t, result.MyVote)
	assert.Len(t, *result.MyVote, 1)
}

func TestService_Vote_CrossTeamBlocked(t *testing.T) {
	t.Parallel()

	otherTeamID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	repo := &mockRepo{
		findByID: func(_ context.Context, id, tid uuid.UUID) (*polls.PollRow, error) {
			assert.Equal(t, pollID, id)
			assert.Equal(t, otherTeamID, tid)
			return nil, pgx.ErrNoRows
		},
		replaceVotes: func(_ context.Context, _, _ uuid.UUID, _ []uuid.UUID, _ bool) error {
			t.Fatal("ReplaceVotes should not be called when the poll does not belong to the team")
			return nil
		},
	}

	svc := polls.NewService(repo, nil, nil, slog.Default())
	_, err := svc.Vote(context.Background(), pollID, otherTeamID, userID, []uuid.UUID{optionID})

	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestService_Vote_SingleChoiceRejectsMultipleOptions(t *testing.T) {
	t.Parallel()

	pr := makePollRow() // Multiple: false
	otherOptionID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	repo := &mockRepo{
		findByID: func(_ context.Context, id, tid uuid.UUID) (*polls.PollRow, error) {
			return pr, nil
		},
		replaceVotes: func(_ context.Context, _, _ uuid.UUID, _ []uuid.UUID, _ bool) error {
			t.Fatal("ReplaceVotes should not be called for an invalid single-choice vote")
			return nil
		},
	}

	svc := polls.NewService(repo, nil, nil, slog.Default())
	_, err := svc.Vote(context.Background(), pollID, teamID, userID, []uuid.UUID{optionID, otherOptionID})

	require.Error(t, err)
	assert.ErrorIs(t, err, polls.ErrSingleChoiceMultipleOptions)
}

func TestService_Vote_SingleChoiceDuplicateOptionID_NotRejected(t *testing.T) {
	t.Parallel()

	pr := makePollRow() // Multiple: false
	opt := makeOptionRow()
	var gotOptionIDs []uuid.UUID

	repo := &mockRepo{
		findByID: func(context.Context, uuid.UUID, uuid.UUID) (*polls.PollRow, error) {
			return pr, nil
		},
		replaceVotes: func(_ context.Context, _, _ uuid.UUID, oids []uuid.UUID, _ bool) error {
			gotOptionIDs = oids
			return nil
		},
		listOptions: func(context.Context, uuid.UUID) ([]*polls.PollOptionRow, error) {
			return []*polls.PollOptionRow{opt}, nil
		},
		listVotes: func(context.Context, uuid.UUID) ([]*polls.PollVoteRow, error) {
			return nil, nil
		},
	}

	svc := polls.NewService(repo, nil, nil, slog.Default())
	// Same option submitted twice — a single-choice vote in substance, must
	// not trip ErrSingleChoiceMultipleOptions (raw, undeduped length is 2).
	_, err := svc.Vote(context.Background(), pollID, teamID, userID, []uuid.UUID{optionID, optionID})

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{optionID}, gotOptionIDs, "duplicate must be deduped before reaching the repository")
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	called := false
	repo := &mockRepo{
		delete: func(_ context.Context, id, tid uuid.UUID) error {
			assert.Equal(t, pollID, id)
			assert.Equal(t, teamID, tid)
			called = true
			return nil
		},
	}

	svc := polls.NewService(repo, nil, nil, slog.Default())
	err := svc.Delete(context.Background(), pollID, teamID)

	require.NoError(t, err)
	assert.True(t, called)
}

func TestService_Delete_CrossTeamBlocked(t *testing.T) {
	t.Parallel()

	otherTeamID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repo := &mockRepo{
		delete: func(_ context.Context, id, tid uuid.UUID) error {
			assert.Equal(t, pollID, id)
			assert.Equal(t, otherTeamID, tid)
			return pgx.ErrNoRows
		},
	}

	svc := polls.NewService(repo, nil, nil, slog.Default())
	err := svc.Delete(context.Background(), pollID, otherTeamID)

	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}
