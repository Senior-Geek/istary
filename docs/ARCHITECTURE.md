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
| `gui` (separate module) | Wails + Svelte desktop prototype: Wizardry I character creation via `app.Controller` | No tcell; uses Wails/WebKit |

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
| `CloseNamePrompt()` | Close the `NAME >` prompt without a name (terminal ESC) |
| `CancelPasswordCheck()` | Back out of an existing character's password prompt (terminal ESC) |
| `SubmitCharacterPassword(pw)` | Answer an existing character's password; opens the edit menu or refuses with `** THATS NOT IT **` |
| `EditCharacter()`, `LeaveCharacterEdit()` | The character open in the edit menu; close the menu |
| `BeginDeleteCharacter()`, `ConfirmDeleteCharacter(yes)` | "ARE YOU SURE YOU WANT TO DELETE"; yes removes from roster and party |
| `ClearRoster()` | Empty roster and party (GUI reset; caller backs up with `engine.BackupRoster` and saves) |
| `RenameCharacter(index, name)` | Utilities rename rules: trimmed, uppercased, not empty, not already in use |
| `ValidateName(name)` | Whole-name check against the creation prompt's rules (15 chars, allowed characters) |

Class guidance for creation: `ClassGuide()` (`app/classguide.go`) reports each
class's minimums, allowed alignments, shortfalls, eligibility and whether it is
reachable with the remaining or redistributed bonus points.

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
| `JoinRefusal(ch)` | Read-only: the Tavern's answer if `ch` tried to join now (`""` = may join, password still asked). Shares its checks with the real join |
| `InParty(town, ch)`, `PartySlot(town, ch)` | Membership from the party list. Never use `InMaze` for this: joining sets it, as does being out in the dungeon |
| `MaxPartySize`, `FrontRowSize` | 6 members; the first 3 are the front row (only they can melee) |

**Combat order entry** (`app/combat.go`, M5a). Orders are given to a
character, not a slot; the controller records them in the character's current
slot and the engine binds them to that character (C1). The rules are those the
terminal applied in `handleCombatChoose`; none is new. Prompts, keys and
moving to the next member stay in the front end. No timers, no random numbers.

| Operation | Effect |
|---|---|
| `CanGiveOrder(ch)` | Whether `ch` is offered an order: OK, asleep or afraid (engine `IsAlive`). Used by the terminal's `findNextActor` |
| `CombatOrderMenu(ch) ([]OrderOption, error)` | Fight, Spell, Parry, Run, Use, Dispel, each with `Available`, a `Reason` when not, a `Note` (Use with nothing usable counts as Parry) and the `Target` to choose next (group, member, spell, item or nothing). `ErrCannotAct` for paralyzed, stoned, dead, ashed or lost members |
| `CombatSpellTarget(ch, name) (OrderTarget, error)` | Checks the spell (`*Refusal` "** UNKNOWN SPELL **" or "** CANT CAST **") and says what it needs: a group, a member or nothing |
| `CombatUsableItems(ch) []int` | Indexes of equipped items that cast a spell |
| `SetOrder(ch, Order) error` | Validates and records the order: Fight only from the front row (1-3); Dispel for priests, bishops above level 3, lords above level 8; Fight, Dispel and group spells at a living group; member spells at any party member; Use of a usable item, aimed by the item's spell; Use with nothing usable recorded as Parry. Errors: `ErrNotInCombat`, `ErrNoSuchPartyMember`, `ErrCannotAct`, `ErrOrderUnavailable` (with the reason), `ErrInvalidTarget`, `ErrNoSuchItem`, `ErrInvalidChoice` |

