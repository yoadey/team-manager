package polls_test

import (
	"context"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/polls"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	listByTeam   func(ctx context.Context, teamID uuid.UUID) ([]*polls.PollRow, error)
	findByID     func(ctx context.Context, id uuid.UUID) (*polls.PollRow, error)
	create       func(ctx context.Context, teamID, creatorID uuid.UUID, question string, multiple, anonymous bool, options []string) (uuid.UUID, error)
	delete       func(ctx context.Context, id uuid.UUID) error
	listOptions  func(ctx context.Context, pollID uuid.UUID) ([]*polls.PollOptionRow, error)
	listVotes    func(ctx context.Context, pollID uuid.UUID) ([]*polls.PollVoteRow, error)
	replaceVotes func(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID, multiple bool) error
}

func (m *mockRepo) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*polls.PollRow, error) {
	return m.listByTeam(ctx, teamID)
}
func (m *mockRepo) FindByID(ctx context.Context, id uuid.UUID) (*polls.PollRow, error) {
	return m.findByID(ctx, id)
}
func (m *mockRepo) Create(ctx context.Context, teamID, creatorID uuid.UUID, question string, multiple, anonymous bool, options []string) (uuid.UUID, error) {
	return m.create(ctx, teamID, creatorID, question, multiple, anonymous, options)
}
func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.delete(ctx, id)
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
		listByTeam: func(_ context.Context, tid uuid.UUID) ([]*polls.PollRow, error) {
			assert.Equal(t, teamID, tid)
			return []*polls.PollRow{pr}, nil
		},
		listOptions: func(_ context.Context, _ uuid.UUID) ([]*polls.PollOptionRow, error) {
			return []*polls.PollOptionRow{opt}, nil
		},
		listVotes: emptyVoteRepo(),
	}

	svc := polls.NewService(repo, nil)
	result, err := svc.ListByTeam(context.Background(), teamID, userID)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, openapi_types.UUID(pollID), result[0].Id)
	assert.Equal(t, "Best player?", result[0].Question)
	require.Len(t, result[0].Options, 1)
	assert.Equal(t, "Alice", result[0].Options[0].Text)
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
		findByID: func(_ context.Context, _ uuid.UUID) (*polls.PollRow, error) {
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

	svc := polls.NewService(repo, nil)
	body := &gen.CreatePollRequest{
		Question: "Vote now?",
		Options:  []string{"Yes", "No"},
	}
	result, err := svc.Create(context.Background(), teamID, userID, body)

	require.NoError(t, err)
	assert.Equal(t, "Vote now?", result.Question)
	assert.Len(t, result.Options, 2)
}

func TestService_Vote(t *testing.T) {
	t.Parallel()

	pr := makePollRow()
	opt := makeOptionRow()
	called := false

	repo := &mockRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*polls.PollRow, error) {
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

	svc := polls.NewService(repo, nil)
	result, err := svc.Vote(context.Background(), pollID, userID, []uuid.UUID{optionID})

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, 1, result.TotalVotes)
	require.NotNil(t, result.MyVote)
	assert.Len(t, *result.MyVote, 1)
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	called := false
	repo := &mockRepo{
		delete: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, pollID, id)
			called = true
			return nil
		},
	}

	svc := polls.NewService(repo, nil)
	err := svc.Delete(context.Background(), pollID)

	require.NoError(t, err)
	assert.True(t, called)
}
