# ADR-001: Split config.go Into Single-Responsibility Files

## Status
Proposed

## Context

Current config.go is 791 lines and contains multiple unrelated responsibilities:
- Configuration loading
- Repository management (git operations)
- Binary management (build/download)
- Path helper functions
- Workspace management
- Utility functions

### Problems This Causes:

1. **File too large to read completely** - Cannot hold entire file in context
2. **Sed/awk editing required** - Must use line-number-based blind edits which frequently break code
3. **Mixed concerns** - Changing fonts handling can break build logic
4. **Hard to understand** - Cannot quickly find relevant code
5. **Violates CLAUDE.md** - Exceeds 200 line maximum, violates single responsibility

### Recent Mistakes Due To File Size:

- Multiple failed attempts to add fontsDir using sed/awk
- Accidentally duplicated closing braces
- Broke function signatures while trying to edit specific lines
- Could not see full context of changes being made

## Decision

Split config.go (791 lines) into focused single-responsibility files:

```
config.go (core only - ~150 lines)
├── Constants (dataDir, srcDir, distDir, fontsDir)
├── Types (config, repoConfig, binSpec, buildTarget, buildResult)
├── loadConfig()
└── finalize()

repos.go (~150 lines)
├── initDataRepos()
├── initFontsRepo()
├── initCodeRepos()
├── addDataRepo()
├── addCodeRepo()
├── ensureRepos()
├── ensureBuildRepos()
├── gitCloneOrUpdate()
├── gitClone()
├── gitUpdate()
└── runGit()

paths.go (~100 lines)
├── getBinaryPath()
├── getDistGlob()
├── getGoBinPath()
├── getExampleDir()
├── getExampleDshPath()
├── getExampleXmlPath()
├── getShellCompletionPath()
├── getShellRCPath()
├── getRepoNameByDir()
└── resolveBinary()

build.go (~200 lines)
├── initToolchain()
├── buildBinary()
├── buildFilename()
├── buildAll()
├── ensureBins()
├── downloadReleaseBinaries()
├── Build target methods (buildEnv, extension)
└── getRequirement()

workspace.go (~50 lines)
└── ensureWorkspace()

util.go (~100 lines)
├── getenvDefault()
├── getenvInt()
├── resolveGoBin()
├── runGoEnv()
├── absPath()
├── expandPath()
├── parseExample()
├── normalizeExampleName()
└── copyFile()
```

### File Descriptions (5-word test):

- **config.go**: Core configuration loading and types
- **repos.go**: Git repository management operations
- **paths.go**: All file path calculations
- **build.go**: Binary building and downloading
- **workspace.go**: Go workspace file generation
- **util.go**: Pure utility helper functions

## Benefits

1. **No more sed/awk mistakes** - Files small enough to read completely
2. **Clear separation of concerns** - Fonts changes only touch repos.go
3. **Easy to find code** - Know exactly which file to open
4. **Follows CLAUDE.md** - All files under 200 lines, single responsibility
5. **Safer refactoring** - Changes isolated to specific domains

## Migration Plan

### Why Move To v1/ First

**Critical insight**: Moving current code to v1/ BEFORE splitting allows reading v1/config.go as reference while writing new files, instead of destructive in-place editing.

#### Without v1/:
```bash
# Risky: Must edit config.go in place with sed/awk
sed -n '340,360p' config.go > repos.go  # Extract functions
sed -i '340,360d' config.go             # Delete from original
# If something breaks, hard to recover
```

#### With v1/:
```bash
# Safe: Read from v1/, write to new files
cat v1/config.go  # Reference (read-only)
# Copy functions to new repos.go, paths.go, etc.
# Old code still works in v1/ for comparison
# No destructive edits with sed/awk
```

#### Benefits:

1. **No more sed/awk editing** - Just read v1/ and write new files
2. **Old code preserved** - v1/ keeps working during migration
3. **Easy comparison** - Can diff v1/ vs new code
4. **Safe rollback** - If new code breaks, v1/ still works
5. **Clear progress** - Can see exactly what's been migrated

### Phase 1: Preserve Current Code
1. Create v1/ directory
2. Move ALL project files to v1/ (not just *.go):
   - *.go (source code)
   - go.mod, go.sum (dependencies)
   - Makefile (build commands)
   - README.md (documentation)
   - CLAUDE.md (development rules)
   - .gitignore (git config)
3. Update v1/go.mod module path if needed
4. Test v1/ works independently: `cd v1 && make test`
5. Commit: "Move current working code to v1/ for safe refactoring"
### Phase 2: Create New Split Files (reading from v1/)
1. Create repos.go - Read repository functions from v1/config.go, copy to new file
2. Test: `go build`
3. Create paths.go - Read path functions from v1/config.go, copy to new file  
4. Test: `go build`
5. Create build.go - Read build functions from v1/config.go, copy to new file
6. Test: `go build`
7. Create workspace.go - Read workspace function from v1/config.go, copy to new file
8. Test: `go build`
9. Create util.go - Read utility functions from v1/config.go, copy to new file
10. Test: `go build`
11. Create config.go - Copy core config struct, constants, loadConfig(), finalize()
12. Test: `go build`

### Phase 3: Validate
1. Run full test suite: `make test`
2. Compare behavior with v1/
3. Commit: "Split config into single-responsibility files"

### Phase 4: Cleanup
1. Delete v1/ directory once confident
2. Commit: "Remove v1/ after successful migration"

### Key Principle

**Never use sed/awk to destructively edit files. Always read from source, write to new destination.**
## Risks

- Must ensure all functions remain accessible (exported names)
- Must not break any imports
- Must maintain behavior exactly

## Compliance With CLAUDE.md

✓ Maximum 200 lines per file
✓ One responsibility per file
✓ Clear file naming (responsibility-based)
✓ 5-word description test passes for all files
✓ Config still calculates ALL file paths (in paths.go)
✓ Keeps things DRY

## Success Criteria

- [ ] config.go under 200 lines
- [ ] All new files under 200 lines
- [ ] Each file has single clear responsibility
- [ ] All tests pass (make test)
- [ ] No behavior changes
- [ ] Easier to edit without sed/awk
