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

1. Create new files (repos.go, paths.go, build.go, workspace.go, util.go)
2. Move functions from config.go to new files
3. Test after each file is created
4. Remove moved functions from config.go
5. Final test that everything still works
6. Commit

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
