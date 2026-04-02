package anthropic

// TaskBudgetParam is output_config.task_budget (services/api/claude.ts TaskBudgetParam; beta task-budgets-2026-03-13).
type TaskBudgetParam struct {
	Type      string `json:"type"` // "tokens"
	Total     int    `json:"total"`
	Remaining *int   `json:"remaining,omitempty"`
}

// OutputConfig maps to Messages API output_config (subset).
type OutputConfig struct {
	TaskBudget *TaskBudgetParam `json:"task_budget,omitempty"`
}
