// Command healthcheck is a tiny standalone binary invoked by Docker's
// HEALTHCHECK / docker-compose's healthcheck.test. It exists because the
// production image (gcr.io/distroless/static-debian12) ships no shell and no
// wget/curl — only the server binary and CA certs — so a CMD-SHELL healthcheck
// like "wget -qO- ... || exit 1" cannot exec at all and always reports
// unhealthy regardless of whether the server is actually up.
package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	port := parsePort(os.Getenv("PORT"))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// The host is always the hardcoded loopback address; only the numeric
	// port varies, and parsePort rejects anything that isn't a valid port
	// number. This is not an SSRF sink despite building the URL from an env
	// var — PORT is trusted deployment config, not attacker-controlled input,
	// and the validated port can't smuggle a different host/path into the URL.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+"/healthz", http.NoBody)
	if err != nil {
		return 1
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

// defaultPort matches the PORT default documented in CLAUDE.md / config.go.
const defaultPort = 8080

// parsePort validates s as a TCP port number (1-65535), falling back to
// defaultPort when s is empty or not a valid port.
func parsePort(s string) int {
	if s == "" {
		return defaultPort
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 65535 {
		return defaultPort
	}
	return n
}
