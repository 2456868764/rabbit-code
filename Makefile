# rabbit-code — Phase 0 Makefile (see docs/phases/PHASE00_SPEC_AND_ACCEPTANCE.md)
BIN := bin/rabbit-code
MODULE := ./...

.PHONY: build test test-race test-phase4 test-phase5 lint e2e e2e-tui clean assets-mascot

# Write assets/rabbit-code-mascot.png using the same resolution order as CLI (assets/rabbit.png when cwd is module root).
assets-mascot:
	go run ./cmd/mascot-export

build:
	mkdir -p bin
	go build -o $(BIN) -ldflags "-X github.com/2456868764/rabbit-code/internal/version.Version=0.0.0-phase0 -X github.com/2456868764/rabbit-code/internal/version.Commit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" ./cmd/rabbit-code

test:
	go test $(MODULE) -count=1

test-phase4:
	go test ./internal/anthropic/... ./internal/cost/... ./internal/compact/... ./internal/features/... ./internal/app/... ./internal/messages/... -race -count=1

test-phase5:
	go test ./internal/query/... ./internal/querydeps/... ./internal/compact/... ./internal/engine/... ./internal/memdir/... ./internal/messages/... ./internal/features/... ./internal/phase5cli/... -race -count=1

test-race:
	go test $(MODULE) -race -count=1

lint:
	golangci-lint run $(MODULE)

e2e:
	@true

e2e-tui:
	@true

clean:
	rm -rf bin
