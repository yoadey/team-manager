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
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/healthz", http.NoBody)
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
