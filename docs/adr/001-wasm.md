# ADR 001: Compile Deck Binaries to WASM/WASI

## Status

Proposed

## Context

The current deck tooling (`decksh`, `pdfdeck`, `pngdeck`, `svgdeck`, `ebdeck`, `dshfmt`, `dshlint`, etc.) is distributed as native Go binaries installed via `go install`. To broaden the environments in which these tools can run, we want to evaluate compiling each binary to WebAssembly (WASM) for in-browser usage and to WASI (WebAssembly System Interface) for headless/sandboxed command-line execution.

Primary drivers:

- Allow deck tooling to run inside browsers (WASM) for interactive demos.
- Support sandboxed environments (CI, serverless, locked-down workstations) via WASI.
- Reduce installation friction by shipping portable artifacts instead of native binaries.

Constraints:

- Some binaries depend on graphical libraries (e.g., `ebdeck` uses Ebiten) that may not yet support WASM/WASI.
- Deckviz examples read files from the local filesystem; we need a strategy for packaging data or providing a virtual filesystem.
- Existing native workflow must remain supported for power users.

## Decision

Compile the deck tooling into portable artifacts with clear responsibilities:

- CLI-oriented binaries (`decksh`, `dshfmt`, `dshlint`) target **WASI** so they can run headless in sandboxed runtimes.
- Rendering/output binaries (`pdfdeck`, `pngdeck`, `svgdeck`) target **WASM** first (and WASI where feasible) by emitting results to in-memory buffers or virtual file systems instead of the host filesystem.
- Graphical viewers (`ebdeck`, `gcdeck`) target **WASM** to paint onto HTML canvas elements; native builds continue for desktop usage.

Implementation will proceed in stages:

1. **WASI-first for CLI tools**: build `decksh`, `dshfmt`, and `dshlint` for WASI where feasible; wrap file IO using WASI preview APIs.
2. **WASM for outputs and viewers**: prototype renderers (`pdfdeck`, `pngdeck`, `svgdeck`) as WASM modules that emit downloadable artifacts (memory buffers streamed to the browser) and graphical viewers (`ebdeck`, `gcdeck`) that paint to HTML canvas.
3. **Packaging**: expose `decktool wasm` and `decktool wasi` commands that produce ready-to-run bundles (WASM/WASI binary plus HTML or stub runner scripts).
4. **Distribution**: host the artifacts via GitHub Releases or a CDN for easy consumption.

Native binaries remain available; WASM/WASI builds supplement them.

## Consequences

Positive:

- Deck tooling becomes portable across browsers and sandboxed runtimes.
- Graphical viewers (`ebdeck`, `gcdeck`) can run in-browser via WASM, enabling interactive previews without native installs.
- Output generators (`pdfdeck`, `pngdeck`, `svgdeck`) no longer require native binaries when run in WASM/WASI; they stream results to the caller for download or further processing.
- CI pipelines/serve → easier to run deck conversions without local Go toolchain.

Negative / Open Questions:

- Ebiten (used by `ebdeck`/`gcdeck`) may have limitations in WASM; requires verification.
- WASM bundles could be large if fonts/assets are embedded.
- Need an approach to provide deckviz data/files in browser and WASI environments (e.g., embed data, fetch over HTTP, or mount WASI virtual FS).
- Viewer UIs must adapt to browser event models and canvas rendering.
- WASI support for file creation must be validated before promising WASI builds for renderers.

Next steps:

1. Audit each deck binary for WASM/WASI compatibility (Go `GOOS=js GOARCH=wasm` / `GOOS=wasip1 GOARCH=wasm`), documenting required build flags, external dependencies, and current blockers in a compatibility matrix.
2. Build proof-of-concept WASI versions of the CLI tools (`decksh`, `dshfmt`, `dshlint`) that exercise the compatibility matrix, wiring file access through WASI preview APIs and validating non-regression with the existing test suite.
3. Prototype WASM renderers (`pdfdeck`, `pngdeck`, `svgdeck`) that stream outputs instead of writing to disk, and WASM viewers (`ebdeck`, `gcdeck`) that paint to canvas.
4. Evaluate WASI feasibility for renderers once streaming/file APIs are proven.
5. Keep `decktool` as a native build orchestrator that produces WASM/WASI bundles.
6. Design packaging/distribution plan for releasing WASM/WASI artifacts.
