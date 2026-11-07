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
| `FinishOrders()` | Order entry → confirmation, once every member who can act has an order (`ErrOrdersIncomplete`) (M5d) |
| `FightFriendly()` / `LeaveInPeace()` | A friendly encounter: order entry, or back to the maze (M5d) |
| `ReturnToOrders()` | B)ack at confirmation: every order cleared, order entry from slot 1 |
| `Commit() (PlaybackStep, error)` | From confirmation only: `ExecuteRound` once, playback begins at the first block, which is returned. A second `Commit` fails (`ErrNotConfirming`) |
| `Advance() (PlaybackStep, error)` | Releases the next block (silent blocks too) and moves the display cursor (`MessageIndex`) to it. After the last block it returns `Done` and changes nothing. `ErrAwaitingChoice` while HAMAN's boon prompt is open |
| `RevealedEvents()` | The round's events released so far, and none beyond |
| `ChooseBoon(n)` | Answers HAMAN or MAHAMAN's boon prompt (Wiz 2/3). `Advance` plays up to the block that shows the prompt and waits there (`ErrAwaitingChoice`); `CombatView` lists the boons (`Boons`) only once that block is on screen |
| `LeaveCombat()` | After the victory or defeat screen: victory to the maze, defeat to the castle ("YOUR PARTY HAS PERISHED IN THE MAZE..."). The PAUSE2 wait before it is the front end's (M5d) |
| `EndRound() (RoundOutcome, error)` | Only after `Advance` has returned `Done` (`ErrPlaybackUnfinished` before, even with the last block on screen): drops the frozen round display and decides `RoundEscaped` (back to the maze, "THE PARTY FLED!"), `RoundVictory` (the chest, or the rewards), `RoundDefeated`, or `RoundContinues` (engine `BeginNextRound`: groups compacted, formation re-sorted, orders cleared, then order entry) |

States: `CombatInit` → (`StartCombat`) `CombatChoose` or `CombatExecute`;
`CombatChoose` → (orders, front end) `CombatConfirm` → (`Commit`)
`CombatExecute` → (`Advance` … `Done`, `EndRound`) `CombatChoose`, the maze,
`CombatChest`, `CombatVictory` or `CombatDefeat`. `ReturnToOrders` goes back
from `CombatConfirm` to `CombatChoose`.

**Combat view** (`app/combat_view.go`, M5c). `CombatView()` describes combat
as the player may see it now, as plain values (no pointers, maps or
interfaces; JSON tags for later serialization):

| Field | Content |
|---|---|
| `Phase`, `Next` | opening, friendly, orders, confirm, playback, chest, victory, defeat; and what to do next: `start`, `friendly`, `orders`, `confirm`, `advance`, `choice` (HAMAN), `end-round`, `chest`, `leave` |
| `Round`, `Frozen` | Round number; `Frozen` during playback, when the values below are the round-start display |
| `Party` | Formation order: slot, name, class, AC, HP, max HP, status, poisoned, can act (from the displayed status) |
| `Groups` | Visible groups (count above zero): number, count, the identified or unidentified name, singular or plural by the count shown |
| `Actor`, `Menu` | Order entry: the member being given an order and their menu (availability, reason, note, what to choose next) |
| `Orders` | Order entry and confirmation: the orders given, with target group, ally name, or item by the name the player knows |
| `Events`, `Current`, `Lines` | Playback: the released events (kind, actor, target, lines, part, silent, effects), the one on screen, and its lines. Order entry: a refusal on screen. Victory: the rewards |

Classic timing and knowledge:
- **Frozen during playback:** the party, monster counts and group names
  (including identification, frozen with the counts in `ExecuteRound`;
  engine `DisplayIdentified`) stay at the round-start display until
  `EndRound`.
- **Formation:** the formation shown is the one the round is played in.
- **Party effects:** they carry the HP change, not absolute HP, so silent
  round-start healing or poison stays hidden until the display update.
