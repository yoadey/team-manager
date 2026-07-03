package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// listenOnPort starts httptest server on a fixed port so PORT/127.0.0.1
// resolution in run() matches, since httptest.NewServer normally picks an
// ephemeral port bound to 127.0.0.1 already — we just need to read it back.
func startServer(t *testing.T, handler http.HandlerFunc) (port string, closeFn func()) {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := httptest.NewUnstartedServer(handler)
	if err := srv.Listener.Close(); err != nil { // replaced by ln below
		t.Fatalf("close default listener: %v", err)
	}
	srv.Listener = ln
	srv.Start()
	_, p, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	return p, srv.Close
}

func TestRun_HealthyServer_ReturnsZero(t *testing.T) {
	port, closeFn := startServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	defer closeFn()

	t.Setenv("PORT", port)
	if got := run(); got != 0 {
		t.Errorf("run() = %d, want 0", got)
	}
}

func TestRun_UnhealthyStatus_ReturnsOne(t *testing.T) {
	port, closeFn := startServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	defer closeFn()

	t.Setenv("PORT", port)
	if got := run(); got != 1 {
		t.Errorf("run() = %d, want 1", got)
	}
}

func TestParsePort(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", defaultPort},
		{"8080", 8080},
		{"3000", 3000},
		{"0", defaultPort},
		{"-1", defaultPort},
		{"70000", defaultPort},
		{"not-a-port", defaultPort},
		{"8080; rm -rf /", defaultPort},
	}
	for _, c := range cases {
		if got := parsePort(c.in); got != c.want {
			t.Errorf("parsePort(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRun_NoServer_ReturnsOne(t *testing.T) {
	// An unused port — connection should fail outright.
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	t.Setenv("PORT", port)
	if got := run(); got != 1 {
		t.Errorf("run() = %d, want 1", got)
	}
}
