package mapping

import (
	"path/filepath"
	"testing"
)

func TestLoadState_NewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if len(s.Events) != 0 || len(s.Absences) != 0 {
		t.Errorf("new state should start empty, got %+v", s)
	}
}

func TestState_SaveAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := LoadState(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Events["sp-101"] = "tv-uuid-1"
	s.Absences["sp-9:2026-06-01"] = "tv-uuid-2"
	if err := s.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("reload LoadState() error = %v", err)
	}
	if reloaded.Events["sp-101"] != "tv-uuid-1" {
		t.Errorf("reloaded Events[sp-101] = %q, want tv-uuid-1", reloaded.Events["sp-101"])
	}
	if reloaded.Absences["sp-9:2026-06-01"] != "tv-uuid-2" {
		t.Errorf("reloaded Absences[sp-9:2026-06-01] = %q, want tv-uuid-2", reloaded.Absences["sp-9:2026-06-01"])
	}
}
