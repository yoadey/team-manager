// Package sqlbuilder builds dynamic "UPDATE ... SET" clauses for partial
// (PATCH-style) updates whose column set is only known at request time and
// therefore can't be expressed as a single static sqlc query.
//
// It replaces the hand-rolled per-repository SET builders that used to fall
// back to a placeholder no-op ("SET id = $N") when nothing was actually
// being updated. That fallback assumed the caller's next placeholder always
// bound to the row's own primary key in the WHERE clause -- an assumption
// that has already broken once (see events.updateSeriesEvents' historical
// bug, guarded today by TestEventRepository_UpdateEvent_Series_OnlyDateSet_DoesNotCorruptSQL):
// a caller with a different WHERE shape can end up with the no-op
// placeholder binding to the wrong column, silently corrupting unrelated
// rows. Builder.Build reports an empty set explicitly via its ok return
// instead, so callers must decide what "nothing to update" means for their
// own query rather than relying on a shared, easy-to-misuse fallback.
package sqlbuilder

import (
	"fmt"
	"strings"
)

// set is one column/value pair queued for a SET clause.
type set struct {
	column string
	value  any
}

// Builder accumulates the columns and values for a dynamic UPDATE ... SET
// clause. WHERE-clause placeholders are never part of a Builder -- Build
// returns the next unused placeholder index so the caller numbers its own
// WHERE args starting there, keeping SET and WHERE argument lists visibly
// separate instead of interleaved by a shared running counter.
type Builder struct {
	sets []set
}

// New creates an empty Builder.
func New() *Builder {
	return &Builder{}
}

// Add queues column = <next placeholder> with value as its bound argument.
// Callers only call Add for fields actually present in the patch (typically
// guarded by a "if patch.Field != nil" check); Add itself does not filter.
// Returns the Builder for chaining.
func (b *Builder) Add(column string, value any) *Builder {
	b.sets = append(b.sets, set{column: column, value: value})
	return b
}

// Empty reports whether no column has been queued yet.
func (b *Builder) Empty() bool {
	return len(b.sets) == 0
}

// Build renders the queued columns into a "col1 = $N, col2 = $N+1, ..." SET
// clause, with placeholders numbered starting at startIdx, and returns the
// bound arguments in the same order. nextIdx is the first placeholder index
// not used by this clause -- pass it as the starting index for the
// caller's own WHERE arguments so the two argument lists compose without
// manual index arithmetic at the call site.
//
// ok is false when no column was queued (Empty() is true); setSQL/args/
// nextIdx are all zero-valued in that case, and callers must not execute a
// query built from them -- there is deliberately no placeholder fallback
// (see the package doc comment for why).
func (b *Builder) Build(startIdx int) (setSQL string, args []any, nextIdx int, ok bool) {
	if len(b.sets) == 0 {
		return "", nil, startIdx, false
	}
	parts := make([]string, 0, len(b.sets))
	args = make([]any, 0, len(b.sets))
	idx := startIdx
	for _, s := range b.sets {
		parts = append(parts, fmt.Sprintf("%s = $%d", s.column, idx))
		args = append(args, s.value)
		idx++
	}
	return strings.Join(parts, ", "), args, idx, true
}
