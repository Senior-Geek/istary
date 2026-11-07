# Istary desktop GUI (prototype)

A Wails v2 + Svelte 5 front end. It covers Wizardry I character creation,
roster management, the party (Gilgamesh's Tavern), and combat (M7a):
practice encounters played through `app.Controller` and `app.CombatView`.

**Practice encounters.** There is no dungeon in the GUI yet.
- **What you fight:** "Practice encounter" in the Party column starts a real
  encounter from the maze entrance, with copies of the party.
- **Nothing is kept:** the results are discarded and the roster is not
  touched.
- **The chest option:** "with a chest" adds a chest to try the chest screen.
- **The screen** (`frontend/src/lib/CombatScreen.svelte`):
  - order entry with the actions each member may take and why not;
  - target choice and the order review;
  - playback one action at a time (Space or Enter, or automatic);
  - the chest, victory, defeat and escape;
  - an expandable "details" log.

All rules come from the Go engine through `app.Controller`. The Svelte code
only renders `View` (see `view.go`) and calls `Session` methods
(`session.go`); it holds no game state or rules of its own.

## Saves

The GUI never uses the terminal game's `~/.config/wizardry`. It saves to
`~/Library/Application Support/Istary/dev-saves`, or to `$ISTARY_SAVE_DIR`
when set, and refuses to start if pointed at the terminal directory.

## Build and run (macOS)

Requires Go, Node 20.19+/22.12+, Xcode command-line tools, and the Wails CLI
(`go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0`).

```bash
cd gui
wails build -platform darwin/arm64   # or `make gui` from the repo root
open build/bin/Istary.app

wails dev                            # live-reload development
```

This directory is a separate Go module (`github.com/Senior-Geek/istary/gui`)
with a `replace` back to the main module, so `go test ./...` at the repo root
and the terminal build do not depend on Wails. Run its tests with
`cd gui && go test ./...`.
