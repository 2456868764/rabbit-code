package bashtool

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BackgroundOutputPath returns the on-disk path used for a task's merged stdout/stderr (RABBIT_TASK_OUTPUT_DIR or temp).
func BackgroundOutputPath(taskID string) string {
	return backgroundOutputPath(taskID)
}

func backgroundOutputPath(taskID string) string {
	if dir := strings.TrimSpace(os.Getenv("RABBIT_TASK_OUTPUT_DIR")); dir != "" {
		return filepath.Join(dir, taskID+".output")
	}
	return filepath.Join(os.TempDir(), "rabbit-code-bash-bg-"+taskID+".output")
}

func newBackgroundTaskID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type bgEntry struct {
	done chan struct{}
	err  error
}

var backgroundRegistry sync.Map // taskID -> *bgEntry

// startBackgroundCommand spawns sh -c in a goroutine; stdout and stderr are merged to outPath. timeoutMs clamps like foreground.
func startBackgroundCommand(cmdStr string, timeoutMs int) (taskID, outPath string, err error) {
	taskID = newBackgroundTaskID()
	outPath = backgroundOutputPath(taskID)
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", "", err
	}
	ent := &bgEntry{done: make(chan struct{})}
	backgroundRegistry.Store(taskID, ent)

	go func() {
		defer f.Close()
		defer close(ent.done)
		cctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
		cmd := exec.CommandContext(cctx, "sh", "-c", cmdStr)
		cmd.Stdout = f
		cmd.Stderr = f
		ent.err = cmd.Run()
	}()

	return taskID, outPath, nil
}

// WaitBackgroundTask blocks until the background shell exits or ctx is cancelled.
func WaitBackgroundTask(ctx context.Context, taskID string) error {
	v, ok := backgroundRegistry.Load(taskID)
	if !ok {
		return errors.New("bashtool: unknown background task id")
	}
	ent := v.(*bgEntry)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ent.done:
		return ent.err
	}
}
