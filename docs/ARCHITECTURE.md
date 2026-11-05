# Istary architecture

Istary is a single Go binary. Game data is embedded; saves are JSON.

```
cmd/istary ──► app ──► engine ──► data
    │           (controller,   (rules, state,   (types, JSON
    │            no terminal)   saves)           loaders)
    ├──► render ──► engine, data       (tcell + Sixel drawing)
    └──► scenarios/wiz1,2,3 ──► data   (embedded game data)
```

| Package | Role | Terminal code? |
|---|---|---|
| `data` | Scenario data types and JSON loaders | No |
| `scenarios/wiz1-3` | Embed each scenario's JSON/binary data and build a `data.Scenario` | No |
| `engine` | Game state (`GameState`, `TownState`, `CombatState`, `CreationState`), rules, save/load | No |
| `app` | Application controller: semantic operations plus a key map for migrated workflows | No (enforced by `app/deps_test.go`) |
| `render` | Draws `engine.GameState` with tcell; Sixel graphics; terminal detection | Yes |
| `cmd/istary` | Terminal front end: flags, event loop, timers, tcell key translation, legacy input handlers | Yes |

## Input flow

```
tcell.EventKey ─► keyFromTcell ─► app.Controller.HandleKey ─┬─ handled ─► done
                  (cmd/istary/    (routes migrated phases   │
                   input.go)       and prompts)             └─ not handled ─► legacyDispatch
                                                                              (tcell handlers in main.go)
```

`dispatchKey` in `cmd/istary/input.go` is the only dispatch entry point. The
main loop and the test driver both call it. The main loop then saves
(`Controller.Save`) and redraws after every key, as before.

The terminal key map in `app` calls the same semantic operations a GUI would
call, so there is one implementation of each rule. Tests in `app` check that
keys and operations reach identical state under the same random seed.

## Controller API (migrated so far)

All operations are methods on `*app.Controller` (`app.New(game)`). Each one
checks that the game is in the right state and returns `app.ErrWrongState`
otherwise. Refusals that the original game shows as an in-game message return
`*app.Refusal{Message}`, and the same text is in `Town.Message`.

**Training Grounds** (`app/training.go`)

| Operation | Effect |
|---|---|
| `EnterTrainingGrounds()` / `LeaveTrainingGrounds()` | Move between town and the Training Grounds |
| `TrainingEnterName(name) (NameLookup, error)` | Look up a name: `NameLeft`, `NameRoster`, `NameNeedsPassword`, `NameOfferCreate`, `NameNotFound` (Wiz 2/3), `NameRosterFull` |
| `ConfirmCreate(yes)` | Answer "create it?"; yes starts creation at the password step |
| `CloseRoster()` | Leave the `*ROSTER` listing |

**Character creation** (`app/creation.go`)

| Operation | Step |
|---|---|
| `Creation() *engine.CreationState` | Read the creation in progress (nil if none) |
| `SetCreationPassword(pw, confirm)` | Password; `ErrPasswordMismatch` restarts entry |
| `ChooseRace(r)` | Race |
| `ChooseAlignment(a)` | Alignment; sets base stats and rolls bonus points |
| `SelectStat(i)`, `NextStat()`, `AddStatPoint()`, `RemoveStatPoint()` | Allocate bonus points |
| `FinishStats()` | Needs 0 points left and at least one eligible class |
| `ChooseClass(c)` | `ErrClassUnavailable` if not eligible |
| `CreationBack()` | Back from race, alignment or class |
| `SetCreationName(name)`, `CancelCreation()` | Name step (reached by going back from race) |
| `ConfirmCreation(keep) (*engine.Character, error)` | Rolls HP, gold and age; adds to roster |

Eligibility for display: `Creation().ClassAvailability()`.

**Tavern** (`app/tavern.go`)

| Operation | Effect |
|---|---|
| `EnterTavern()` / `LeaveTavern()` | Move between the castle and the Tavern |
| `AddMember(name) (needsPassword, error)` | Validate a join; `*Refusal` for WHO?, OUT, BAD ALIGNMENT, ... |
| `SubmitTavernPassword(pw)` / `CancelTavernPassword()` | Finish or abandon a join |
| `RemoveMember(slot)` | Remove party member (0-based); stays in the roster |
| `InspectMember(slot)` | Open the character sheet (sheet itself is terminal-only) |
| `BeginAddMember()`, `BeginRemoveMember()` | Open the prompts (used by the key map) |
| `PartyAlignment(town)` | Party alignment rule (last non-neutral member) |

Persistence: call `Controller.Save()` after each action.

## Still terminal-only

- Every other phase and prompt goes through `legacyDispatch`: title, castle
  menu, inn, Boltac's, temple, Edge of Town, camp, maze, combat, utilities,
  and the Training Grounds password check, character edit menu, class change,
  reroll, delete and Rite of Passage.
- Timers live in the main loop: combat auto-advance (`combatTimerTick`), the
  Wiz 3 story pages, inn healing ticks and the title animation goroutine.
- `handleMazeInput` takes a `*render.Screen` only to beep.
- `engine.TitleState.Anim` holds a `*render.WTAnimation` as `interface{}`.
- `render` keeps package-level state (`SixelSupported`, `ColorMode`,
  `CellWidth`, `CellHeight`).

## Randomness

All game randomness uses the global `math/rand` source and is unchanged.
Tests restore `rand.Seed` with a test-only `//go:debug randseednop=0`
directive and compare against golden files, which also catch changes in the
order or number of random draws. Game logic must stay on one goroutine.

## Saves

`~/.config/wizardry/roster<N>.json` (the upstream directory name is kept for
compatibility). The legacy `~/.config/wizardry/wiz<N>/roster.json` is still
read. The format is pinned by `engine/testdata/save_roster1.golden.json`.