- **Never in a View:** monster IDs and HP, initiative, the victims of unshown
  actions, the chest trap, and the round's outcome.
- **Last block:** the View on the last block still says `advance`. Only after
  `Advance` returns `Done` does it say `end-round`, so it does not reveal
  that an action is the last.
- **Stricter than the terminal:** the terminal's renderer, unchanged, names
  groups from live identification and live counts, which can show a
  LATUMAPIC identification or a kill's singular name early.

**Chest** (`app/chest.go`, M5e). After a won fight with a chest, the
mechanics, chances and messages are the engine's (Pascal REWARDS),
unchanged. The controller checks the phase and that the member can act. The
"WHO (#) WILL ...?" prompts and the PAUSE2 waits stay in the front end.

| Operation | Effect |
|---|---|
| `OpenChest(ch)`, `InspectChest(ch)`, `CastCalfo(ch)`, `DisarmChest(ch)` | The member acts; the result is shown (`CombatChestResult`). Errors: `ErrNotAtChest`, `ErrNoSuchPartyMember`, `ErrCannotAct`. Anyone may try CALFO ("CAN'T CAST CALFO!"); a second inspection gets "YOU ALREADY LOOKED!" |
| `LeaveChest()` | Leave it unopened: experience, no treasure |
| `AdvanceChest() (done, error)` | Shows the next block of a result; `done` when the last is on screen |
| `EndChestResult() (complete, error)` | After the last block: if the chest was opened or left, the rewards are shared and combat moves to victory (`complete`); otherwise back to the chest menu |

`CombatView` in the chest phase has a `Chest` section:
- **Stage:** menu, who-open, who-calfo, who-inspect, who-disarm, or result.
- **Actions:** with availability and notes, for example "no one here can
  cast CALFO".
- **Members:** whether each can act, has inspected, and can cast CALFO.
- **Reports:** every trap the party has been told about, by inspection,
  CALFO, a disarm that set it off, or opening. Reports are kept as told,
  right or wrong, with whether the trap went off.
- **Complete:** whether the chest is finished.

At victory, `XP` and `Gold` hold the revealed rewards. The true trap, the
chances and the treasure before victory are never in a View. The engine
records the reports as it writes their messages (`ChestReports`), so nothing
is read back from text.

**Information boundary.** The playback operations and `CombatView` expose
only released events: no message, actor, target or effect of a block not yet shown. This
does not isolate the engine state itself: `Controller.Game` is public, and
`engine.CombatState` (messages, `Events()`, groups, victims) is public too.
The terminal renderer reads the round's messages from the display cursor
(`MessageIndex`), as before. The guarantee therefore holds for callers that
use only the controller's playback operations. The desktop GUI already shows
its frontend only `View` values built in Go (`gui/view.go`), never the engine
state; GUI combat must use `CombatView` alone.
Enforcing the boundary for all Go callers would mean hiding the engine state
behind the controller, a larger redesign than M5b.

**Maze and Camp (M6a).** The terminal's maze handler moved into the
controller unchanged (`app/maze.go`, `mazeKey`, with `checkSquare`,
`hazardDamage` and `ouchyDamage`). Only key decoding changed, and the beep
is `Controller.Beep`, set by the terminal to `screen.Beep`. A step is one
transaction, in the original order: the move, the square's effect, healing
and poison (25% per step), then the encounter check (1 in 99, a fight zone,
or a chest alarm). Nothing is skipped, so the random numbers are the
terminal's. A step can end in combat (`PhaseCombat`).

| Operation | Effect |
|---|---|
| `EnterMaze()` | Edge of Town M)AZE: Camp at the entrance (floor 1, 0,0, facing north), fight zones set. `ErrPartyEmpty` without a party |
| `LeaveCamp()` | Camp L)EAVE; at the stairs the question is asked at once. Only from Camp's menu (`ErrWrongState` while one of Camp's screens is open) |
| `Forward()`, `TurnLeft()`, `TurnRight()`, `Kick()`, `Camp()`, `Inspect()` | One maze command (F, L, R, K, C, I). Refused with `ErrPromptPending` while the maze waits for an answer |
| `ContinueMessage()` | [RET] on a message: next page, or dismiss (then an item, a search question or a return to the castle may follow, as in the terminal) |
| `Answer(yes)` | The stairs or a search question (Y/N). Y on the stairs up returns to the castle |
| `PressButton(n)` | Button n (1 is A). `ErrInvalidChoice` outside A to the last button |
| `PickUp()`, `LeaveInspect()` | Inspect's P)ICK UP and L)EAVE |

