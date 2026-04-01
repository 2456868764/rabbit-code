// Package migrations runs versioned config transforms (schema_version) on merged settings maps.
package migrations

import (
	"fmt"
)

// CurrentSchemaVersion is written to new saves and is the target after Apply.
const CurrentSchemaVersion = 2

// Apply runs migrations from the map's schema_version up to CurrentSchemaVersion.
// If "schema_version" is absent, the map is treated as already current (no migration;
// handwritten configs without a version field are left unchanged).
func Apply(m map[string]interface{}) error {
	if len(m) == 0 {
		return nil
	}
	v, explicit := readVersionExplicit(m)
	if !explicit {
		return nil
	}
	if v < 1 {
		v = 1
	}
	for v < CurrentSchemaVersion {
		fn, ok := steps[v]
		if !ok {
			return fmt.Errorf("migrations: no step from version %d", v)
		}
		if err := fn(m); err != nil {
			return fmt.Errorf("migrations: v%d→v%d: %w", v, v+1, err)
		}
		v++
		setVersion(m, v)
	}
	return nil
}

func readVersionExplicit(m map[string]interface{}) (v int, explicit bool) {
	raw, ok := m["schema_version"]
	if !ok {
		return CurrentSchemaVersion, false
	}
	switch x := raw.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	default:
		return 1, true
	}
}

func setVersion(m map[string]interface{}, v int) {
	m["schema_version"] = float64(v) // JSON-friendly
}

// steps[v] migrates from v to v+1.
var steps = map[int]func(map[string]interface{}) error{
	1: migrate1To2,
}

// migrate1To2 example: ensure Phase-2 canonical keys exist (no-op rename sample for tests).
func migrate1To2(m map[string]interface{}) error {
	if _, ok := m["migrated_from_v1"]; !ok {
		m["migrated_from_v1"] = true
	}
	return nil
}
