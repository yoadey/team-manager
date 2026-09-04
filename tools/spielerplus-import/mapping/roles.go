// Package mapping handles the operator-supplied SpielerPlus-role ->
// Teamverwaltung-role table and the local idempotency state file.
package mapping

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RoleMapping maps a SpielerPlus role name (as scraped, e.g. "Trainer") to
// the name of an existing Teamverwaltung role in the target team.
type RoleMapping map[string]string

// LoadRoleMapping reads a YAML file of the form:
//
//	Trainer: Trainer
//	Co-Trainer: Trainer
//	Spieler: Member
func LoadRoleMapping(path string) (RoleMapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("mapping: read role mapping %s: %w", path, err)
	}
	var m RoleMapping
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("mapping: parse role mapping %s: %w", path, err)
	}
	if len(m) == 0 {
		return nil, fmt.Errorf("mapping: role mapping %s is empty", path)
	}
	return m, nil
}

// Resolve looks up the Teamverwaltung role name for a SpielerPlus role,
// failing loudly (per spec: "Unmapped SpielerPlus roles fail loudly") rather
// than defaulting silently.
func (m RoleMapping) Resolve(spielerPlusRole string) (string, error) {
	role, ok := m[spielerPlusRole]
	if !ok {
		return "", fmt.Errorf("mapping: no role mapping entry for SpielerPlus role %q - add it to the role mapping file", spielerPlusRole)
	}
	return role, nil
}
