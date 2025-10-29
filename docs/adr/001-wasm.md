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

Compile each deck tool to both WASM and WASI targets, producing portable binaries alongside the existing native builds. The implementation will proceed in stages:

1. **WASI-first for CLI tools**: build `decksh`, `dshfmt`, and `dshlint` for WASI where feasible; wrap file IO using WASI preview APIs.
2. **WASM for outputs and viewers**: prototype renderers (`pdfdeck`, `pngdeck`, `svgdeck`) as WASM modules that emit downloadable artifacts, and graphical viewers (`ebdeck`, `gcdeck`) as WASM modules that paint to HTML canvas.
3. **Packaging**: expose `decktool wasm` and `decktool wasi` commands that produce ready-to-run bundles (WASM/WASI binary plus HTML or stub runner scripts).
4. **Distribution**: host the artifacts via GitHub Releases or a CDN for easy consumption.

Native binaries remain available; WASM/WASI builds supplement them.

## Consequences

Positive:

- Deck tooling becomes portable across browsers and sandboxed runtimes.
- Graphical viewers (`ebdeck`, `gcdeck`) can run in-browser via WASM, enabling interactive previews without native installs.
- Output generators (`pdfdeck`, `pngdeck`, `svgdeck`) produce artifacts directly in WASM/WASI without native binaries.
- CI pipelines/serve → easier to run deck conversions without local Go toolchain.

Negative / Open Questions:

- Ebiten (used by `ebdeck`/`gcdeck`) may have limitations in WASM; requires verification.
- WASM bundles could be large if fonts/assets are embedded.
- Need an approach to provide deckviz data/files in browser and WASI environments.
- Viewer UIs must adapt to browser event models and canvas rendering.

Next steps:

1. Audit each deck binary for WASM/WASI compatibility (Go `GOOS=js GOARCH=wasm` / `GOOS=wasip1 GOARCH=wasm`).
2. Build proof-of-concept WASI versions of the CLI tools (`decksh`, `dshfmt`, `dshlint`).
3. Prototype WASM renderers (`pdfdeck`, `pngdeck`, `svgdeck`) that output files, and WASM viewers (`ebdeck`, `gcdeck`) that paint to canvas.
4. Keep `decktool` as a native build orchestrator that produces WASM/WASI bundles.
5. Design packaging/distribution plan for releasing WASM/WASI artifacts.
