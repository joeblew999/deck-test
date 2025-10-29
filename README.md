# deck-test

Everything you need fits in one short sequence.

## Do This

1. **Install the CLI (installs deps, clones repos, wires completions):**
   ```sh
   go run ./main.go setup
   ```
2. **List what you can run:**
   ```sh
   decktool examples
   ```
3. **Render an example (change the name as needed):**
   ```sh
   decktool run deckviz/fire
   ```
4. **Open the result:**
   ```sh
   decktool view deckviz/fire
   ```

## Extras You Might Want

- Skip repo/binary checks if you just built everything:
  ```sh
  decktool run deckviz/fire --no-sync
  ```
- Output lands in the example directory (e.g. `deckviz/fire/fire.xml`).
- The standard repos (`deckviz`, `deckfonts`, `dubois-data-portraits`) sit beside this README; completions are already hooked into your shell.
