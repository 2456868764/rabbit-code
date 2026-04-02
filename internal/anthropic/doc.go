// Package anthropic implements the Anthropic Messages HTTP client surface aligned with
// claude-code-sourcemap/restored-src/src/services/api (streaming, retry, errors, auth, betas, preconnect).
//
// Backpressure (P4.1.2 / AC4-1c): StreamEvents uses a bounded channel; the SSE reader goroutine
// blocks on send when the buffer is full, so a slow consumer does not grow unbounded memory.
//
// Bedrock: betas in BEDROCK_EXTRA_PARAMS_HEADERS are JSON-encoded as "anthropic_beta" on the request
// body when using Client.SetBetaNames (HTTP anthropic-beta gets the remainder).
//
// Proxy (P4.3.3): HTTPTransportWithProxyFromEnv (and HTTPTransportWithProxyFromEnvAndRoots for Bootstrap TLS pool)
// as the base RoundTripper for NewTransportChain / NewClient.
// mTLS: HTTPTransportAPIOutbound / HTTPTransportAPIOutboundWithRoots load CLAUDE_CODE_CLIENT_CERT + CLAUDE_CODE_CLIENT_KEY when both are set.
// Cloud: SigningTransport + CloudRequestSigner hook for Bedrock/Vertex signing (StubBedrockSigner / StubVertexSigner until AC4-6 real signing).
// Vertex: ANTHROPIC_VERTEX_PROJECT_ID + CLOUD_ML_REGION (or ANTHROPIC_VERTEX_BASE_URL) enable streamRawPredict path and vertex JSON body (see vertex-sdk).
// Foundry: ANTHROPIC_FOUNDRY_RESOURCE builds https://{res}.services.ai.azure.com/anthropic (foundry-sdk).
// ReadAssistantStream: optional WithThinkingAccumulator / WithCompactionAccumulator / WithToolInputAccumulators (Client fields).
// Client.SetOnStreamUsageBootstrap wires OnStreamUsage to cost.ApplyUsageToBootstrap (P4.4.1).
// Retry: Policy.StrictForeground529 mirrors withRetry.ts FOREGROUND_529_RETRY_SOURCES; RABBIT_CODE_STRICT_FOREGROUND_529 enables it in DefaultPolicy.
// Policy.InitialConsecutive529Errors mirrors withRetry.ts initialConsecutive529Errors (529 budget pre-seed).
// Unattended: Policy.Unattended (RABBIT_CODE_UNATTENDED_RETRY) after MaxAttempts on HTTP 429/529 enters withRetry.ts persistent backoff + HEARTBEAT_INTERVAL_MS chunked waits.
// Attribution: AttributionSystemPromptLine matches system.ts getAttributionHeader (cc_version / cc_entrypoint / optional cch=00000 / cc_workload); inject into system prompt when wiring messages (Phase 5).
// Prompt cache break: ReadAssistantStream WithOnPromptCacheBreak telemetry hook (AC4-F3).
// Anti-distillation: RABBIT_CODE_ANTI_DISTILLATION_CC + RABBIT_CODE_ANTI_DISTILLATION_FAKE_TOOLS adds JSON anti_distillation: ["fake_tools"] (getExtraBodyParams); optional header envs remain.
// Stale connection: RABBIT_CODE_DISABLE_KEEPALIVE_ON_ECONNRESET wraps a dedicated *http.Transport (not http.DefaultTransport) to set DisableKeepAlives after ECONNRESET/EPIPE.
package anthropic
