# deck-test

Test harness for deck examples with WASM/WASI support.

**Releases:** https://github.com/joeblew999/deck-test/releases


## What It Does

- Downloads deck example data (deckviz, dubois, deckfonts)
- Clones & builds deck Go repos (decksh, pdfdeck, gift, etc.)
- Compiles to native, WASM, and WASI targets
- Runs examples and opens results

## Quick Start

```bash
# Get binaries and data ( that has exmales)
go run . ensure

# List examples
go run . examples

# Run an example
go run . run deckviz/fire
# View an example 
go run . view deckviz/fire
```

## Build & Release

```bash
# Build all binaries (native, WASM, WASI)
go run . dev-build

# Create GitHub release ( that ensure can use later to bring them back down)
go run . dev-release
```


