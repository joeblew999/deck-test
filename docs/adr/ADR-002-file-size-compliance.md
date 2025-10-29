# ADR-002: File Size Compliance Strategy

**Status**: Proposed
**Date**: 2025-10-29
**Context**: After ADR-001 config.go split, three files still violate CLAUDE.md's 200-line limit

## Context

CLAUDE.md mandates:
> **Maximum 200 lines per file** - Files must be small enough to read completely

### Current Violations

| File | Lines | Over Limit | Functions |
|------|-------|------------|-----------|
| build.go | 267 | +67 | 9 functions |
| commands_dev.go | 244 | +44 | 3 functions |
| commands.go | 230 | +30 | 7 functions |

### Compliant Files
- config.go: 169 lines ✓
- repos.go: 164 lines ✓
- examples.go: 186 lines ✓
- setup.go: 172 lines ✓
- util.go: 116 lines ✓
- paths.go: 72 lines ✓
- workspace.go: 42 lines ✓
- main.go: 20 lines ✓

## Analysis

### 1. build.go (267 lines, 9 functions)

**Responsibilities**:
- Building binaries (buildBinary, buildFilename, buildAll)
- Binary resolution (getBinaryPath, resolveBinary, ensureBins)
- GitHub release downloads (ensureGhCli, downloadReleaseBinaries)

**Split Strategy**:
```
build.go (~115 lines) - Building operations
  - buildBinary() - Compile single binary
  - buildFilename() - Generate filename with target suffix
  - buildAll() - Build all binaries for all targets
  - getRequirement() - Format requirement string for errors
  - getBinaryPath() - Get path to binary in dist/
  - ensureBins() - Orchestrate ensuring binaries exist

binaries.go (~152 lines) - Binary acquisition
  - resolveBinary() - Find binary in dist/PATH/GOBIN (~26 lines)
  - ensureGhCli() - Auto-install gh CLI (~23 lines)
  - downloadReleaseBinaries() - Download from GitHub releases (~103 lines)
```

**Rationale**:
- Build operations (compile) vs binary acquisition (download/resolve) are distinct concerns
- ensureBins() orchestrates both, stays in build.go as the entry point
- resolveBinary() finds existing binaries (PATH/GOBIN/dist)
- downloadReleaseBinaries() gets binaries from GitHub releases
- Both resulting files under 200 lines ✓

### 2. commands_dev.go (244 lines, 3 functions)

**Current Structure**:
- newDevBuildCommand() - 68 lines
- newDevReleaseCommand() - 132 lines (LARGE!)
- newDevCleanCommand() - 30 lines

**Split Strategy**:
```
commands_dev.go (~112 lines) - Dev build/clean commands
  - newDevBuildCommand() - 68 lines
  - newDevCleanCommand() - 30 lines
  - ~14 lines imports/package

commands_release.go (~132 lines) - Release command
  - newDevReleaseCommand() - 132 lines
```

**Rationale**:
- newDevReleaseCommand() is 132 lines - more than half the file
- Release process (GitHub releases, asset uploads) is distinct from local builds
- Clear separation: local dev operations vs release publishing

**Alternative (RECOMMENDED)**: Extract release logic into release.go methods
- Could create cfg.createRelease() method in new release.go
- Would make commands_release.go much smaller (~30-40 lines)
- Follows pattern of other commands (thin wrapper calling cfg methods)
- Would need to split 132-line function into logical methods

### 3. commands.go (230 lines, 7 functions)

**Current Structure**:
- newRootCommand() - 27 lines
- newEnsureCommand() - 18 lines
- newExamplesCommand() - 21 lines
- newRunCommand() - 30 lines
- newViewCommand() - 43 lines
- newCompletionCommand() - 23 lines
- newSetupCommand() - 56 lines (LARGEST)

**Split Strategy Option A (RECOMMENDED)**: Extract setup orchestration to setup.go
```
1. Create cfg.setupTool() method in setup.go
2. Consolidate setup orchestration logic there
3. Keep newSetupCommand() as thin ~15-20 line wrapper

Result: commands.go ~175-180 lines ✓
```

**Split Strategy Option B**: Split into commands.go + commands_util.go
```
commands.go (~140 lines) - Core user commands
  - newRootCommand()
  - newEnsureCommand()
  - newExamplesCommand()
  - newRunCommand()
  - newViewCommand()

commands_util.go (~90 lines) - Utility commands
  - newCompletionCommand()
  - newSetupCommand()
```

