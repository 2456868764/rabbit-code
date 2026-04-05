package features

import (
	"testing"
)

func TestWebSearchToolEnabled(t *testing.T) {
	t.Setenv(EnvDisableWebSearch, "")
	t.Setenv(EnvForceWebSearch, "")
	t.Setenv(EnvUseBedrock, "")
	t.Setenv(EnvUseVertex, "")
	t.Setenv(EnvUseFoundry, "")

	if !WebSearchToolEnabled("") {
		t.Fatal("default anthropic should enable")
	}
	if !WebSearchToolEnabled("claude-3-5-haiku-20241022") {
		t.Fatal("anthropic any model")
	}

	t.Setenv(EnvUseBedrock, "1")
	if WebSearchToolEnabled("claude-sonnet-4-20250514") {
		t.Fatal("bedrock off")
	}
	t.Setenv(EnvForceWebSearch, "1")
	if !WebSearchToolEnabled("x") {
		t.Fatal("force on bedrock")
	}
	t.Setenv(EnvForceWebSearch, "")
	t.Setenv(EnvUseBedrock, "")

	t.Setenv(EnvUseVertex, "1")
	if WebSearchToolEnabled("claude-3-5-haiku-20241022") {
		t.Fatal("vertex old model off")
	}
	if !WebSearchToolEnabled("claude-sonnet-4-20250514") {
		t.Fatal("vertex sonnet 4 on")
	}
	t.Setenv(EnvUseVertex, "")

	t.Setenv(EnvUseFoundry, "1")
	if !WebSearchToolEnabled("") {
		t.Fatal("foundry on")
	}
	t.Setenv(EnvUseFoundry, "")

	t.Setenv(EnvDisableWebSearch, "1")
	if WebSearchToolEnabled("claude-sonnet-4-20250514") {
		t.Fatal("disable wins over anthropic")
	}
}
