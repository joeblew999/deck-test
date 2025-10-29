# Development Guidelines

## Core Principles

1. **No hacky code** - Write professional, maintainable code.
2. **Keep it DRY** - config.go is single source of truth
3. **Minimize flags** - Flags create permutations and bugs
4. **Use Makefile** - Always test commands in correct order

Config system needs to calculate ALL file paths !! 

## Testing Order

```bash
make test  # Runs: build → ensure → examples
```

## Dot folders

- We use these for non core code. 

## Makefile

- Always use Makefile with `go run .` for development
- Test with Makefile before pushing
- Keep changes simple and focused
- MUST keep it aligned with the CLI commands. 

## README

- Always use  `go run .` for examples.
- Keep changes simple and focused
- MUST keep the Makefile and README aligned.

## File Size

- **Maximum 200 lines per file** - Files must be small enough to read completely
- If a file exceeds 200 lines, split it by responsibility into separate files
- Small files prevent mistakes from blind sed/awk edits
- Use clear file names that indicate their single responsibility (repos.go, paths.go, build.go, etc)

## File Responsibilities

- **One responsibility per file** - Each file should have a single, clear purpose
- **File naming**: Use the responsibility as the filename (repos.go for repository management, paths.go for path helpers)
- **No mixing concerns** - Don't put git operations and build operations in the same file
- **Types belong with behavior** - Put struct definitions in the file where they're primarily used
- **Shared types go in config.go** - Only types used across multiple files belong in the core config file

### Example File Organization:

```
config.go     - Core config struct, constants, loadConfig(), finalize()
repos.go      - Repository management: git clone, update, sync operations
paths.go      - ALL path calculations and file path helpers
build.go      - Binary building and downloading
workspace.go  - Go workspace management
util.go       - Pure utility functions (no side effects)
```

### Rule: If you can't describe a file's purpose in 5 words, it has too many responsibilities.
