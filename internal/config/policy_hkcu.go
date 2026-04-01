package config

// policyHKCUSnapshot returns registry-backed policy on Windows when implemented; empty elsewhere (P2.3.1).
func policyHKCUSnapshot() map[string]interface{} {
	return map[string]interface{}{}
}
