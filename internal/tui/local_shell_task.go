package tui

// LocalShellTask mirrors the LocalShellTask shape from Claude Code query layer: a host-run shell job
// with stable id and merged output path (see query.ts “LocalShellTask completions”).
type LocalShellTask struct {
	ID           string
	Command      string
	WorkingDir   string
	OutputPath   string
	ExitCode     int
	Interrupted  bool
	StartedUnix  int64
	FinishedUnix int64
}

// TaskStatus is a coarse lifecycle for Bubble Tea / CLI views.
type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskDone
	TaskError
)

func (t *LocalShellTask) Status() TaskStatus {
	switch {
	case t.ID == "":
		return TaskPending
	case t.FinishedUnix == 0 && t.StartedUnix != 0:
		return TaskRunning
	case t.Interrupted || t.ExitCode != 0:
		return TaskError
	default:
		return TaskDone
	}
}
