package main

import (
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
