package polls_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/polls"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestPollRepository_CreateAndList(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Poll User', 'poll@example.com', '#123456')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Poll Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	creatorID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, creatorID, "Best player?", false, false, []string{"Alice", "Bob", "Charlie"})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, pollID)

	pr, err := repo.FindByID(ctx, pollID, teamID)
	require.NoError(t, err)
	require.NotNil(t, pr)
	assert.Equal(t, "Best player?", pr.Question)
	assert.False(t, pr.Multiple)
	assert.False(t, pr.Anonymous)

	opts, err := repo.ListOptions(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, opts, 3)
	assert.Equal(t, "Alice", opts[0].Text)
	assert.Equal(t, "Bob", opts[1].Text)
	assert.Equal(t, "Charlie", opts[2].Text)

	list, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, pollID, list[0].Id)
}

func TestPollRepository_Vote(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Voter', 'voter@example.com', '#aabbcc')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Vote Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, userID, "Vote?", false, false, []string{"Yes", "No"})
	require.NoError(t, err)

	opts, err := repo.ListOptions(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, opts, 2)

	yesID := opts[0].Id

	err = repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{yesID}, false)
	require.NoError(t, err)

	votes, err := repo.ListVotes(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, votes, 1)
	assert.Equal(t, yesID, votes[0].OptionId)
	assert.Equal(t, userID, votes[0].UserId)

	// Replace with the other option.
	noID := opts[1].Id
	err = repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{noID}, false)
	require.NoError(t, err)

	votes, err = repo.ListVotes(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, votes, 1)
	assert.Equal(t, noID, votes[0].OptionId)

	// Regression: VotePollRequest.optionIds must accept an empty array to
	// retract a vote entirely (the openapi.yaml schema previously declared
	// minItems: 1, contradicting this actually-relied-upon behavior — a
	// multi-select poll UI lets a user un-select their last remaining
	// choice, which submits optionIds: []).
	err = repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{}, false)
	require.NoError(t, err)

	votes, err = repo.ListVotes(ctx, pollID)
	require.NoError(t, err)
	assert.Empty(t, votes)
}

// A client submitting the same option ID twice (the OpenAPI schema doesn't
// declare uniqueItems on optionIds) must not be misread as "an option that
// doesn't belong to the poll" — the second insert of a duplicate hits ON
// CONFLICT DO NOTHING (RowsAffected=0), which without deduping the loop
// would mistake for ErrOptionNotInPoll and abort the whole vote.
func TestPollRepository_ReplaceVotes_DuplicateOptionIDs_DoesNotError(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Dup Voter', 'dup-voter@example.com', '#aabbcc')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Dup Vote Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, userID, "Multi?", true, false, []string{"A", "B", "C"})
	require.NoError(t, err)
	opts, err := repo.ListOptions(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, opts, 3)

	err = repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{opts[0].Id, opts[0].Id, opts[1].Id}, true)
	require.NoError(t, err)

	votes, err := repo.ListVotes(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, votes, 2, "duplicate option ID must be deduped, and the other legitimate option must not be dropped")
}

func TestPollRepository_ReplaceVotes_RejectsOptionFromOtherPoll(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Voter2', 'voter2@example.com', '#aabbcc')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Vote Team 2')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, userID, "Poll A?", false, false, []string{"Yes", "No"})
	require.NoError(t, err)
	otherPollID, err := repo.Create(ctx, teamID, userID, "Poll B?", false, false, []string{"Only Option"})
	require.NoError(t, err)

	otherOpts, err := repo.ListOptions(ctx, otherPollID)
	require.NoError(t, err)
	require.Len(t, otherOpts, 1)

	err = repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{otherOpts[0].Id}, false)
	require.ErrorIs(t, err, polls.ErrOptionNotInPoll)

	votes, err := repo.ListVotes(ctx, pollID)
	require.NoError(t, err)
	assert.Empty(t, votes, "no vote row should have been created for the cross-poll option")
}

// TestPollRepository_ReplaceVotes_PollDeleted_ReturnsErrNoRows regression-tests
// a race where a poll deleted between Service.Vote's FindByID (poll still
// exists) and ReplaceVotes committing (DeletePoll cascades poll_options/
// poll_votes away) used to be misreported as ErrOptionNotInPoll (422 "option
// does not belong to poll") -- the WHERE EXISTS guard can't tell "this
// option isn't in that poll" apart from "that poll doesn't exist anymore"
// on its own. The voter must see 404 "poll not found" (via pgx.ErrNoRows,
// already wired up in VotePoll's handler), not a misleading 422.
func TestPollRepository_ReplaceVotes_PollDeleted_ReturnsErrNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Voter3', 'voter3@example.com', '#ccddee')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Vote Team 3')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, userID, "Poll C?", false, false, []string{"Yes", "No"})
	require.NoError(t, err)
	opts, err := repo.ListOptions(ctx, pollID)
	require.NoError(t, err)
	require.NotEmpty(t, opts)

	// Simulate the poll being deleted concurrently, after Service.Vote's
	// FindByID already confirmed it existed but before ReplaceVotes commits.
	require.NoError(t, repo.Delete(ctx, pollID, teamID))

	err = repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{opts[0].Id}, false)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.NotErrorIs(t, err, polls.ErrOptionNotInPoll)
}

