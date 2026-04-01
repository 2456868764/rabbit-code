# rabbit-code — Phase 0 Makefile (see docs/phases/PHASE00_SPEC_AND_ACCEPTANCE.md)
BIN := bin/rabbit-code
MODULE := ./...

.PHONY: build test test-race lint e2e e2e-tui clean

build:
	mkdir -p bin
	go build -o $(BIN) -ldflags "-X github.com/2456868764/rabbit-code/internal/version.Version=0.0.0-phase0 -X github.com/2456868764/rabbit-code/internal/version.Commit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" ./cmd/rabbit-code

test:
	go test $(MODULE) -count=1

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
