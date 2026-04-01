package app

import (
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestNewSlowLogger_disabled(t *testing.T) {
	t.Setenv(features.EnvSlowOperationLogging, "")
	log, closeFn, err := NewSlowLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()
	if log != nil {
		t.Fatal("expected nil logger when feature off")
	}
}

func TestNewSlowLogger_enabled(t *testing.T) {
	t.Setenv(features.EnvSlowOperationLogging, "1")
	t.Setenv(envSlowLogFile, "")
	log, closeFn, err := NewSlowLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()
	if log == nil {
		t.Fatal("expected logger")
	}
}

func TestNewSlowLogger_fileSink(t *testing.T) {
	t.Setenv(features.EnvSlowOperationLogging, "true")
	dir := t.TempDir()
	path := filepath.Join(dir, "slow.log")
	t.Setenv(envSlowLogFile, path)
	log, closeFn, err := NewSlowLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()
	if log == nil {
		t.Fatal("expected logger")
	}
	log.Info("test slow line")
	closeFn()
}
