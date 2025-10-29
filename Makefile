.PHONY: all build ensure examples run view clean dev-clean test help

# Variables
GO_RUN := go run .

TEST_EXAMPLE := deckviz/fire
TEST_EXAMPLE_PATH := .data/deckviz/fire

# Default target
all: dev-build


# Build all binaries (native, wasm, wasi)
dev-build:
	$(GO_RUN) dev-build

# Build and create GitHub release
dev-release:
	$(GO_RUN) dev-release

# Create GitHub release (skip build if already built)
dev-release-fast:
	$(GO_RUN) dev-release --skip-build

# Clean all dot folders (data, src, dist, fonts) for fresh start
# WARNING: This removes ALL repos and takes a long time to re-clone
dev-clean:
	@echo "WARNING: This will remove .data, .src, .dist, and .fonts folders"
	@echo "It takes a long time to re-clone all repositories!"
	@read -p "Are you sure? (yes/no): " answer && [ "$$answer" = "yes" ]
	$(GO_RUN) dev-clean



# Ensure binaries and repositories are up to date
ensure:
	$(GO_RUN) ensure

# List all available examples
examples:
	$(GO_RUN) examples

# Run a specific example (requires EXAMPLE variable)
# Usage: make run EXAMPLE=deckviz/aapl
run:
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "Error: EXAMPLE variable is required"; \
		echo "Usage: make run EXAMPLE=deckviz/aapl"; \
		exit 1; \
	fi
	$(GO_RUN) run $(EXAMPLE)
run-test:
	$(GO_RUN) run $(TEST_EXAMPLE)
run-test-path:
	$(GO_RUN) run $(TEST_EXAMPLE_PATH)

# View a specific example (requires EXAMPLE variable)
# Usage: make view EXAMPLE=deckviz/aapl
view:
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "Error: EXAMPLE variable is required"; \
		echo "Usage: make view EXAMPLE=deckviz/aapl"; \
		exit 1; \
	fi
	$(GO_RUN) view $(EXAMPLE)
view-test:
	$(GO_RUN) view $(TEST_EXAMPLE)
view-test-path:
	$(GO_RUN) view $(TEST_EXAMPLE_PATH)





# Test all commands in the correct order (from CLAUDE.md)
test: build ensure examples
	@echo "✓ All core commands tested successfully"
	@echo ""
	@echo "To test run/view commands:"
	@echo "  make run EXAMPLE=deckviz/aapl"
	@echo "  make view EXAMPLE=deckviz/aapl"

