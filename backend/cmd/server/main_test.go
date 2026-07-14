package main

import (
	"runtime/debug"
	"testing"
	"time"
)

// Regression test: httpSrv.WriteTimeout used to equal chimiddleware.Timeout's
// requestTimeout exactly (both 30s). chi's Timeout doesn't preemptively abort
// a running handler -- it only writes a 504 in a defer AFTER next.ServeHTTP
// returns, so with WriteTimeout equal to requestTimeout, the connection's
// write deadline (reset when headers are read, so it starts at roughly the
// same instant) can elapse before that deferred write happens, silently
// dropping the RFC 9457 error body in favor of a bare connection reset.
// httpWriteTimeout must stay strictly greater than requestTimeout with real
// margin.
func TestHTTPWriteTimeout_ExceedsRequestTimeout(t *testing.T) {
	if httpWriteTimeout <= requestTimeout {
		t.Fatalf("httpWriteTimeout (%s) must be greater than requestTimeout (%s)", httpWriteTimeout, requestTimeout)
	}
	const minMargin = 5 * time.Second // well under the 10s actually configured
	if httpWriteTimeout-requestTimeout < minMargin {
		t.Fatalf("httpWriteTimeout (%s) leaves too little margin over requestTimeout (%s)", httpWriteTimeout, requestTimeout)
	}
}

// TestApplyMemoryLimitHeadroom_ScalesDownFromContainerLimit is a regression
// test for GOMEMLIMIT being sourced straight from the container's hard
// cgroup memory limit with no headroom (deployment.yaml's downward-API env
// var, see gomemlimitHeadroomFactor's doc comment) -- GOMEMLIMIT only
// accounts for Go-runtime-tracked memory, not other RSS contributors, so
// setting it equal to the hard limit undermines the OOM protection it
// exists to provide. Restores the runtime's previous memory limit
// afterward so this test doesn't leak state into siblings in the same
// process.
func TestApplyMemoryLimitHeadroom_ScalesDownFromContainerLimit(t *testing.T) {
	prev := debug.SetMemoryLimit(-1) // -1 reads the current limit without changing it
	t.Cleanup(func() { debug.SetMemoryLimit(prev) })

	var rawLimit int64 = 268435456 // 256Mi, matching the default resources.limits.memory profile
	t.Setenv("GOMEMLIMIT", "268435456")

	applyMemoryLimitHeadroom()

	got := debug.SetMemoryLimit(-1)
	want := int64(float64(rawLimit) * gomemlimitHeadroomFactor)
	if got != want {
		t.Fatalf("applyMemoryLimitHeadroom() set the runtime limit to %d, want %d (%.0f%% of GOMEMLIMIT)", got, want, gomemlimitHeadroomFactor*100)
	}
	if got >= 268435456 {
		t.Fatalf("applyMemoryLimitHeadroom() left the runtime limit (%d) at or above the raw container limit (268435456) -- no headroom", got)
	}
}

// Companion test: an unset GOMEMLIMIT (local dev, tests, cmd/healthcheck)
// must leave the runtime's memory limit untouched rather than panicking or
// setting it to zero.
func TestApplyMemoryLimitHeadroom_NoopWhenUnset(t *testing.T) {
	prev := debug.SetMemoryLimit(-1)
	t.Cleanup(func() { debug.SetMemoryLimit(prev) })

	t.Setenv("GOMEMLIMIT", "")

	applyMemoryLimitHeadroom()

	if got := debug.SetMemoryLimit(-1); got != prev {
		t.Fatalf("applyMemoryLimitHeadroom() changed the runtime limit from %d to %d despite GOMEMLIMIT being unset", prev, got)
	}
}
