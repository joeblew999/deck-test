# deck-test

Test harness for deck examples with WASM/WASI support.

**Releases:** https://github.com/joeblew999/deck-test/releases

## Quick Start

```bash
# Get binaries
go run . ensure

# List examples
decktool examples

# Run an example
decktool run deckviz/fire
```

## Build & Release

```bash
# Build all binaries (native, WASM, WASI)
decktool dev-build

# Create GitHub release
decktool dev-release
```

## What It Does

- Downloads deck example data (deckviz, dubois, deckfonts)
- Clones & builds deck Go repos (decksh, pdfdeck, gift, etc.)
- Compiles to native, WASM, and WASI targets
- Runs examples and opens results

## Commands

- `ensure` - Get binaries from GitHub release or build locally
- `examples` - List all available examples
- `run [example]` - Lint and render an example
- `view [example]` - Open example in ebdeck
- `dev-build` - Build binaries for native/WASM/WASI
- `dev-release` - Create GitHub release with all binaries
