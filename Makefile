TEMP_DIR := ./.tmp

# Tool versions #################################
GOLICENSES_VERSION := v5.0.1

# Formatting variables #################################
BOLD := $(shell tput -T linux bold)
PURPLE := $(shell tput -T linux setaf 5)
GREEN := $(shell tput -T linux setaf 2)
CYAN := $(shell tput -T linux setaf 6)
RED := $(shell tput -T linux setaf 1)
RESET := $(shell tput -T linux sgr0)
TITLE := $(BOLD)$(PURPLE)
SUCCESS := $(BOLD)$(GREEN)

define title
	@printf "$(TITLE)%s$(RESET)\n" "$(1)"
endef

# Test variables #################################
COVERAGE_THRESHOLD := 28  # the quality gate lower threshold for unit test total % coverage (by function statements)


## Bootstrapping targets #################################

.PHONY: bootstrap
bootstrap: $(TEMP_DIR) bootstrap-go bootstrap-tools ## Download and install all tooling dependencies (+ prep tooling in the ./tmp dir)
	$(call title,Bootstrapping dependencies)

.PHONY: bootstrap-go
bootstrap-go:
	go mod download

$(TEMP_DIR):
	mkdir -p $@

.PHONY: bootstrap-tools
bootstrap-tools: $(TEMP_DIR)
	GOBIN=$(realpath $(TEMP_DIR)) go install golang.org/x/perf/cmd/benchstat@latest
	curl -sSfL https://raw.githubusercontent.com/khulnasoft/go-licenses/master/golicenses.sh | sh -s -- -b $(TEMP_DIR)/ $(GOLICENSES_VERSION)

BINARY := kotha
BIN_DIR := bin
MAIN := ./main.go
MODULE := github.com/kothagpt/kotha
VERSION_PKG := $(MODULE)/internal/version
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(VERSION_PKG).Version=$(VERSION)

GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: help build bin lint vet fmt fmt-check test test-race unit cover bench bootstrap bootstrap-go bootstrap-tools check-licenses check-go-mod-tidy tidy clean install release-snapshot

.PHONY: help
help:  ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(BOLD)$(CYAN)%-25s$(RESET)%s\n", $$1, $$2}'

build: ## Build binary to ./bin/kotha
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(MAIN)

bin: build ## Alias for build

lint: vet fmt-check ## Run vet, fmt check, golangci-lint and staticcheck
	$(GOLANGCI_LINT) run ./... 2>/dev/null || $(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format code
	$(GO) fmt ./...

fmt-check: ## Check formatting
	@test -z "$$($(GO) fmt -l ./...)" || (echo "Unformatted files:"; $(GO) fmt -l ./...; exit 1)

test: ## Run tests
	$(GO) test ./...

## Testing targets #################################

.PHONY: unit
unit: $(TEMP_DIR)  ## Run unit tests (with coverage)
	$(call title,Running unit tests)
	go test -coverprofile $(TEMP_DIR)/unit-coverage-details.txt ./...
	@.github/scripts/coverage.py $(COVERAGE_THRESHOLD) $(TEMP_DIR)/unit-coverage-details.txt

test-race: ## Run tests with race detector
	$(GO) test -race ./...

cover: ## Run tests with coverage
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

bench: $(TEMP_DIR) ## Run benchmarks (health, ratelimit, event system) and compare with benchstat
	$(GO) test -bench=. -benchtime=1s -count=5 -run=^$$ ./internal/health/... ./internal/ratelimit/... ./internal/pubsub/... | tee $(TEMP_DIR)/bench.txt
	$(TEMP_DIR)/benchstat $(TEMP_DIR)/bench.txt

.PHONY: check-licenses
check-licenses:  ## Ensure transitive dependencies are compliant with the current license policy
	$(call title,Checking for license compliance)
	$(TEMP_DIR)/golicenses check ./...

.PHONY: check-go-mod-tidy
check-go-mod-tidy:
	@ .github/scripts/go-mod-tidy-check.sh && echo "go.mod and go.sum are tidy!"

tidy: ## Tidy go modules
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) coverage.out dist/ $(TEMP_DIR)

install: ## Install binary to GOPATH/bin
	CGO_ENABLED=0 $(GO) install -ldflags "$(LDFLAGS)" ./...

release-snapshot: ## Goreleaser snapshot build
	goreleaser release --snapshot --clean

## SDK regen targets #################################

.PHONY: regen-sdk
regen-sdk: ## Validate openapi.yaml against the Go SDK surface
	$(GO) run ./scripts/regen-sdk

.PHONY: regen-sdk-check
regen-sdk-check: regen-sdk ## Alias used by CI
