package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yoadey/team-manager/tools/spielerplus-import/spielerplus"
)

// Config holds everything the importer needs, sourced from environment
// variables so no secrets end up on the command line / in shell history.
type Config struct {
	// DatabaseURL is the Teamverwaltung Postgres DSN (same as the backend's
	// DATABASE_URL).
	DatabaseURL string
	// TeamID is the Teamverwaltung team the imported data is written into.
	// The team must already exist.
	TeamID string
	// SpielerPlusSessionCookie is the raw "Cookie:" header value captured
	// from a logged-in browser session against spielerplus.de.
	SpielerPlusSessionCookie string
	// RoleMappingPath points at a YAML file mapping SpielerPlus role names
	// to existing Teamverwaltung role names in TeamID.
	RoleMappingPath string
	// StatePath points at the local JSON idempotency state file.
	StatePath string
	// ExpectedTeamName, if set, is compared against the team name detected
	// as currently active in the SpielerPlus session (see
	// SPIELERPLUS_EXPECTED_TEAM_NAME): the run proceeds automatically on a
	// match and aborts on a mismatch, without an interactive prompt. Left
	// empty, the detected name is shown for interactive
	// confirmation instead - either way, this guards against an account
	// that manages more than one SpielerPlus team having the wrong one
	// active when its session cookie was captured (SpielerPlus scopes
	// every page this importer reads by that session-level active team,
	// with no team id in the URL to double check against).
	ExpectedTeamName string
	// RequestDelay is the minimum gap enforced between requests to
	// SpielerPlus (see SPIELERPLUS_REQUEST_DELAY), so a long member/event
	// list can't hammer it in a tight loop and draw attention or trip a
	// rate limit.
	RequestDelay time.Duration
	// DryRun, when true, performs the full read/scrape + mapping pipeline
	// but writes nothing to the database.
	DryRun bool

	// S3Endpoint/S3Region/S3Bucket/S3AccessKeyID/S3SecretAccessKey/
	// S3UsePathStyle configure the object store member photos are uploaded
	// to, using the same env var names and semantics as the backend's own
	// S3-compatible object storage (see CLAUDE.md) - point this at the same
	// bucket the backend uses so an imported photo is retrievable through
	// the normal app. Photo import is entirely optional: leaving
	// S3_ENDPOINT/S3_BUCKET unset (see PhotoImportEnabled) skips it, rather
	// than failing the run - a club can always add photos later.
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3UsePathStyle    bool
}

// PhotoImportEnabled reports whether enough S3 configuration was provided
// to attempt member photo import at all.
func (c *Config) PhotoImportEnabled() bool {
	return c.S3Endpoint != "" && c.S3Bucket != ""
}

func loadConfig(dryRun bool) (*Config, error) {
	cfg := &Config{
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		TeamID:                   os.Getenv("TEAM_ID"),
		SpielerPlusSessionCookie: os.Getenv("SPIELERPLUS_SESSION_COOKIE"),
		RoleMappingPath:          os.Getenv("ROLE_MAPPING_PATH"),
		StatePath:                os.Getenv("STATE_PATH"),
		ExpectedTeamName:         os.Getenv("SPIELERPLUS_EXPECTED_TEAM_NAME"),
		DryRun:                   dryRun,
		S3Endpoint:               os.Getenv("S3_ENDPOINT"),
		S3Region:                 os.Getenv("S3_REGION"),
		S3Bucket:                 os.Getenv("S3_BUCKET"),
		S3AccessKeyID:            os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretAccessKey:        os.Getenv("S3_SECRET_ACCESS_KEY"),
		S3UsePathStyle:           os.Getenv("S3_USE_PATH_STYLE") == "true",
	}
	if cfg.StatePath == "" {
		cfg.StatePath = "spielerplus-import-state.json"
	}

	cfg.RequestDelay = spielerplus.DefaultRequestDelay
	if raw := os.Getenv("SPIELERPLUS_REQUEST_DELAY"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid SPIELERPLUS_REQUEST_DELAY %q: %w (expected a Go duration like \"500ms\" or \"2s\"; use \"0\" to disable throttling)", raw, err)
		}
		if d < 0 {
			return nil, fmt.Errorf("invalid SPIELERPLUS_REQUEST_DELAY %q: must not be negative", raw)
		}
		cfg.RequestDelay = d
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.TeamID == "" {
		missing = append(missing, "TEAM_ID")
	}
	if cfg.SpielerPlusSessionCookie == "" {
		missing = append(missing, "SPIELERPLUS_SESSION_COOKIE")
	}
	if cfg.RoleMappingPath == "" {
		missing = append(missing, "ROLE_MAPPING_PATH")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variable(s): %v", missing)
	}
	return cfg, nil
}
