# Istary desktop GUI (prototype)

A Wails v2 + Svelte 5 front end. It covers Wizardry I character creation,
roster management, the party (Gilgamesh's Tavern), combat (M7a) and the
dungeon (M6a): real expeditions (M6b-1), practice expeditions and practice
encounters played through `app.Controller`, `app.DungeonView` and
`app.CombatView`.

**Real expeditions.** "Real expedition" in the Party column takes the party
itself into the maze, to Camp at the entrance. It plays like a practice
expedition (below), but everything is kept: the game is saved after every
action. It ends in the castle, by the stairs or with the party's defeat
(the engine returns the dead party there); it cannot be abandoned. If
Istary closes or crashes during an expedition, it resumes where it was last
saved; an encounter it was in is over and its unclaimed rewards are lost.
While the party is out, the roster, Tavern and Training Grounds wait. A
save that fails is shown until one succeeds. Only one Istary window can use
a save folder at a time.

**Practice expeditions.** "Practice expedition" in the Party column takes
copies of the party into the real maze, to Camp at the entrance.
- **Moving:** the buttons or the terminal's keys (F/W/↑ forward, L/A/←
  left, R/D/→ right, K kick, C camp, I inspect, Enter, Y/N, button letters).
  Doors need a kick, as in the engine.
- **Everything happens:** square effects, healing and poison, and
  encounters, exactly as in the terminal. An encounter opens the combat
  screen; after a victory the walk goes on.
- **Nothing is kept:** taking the stairs up, perishing or Abandon ends the
  expedition and discards it; the roster is not touched.
- **The screen** (`frontend/src/lib/DungeonScreen.svelte`) draws a simple
  wireframe of the corridor, not the Classic picture. Camp offers only
  Leave so far.

**Practice encounters.**
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
