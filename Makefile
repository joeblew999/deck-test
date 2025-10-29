.PHONY: all build ensure examples run view clean test help

# Variables
GO_RUN := go run .
NO_SYNC := --no-sync

# Default target
all: build

# Build all binaries (native, wasm, wasi)
build:
	$(GO_RUN) dev-build $(NO_SYNC)

# Build and create GitHub release
release:
	$(GO_RUN) dev-release

# Create GitHub release (skip build if already built)
release-fast:
	$(GO_RUN) dev-release --skip-build

# Ensure binaries and repositories are up to date
ensure:
	$(GO_RUN) ensure $(NO_SYNC)

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
	$(GO_RUN) run $(EXAMPLE) $(NO_SYNC)

# View a specific example (requires EXAMPLE variable)
# Usage: make view EXAMPLE=deckviz/aapl
view:
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "Error: EXAMPLE variable is required"; \
		echo "Usage: make view EXAMPLE=deckviz/aapl"; \
		exit 1; \
	fi
	$(GO_RUN) view $(EXAMPLE) $(NO_SYNC)

# Clean build artifacts
clean:
	rm -rf dist/ .dist/ bin/ .src/ .data/

# Test all commands in the correct order (from CLAUDE.md)
test: build ensure examples
	@echo "✓ All core commands tested successfully"
	@echo ""
	@echo "To test run/view commands:"
	@echo "  make run EXAMPLE=deckviz/aapl"
	@echo "  make view EXAMPLE=deckviz/aapl"