**Rationale for Option A**:
- setup.go already has buildSelf(), installSelf(), writeCompletion()
- More consistent with architecture (thin command wrappers)
- Fewer files (11 → 13 vs 11 → 14)
- Better separation: commands do CLI glue, setup.go does setup logic

**Rationale for Option B**:
- Simpler mechanical split (no refactoring)
- Clear user-facing vs meta-operation split
- Both files well under 200 lines

## Decision

### Phase 1: Split build.go → build.go + binaries.go
**Priority**: HIGH (67 lines over limit)
**Approach**: Mechanical split - move 3 functions to new file

1. Create binaries.go with:
   - resolveBinary() - Find binary in dist/PATH/GOBIN
   - ensureGhCli() - Auto-install gh CLI
   - downloadReleaseBinaries() - Download from GitHub releases

2. Keep in build.go:
   - buildBinary() - Compile single binary
   - buildFilename() - Generate filename
   - buildAll() - Build all binaries
   - getRequirement() - Format requirement string
   - getBinaryPath() - Get path in dist/
   - ensureBins() - Orchestrate ensuring binaries exist

**Result**: build.go ~115 lines, binaries.go ~152 lines ✓

### Phase 2: Refactor commands_dev.go release command
**Priority**: MEDIUM (44 lines over limit)
**Approach**: Two options, defer decision to implementation

**Option A (RECOMMENDED)**: Extract logic to release.go
1. Create release.go with methods:
   - cfg.createRelease(ctx, tag, prerelease, draft) - Create GitHub release
   - cfg.uploadReleaseAssets(ctx, tag, files) - Upload assets
   - cfg.buildReleaseBody() - Generate release notes

2. Slim down newDevReleaseCommand() to ~30-40 lines:
   - Parse flags
   - Call cfg.buildAll()
   - Call cfg.createRelease()
   - Call cfg.uploadReleaseAssets()

**Result**: commands_dev.go ~112 lines, release.go ~150 lines ✓

**Option B**: Simple split into commands_dev.go + commands_release.go
- Mechanical split, no refactoring
- Result: commands_dev.go ~112 lines, commands_release.go ~132 lines ✓

### Phase 3: Slim down commands.go
**Priority**: LOW (30 lines over limit)
**Approach**: Extract setup orchestration to setup.go (RECOMMENDED)

1. Create cfg.setupTool(ctx, opts setupOptions) method in setup.go
2. Move orchestration logic from newSetupCommand() to setupTool()
3. Keep newSetupCommand() as thin ~15-20 line wrapper

**Result**: commands.go ~175-180 lines ✓

**Alternative**: Split into commands.go + commands_util.go
- Result: commands.go ~140 lines, commands_util.go ~90 lines ✓

## Consequences

### Positive
- All files under 200 lines
- Better separation of concerns
- Easier to understand and modify individual files
- Follows single responsibility principle
- Consistent architecture (thin command wrappers → config methods)

### Negative
- More files to navigate (11 → 13-14 files)
- May need to jump between files for related operations
- Build/binary acquisition split may feel artificial to some
- Phase 2 Option A requires refactoring, not just moving code

### Neutral
- Maintains DRY principle
- No functional changes
- Existing tests still work
- Still manageable without folder structure (13-14 files)

## Implementation Order

1. **Phase 1: build.go → build.go + binaries.go** (highest priority, simple)
2. **Phase 2: commands_dev.go refactor** (medium priority, may need refactoring)
3. **Phase 3: commands.go refactor** (lowest priority, only 30 lines over)

## Alternatives Considered

### Alternative 1: Increase Line Limit
- Rejected: CLAUDE.md principle is sound - small files prevent mistakes
- 200 lines is reasonable for complete comprehension

### Alternative 2: Inline Small Functions
- Rejected: Reduces readability and reusability
- Functions are already appropriately sized

### Alternative 3: Do Nothing
- Rejected: Violates project guidelines in CLAUDE.md
- Makes codebase less maintainable over time

### Alternative 4: Create Folder Structure
- Not needed yet: 13-14 files is still manageable in root
- Natural grouping already visible by filename (commands*.go, build/binaries/release.go, etc)
- Revisit if file count exceeds 20

## Notes

- Commands are thin wrappers - real logic lives in config methods
- Most flow logic is in: build.go, repos.go, examples.go, setup.go
- This is good architecture - commands do orchestration, not implementation
- After all phases, file count: 11 → 13-14 files (still manageable without folders)
