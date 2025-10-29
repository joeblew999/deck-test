# ADR 001: Compile Deck Binaries to WASM/WASI

## Status

Proposed

## Context

The current deck tooling (`decksh`, `pdfdeck`, `pngdeck`, `svgdeck`, `ebdeck`, `dshfmt`, `dshlint`, etc.) is distributed as native Go binaries installed via `go install`. We want the decktool CLI to produce WebAssembly (WASM) and WASI artifacts for every deck binary so they can be consumed outside of native environments. Whether individual binaries succeed or fail to compile will be discovered during implementation—this ADR establishes the tooling changes needed to attempt both targets for all binaries.

Primary drivers:

- Allow deck tooling to run inside browsers (WASM) for interactive demos.
- Support sandboxed environments (CI, serverless, locked-down workstations) via WASI.
- Reduce installation friction by shipping portable artifacts instead of native binaries.

Constraints:

- Decktool must clone/update the required repositories before building—`go install` alone is insufficient once we produce custom WASM/WASI bundles.
- Deckviz examples read files from the local filesystem; we need a strategy for packaging data or providing a virtual filesystem when running under WASM/WASI.
- Existing native workflow must remain supported for power users.

## Decision

Teach the native `decktool` CLI to orchestrate builds that output both WASM and WASI artifacts for *every* deck binary. The CLI will:

- Clone/update all required deck repositories (deck, decksh, ebcanvas, etc.) into a workspace so the sources are available.
- Run `go build` for each binary with `GOOS=js GOARCH=wasm` to produce WASM modules.
- Run `go build` for each binary with `GOOS=wasip1 GOARCH=wasm` to produce WASI modules (still recorded even if compilation fails; failures are surfaced to the user).
- Package the resulting artifacts in a deterministic output directory so downstream consumers (and future commands) can fetch them.

This replaces the previous expectation that end users run `go install` directly; `decktool` becomes the single entry point for producing portable builds while native binaries remain available.

Implementation will proceed in stages:

1. Extend `decktool` with build commands (e.g., `decktool build wasm`, `decktool build wasi`) that iterate over the deck binary list, clone necessary repos, and run the appropriate `go build` invocations.
2. Capture per-binary results (success/failure, artifact paths) so users can see which builds succeeded without manually retrying.
3. Package outputs into a structured directory (or archive) ready for publishing; future automation can upload these artifacts.
4. Keep native builds intact; this ADR does not deprecate the existing workflow.

Native binaries remain available; WASM/WASI builds supplement them.

## Consequences

Positive:

- One command (`decktool build wasm|wasi`) attempts builds for every deck binary, simplifying distribution of portable artifacts.
- Build outputs are reproducible because decktool manages repository checkout and build flags.
- Portable artifacts remove the dependency on `go install` for consumers that need WASM/WASI builds.

Negative / Open Questions:

- Some binaries may fail to compile for WASM or WASI; decktool must report failures and continue building the rest.
- WASM bundles could be large if fonts/assets are embedded.
- Need an approach to provide deckviz data/files in browser and WASI environments (e.g., embed data, fetch over HTTP, or mount WASI virtual FS).
- Viewer UIs must adapt to browser event models and canvas rendering.
- WASI support for file creation must be validated before promising WASI builds for renderers.

Next steps:

1. Inventory the deck binaries and define a compatibility matrix capturing required repos, build commands, and expected artifact names.
2. Implement the new decktool build commands that clone repos, invoke `go build` for WASM/WASI, and store artifacts/results.
3. Ensure build logs clearly report success/failure per binary so future ADRs can address any gaps.
4. Investigate strategies for handling data/assets in WASM/WASI (virtual FS, HTTP fetching, embedding) and document follow-up work.
5. Design packaging/distribution plan for releasing WASM/WASI artifacts produced by decktool.
