.PHONY: build run test test-quick test-play e2e clean install lint fmt vet help release

# Build variables
BINARY_NAME=gochi
MAIN_PATH=./cmd/gochiserver
BUILD_DIR=./bin
GO=go
GOFLAGS=-v

# Version info
VERSION?=0.1.0-alpha
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

help: ## Display this help message
	@echo "Gochi - Advanced Digital Pet System"
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the game and the standalone hub
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/gochihub ./cmd/gochihub
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME) and $(BUILD_DIR)/gochihub"

# Platforms for release builds. Phones and tablets play in their browser,
# connected to a computer running the game.
RELEASE_PLATFORMS=windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
DIST_DIR=./dist

release: ## Build the game and hub for Windows, macOS and Linux into dist/
	@rm -rf $(DIST_DIR)
	@for platform in $(RELEASE_PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out=$(DIST_DIR)/gochi-$(VERSION)-$$os-$$arch; \
		echo "Building $$out"; \
		mkdir -p $$out || exit 1; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build $(LDFLAGS) -o $$out/$(BINARY_NAME)$$ext $(MAIN_PATH) || exit 1; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -o $$out/gochihub$$ext ./cmd/gochihub || exit 1; \
		cp -R configs README.md $$out/ || exit 1; \
	done
	@echo "Release builds are in $(DIST_DIR)/"

run: build ## Build and run the application
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

test: ## Run all tests
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "Tests complete"

test-quick: ## Run the tests, without the long ones
	$(GO) test -short ./...

test-play: ## Play the game at random for a long while (PLAY_SEEDS seeds; PLAY_SEED replays one)
	GOCHI_PLAY_SEEDS=$(or $(PLAY_SEEDS),100) GOCHI_PLAY_SEED=$(PLAY_SEED) $(GO) test ./cmd/gochiserver -run 'TestRandomPlay$$' -timeout 2h -v

e2e: ## Play the page in a headless browser (needs node and Playwright; NODE_PATH finds Playwright)
	GOCHI_E2E=1 $(GO) test ./cmd/gochiserver -run TestBrowserSmoke -timeout 10m -v

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt coverage.html
	@$(GO) clean
	@echo "Clean complete"

install: ## Install dependencies
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "Dependencies updated"

dev: ## Run in development mode with live reload (requires air)
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t gochi:$(VERSION) .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -it --rm gochi:$(VERSION)

all: clean install fmt vet lint test build ## Run all checks and build

.DEFAULT_GOAL := help
