package main

import (
	"fmt"
	"os"
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
	// DryRun, when true, performs the full read/scrape + mapping pipeline
	// but writes nothing to the database.
	DryRun bool
}

func loadConfig(dryRun bool) (*Config, error) {
	cfg := &Config{
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		TeamID:                   os.Getenv("TEAM_ID"),
		SpielerPlusSessionCookie: os.Getenv("SPIELERPLUS_SESSION_COOKIE"),
		RoleMappingPath:          os.Getenv("ROLE_MAPPING_PATH"),
		StatePath:                os.Getenv("STATE_PATH"),
		DryRun:                   dryRun,
	}
	if cfg.StatePath == "" {
		cfg.StatePath = "spielerplus-import-state.json"
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