`HandleKey` routes all maze keys, Camp's L and Edge of Town's M to these; the
rest of Camp and the Edge of Town is still the terminal's. The operations are
the keys: tests drive one game with operations and one with keys through
`HandleKey` from the same random sequence and compare after every step, and
a differential run of 1,500 key sequences on every floor matched `main` line
for line (state and screen text).

`DungeonView()` (`app/dungeon_view.go`) is the read-only, JSON-ready view of
the maze and Camp:
- **Phase:** camp, maze, combat, town or other.
- **Party:** as the maze screen lists it.
- **Corridor:** nearest first, each square's left, right and front walls
  (open, wall or door), and the wall ahead of an open side. It reaches as
  far as the Classic view draws: five squares with light, three with QUICK
  plot, two without, stopping at darkness or the first wall or door ahead.
- **Viewport, Lines, More:** "OUCH!" and "IT'S VERY DARK HERE", the message
  area, and "[RET] FOR MORE".
- **Prompt:** what the maze waits for (messages, stairs, search, buttons,
  inspect, and the terminal-only map and delay). **Buttons** and **Found**
  go with it.
- **Spells:** LIGHT and PROTECT, as on the maze screen.
- **Actions:** the operations open now (the `Act...` names).

The view shows nothing the maze screen does not: no position, facing or
floor (only DUMAPIC tells those), no map, nothing past the light, no
encounter state. A hidden door is a wall without light and a door with it.
The Classic view also lets one flicker into view one draw in six without
light; that needs a random number, and building the view never draws one,
so it is left out (`WIZARDRY-COMPATIBILITY.md` #23). A test checks that
changing any wall changes the terminal's drawing if and only if it changes
the corridor.

Persistence: call `Controller.Save()` after each action.

## Still terminal-only

- Every other phase and prompt goes through `legacyDispatch`: title, castle
  menu, inn, Boltac's, temple, the Edge of Town (except M), Camp (except L),
  combat (whose order legality now comes from the controller, above), utilities,
  and the rest of the Training Grounds edit menu (inspect, class change,
  reroll, set password, Rite of Passage).
- Timers live in the main loop: combat pacing (`combatTimerTick` decides
  when; the controller does the rest, above), the Wiz 3 story pages, inn
  healing ticks and the title animation goroutine.
- Combat still in the terminal: order-entry navigation (which member is
  being ordered, `findNextActor`), the chest's "WHO (#)" prompts, and the
  PAUSE1/PAUSE2 timing.
  Everything else goes through the controller, and `CombatView` is the
  presentation contract for GUI combat. The terminal still takes HAMAN's
  1-3 as soon as the choice opens, before its block is shown
  (`WIZARDRY-COMPATIBILITY.md` #19); `ChooseBoon` accepts that.
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

**GUI combat (M7a).** `gui/combat.go` is the combat bridge.
- **What the frontend gets:** while an encounter runs, `View.Screen` is
  `combat` and `View.Combat` is the controller's `app.CombatView`. That is
  the only combat data the frontend receives.
- **Methods:** `CombatStart`, `CombatFriendly`, `CombatSelect`,
  `CombatOrder`, `CombatFinishOrders`, `CombatBack`, `CombatCommit`,
  `CombatAdvance`, `CombatChooseBoon`, `CombatEndRound`, `CombatLeave`,
  `ChestAction`, `ChestContinue` and `AbandonPractice`. Each calls the
  matching controller operation.
- **Pacing:** the frontend's: Space or Enter for the next step, or automatic
  at a chosen speed.
- **How combat is reached:** through `PracticeEncounter(withChest)`, a
  real encounter from the maze entrance, a practice expedition, or a real
  expedition (below). Practice is played by copies of the party in a
  separate game that is never saved, so the roster is unchanged; a real
  expedition fights in the saved game. `withChest` gives the encounter a chest
  for practice (the entrance's encounters never have one); it is a practice
  setting, not a rule.

**GUI dungeon (M6a).** `gui/dungeon.go` is the dungeon bridge.
- **Starting:** `PracticeExpedition()` copies the party into a separate,
  never-saved game and calls `EnterMaze`.
- **What the frontend gets:** `View.Screen` is `dungeon` and `View.Dungeon`
  is the controller's `app.DungeonView`, the only maze data it receives.
  `View.Expedition` says an expedition is under way.
- **Methods:** `DungeonAction(action, button)` performs one of the view's
  actions through the matching operation; `AbandonPractice` ends it.
- **Encounters:** a step that meets one leaves `PhaseCombat`, so `View`
  switches to the combat screen on the same expedition. After a victory the
  walk goes on, with its losses, in the practice game only.
- **The end:** when the party is back in town (the stairs, or perished) or
  the expedition is abandoned, the practice game is discarded.
- **The screen** (`DungeonScreen.svelte`): a wireframe of the corridor
  (`lib/dungeon.js`), the messages, the party, and the actions with the
  terminal's keys. It is a placeholder, not the Classic renderer.
- **Not yet:** saved positions and resuming (M6b-2), rescue and recovery
  (M6b-4); Camp's other screens (M6b-3).

**Real expeditions (M6b-1).** `RealExpedition()` takes the party itself,
in the saved game, through the same dungeon, combat and chest operations.
- **Saving:** every action is saved (atomically), whatever happened, so
  injuries, spells spent, experience, gold and items are kept. A failed
  save is in `View.Unsaved` until one succeeds; the action is never
  repeated, the next successful save writes it.
- **The end:** only in the castle: the stairs, or the engine's defeat,
  which returns the dead party there. It cannot be abandoned.
- **A restart during an expedition** finds the party in the castle with
  everything saved up to the last action (no position is saved until
  M6b-2), as in the terminal. An interrupted encounter is gone; rewards
  are kept only if they had been given.
- **While the party is out** the Tavern, the Training Grounds, rename,
  reset, restore, reload and practice are refused.
- **View:** `View.Real` marks it; the screens label it and offer Abandon
  only for practice.

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

The file holds the roster and the party's names, nothing about an
expedition: after a restart the party is in the castle (M6b will add an
optional expedition section; see `design/m6b/EXPEDITION-PERSISTENCE.md`).

**Atomic replacement (M6b-0).** `Save`, `RestoreRoster` and `BackupRoster`
write through `engine.writeFileAtomic`: the new contents go to a temporary
file `.<name>.tmp-<random>` in the same folder, which is synced, given the
old file's permissions (0644 less the umask for a new file) and renamed over
the old one; then the folder is synced. A failure before the rename removes
the temporary file and leaves the previous file byte-identical; a failed
folder sync is reported although the new file is already in place. A roster
that is a symbolic link is written through. The bytes written are unchanged
(`save_roster1.resaved.golden.json` pins a re-save of an existing roster,
produced with the old writer). Limits:

- Saving needs write permission on the save folder, not just the file.
- The file is replaced by a new one owned by the user saving; an existing
  owner or extended attributes on the old file are not carried over.
- How durable a save is after a crash or power loss depends on the platform:
  on macOS Go's `File.Sync` uses `F_FULLFSYNC`, and the folder is synced with
  `fsync`. Tests inject failures at every step but cannot simulate power loss.
- A temporary file left by a crash between creating it and the rename is
  never read as a save (its name does not end in `.json`), but nothing
  removes it yet.
- The terminal ignores save errors as before; the GUI reports them.
