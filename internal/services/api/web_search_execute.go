package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools/websearchtool"
)

// WebSearchCallParams configures ExecuteWebSearchToolCall (WebSearchTool.call analogue).
type WebSearchCallParams struct {
	Policy         Policy
	MainLoopModel  string
	SmallFastModel string
	MaxTokens      int
	// OnWebSearchProgress optional stream progress (query_update, search_results_received).
	OnWebSearchProgress func(websearchtool.WebSearchProgress)
}

// ExecuteWebSearchToolCall mirrors WebSearchTool.call: inner Messages stream with web_search_20250305 tool,
// then makeOutputFromSearchResponse via websearchtool.MakeOutputFromContentBlocks.
func ExecuteWebSearchToolCall(ctx context.Context, c *Client, in websearchtool.Input, p WebSearchCallParams) ([]any, error) {
	if c == nil {
		return nil, fmt.Errorf("anthropic: nil client")
	}
	start := time.Now()
	pol := p.Policy
	if pol.MaxAttempts == 0 {
		pol = DefaultPolicy()
	}
	// Tag inner call like upstream querySource: 'web_search_tool' so StrictForeground529 does not treat it as repl_main_thread (529 retry policy).
	pol.QuerySource = QuerySourceWebSearchTool

	model := strings.TrimSpace(p.MainLoopModel)
	if model == "" {
		model = "claude-3-5-haiku-20241022"
	}
	maxTok := p.MaxTokens
	if maxTok <= 0 {
		maxTok = 1024
	}

	usePlum := features.WebSearchPlumVx3()
	small := strings.TrimSpace(p.SmallFastModel)
	if small == "" {
		small = smallFastModelFromEnv()
	}

	var toolChoice, thinking json.RawMessage
	var temperature *float64
	if usePlum {
		model = small
		toolChoice = json.RawMessage(`{"type":"tool","name":"web_search"}`)
		thinking = json.RawMessage(`{"type":"disabled"}`)
		t := 1.0
		temperature = &t
	}

	userLine := websearchtool.InnerSearchUserContent(in.Query)
	msgs, err := json.Marshal([]map[string]any{{
		"role": "user",
		"content": []any{map[string]any{
			"type": "text",
			"text": userLine,
		}},
	}})
	if err != nil {
		return nil, err
	}
	sys, err := json.Marshal(websearchtool.InnerSearchSystemPrompt)
	if err != nil {
		return nil, err
	}
	schema := websearchtool.WebSearchToolSchemaFromInput(in)
	tools, err := json.Marshal([]any{schema})
	if err != nil {
		return nil, err
	}

	body := MessagesStreamBody{
		Model:       model,
		MaxTokens:   maxTok,
		Messages:    msgs,
		System:      sys,
		Tools:       tools,
		ToolChoice:  toolChoice,
		Thinking:    thinking,
		Temperature: temperature,
	}
	body.AnthropicBeta = AppendBetaUnique(body.AnthropicBeta, BetaWebSearch)

	resp, err := c.PostMessagesStream(ctx, body, pol)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	blocks, u, err := ReadWebSearchAssistantBlocks(ctx, resp.Body,
		WebSearchReadFallbackQuery(in.Query),
		WebSearchReadOnProgress(p.OnWebSearchProgress),
	)
	if err != nil {
		return nil, err
	}
	if c.OnStreamUsage != nil {
		c.OnStreamUsage(u)
	}

	results, err := websearchtool.MakeOutputFromContentBlocks(blocks, in.Query, time.Since(start).Seconds())
	if err != nil {
		return nil, err
	}
	return results, nil
}

func smallFastModelFromEnv() string {
	if s := strings.TrimSpace(os.Getenv("RABBIT_CODE_SMALL_FAST_MODEL")); s != "" {
		return s
	}
	return "claude-3-5-haiku-20241022"
}
