package features

import "testing"

func TestHardFailEnabled(t *testing.T) {
	t.Setenv(EnvHardFail, "")
	if HardFailEnabled() {
		t.Fatal("expected false")
	}
	t.Setenv(EnvHardFail, "1")
	if !HardFailEnabled() {
		t.Fatal("expected true")
	}
}

func TestSlowOperationLoggingEnabled(t *testing.T) {
	t.Setenv(EnvSlowOperationLogging, "true")
	if !SlowOperationLoggingEnabled() {
		t.Fatal("expected true")
	}
}

func TestFilePersistenceEnabled(t *testing.T) {
	t.Setenv(EnvFilePersistence, "0")
	if FilePersistenceEnabled() {
		t.Fatal("expected false")
	}
}

func TestLodestoneEnabled(t *testing.T) {
	t.Setenv(EnvLodestone, "yes")
	if !LodestoneEnabled() {
		t.Fatal("expected true")
	}
}
