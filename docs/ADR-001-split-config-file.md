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

### Why v2/ Folder Instead of v1/

**Critical insight**: Keep current working code in root, create new split files in v2/ subdirectory.

#### Benefits:
1. **Current code keeps working** - Root directory unchanged, all commands work
2. **No dependency issues** - v2/ can reference ../.data, ../.src, ../.dist
3. **Clear naming** - Root = current, v2/ = new refactored version
4. **Easy comparison** - Can diff root vs v2/ files
5. **Safe migration** - When v2/ works, just move files up and delete old ones

### Phase 1: Create v2/ Directory Structure
1. Create v2/ directory
2. Copy go.mod, go.sum to v2/ (needed for imports)
3. Update v2/go.mod if needed for module path
4. Create empty files: v2/repos.go, v2/paths.go, v2/build.go, v2/workspace.go, v2/util.go, v2/config.go

### Phase 2: Populate v2/ Files (reading from root)
1. Create v2/repos.go - Read repository functions from root config.go, copy to new file
2. Test: `cd v2 && go build`
3. Create v2/paths.go - Read path functions from root config.go, copy to new file  
4. Test: `cd v2 && go build`
5. Create v2/build.go - Read build functions from root config.go, copy to new file
6. Test: `cd v2 && go build`
7. Create v2/workspace.go - Read workspace function from root config.go, copy to new file
8. Test: `cd v2 && go build`
9. Create v2/util.go - Read utility functions from root config.go, copy to new file
10. Test: `cd v2 && go build`
11. Create v2/config.go - Copy core config struct, constants, loadConfig(), finalize()
12. Test: `cd v2 && go build`
13. Copy v2/commands.go, v2/examples.go, v2/setup.go, v2/main.go from root
14. Update imports in v2/*.go to reference other v2 files
15. Test: `cd v2 && go build`

### Phase 3: Validate v2/ Works
1. Copy Makefile to v2/
2. Run full test suite: `cd v2 && make test`
3. Compare behavior with root version
4. Commit: "Create v2/ with split single-responsibility files"

### Phase 4: Replace Root With v2/
1. Delete old root *.go files (except those in v2/)
2. Move v2/*.go to root
3. Test root still works: `make test`
4. Delete v2/ directory
5. Commit: "Replace monolithic files with split v2/ version"

### Key Principle

**Never use sed/awk to destructively edit files. Always read from source (root config.go), write to new destination (v2/).**
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
