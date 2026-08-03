package mapping

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRoleMapping(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roles.yaml")
	content := "Trainer: Trainer\nCo-Trainer: Trainer\nSpieler: Member\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	m, err := LoadRoleMapping(path)
	if err != nil {
		t.Fatalf("LoadRoleMapping() error = %v", err)
	}

	got, err := m.Resolve("Co-Trainer")
	if err != nil {
		t.Fatalf("Resolve(Co-Trainer) error = %v", err)
	}
	if got != "Trainer" {
		t.Errorf("Resolve(Co-Trainer) = %q, want Trainer", got)
	}
}

func TestRoleMapping_Resolve_Unmapped(t *testing.T) {
	m := RoleMapping{"Trainer": "Trainer"}
	if _, err := m.Resolve("Zeugwart"); err == nil {
		t.Fatal("expected an error for an unmapped SpielerPlus role")
	}
}

func TestLoadRoleMapping_MissingFile(t *testing.T) {
	if _, err := LoadRoleMapping("/nonexistent/roles.yaml"); err == nil {
		t.Fatal("expected an error for a missing role mapping file")
	}
}

func TestLoadRoleMapping_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRoleMapping(path); err == nil {
		t.Fatal("expected an error for an empty role mapping file")
	}
}