// TestPollRepository_ReplaceVotes_RejectsNonMemberUser regression-tests a bug
// where ReplaceVotes checked that the option belonged to the poll but never
// checked that userID was currently a member of the poll's team.
// polls/vote is self-service (see authz.go), so RequireMembership only
// checks membership once at the start of the request -- a membership
// removal racing this call could otherwise still commit a vote for a
// non-member, and for a non-anonymous poll that ex-member's name/avatar/
// photo would then be displayed alongside their vote to every remaining
// team member indefinitely.
func TestPollRepository_ReplaceVotes_RejectsNonMemberUser(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	memberID := uuid.New().String()
	outsiderID := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES
			($1, 'Member', 'poll-member@example.com', '#aabbcc'),
			($2, 'Outsider', 'poll-outsider@example.com', '#ccddee')`,
		memberID, outsiderID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Non-Member Vote Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, memberID)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	member := uuid.MustParse(memberID)
	outsider := uuid.MustParse(outsiderID)

	pollID, err := repo.Create(ctx, teamID, member, "Non-member poll?", false, false, []string{"Yes", "No"})
	require.NoError(t, err)
	opts, err := repo.ListOptions(ctx, pollID)
	require.NoError(t, err)
	require.NotEmpty(t, opts)

	err = repo.ReplaceVotes(ctx, pollID, outsider, []uuid.UUID{opts[0].Id}, false)
	require.ErrorIs(t, err, pgx.ErrNoRows, "ReplaceVotes must reject a userID that is not a member of the poll's team")

	votes, err := repo.ListVotes(ctx, pollID)
	require.NoError(t, err)
	assert.Empty(t, votes, "no vote row should have been created for the non-member user")

	// A real member of the team is unaffected.
	err = repo.ReplaceVotes(ctx, pollID, member, []uuid.UUID{opts[0].Id}, false)
	require.NoError(t, err, "ReplaceVotes scoped to an actual team member must succeed")
}

// TestPollRepository_ReplaceVotes_ConcurrentSingleChoice_NoDoubleVote is a
// regression test for a race where two concurrent ReplaceVotes calls for the
// same user on a single-choice poll could each observe an empty poll_votes
// table (both DELETEs run before either INSERT commits under Read Committed
// isolation) and both succeed, leaving the user with two votes on a poll
// that's supposed to allow only one. The advisory lock in ReplaceVotes must
// serialize these calls so exactly one vote survives.
func TestPollRepository_ReplaceVotes_ConcurrentSingleChoice_NoDoubleVote(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Racer', 'racer@example.com', '#ff00ff')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Race Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, userID, "Race?", false, false, []string{"A", "B"})
	require.NoError(t, err)

	opts, err := repo.ListOptions(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, opts, 2)
	optA, optB := opts[0].Id, opts[1].Id

	const rounds = 20
	for i := 0; i < rounds; i++ {
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs <- repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{optA}, false)
		}()
		go func() {
			defer wg.Done()
			errs <- repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{optB}, false)
		}()
		wg.Wait()
		close(errs)
		for e := range errs {
			require.NoError(t, e)
		}

		votes, err := repo.ListVotes(ctx, pollID)
		require.NoError(t, err)
		require.Lenf(t, votes, 1, "round %d: user must have exactly one vote on a single-choice poll, got %d", i, len(votes))
	}
}

func TestPollRepository_Delete(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del Voter', 'del-poll@example.com', '#ffffff')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Poll Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	creatorID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, creatorID, "To Delete?", false, false, []string{"Yes"})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, pollID, teamID))

	list, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestPollRepository_FindByID_CrossTeamBlocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	otherTid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Cross Team User', 'cross-poll@example.com', '#123123')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Cross Team A')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Cross Team B')`,
		otherTid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	otherTeamID := uuid.MustParse(otherTid)
	creatorID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, creatorID, "Team A only?", false, false, []string{"Yes", "No"})
	require.NoError(t, err)

	// A poll belonging to team A must not be found or deletable when scoped to team B.
	_, err = repo.FindByID(ctx, pollID, otherTeamID)
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	err = repo.Delete(ctx, pollID, otherTeamID)
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	// It should still exist and be found/deletable when scoped correctly.
	pr, err := repo.FindByID(ctx, pollID, teamID)
	require.NoError(t, err)
	assert.Equal(t, pollID, pr.Id)
	require.NoError(t, repo.Delete(ctx, pollID, teamID))
}
