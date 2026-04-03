// Package anthropic implements the Anthropic Messages HTTP client surface aligned with
// claude-code-sourcemap/restored-src/src/services/api (streaming, retry, errors, auth, betas, preconnect).
// Go import path: github.com/2456868764/rabbit-code/internal/services/api (directory mirrors src/services/api).
//
// Phase 4 scope (this module + app wiring): outbound, signing (Bedrock/Vertex), preconnect, Client
// factories, usage hook, retry/stream behavior, services/api probe shapes (AC4-7 best-effort paths),
// and env flags in internal/features/rabbit_env.go. Foundry
// outbound Azure AD signing is excluded. Main-process Messages loop is Phase 5.
// Parallel Keychain/OAuth prefetch in Bootstrap is an optional Phase 4 tail if product AC requires
// full main.tsx-style warmup (see package features doc and app.ParallelPrefetch).
//
// Backpressure (P4.1.2 / AC4-1c): StreamEvents uses a bounded channel; the SSE reader goroutine
// blocks on send when the buffer is full, so a slow consumer does not grow unbounded memory.
//
// Bedrock: betas in BEDROCK_EXTRA_PARAMS_HEADERS are JSON-encoded as "anthropic_beta" on the request
// body when using Client.SetBetaNames (HTTP anthropic-beta gets the remainder).
//
// App wiring: app.NewAnthropicClient uses NewClientWithPool + ReadAPIKey + SetOnStreamUsageBootstrap (same outbound as Bootstrap preconnect).
// CLI: rabbit-code probe [services/api ts name] uses app.RunProbe (P4.6.1); main repeats RunAPIPreconnect after merged config when trust is on.
// Proxy (P4.3.3): HTTPTransportWithProxyFromEnv (and HTTPTransportWithProxyFromEnvAndRoots for Bootstrap TLS pool)
// as the base RoundTripper for NewTransportChain / NewClient.
// mTLS: HTTPTransportAPIOutbound / HTTPTransportAPIOutboundWithRoots load RABBIT_CODE_CLIENT_CERT + RABBIT_CODE_CLIENT_KEY when both are set.
// Preconnect skip: RABBIT_CODE_UNIX_SOCKET or RABBIT_CODE_CLIENT_CERT / RABBIT_CODE_CLIENT_KEY (ShouldSkipPreconnect).
// Cloud: SigningTransport + CloudRequestSigner — BedrockSigV4Signer (bedrock-runtime SigV4), VertexTokenSigner (GCP ADC Bearer), StubFoundrySigner; NewSigningTransportForProvider / NewAPIOutboundTransport; ResolveAPIOutboundTransport falls back to proxy+roots on error (Bootstrap + NewClientWithPool / NewClientWithPoolOAuth). RABBIT_CODE_SKIP_BEDROCK_AUTH / RABBIT_CODE_SKIP_VERTEX_AUTH no-op signers for mocks.
// Vertex: ANTHROPIC_VERTEX_PROJECT_ID + CLOUD_ML_REGION (or ANTHROPIC_VERTEX_BASE_URL) enable streamRawPredict path and vertex JSON body (see vertex-sdk).
// Foundry: ANTHROPIC_FOUNDRY_RESOURCE builds https://{res}.services.ai.azure.com/anthropic (foundry-sdk). Azure AD signing for outbound requests is not in Phase 4 (RABBIT_CODE_SKIP_FOUNDRY_AUTH reserved for future mocks).
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
