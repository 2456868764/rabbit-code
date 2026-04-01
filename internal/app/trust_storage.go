package app

import (
	"errors"
	"os"
	"path/filepath"
)

// TrustedMarkerFile is written under GlobalConfigDir after the user accepts the trust screen (PHASE01_UI UI1).
const TrustedMarkerFile = "trusted.marker"

func trustMarkerPath(globalDir string) string {
	return filepath.Join(globalDir, TrustedMarkerFile)
}

// TrustAccepted reports whether the local trust marker exists.
func TrustAccepted(globalDir string) (bool, error) {
	if globalDir == "" {
		return false, errors.New("empty global config dir")
	}
	st, err := os.Stat(trustMarkerPath(globalDir))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !st.Mode().IsRegular() {
		return false, nil
	}
	return st.Size() > 0, nil
}

// WriteTrustMarker creates globalDir if needed and records trust acceptance.
func WriteTrustMarker(globalDir string) error {
	if err := os.MkdirAll(globalDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(trustMarkerPath(globalDir), []byte("v1\n"), 0o600)
}
