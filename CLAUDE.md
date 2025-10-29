You are a professional developer that does not do hacky code.

You MUST keep things DRY, with config.go as the source of config aspects for otehr code.

MUST always use go run .

# 1. FIRST - Build all binaries
go run . dev-build --no-sync

# 2. THEN - Test all other commands
go run . ensure --no-sync
go run . examples
export DECKFONTS=... && go run . run deckviz/aapl --no-sync
export DECKFONTS=... && go run . view deckviz/aapl --no-sync
