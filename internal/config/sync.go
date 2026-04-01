package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnvSettingsSyncAuthorization is optional; when set, added as HTTP Authorization header for sync requests.
const EnvSettingsSyncAuthorization = "RABBIT_CODE_SETTINGS_SYNC_AUTHORIZATION"

// SyncPullToUserFile GETs JSON from url and deep-merges into the user config file (download overlays existing keys).
func SyncPullToUserFile(ctx context.Context, url, globalConfigDir string) error {
	if globalConfigDir == "" {
		return fmt.Errorf("config: globalConfigDir required")
	}
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("sync: url required")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(url), nil)
	if err != nil {
		return err
	}
	if h := strings.TrimSpace(os.Getenv(EnvSettingsSyncAuthorization)); h != "" {
		req.Header.Set("Authorization", h)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("sync pull: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	var downloaded map[string]interface{}
	if err := json.Unmarshal(body, &downloaded); err != nil {
		return fmt.Errorf("sync pull: response JSON: %w", err)
	}
	if downloaded == nil {
		downloaded = map[string]interface{}{}
	}

	path := filepath.Join(globalConfigDir, UserConfigFileName)
	user, err := ReadJSONFile(path)
	if err != nil {
		return err
	}
	DeepMerge(user, downloaded)
	if verrs := Validate(user); len(verrs) > 0 {
		return fmt.Errorf("validation after merge: %v", verrs)
	}
	return AtomicWriteJSON(path, user)
}

// SyncPushFromUserFile POSTs the user config JSON to url (P2.F.3 minimal).
func SyncPushFromUserFile(ctx context.Context, url, globalConfigDir string) error {
	if globalConfigDir == "" {
		return fmt.Errorf("config: globalConfigDir required")
	}
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("sync: url required")
	}
	path := filepath.Join(globalConfigDir, UserConfigFileName)
	user, err := ReadJSONFile(path)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(url), bytes.NewReader(b))
	if err != nil {
		return err
	}
	if h := strings.TrimSpace(os.Getenv(EnvSettingsSyncAuthorization)); h != "" {
		req.Header.Set("Authorization", h)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("sync push: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	return nil
}
