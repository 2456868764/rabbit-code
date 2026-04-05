package anthropic

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func newRandomHexID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

const rabbitConfigDirName = "rabbit-code"
const deviceIDFileName = "device_id"

var (
	deviceIDMu     sync.Mutex
	deviceIDCached string
)

// deviceIDStateDir returns a per-user config directory for persisted CLI state (getOrCreateUserID analogue).
func deviceIDStateDir() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, rabbitConfigDirName), nil
}

// LoadOrCreateDeviceID returns a stable device id: RABBIT_CODE_DEVICE_ID if set, else file-backed UUID in user config.
func LoadOrCreateDeviceID() string {
	if s := strings.TrimSpace(os.Getenv(EnvRabbitDeviceID)); s != "" {
		return s
	}
	deviceIDMu.Lock()
	defer deviceIDMu.Unlock()
	if deviceIDCached != "" {
		return deviceIDCached
	}
	dir, err := deviceIDStateDir()
	if err != nil {
		deviceIDCached = newRandomHexID()
		return deviceIDCached
	}
	path := filepath.Join(dir, deviceIDFileName)
	if b, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(b))
		if id != "" {
			deviceIDCached = id
			return id
		}
	}
	id := newRandomHexID()
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(path, []byte(id), 0o600)
	deviceIDCached = id
	return id
}
