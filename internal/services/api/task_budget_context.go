package anthropic

import "context"

type perTurnTaskBudgetKey struct{}

// WithPerTurnTaskBudget attaches output_config.task_budget for Messages API calls on this context
// (query.ts QueryParams.taskBudget / QueryEngine.ts QueryEngineConfig.taskBudget).
func WithPerTurnTaskBudget(ctx context.Context, total int) context.Context {
	if total <= 0 {
		return ctx
	}
	return context.WithValue(ctx, perTurnTaskBudgetKey{}, total)
}

func perTurnTaskBudgetTotal(ctx context.Context) (int, bool) {
	v := ctx.Value(perTurnTaskBudgetKey{})
	if v == nil {
		return 0, false
	}
	n, ok := v.(int)
	if !ok || n <= 0 {
		return 0, false
	}
	return n, true
}

// PerTurnTaskBudgetFromContext returns the total from WithPerTurnTaskBudget, if any (tests / loop wiring).
func PerTurnTaskBudgetFromContext(ctx context.Context) (total int, ok bool) {
	return perTurnTaskBudgetTotal(ctx)
}
