package todowritetool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// TodoWrite implements tools.Tool for TodoWriteTool.ts.
type TodoWrite struct {
	// NonInteractive mirrors LoopDriver.NonInteractive / isNonInteractiveSession when tool runs without engine wiring.
	NonInteractive bool
}

// New returns a TodoWrite tool (enablement uses features.TodoWriteToolEnabled when NonInteractive is set from host).
func New() *TodoWrite { return &TodoWrite{} }

func (t *TodoWrite) Name() string { return TodoWriteToolName }

func (t *TodoWrite) Aliases() []string { return nil }

// TodoItem mirrors utils/todo/types.ts TodoItem.
type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm"`
}

type todoWriteInput struct {
	Todos []TodoItem `json:"todos"`
}

var verifHint = regexp.MustCompile(`(?i)verif`)

func validateTodos(list []TodoItem) error {
	for i, it := range list {
		// z.string().min(1): length in code units; do not trim (whitespace-only strings are valid).
		if len(it.Content) < 1 {
			return fmt.Errorf("todos[%d].content: Content cannot be empty", i)
		}
		if len(it.ActiveForm) < 1 {
			return fmt.Errorf("todos[%d].activeForm: Active form cannot be empty", i)
		}
		switch it.Status {
		case "pending", "in_progress", "completed":
		default:
			return fmt.Errorf("todos[%d].status: must be pending, in_progress, or completed", i)
		}
	}
	return nil
}

func allCompleted(list []TodoItem) bool {
	for _, it := range list {
		if it.Status != "completed" {
			return false
		}
	}
	return true
}

func anyVerificationMention(list []TodoItem) bool {
	for _, it := range list {
		if verifHint.MatchString(it.Content) {
			return true
		}
	}
	return false
}

// Run implements tools.Tool.
func (t *TodoWrite) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	rc := RunContextFrom(ctx)
	nonInteractive := t.NonInteractive
	if rc != nil {
		nonInteractive = rc.NonInteractive
	}
	if !features.TodoWriteToolEnabled(nonInteractive) {
		return nil, errors.New("todowritetool: TodoWrite is disabled for this session (non-interactive or CLAUDE_CODE_ENABLE_TASKS is set)")
	}

	dec := json.NewDecoder(bytes.NewReader(inputJSON))
	dec.DisallowUnknownFields()
	var in todoWriteInput
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("todowritetool: invalid json: %w", err)
	}
	if in.Todos == nil {
		return nil, errors.New("todowritetool: todos must be a non-null array")
	}
	if err := validateTodos(in.Todos); err != nil {
		return nil, err
	}

	var old []TodoItem
	key := TodoKey(rc)
	if rc != nil && rc.Store != nil {
		old = rc.Store.Get(key)
	}

	allDone := allCompleted(in.Todos)
	storedNew := in.Todos
	if allDone {
		storedNew = []TodoItem{}
	}
	if rc != nil && rc.Store != nil {
		rc.Store.Set(key, storedNew)
	}

	verificationNudgeNeeded := false
	if features.TodoWriteVerificationNudgeEnabled() &&
		(rc == nil || strings.TrimSpace(rc.AgentID) == "") &&
		allDone &&
		len(in.Todos) >= 3 &&
		!anyVerificationMention(in.Todos) {
		verificationNudgeNeeded = true
	}

	oldOut := old
	if oldOut == nil {
		oldOut = []TodoItem{}
	}
	out := map[string]any{
		"oldTodos":                oldOut,
		"newTodos":                in.Todos,
		"verificationNudgeNeeded": verificationNudgeNeeded,
	}
	return json.Marshal(out)
}
