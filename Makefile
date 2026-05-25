## ─────────────────────────────────────────────────────────────────────────────
##  go-tagger  ·  Makefile
##  Author: Hadi Cahyadi <cumulus13@gmail.com>
## ─────────────────────────────────────────────────────────────────────────────

MODULE    := github.com/cumulus13/go-tagger
BINARY    := go-tagger
CMD       := ./cmd/go-tagger

# Version info (overridden by CI via ldflags)
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE      ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS   := -X $(MODULE)/internal/config.Version=$(VERSION) \
             -X $(MODULE)/internal/config.GitCommit=$(COMMIT) \
             -X $(MODULE)/internal/config.BuildDate=$(DATE)

# Build output directories
DIST      := dist
PLATFORMS := \
  linux/amd64 \
  linux/arm64 \
  linux/386 \
  darwin/amd64 \
  darwin/arm64 \
  windows/amd64 \
  windows/arm64

# Colors for pretty output
GREEN  := \033[0;32m
CYAN   := \033[0;36m
YELLOW := \033[0;33m
RESET  := \033[0m

.PHONY: all build test lint fmt vet clean dist help install release

## ── Default ──────────────────────────────────────────────────────────────────

all: fmt vet test build

## ── Build ────────────────────────────────────────────────────────────────────

build: ## Build the binary for the current platform
	@printf "$(CYAN)▶ Building $(BINARY) $(VERSION)…$(RESET)\n"
	@go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)
	@printf "$(GREEN)✔ Built: ./$(BINARY)$(RESET)\n"

build-static: ## Build a fully static binary (Linux only)
	@printf "$(CYAN)▶ Building static $(BINARY) $(VERSION)…$(RESET)\n"
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS) -extldflags=-static" -o $(BINARY) $(CMD)
	@printf "$(GREEN)✔ Built: ./$(BINARY)$(RESET)\n"

install: ## Install to GOPATH/bin
	@printf "$(CYAN)▶ Installing $(BINARY)…$(RESET)\n"
	go install -trimpath -ldflags "$(LDFLAGS)" $(CMD)
	@printf "$(GREEN)✔ Installed$(RESET)\n"

## ── Test ─────────────────────────────────────────────────────────────────────

test: ## Run all unit tests
	@printf "$(CYAN)▶ Running tests…$(RESET)\n"
	go test -race -count=1 ./...
	@printf "$(GREEN)✔ All tests passed$(RESET)\n"

test-verbose: ## Run all tests with verbose output
	go test -race -v -count=1 ./...

test-cover: ## Run tests with coverage report
	@printf "$(CYAN)▶ Running tests with coverage…$(RESET)\n"
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@printf "$(GREEN)✔ Coverage report: coverage.html$(RESET)\n"

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

## ── Code quality ─────────────────────────────────────────────────────────────

fmt: ## Format all Go source files
	@printf "$(CYAN)▶ Formatting…$(RESET)\n"
	@gofmt -l -w .
	@printf "$(GREEN)✔ Formatted$(RESET)\n"

vet: ## Run go vet
	@printf "$(CYAN)▶ Vetting…$(RESET)\n"
	@go vet ./...
	@printf "$(GREEN)✔ Vet OK$(RESET)\n"

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not found — install via https://golangci-lint.run/usage/install/"; exit 1)
	golangci-lint run ./...

tidy: ## Tidy go.mod / go.sum
	go mod tidy

## ── Cross-platform dist ──────────────────────────────────────────────────────

dist: clean ## Build binaries for all platforms into dist/
	@printf "$(CYAN)▶ Building cross-platform binaries…$(RESET)\n"
	@mkdir -p $(DIST)
	@$(foreach PLATFORM,$(PLATFORMS), \
		$(eval OS    := $(word 1,$(subst /, ,$(PLATFORM)))) \
		$(eval ARCH  := $(word 2,$(subst /, ,$(PLATFORM)))) \
		$(eval EXT   := $(if $(filter windows,$(OS)),.exe,)) \
		$(eval OUT   := $(DIST)/$(BINARY)_$(VERSION)_$(OS)_$(ARCH)$(EXT)) \
		printf "  $(YELLOW)→$(RESET) $(OS)/$(ARCH) …\n"; \
		GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 \
			go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT) $(CMD) \
			&& printf "    $(GREEN)✔ $(OUT)$(RESET)\n" \
			|| printf "    $(YELLOW)⚠ FAILED$(RESET)\n"; \
	)
	@printf "\n$(CYAN)▶ Creating checksums…$(RESET)\n"
	@cd $(DIST) && sha256sum * > SHA256SUMS && printf "$(GREEN)✔ dist/SHA256SUMS$(RESET)\n"

## ── Release archive ──────────────────────────────────────────────────────────

release: dist ## Package each binary as a .tar.gz / .zip
	@printf "$(CYAN)▶ Packaging archives…$(RESET)\n"
	@cd $(DIST) && for f in $(BINARY)_*; do \
		case "$$f" in \
			*.exe) zip "$${f%.exe}.zip"   "$$f" && printf "  $(GREEN)✔ $${f%.exe}.zip$(RESET)\n" ;; \
			*.tar.gz|*.zip|SHA256SUMS) ;; \
			*)     tar czf "$$f.tar.gz" "$$f" && printf "  $(GREEN)✔ $$f.tar.gz$(RESET)\n" ;; \
		esac; \
	done
	@cd $(DIST) && sha256sum *.tar.gz *.zip >> SHA256SUMS 2>/dev/null || true
	@printf "$(GREEN)✔ Packaging done$(RESET)\n"

## ── Clean ────────────────────────────────────────────────────────────────────

clean: ## Remove build artefacts
	@printf "$(CYAN)▶ Cleaning…$(RESET)\n"
	@rm -rf $(DIST) $(BINARY) coverage.out coverage.html
	@printf "$(GREEN)✔ Clean$(RESET)\n"

## ── Help ─────────────────────────────────────────────────────────────────────

help: ## Show this help
	@printf "\n$(CYAN)go-tagger$(RESET) — MP3 ID3v2 Tag Editor\n"
	@printf "$(YELLOW)Version:$(RESET) $(VERSION)  $(YELLOW)Commit:$(RESET) $(COMMIT)\n\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@printf "\n"
