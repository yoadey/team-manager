package sqlbuilder_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/db/sqlbuilder"
)

func TestBuilder_Empty_ReportsNotOk(t *testing.T) {
	t.Parallel()
	b := sqlbuilder.New()
	assert.True(t, b.Empty())

	setSQL, args, nextIdx, ok := b.Build(1)
	assert.False(t, ok)
	assert.Empty(t, setSQL)
	assert.Nil(t, args)
	assert.Equal(t, 1, nextIdx)
}

func TestBuilder_SingleColumn(t *testing.T) {
	t.Parallel()
	b := sqlbuilder.New()
	b.Add("title", "New Title")
	assert.False(t, b.Empty())

	setSQL, args, nextIdx, ok := b.Build(1)
	require.True(t, ok)
	assert.Equal(t, "title = $1", setSQL)
	assert.Equal(t, []any{"New Title"}, args)
	assert.Equal(t, 2, nextIdx)
}

func TestBuilder_MultipleColumns_NumberedInOrder(t *testing.T) {
	t.Parallel()
	b := sqlbuilder.New()
	b.Add("title", "New Title").Add("body", "New Body").Add("pinned", true)

	setSQL, args, nextIdx, ok := b.Build(1)
	require.True(t, ok)
	assert.Equal(t, "title = $1, body = $2, pinned = $3", setSQL)
	assert.Equal(t, []any{"New Title", "New Body", true}, args)
	assert.Equal(t, 4, nextIdx)
}

// TestBuilder_StartIdx_ComposesWithPriorPlaceholders verifies a non-1
// startIdx (e.g. after an UPDATE ... FROM subquery or a prior CTE already
// consumed placeholders $1/$2) numbers correctly and returns the right
// nextIdx for the caller's WHERE clause to continue from.
func TestBuilder_StartIdx_ComposesWithPriorPlaceholders(t *testing.T) {
	t.Parallel()
	b := sqlbuilder.New()
	b.Add("amount", int64(500))

	setSQL, args, nextIdx, ok := b.Build(3)
	require.True(t, ok)
	assert.Equal(t, "amount = $3", setSQL)
	assert.Equal(t, []any{int64(500)}, args)
	assert.Equal(t, 4, nextIdx)
}

// TestBuilder_WhereArgsStaySeparate documents the intended call-site
// composition: WHERE args are appended by the caller after Build returns,
// using nextIdx -- never interleaved into the Builder itself. This is what
// replaces the old hand-rolled builders' shared running counter (the source
// of the historical id=$N fallback bug), and this test pins that no
// Builder API accidentally reintroduces WHERE-arg awareness into Build.
func TestBuilder_WhereArgsStaySeparate(t *testing.T) {
	t.Parallel()
	b := sqlbuilder.New()
	b.Add("label", "New Label")

	setSQL, setArgs, nextIdx, ok := b.Build(1)
	require.True(t, ok)

	whereArgs := []any{"row-id", "team-id"}
	allArgs := append(append([]any{}, setArgs...), whereArgs...)

	assert.Equal(t, "label = $1", setSQL)
	assert.Equal(t, 2, nextIdx)
	assert.Equal(t, []any{"New Label", "row-id", "team-id"}, allArgs)
}
