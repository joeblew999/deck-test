# Development Guidelines

## Core Principles

1. **No hacky code** - Write professional, maintainable code
2. **Keep it DRY** - config.go is single source of truth
3. **Minimize flags** - Flags create permutations and bugs
4. **Use Makefile** - Always test commands in correct order

Config needs to calculate ALL file paths !! 

## Testing Order

```bash
make test  # Runs: build → ensure → examples
```

## Makefile

- Always use Makefile with `go run .` for development
- Test with Makefile before pushing
- Keep changes simple and focused
