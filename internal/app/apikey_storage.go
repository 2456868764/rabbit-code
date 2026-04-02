package app

import (
	"os"
	"path/filepath"
	"strings"
)

const apiKeyFileName = "api_key"

// API key resolution order: ANTHROPIC_API_KEY, RABBIT_CODE_API_KEY, then file under global config dir.
var apiKeyEnvNames = []string{"ANTHROPIC_API_KEY", "RABBIT_CODE_API_KEY"}

func apiKeyFilePath(globalDir string) string {
	return filepath.Join(globalDir, apiKeyFileName)
}

// HasAPIKeyConfigured returns true if any supported env var or the saved api_key file is non-empty.
func HasAPIKeyConfigured(globalDir string) bool {
	return ReadAPIKey(globalDir) != ""
}

// ReadAPIKey returns the API key from the first non-empty env (ANTHROPIC_API_KEY, RABBIT_CODE_API_KEY),
// else from globalDir/api_key. Empty string if unset.
func ReadAPIKey(globalDir string) string {
	for _, name := range apiKeyEnvNames {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	b, err := os.ReadFile(apiKeyFilePath(globalDir))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// WriteAPIKeyFile stores the key for later phases (mode 0600). Trims surrounding whitespace.
func WriteAPIKeyFile(globalDir, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(globalDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(apiKeyFilePath(globalDir), []byte(key+"\n"), 0o600)
}
