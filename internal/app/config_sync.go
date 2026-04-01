package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/config"
)

// RunConfigSyncPull downloads JSON from url (or merged download_user_settings_url) into user config.json.
func RunConfigSyncPull(ctx context.Context, urlOverride string) error {
	globalDir, err := GlobalConfigDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := FindProjectRoot(cwd)
	m, err := config.LoadMerged(config.Paths{GlobalConfigDir: globalDir, ProjectRoot: root})
	if err != nil {
		return err
	}
	url := strings.TrimSpace(urlOverride)
	if url == "" {
		if u, ok := m["download_user_settings_url"].(string); ok {
			url = strings.TrimSpace(u)
		}
	}
	if url == "" {
		return fmt.Errorf("sync pull: set download_user_settings_url in config or pass URL argument")
	}
	return config.SyncPullToUserFile(ctx, url, globalDir)
}

// RunConfigSyncPush POSTs user config.json to url (or merged upload_user_settings_url).
func RunConfigSyncPush(ctx context.Context, urlOverride string) error {
	globalDir, err := GlobalConfigDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := FindProjectRoot(cwd)
	m, err := config.LoadMerged(config.Paths{GlobalConfigDir: globalDir, ProjectRoot: root})
	if err != nil {
		return err
	}
	url := strings.TrimSpace(urlOverride)
	if url == "" {
		if u, ok := m["upload_user_settings_url"].(string); ok {
			url = strings.TrimSpace(u)
		}
	}
	if url == "" {
		return fmt.Errorf("sync push: set upload_user_settings_url in config or pass URL argument")
	}
	return config.SyncPushFromUserFile(ctx, url, globalDir)
}