Two cases are kept as found (`WIZARDRY-COMPATIBILITY.md` #17, #18): a group
spell is recorded with no group when no group is alive (its known route, #17,
is corrected in M5b), and an item whose spell targets a single monster
(SCROLL/HALITO) is aimed at its user.

**Combat round lifecycle** (`app/combat_round.go`, M5b). The controller starts
combat, executes a confirmed round once, releases its events one block at a
time and ends the round. The terminal keeps the keys, the display and the
pacing: its timer (`combatTimerTick`, PAUSE1) calls `Advance`, and `EndRound`
after the last block. No timers or random numbers in the controller.

| Operation | Effect |
|---|---|
| `StartCombat() (playback, error)` | From the opening pause (`CombatInit`): order entry from the first member who can act, or the monsters' surprise round executed at once (`playback` true) |
| `ReturnToOrders()` | B)ack at confirmation: every order cleared, order entry from slot 1 |
| `Commit() (PlaybackStep, error)` | From confirmation only: `ExecuteRound` once, playback begins at the first block, which is returned. A second `Commit` fails (`ErrNotConfirming`) |
| `Advance() (PlaybackStep, error)` | Releases the next block (silent blocks too) and moves the display cursor (`MessageIndex`) to it. After the last block it returns `Done` and changes nothing. `ErrAwaitingChoice` while HAMAN's boon prompt is open |
| `RevealedEvents()` | The round's events released so far, and none beyond |
| `ChooseBoon(n)` | Answers HAMAN or MAHAMAN's boon prompt (Wiz 2/3) |
| `EndRound() (RoundOutcome, error)` | Only when every block has been released (`ErrPlaybackUnfinished`): drops the frozen round display and decides `RoundEscaped` (back to the maze, "THE PARTY FLED!"), `RoundVictory` (the chest, or the rewards), `RoundDefeated`, or `RoundContinues` (engine `BeginNextRound`: groups compacted, formation re-sorted, orders cleared, then order entry) |

States: `CombatInit` → (`StartCombat`) `CombatChoose` or `CombatExecute`;
`CombatChoose` → (orders, front end) `CombatConfirm` → (`Commit`)
`CombatExecute` → (`Advance` … `Done`, `EndRound`) `CombatChoose`, the maze,
`CombatChest`, `CombatVictory` or `CombatDefeat`. `ReturnToOrders` goes back
from `CombatConfirm` to `CombatChoose`.

**Information boundary.** The playback operations expose only released
events: no message, actor, target or effect of a block not yet shown. This
does not isolate the engine state itself: `Controller.Game` is public, and
`engine.CombatState` (messages, `Events()`, groups, victims) is public too.
The terminal renderer reads the round's messages from the display cursor
(`MessageIndex`), as before. The guarantee therefore holds for callers that
use only the controller's playback operations. The desktop GUI already shows
its frontend only `View` values built in Go (`gui/view.go`), never the engine
state; GUI combat must build its combat view from `RevealedEvents` alone.
Enforcing the boundary for all Go callers would mean hiding the engine state
behind the controller, a larger redesign than M5b.

Persistence: call `Controller.Save()` after each action.

## Still terminal-only

- Every other phase and prompt goes through `legacyDispatch`: title, castle
  menu, inn, Boltac's, temple, Edge of Town, camp, maze, combat (whose order
  legality now comes from the controller, above), utilities,
  and the rest of the Training Grounds edit menu (inspect, class change,
  reroll, set password, Rite of Passage).
- Timers live in the main loop: combat pacing (`combatTimerTick` decides
  when; the controller does the rest, above), the Wiz 3 story pages, inn
  healing ticks and the title animation goroutine.
- Combat still in the terminal: the friendly-encounter choice, the chest
  (and its message pacing), and leaving combat after the victory or defeat
  pause.
- `handleMazeInput` takes a `*render.Screen` only to beep.
- `engine.TitleState.Anim` holds a `*render.WTAnimation` as `interface{}`.
- `render` keeps package-level state (`SixelSupported`, `ColorMode`,
  `CellWidth`, `CellHeight`).

## Desktop GUI

`gui/` is its own Go module so Wails stays out of the terminal build. Wails v2
has a single native window, so the character sheet is a self-contained Svelte
component (`gui/frontend/src/lib/CharacterSheet.svelte`) shown in a movable,
resizable in-window panel; it takes data and callbacks only, ready to move to a
native window under Wails v3. Roster backups live in `<saveDir>/backups`. Its
`Session` type serialises all calls with a mutex, calls controller operations,
saves after each one, and returns a `View` for Svelte to render. The GUI shows
roster, Training Grounds and party together while the game has one current
town location; `gui/location.go` (`goTo`) is the only code that changes it,
always through the game's own leave/enter steps. See
`gui/README.md`.

## Randomness

All game randomness uses the global `math/rand` source and is unchanged.
Tests restore `rand.Seed` with a test-only `//go:debug randseednop=0`
directive and compare against golden files, which also catch changes in the
order or number of random draws. Game logic must stay on one goroutine.

## Saves

`~/.config/wizardry/roster<N>.json` (the upstream directory name is kept for
compatibility). `engine.SetSaveDir` redirects saves for other front ends; the
GUI uses `~/Library/Application Support/Istary/dev-saves` and refuses the
terminal directory. The legacy `~/.config/wizardry/wiz<N>/roster.json` is still
read. The format is pinned by `engine/testdata/save_roster1.golden.json`.
