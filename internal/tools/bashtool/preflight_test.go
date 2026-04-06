package bashtool_test

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

func TestValidatePipePermissionPreflight_cdGitPipe(t *testing.T) {
	err := bashtool.ValidatePipePermissionPreflight("cd /tmp | git status")
	if err == nil || !strings.Contains(err.Error(), "cd and git") {
		t.Fatalf("got %v", err)
	}
	if bashtool.ValidatePipePermissionPreflight("git status") != nil {
		t.Fatal("single segment should pass")
	}
}

func TestValidatePipePermissionPreflight_multiCdPipe(t *testing.T) {
	err := bashtool.ValidatePipePermissionPreflight("cd a | cd b")
	if err == nil || !strings.Contains(err.Error(), "multiple directory") {
		t.Fatalf("got %v", err)
	}
}

func TestStripSafeWrappers_git(t *testing.T) {
	if !bashtool.IsNormalizedGitCommand("NO_COLOR=1 git status") {
		t.Fatal("expected git after strip")
	}
}

func TestBash_readOnlyRejectsRm(t *testing.T) {
	t.Setenv(features.EnvBashReadOnly, "1")
	t.Setenv(features.EnvBashExec, "1")
	_, err := bashtool.New().Run(t.Context(), []byte(`{"command":"rm -f x"}`))
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("got %v", err)
	}
}

func TestExtractBashCommentLabel(t *testing.T) {
	if bashtool.ExtractBashCommentLabel("# sync\nls") != "sync" {
		t.Fatal()
	}
	if bashtool.ExtractBashCommentLabel("#!/bin/sh\nls") != "" {
		t.Fatal()
	}
}

func TestGetDestructiveCommandWarning(t *testing.T) {
	if bashtool.GetDestructiveCommandWarning("git reset --hard") == "" {
		t.Fatal()
	}
}

func TestStripEmptyLines(t *testing.T) {
	if bashtool.StripEmptyLines("\n\na\n\n") != "a" {
		t.Fatal(bashtool.StripEmptyLines("\n\na\n\n"))
	}
}

func TestCheckPermissionMode_acceptEdits(t *testing.T) {
	r := bashtool.CheckPermissionMode("mkdir foo", bashtool.ModeAcceptEdits)
	if !r.Allow {
		t.Fatal(r)
	}
}
