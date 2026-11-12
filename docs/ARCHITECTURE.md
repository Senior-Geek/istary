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

**Boltac's Trading Post** (`app/shop.go`, `app/shop_ops.go`, `app/shop_view.go`, M8-2).
The terminal's shop moved here intact (the castle's B, the menus, the
catalog pages, the PURCHASE WHICH ITEM prompt, RETURN); the terminal reaches
it through `HandleKey` and keeps no shop code. An old-vs-new comparison
(random parties, items, gold and stock; random keys; 650,000 key presses)
found the whole game state identical after every key. Boltac's answers are
decided once, in `buyQuote`, `sellQuote`, `uncurseQuote` and
`identifyQuote`, which the keys act on and `ShopView` reports. Stock is the
game's own (`GameState.Stock`, M8-1). Nothing here draws a random number.

| Operation | Effect |
|---|---|
| `ShopEnter(slot)` | Choose the customer (WHO WILL ENTER) |
| `ShopLeaveCustomer()`, `LeaveShop()` | Back to WHO WILL ENTER; out to the castle (RETURN steps) |
| `Buy(item)` | P and the item's place: refused (`*Refusal`: last one, can't carry, can't afford) or bought; an item the customer's class cannot use is asked about first. Takes the item itself: the terminal's page and catalog cursor are left as they are, except that an unusable item becomes the cursor, as with the keys |
| `ConfirmUnusable(yes)` | Answer UNUSABLE ITEM - CONFIRM BUY |
| `Sell(n)`, `Uncurse(n)`, `ShopIdentify(n)` | S, U or I and the item's number; refusals as `*Refusal`. Selling an equipped item asks first (B7, Player's Guide p.31): step `confirmSell` |
| `ConfirmSale(yes)` | Answer EQUIPPED ITEM - CONFIRM SELL: Y sells it (taking it off; AC worked out again), N keeps it worn |
| `ShopView()` | Read-only: the step, Boltac's words, the customer (slot, name, class, gold, room), the item he asks about (`Confirm`, `SellConfirm`), the catalog (price, stock, usable, his answer now) and the customer's items (name as known, equipped, cursed as far as the player knows, price or why not, whether selling asks, fee or why not) |

Tests: operations against the terminal's keys on a twin (random, 300
seeds), Buy on the page shown exactly P and its place, `ShopView`'s answers
against what the operations then do, leaving from every step, and the M8-0
characterization (`cmd/istary/shop_test.go`, goldens unchanged).

**Curses and armour class** (`engine/curse.go`, `engine/ac.go`, M8,
findings #34 and #35). A possession's `cursed` flag means *stuck*: equipping
a cursed type, or an item of another alignment, sets it (Player's Guide
pp.4-5); a stuck item holds its category and cannot be taken off, dropped,
traded or sold; Boltac's U)NCURSE removes it, ending the curse.
`engine.KnownCursed` is what a front end may show (identified or worn).
`Character.RecalcAC` is the one place AC is worked out (10 less each worn
item's modifier); it runs at each step of the equip walk, after a sale, an
uncursing and a drop, and on every load, so the stored AC is never stale.
Tests: `app/curse_test.go`, `app/ac_test.go`.

**Equipping in town** (`app/equip.go`, M8). `Equipment(index)`,
`TownEquip(index, n)` (reports whether the item stuck) and
`TownUnequip(index, n)` equip a roster character's items one at a time, by
the equip walk's rules (one per category, the class's usable items, special
items carried not equipped, stuck items holding their place, curses on
wearing, AC worked out); each item's view carries the answer to equipping
and taking it off. Refused outside town and for a member of a saved
expedition. `TestTownEquippingIsTheWalk` holds 150 random characters'
item-by-item equipping to a copy doing the same through Camp's walk.

**Town navigation** (`app/navigation.go`, Castle-0). The terminal still reads
the castle menu and the Edge of Town's T and C itself; these make the same
changes, for front ends without keys. `cmd/istary/navigation_test.go` holds
each move to its key (same town state, no random number, same refusal) and
checks that no other key on those menus moves anywhere.

| Operation | Effect |
|---|---|
| `Exits() []Exit` | The ways out of the current town location in menu order: the castle (Tavern, Inn, Boltac's, Temple, Edge of Town), the Edge of Town (Training Grounds, castle), and back to the castle from the Training Grounds, the Tavern and Boltac's. `Exit.Refusal` is the game's answer if the move is refused now (`YOU HAVE NO PARTY!` for the Inn, Boltac's and the Temple) |
| `GoTo(loc)` | Make one of those moves as its key does; a refusal is in `Town.Message` and returned as `*Refusal`. Boltac's is entered as by B; it is left for the castle as by RETURN at WHO WILL ENTER (`LeaveShop` from anywhere in it). The Inn and the Temple are still the terminal's: `ErrLocationNotMigrated`, nothing changed |
| `EnterEdgeOfTown()` / `LeaveEdgeOfTown()` | Castle E; Edge of Town C |

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
| `LeaveCombat()` | After the victory or defeat screen: victory to the maze; defeat leaves the party stranded on its square and the player in the castle with no party ("YOUR PARTY HAS PERISHED IN THE MAZE...", R-1, `design/r1/STRANDED-PARTIES.md`). A victory after which no one can act is stranded the same way. The PAUSE2 wait before it is the front end's (M5d) |
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
| `EnterMaze()` | Edge of Town M)AZE: Camp at the entrance (floor 1, 0,0, facing north), fight zones set. `ErrPartyEmpty` without a party; `ErrNoOneCanGoOn` when no one in the party can act (R-2: the incapacitated may come along only with someone who can) |
| `LeaveCamp()` | Camp L)EAVE; at the stairs the question is asked at once. Only from Camp's menu (`ErrWrongState` while one of Camp's screens is open) |
| `Forward()`, `TurnLeft()`, `TurnRight()`, `Kick()`, `Camp()`, `Inspect()` | One maze command (F, L, R, K, C, I). Refused with `ErrPromptPending` while the maze waits for an answer |
| `ContinueMessage()` | [RET] on a message: next page, or dismiss (then an item, a search question or a return to the castle may follow, as in the terminal) |
| `Answer(yes)` | The stairs or a search question (Y/N). Y on the stairs up returns to the castle |
| `PressButton(n)` | Button n (1 is A). `ErrInvalidChoice` outside A to the last button |
| `PickUp(n)`, `LeaveInspect()` | Inspect's P)ICK UP of the n-th character found (from 1), and L)EAVE. Inspect lists everyone stranded in the party's area (`engine.GameState.FindStranded`; R-2, `design/r2/RESCUE.md`); `DungeonView.Inspect` gives each with the room left. A full party is refused (`YOUR PARTY HAS 6; NAME STAYS HERE.`), a number not on the list is `ErrInvalidChoice`; either way nothing changes. The terminal asks `GET WHO (0=EXIT) >` |

`HandleKey` routes all maze keys, Camp's L and Edge of Town's M to these
(all of Camp since M6b-3A, below); the rest of the Edge of Town is still the
terminal's. The operations are
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
- **Default:** at the stairs only, the answer the screen highlights:
  `no` when the party came off the stairs onto the square (down from the
  castle, or off a staircase taken), otherwise `yes`. It tells how the
  party came, never where; it is kept in the controller, not saved, and no
  key takes it (`app/stairs.go`, `design/mazenav/MAZE-NAVIGATION.md`).

The view shows nothing the maze screen does not: no position, facing or
floor (only DUMAPIC tells those), no map, nothing past the light, no
encounter state. (The maps of M-1 and UX-1's coordinate readout,
`View.Position`, are separate views, below.) The corridor
itself is the engine's `GameState.Sight()` (`engine/sight.go`), which the
exploration record reads too. A hidden door is a wall without light and a door with it.
The Classic view also lets one flicker into view one draw in six without
light; that needs a random number, and building the view never draws one,
so it is left out (`WIZARDRY-COMPATIBILITY.md` #23). A test checks that
changing any wall changes the terminal's drawing if and only if it changes
the corridor.

Persistence: call `Controller.Save()` after each action.

**Camp and the inspect screen (M6b-3A).** The terminal's Camp handler moved
intact to `app/camp.go`: reorder, equip the whole party, disband, inspect a
member, cast a spell and choose its target, MALOR, DUMAPIC and use an item.
The inspect screen it shares with the castle moved to `app/inspect.go`:
equip, drop, trade gold and items, the spell books, and Camp's identify.
Only key decoding changed. `HandleKey` takes every Camp key, and in town the
inspect screens. `RecalcAC` is exported for the terminal's startup and class
change. A differential run of 400 seeds of random Camp keys matched
`d7bd206` exactly: 71,130 states and the final random-number state.

Operations (`app/camp_ops.go`). Each one checks that its screen is open,
then sends the terminal's keys; the game's refusals stay in `Town.Message`.

| Screen | Operations |
|---|---|
| Camp menu | `CampInspect(slot)`, `BeginReorder()` then `ReorderPick(slot)`, `EquipParty()`, `Disband()`, `LeaveCamp()` |
| Inspect (Camp or town) | `LeaveInspection()`, `BeginEquip()` then `EquipChoose(n)` / `EquipSkip()`, `BeginDrop()` then `DropItem(n)` / `EndDrop()`, `BeginTrade()` then `TradeWith(slot)`, `TradeGold(amount)`, `TradeItem(n)`, `EndTrade()`, `OpenSpellBooks()` then `ShowSpellBook(priest)` / `CloseSpellBook()` |
| Inspect, Camp only | `CastSpell(name)` then `CastOn(slot)` / `CancelCast()`, `LeaveDumapic()`, `Identify()`, `UseItem(n)` |

- **MALOR:** `CastSpell` refuses it (`ErrMalorUnavailable`, finding #25).
  The terminal's keys still cast it.
- **Use:** the engine's stub (#30).
- **Disband:** the engine's (#26).

`CampView()` (`app/camp_view.go`) gives a front end:
- the screen and the message;
- equip choices, with unknown names until an item is identified;
- reorder and trade progress, and the spell waiting for its target;
- the actions open now.

DUMAPIC's result (facing, squares east and north of the stairs, levels
below) appears only while its screen is open; nothing else gives a
position.

A Camp checkpoint records Camp's screen and every Camp field
(`engine.CampScreen`, by name), and `Resume` restores them exactly. An equip
or trade interrupted half-way, or an open DUMAPIC, comes back as it was.

**Camp in the GUI (M6b-3B).** `Session.CampAction(CampRequest)` (`gui/camp.go`)
runs one of the actions `CampView` lists, through its operation above; like
every dungeon action it is saved with its checkpoint on a real expedition and
saves nothing on a practice one. `View.Camp` (`CampPanel`) carries the
`CampView`, each party member's sheet (`CharacterView`, built from the game
being played, so a practice party's copies too) and, for the menu, why an
action is unavailable. `CampView` also lists the inspected character's known
spells (`CampSpellView`): castable, or why not (no effect in Camp, no slots
left, not the character's book, MALOR). Anti-magic is not shown; only casting
reveals it.

- **Not offered:** D)ISBAND (finding #26, until M6b-4), MALOR (#25) and
  spells with no effect in Camp (#33, which the game would waste). The
  bridge refuses each with its reason before anything is pressed, and the
  page shows them disabled with that reason. No bound method takes keys.
- **The page** (`CampScreen.svelte`, helpers in `lib/camp.js`): the party
  bar at the bottom of the dungeon screen (since C0-5; click a member, or
  press 1-6, to pick a member where the screen asks for one), the
  operation under way, the inspected character's sheet, and each screen's
  controls; Escape takes the screen's way back. Choosing which spell to cast
  or item to use is a picker only; nothing is sent until the choice.
- **Tests:** the bridge against the terminal's keys on a copy of the same
  save (states, random numbers and panels), restarts on every Camp screen
  against uninterrupted play, the refusals (the save unchanged), practice
  isolation, unknown item names, and a recorded fixture for the page's
  helpers (`testdata/camp-views.json`).

## Still terminal-only

- Every other phase and prompt goes through `legacyDispatch`: title, castle
  menu (except B), inn, temple, the Edge of Town (except M),
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
town location, which only `Navigate` (below) changes; an action is accepted
only at its own location (`gui/location.go`). See `gui/README.md`.

**Town navigation (Castle-0, C0-1).** `Session.Navigate(dest)`
(`gui/navigation.go`) moves along one of `View.Exits` with
`app.Controller.GoTo`, following the original's layout: the Castle is the
hub; the Training Grounds, the maze and Settings (Istary's Utilities) are at
the Edge of Town; the Training Grounds, the Tavern and Settings lead back to
the Castle (Player's Guide pp.22, 36, 38).
- **View:** `Location` (a town location, `settings`, or `maze` while an
  expedition or practice is under way), `Exits` (`{dest, open, reason}`;
  the Inn and the Temple are closed with the game's refusal or "not open
  in Istary's window yet"; Boltac's is open for a party since M8-3), and `PartyBar`: the six positions of
  the party being played (the practice copies during practice), empty ones
  included, with level, HP, status and poison.
- **Boltac's layout (UX-2, `design/ux2/UX-2.md`):** arriving serves the
  first party member (`serveFirst`); the frame's `fill` layout keeps the
  page from scrolling; the merchandise list and the Selected Character
  panel (the customer's roster entry, equipping by `EquipAction`) scroll on
  their own; the party bar is always shown there and is the customer
  picker. Equipping is refused while Boltac waits for an answer.
- **Settings** is the window's own, not a game location: a session flag
  while the game stands at the Edge of Town, cleared by any other move.
- **Never saved:** the town location is not in the save, so `Navigate`
  writes nothing and draws no random number. It is refused while an
  expedition (real, resumed or practice) is under way and during creation;
  an open town prompt is closed first by the game's cancel steps.
- **Tests:** every destination from every location (the save folder's
  files and times and the next random number unchanged, refusals leave the
  view as it was), refusals out of town and after a restart, an
  unresumable expedition left as saved, prompts closed, Settings left, and
  the party bar against the Tavern's and the dungeon's party.
- **Actions at their locations (C0-2).** Nothing moves the player for an
  action any more (`goTo` is gone): joining and leaving the party are
  accepted at the Tavern; creating, renaming and deleting at the Training
  Grounds; real and practice expeditions at the Edge of Town (so a GUI
  expedition stands at the Edge of Town, as the terminal's M does); reload,
  reset and restore in Settings, which a reload or restore keeps the player
  in. Elsewhere each is refused with where to go ("... Go there first."),
  before anything is changed or drawn. An open town prompt at the location
  is closed by the game's cancel steps, as before. Returning an
  unresumable expedition's party stays open on every town screen.
- **Each place's own screen (C0-4)** offers only the actions accepted
  there; Go still refuses them anywhere else.
- **Tests:** `TestActionsOnlyAtTheirLocations` tries every such action at
  every location (refused elsewhere with the view, the saved roster and
  the random numbers unchanged); the bridge tests walk explicitly with
  `walkTo` before each action.

**The Adventurer's Inn** (`app/inn.go`, `app/inn_ops.go`, `app/inn_view.go`,
I-1, I-2). The terminal's Inn moved here intact (the castle's A, WHO WILL
STAY, the rooms, the heal loop, the level check, RETURN); the terminal
reaches it through `HandleKey`, and its 0.7-second timer calls `InnTick`.
The rooms' letters, names and rates are one table, `innRooms` (the same
rates as `engine.InnRoomQuality` and `InnRoomCosts`). A week heals the
room's rate, to the maximum at most, and costs a full week, while the guest
is hurt and can pay; spell points come back when any room is chosen; the
level check follows every stay (one level at most; age-weighted attribute
changes; death of old age at vitality 2, LOST). Healing draws no random
number; the level check does. Inn stays age no one (finding #11).

| Operation | Effect |
|---|---|
| `InnEnter(slot)` | Choose the guest (WHO WILL STAY); only an OK guest (`*Refusal` "NAME IS NOT OK") |
| `InnLeaveGuest()`, `LeaveInn()` | Back to WHO WILL STAY; out to the castle (RETURN steps) |
| `Stay(room, weeks)` | The room's letter, its weeks and the level check, in one call. Stricter than the keys in two cases (I-4 acceptance): a paid room for a hurt guest who cannot pay one week is refused, naming the free stables, and a guest no longer OK (LOST of old age) may not stay again; the keys keep the original (the room taken, nothing healed; a LOST guest staying on). `weeks` 0: until healed or out of gold (the original's stay); more: that many weeks of healing, then checking out early (PG p.34), refused (`*Refusal`, nothing changed) unless needed and paid for, and never in the stables (one free night). Returns `InnStay`: room, weeks, gold charged, and the guest before and after (HP, maximum, gold, spells, level, attributes, age, learned spells, the level check's words, death) |
| `Quote(room, weeks)` | Read-only: weeks, cost, HP after, healed, weeks needed and affordable, refusal |
| `InnView()` | Read-only: the step, the guests (refusals), the guest, and, for a guest who may stay, each room's rate, quote until healed and most weeks; otherwise `GuestRefusal` and no room |
| `PoolGold(slot)`, `DivvyGold()` (`app/gold.go`, I-3) | Pool every other member's gold to one (the Tavern, Boltac's, the Inn); share it evenly (the Tavern). `GoldMove`: everyone before and after; the total kept; every member takes part; a remainder one piece each in party order |

Tests: the I-0 characterization (`cmd/istary/inn_test.go`: a transcript and
the screens as goldens, every room, running out of gold, spells, levels,
death of old age, no aging, no early checkout); `Stay` against the
terminal's keys and ticks on twins (1,500 random parties, two stays each:
the same town and the same random numbers); quotes against stays; chosen
weeks against their quotes and the most weeks against the stay until
healed; refusals changing nothing; pooling and sharing on random parties
(`app/inn_test.go`, `app/gold_test.go`). `engine.RestAtInn`, unused and at
odds with the terminal (its stables restored no spells), was removed.

**The Temple of Cant** (`app/temple.go`, `app/temple_ops.go`,
`app/temple_view.go`, T-1, T-2). The terminal's Temple moved here intact
(the castle's C, WHO ARE YOU HELPING, the donation, WHO WILL TITHE, the
ritual, RETURN); the terminal reaches it through `HandleKey`. The
treatment is `engine.TempleHeal` (paralysis and stone always cured; dead
fails above 50+3×VIT into ashes, ashes above 40+3×VIT into LOST; 1 HP from
the dead, full HP from ashes; a success ages 1-52 weeks) and the donation
`engine.TempleDonation` (100, 200, 250, 500 × level), taken before the
treatment and kept when it fails. The treatment no longer saves the game
itself (the terminal's main loop saves after every key, as before).

| Operation | Effect |
|---|---|
| `TempleQuote(index, slot)` | Read-only: how the donation would be met: `payer` (alone), `party` (the payer short, the party's gold together enough: each other member's share), or a refusal (the patient, a payer not OK, or even the party short: "CHEAP APOSTATES! OUT!") |
| `Treat(index, slot, party)` | One action: the patient's name, RETURN, the payer's number and a key at the ritual, pressed for the player; returns `TempleReceipt` (conditions, donation, the payer's gold, HP and age before and after, the Temple's words). Stricter than the keys, on purpose: the patient must be at the castle (not on a saved expedition; not OUT in the maze by the Tavern's rule: INMAZE or a maze level, for anyone outside the party, whose members are INMAZE by joining), the payer must be OK, and a payer short of the donation is refused before anything happens unless `party` is set and the party's gold together covers it (PG p.33): then the entire party's gold goes into the fund (Pool Gold to the payer), the donation is taken from it, and the treatment follows, all in the one action; the receipt's `Shares` and `Party` say what each member put in. Refusals change nothing and draw nothing |
| `TempleView()` | Read-only: the patients (treatable conditions and the LOST) with kind (cure, raise, ashes), donation, refusal and funding (a payer, only the party together, short), the payers (refusal when not OK) and the party's gold. No chance of success |
| `LeaveTemple()` | Out to the castle by RETURN from any step but the ritual |
| `PoolGold(slot)` | Open at the Temple too (PG p.33), as an explicit action |

Tests: `engine/temple_test.go` (TempleHeal replayed from the numbers it
draws for every condition and vitality: outcomes, boundaries, HP, age,
consumption; the donation table), `cmd/istary/temple_test.go` (the
terminal's transcript and screens as goldens, and its rules), and
`app/temple_test.go` (Treat against the keys on 1,500 random rosters, two
treatments each, identical with the same random numbers and every
outcome met; the view; each refusal; the keys' original where Treat is
stricter; the receipts; pooling then treating).

**Boltac's Trading Post in the window (M8-3).** `gui/shop.go` is the bridge
to the controller's Boltac's operations (M8-2); the screen is M8-4 (below).
- **Going there and back:** the Castle's Boltac's exit is open for a party
  (the castle's `YOU HAVE NO PARTY!` without one). Its one exit, "← Castle"
  and Escape leave it with `LeaveShop` from any step, as RETURN does: the
  purchase prompt closes, a question about an unusable item is answered no
  (`View.Return.Note` says so), the customer leaves. Never saved, no RNG.
- **View:** `View.Shop` (nil elsewhere) is `{view, last}`: `view` is
  `app.ShopView` unchanged (step, Boltac's words, the customer, the room
  left, the catalog with price, stock, usability and Boltac's answer now,
  the customer's items with sale price, fee and each refusal); `last` is the
  receipt of the transaction the last shop action completed (action,
  customer, item, gold change, `saved`, `durable`), until the next shop
  action or visit.
- **Actions:** `ShopAction(ShopRequest{action, slot, item, n, yes})` with
  `enter` (slot), `leave-customer`, `buy` (item: a catalog offer's item),
  `confirm` (yes), `sell`, `uncurse`, `identify` (n: the item's number). Each
  is the controller's operation; out-of-turn requests (buying with nobody at
  the counter, anything but an answer while Boltac asks) are refused first in
  words. Refused away from Boltac's, during practice and expeditions, and
  during creation, with nothing changed.
- **Saving:** a completed transaction, any change to a character, is saved
  at once, in one atomic write of the roster (gold and item together).
  Choosing the customer, a refusal, Boltac's question and an answer of no
  write nothing; neither does looking. Stock is not saved (finding #12).
- **A failed save** keeps the transaction in the window (as every action):
  `Error` "Could not save: ...", `Unsaved` until a save succeeds, the
  receipt's `saved` false and a notice that says it is NOT saved; a restart
  meanwhile finds the last save. A save that could not sync the folder is
  saved but not durable (`durable` false, `Warning`). The next transaction
  saves everything. `saveGame` (session.go) is the tests' seam for a sync
  failure.
- **Tests:** `gui/shop_test.go`: each transaction and refusal (funds, room,
  cursed, identified, not cursed, no such item or offer), the unusable-item
  question both ways, stock down and up, switching customers, the receipt's
  lifetime, refusals at every other place and out of town, the failed and
  the non-durable save, restart and reload (exactly what the window had,
  once; the save's keys unchanged; stock from the scenario again), and 40
  seeds of random requests against a twin game driven by the same
  controller operations (equal after every request; a write exactly when a
  character changed; the save then equal to the game). The return from every
  step is `TestReturnFromEveryStepOfBoltacs`. None draws a random number.

**The Trading Post screen (M8-4).** `TradingScreen.svelte` in the location
frame, from `View.Shop` only; `lib/shop.js` reshapes it and decides nothing.
- **Customer:** chosen from the shared party bar (the frame passes
  PartyBar's `pick`/`onpick`/`current` through; the same 3x2 bar), marked
  there as current; with the party hidden, from a list in the actions
  column. Choosing someone else while a customer is at the counter switches
  in one request (`enter` lets the customer leave first, by RETURN; refused
  while Boltac asks). The actions column shows name, class, gold, items
  carried and room, the character sheet, and Leave the counter.
- **Tabs** Buy, Sell, Identify, Uncurse are the page's own state: switching
  calls nothing, writes nothing, draws nothing (checked live by counting
  calls into Go and the save file's time). Each line shows Go's price or
  fee, stock (unlimited or left), usability ("MAGE cannot use it; Boltac
  will ask first", `Buy…`), and marks from Go's equipped/cursed/identified
  flags. A button is enabled exactly when Go gives no refusal; the refusal
  is shown beside it in Boltac's words.
- **The counter** sticks above the list while it scrolls: the tabs, a
  customer line (gold, room), Boltac's words or his question about
  unusable equipment (Buy anyway / No), and the receipt with its outcome:
  Saved, saved but perhaps not durably, or NOT saved. The notice banner is
  not repeated while the receipt shows; the error and Unsaved banners are.
- **← Castle / Escape** leave from any tab and step; with the question open
  the frame's note says it is answered no.
- **Tests:** `TestShopViewFixture` records real views
  (`testdata/shop-views.json`: every step, every refusal, the three save
  outcomes); `shopViews.test.js` holds the helpers to them. Live
  (`wails dev` + headless Chrome) at 760x532, 980x692 and 1512x945: entering
  and switching customers by the bar and the list, tabs and scrolling with
  no calls or writes, a purchase, the question from far down the list (in
  view), Escape from it (nothing bought, nothing written), No and Buy anyway,
  an equipped sale, the cursed refusal, identify and uncurse, a failed save
  (NOT saved, file unchanged) then the next save matching the window, no
  sideways scrolling, the party bar 3x2 and in view, item names whole.
  Stills in `docs/design/m2-exploration/stills/gui-trading-m8-4/`.

**M8 completion: Boltac's question, equipping, saving again.**
- **Selling a worn item** (B7): `ShopRequest` `confirm-sale`; anything else
  waits for the answer; ← Castle notes the item is not sold. The screen
  marks worn items `Sell…` ("equipped: Boltac will ask first") and draws the
  question (the item, that it is worn and will be taken off, the price;
  Sell it / No, keep it).
- **Equipping in town** (`gui/equip.go`): `Session.EquipAction(EquipRequest
  {index, n, equip})` from the character sheet, anywhere in town; each
  `ItemView` carries `n`, `kind`, `equipRefusal` and `unequipRefusal`, and
  `CharacterView.EquipNote` why equipping is closed now (an expedition,
  practice, a character being created). `lib/equip.js` makes the Equip /
  Take off buttons from them. A change is saved at once; a refusal writes
  nothing; a stuck item's notice says so.
- **Save failures:** a failed save never undoes the change (the window has
  it, once; the file keeps the previous save whole, one atomic write);
  `Session.SaveNow` writes the window's game as it is, so retrying never
  repeats a purchase, sale, equip or step (on an expedition the place is
  recorded first). The banner says closing now would lose what is unsaved
  and offers Try saving again; Boltac's NOT saved receipt offers it too and
  becomes Saved when it works.
- **Closing the window** (`gui/close.go`, Wails's `OnBeforeClose`): with
  everything saved the close button, ⌘Q and Quit close at once. With the
  last save failed a native dialog asks: Keep playing (default, Escape),
  Try saving again (`SaveNow`; asks again if it fails, closes once it
  works) or Quit without saving. A close while the dialog is open is
  refused without a second dialog. Leave the game still refuses while
  unsaved, so the close it starts never asks. Tests: `gui/close_test.go`;
  verified live in the app (2026-11-09).
- **Tests:** `gui/equip_test.go` (with `equip-views.json` and
  `equipViews.test.js`), `TestShopSellingAWornItemAsks`,
  `TestSaveNowAfterAFailedSave`, `TestSaveNowOnAnExpedition`; the shop
  fixture gains Boltac's sell question. Live at 760x532, 980x692 and
  1512x945: the sell question (No, Escape and Sell it), a failed save
  retried, and the sheet equipping chain mail and an evil shield that
  sticks. Stills in `docs/design/m2-exploration/stills/gui-m8-completion/`.

**The Adventurer's Inn in the window (I-4).** `gui/inn.go` is the bridge;
`InnScreen.svelte` the screen, from `View.Inn` only; `lib/inn.js` reshapes
it and decides nothing.
- **View and actions:** `View.Inn` (nil elsewhere) is `{view, last}`:
  `app.InnView` unchanged and the receipt of the last stay or pooling
  (`stay` or `gold`, `saved`, `durable`). `InnAction(InnRequest{action,
  slot, room, weeks})`: `enter` (in place of any guest), `leave-guest`,
  `stay` (a room; weeks 0 until healed) and `pool` (the party's gold to the
  guest). `InnQuote(room, weeks)` quotes chosen weeks, read-only.
  `GoldAction(GoldRequest{action, slot})`: `divvy` at the Tavern, `pool`;
  its receipt is `View.Gold`, at the Tavern.
- **Saving:** a stay and its level check are one action and one atomic
  write of the roster, so a stay is never half saved and no step lies
  between payment and the level check's rolls for a closed window to undo.
  Pooling and sharing are one write each. Choosing or switching the guest,
  checking out, a refusal, quoting and looking write nothing; a stay that
  changes nothing (a night in the stables short of a level) writes nothing
  but keeps its receipt. A refusal keeps the last receipt (a death of old
  age stays in view); a change of guest clears it. A failed save keeps the stay in the window, the
  receipt NOT saved; Try saving again (`SaveNow`) marks it saved and repeats
  nothing (no healing, charge or roll); closing the window asks first.
- **The screen:** the guest from the party bar (marked there) or, with the
  party hidden or for a guest who may not stay, a list with Go's refusal;
  the guest card (class, level, HP, gold, age, spell points, experience and
  the next level's), the character sheet, Pool party gold, Check out. Two
  modes: Rest until healed (each room's quote: weeks, gold, HP after, and
  when gold falls short, the weeks it pays for of those needed) and Choose
  weeks (Go's quote per room for the weeks typed; a disabled Stay says
  Go's refusal); the stables are one free night in either. The receipt
  lists room and weeks, gold, HP, maximum HP, spell points restored, age,
  level, attribute changes, learned spells and death of old age, with the
  level check's words. A paid room the guest cannot pay a week of is
  disabled with Go's refusal, naming the free stables; a guest LOST of old
  age is shown why, in place of the rooms, and cannot be checked in again. A note says spell points come back with every stay
  and that Istary does not yet age characters for Inn stays. At the
  Tavern, Divvy gold and its receipt (everyone before and after).
- **Pool Gold at Boltac's:** `ShopRequest` `pool`: the same operation,
  through Boltac's save and receipt (`ShopReceipt.Move`); refused with
  nobody at the counter or while Boltac asks (`TestShopPoolGold`).
- **Tests:** `gui/inn_test.go` (one save per stay or pooling and none
  otherwise, a failed save retried with nothing repeated and found after a
  restart, a guest LOST of old age refused from then on with the receipt
  kept, Divvy at the Tavern only) and `TestInnViewFixture`
  (`testdata/inn-views.json`, held by `innViews.test.js`). Live (`wails dev`
  + headless Chrome) at 760x532, 980x692 and 1512x945. Stills in
  `docs/design/m2-exploration/stills/gui-inn-i4/`.

**The Temple of Cant in the window (T-3).** `gui/temple.go` is the bridge;
`TempleScreen.svelte` the screen, from `View.Temple` only; `lib/temple.js`
reshapes it and decides nothing.
- **View and actions:** `View.Temple` (nil elsewhere) is `{view, last}`:
  `app.TempleView` unchanged and the receipt of the last treatment or
  pooling (`treat` or `gold`, `saved`, `durable`).
  `TempleAction(TempleRequest{action, patient, payer})`: `treat` (a roster
  index and a party slot) and `pool` (the party's gold to the payer).
- **Saving:** a treatment, donation and roll together, is one action and
  one atomic write of the roster: never half saved, never rolled again by
  closing the window. Pooling is one write. A refusal and looking write
  nothing; a refusal keeps the last receipt. A failed save keeps the
  treatment, the receipt NOT saved; Try saving again marks it saved,
  charging and rolling nothing again; closing the window asks first.
- **The party's gold:** when the payer is short, the confirmation shows
  Go's quote (`Session.TempleQuote`): the entire party's gold into the
  fund, each member's share, the fund and the donation, what the payer
  keeps; confirming sends `party: true`. The receipt itemises the
  contributions. A failed save retried moves no gold and rolls nothing
  again (`TestTempleWithThePartysGold`).
- **The screen:** the patients (name, class and level, condition,
  donation, whether the gold is there, Go's reason when not treatable; the
  LOST as beyond mortal help), the payer from the party bar or a list
  (only those Go allows; the choice stays), Pool party gold. Treat…
  opens the screen's own confirmation (patient, condition, donation,
  payer and gold, outcomes, the risk in words: assured; uncertain, ashes
  on failure; the final chance, LOST on failure, with a permanent-loss
  warning; the donation charged either way). It calls nothing: Cancel and
  Escape change nothing. The receipt: well again, now ashes, or lost
  forever; conditions, donation and the payer's gold, hit points, age,
  saved or not.
- **Tests:** `gui/temple_test.go` (one save per treatment or pooling, none
  for refusals; a failed save retried, nothing charged or rolled twice,
  found after a restart) and `TestTempleViewFixture`
  (`testdata/temple-views.json`, every outcome), held by
  `templeViews.test.js`. Live at 760x532, 980x692 and 1512x945. Stills in
  `docs/design/m2-exploration/stills/gui-temple-t3/`.

**The location frame (Castle-0 C0-3).** Every town place is drawn in one
frame (`LocationFrame.svelte`), in grid areas that never overlap:
- **head:** the place's title, "← Castle" (`ReturnToCastle`, also
  Escape; since C0-6) and the Show/Hide party toggle;
- **scene:** (a definite height, about 16:9 of its column between 120px
  and 46vh, since a height from aspect-ratio was left out of the row in a
  scrolling body) the place's artwork (`LocationScene.svelte`, `lib/scenes.js`):
  a picture cropped around its focus, or, while there is none (all places
  for now) or it fails to load, a themed panel with the place's emblem;
- **nav:** the other destinations from `View.Exits` (closed ones with Go's
  reason written out), then the place's actions;
- **content:** anything wider, below both;
- **party:** `PartyBar.svelte`, from `View.PartyBar` only (the practice
  copies during practice): three across, the front row (1-3) above the
  back row (4-6), empty positions included, two lines a card (name, class
  and row; level, HP, AC and condition) with the HP as a thin bar (C0-6).
  It stays in view at the bottom; a member of the
  saved roster opens the existing character sheet, which offers Rename and
  Delete at the Training Grounds and Join/Leave at the Tavern only.
- **Sizes:** the artwork beside the destinations from 700px wide (16:9, at
  most 46% of the height, 36% in short windows); stacked below 700px with
  the artwork as a 21:9 banner. Checked live at 760x532 (the minimum
  window), 980x692 and full screen (1512x945, 1920x1080): no overlap, no
  sideways scrolling, every destination and the party bar in view.
- **Later:** a graphical maze can draw into the scene area (the `scene`
  snippet) and an automap into an `aside` beside the nav, without changing
  how places are reached. The maze, Camp and combat screens are not in the
  frame yet.
- **Settings file:** `Session.SetPartyVisible` keeps the toggle in
  `gui-settings.json` in the save folder (`gui/settings.go`), written with
  `engine.WriteFileAtomic`, never in the roster file, a checkpoint or a
  backup. It is accepted anywhere and changes nothing of the game. A
  missing file is the defaults; a damaged or newer one is left as it is,
  the defaults are used with a note (`View.Settings.Note`), and the next
  change replaces it.
- **The town screens (C0-4):** `CastleScreen`, `EdgeScreen`,
  `TrainingScreen`, `TavernScreen` and `SettingsScreen`, each in the
  frame; `App.svelte` only routes by `View.Location` and keeps the
  character sheet. In town the page is one window-high shell: the head and
  the party bar stay in view and the body between them scrolls, so no
  control is ever drawn over another. The old all-in-one screen and its
  `PartyPanel` are gone; the party bar is the only party display.
  - **Castle:** destinations only.
  - **Edge of Town:** Enter the maze, practice (Istary only) and Leave the
    game as actions, each closed with Go's reason (`View.Departures`, from
    the same `realRefusal`/`practiceRefusal` the actions use);
    `LeaveGame` closes the window (Wails's Quit), refused elsewhere, while
    out or creating, and while the last save has failed.
  - **Training Grounds:** the roster and the Create form; the creation
    steps unchanged.
  - **Tavern:** Join, Remove, the Tavern's refusals and the password.
  - **Settings:** the save folder, Reload, Reset and Restore.
  - **The recovery banner's "Try resuming again"** is `TryResumeAgain`,
    accepted anywhere in town while an expedition cannot be resumed.
  - **Checked live** (Wails dev, headless Chrome) at 760x532, 980x692 and
    1512x945: the Castle, the Training Grounds and a full character
    creation, roster inspection, Tavern party assembly, the Edge of Town,
    an expedition out and back by the stairs, Settings and Leave the game;
    on every screen every control could be brought into view and was not
    covered, and the page itself never scrolled.

**The party bar on every screen (Castle-0 C0-5).** The maze and Camp show
the same `PartyBar` at the bottom of the dungeon screen, in place of their
own party strip and cards; combat keeps its own formation display and
shows no bar. `View.PartyBar` carries AC as well.
- **Camp picks through the bar:** a click on a member does what Camp asks
  (`pickLabel`, from `partyPick`), the member Camp is working with is
  marked, and the new places while reordering (`placeMark`); the keys 1-6
  still pick, and with the party hidden Camp says so.
- **Combat shows no bar** on purpose: it shows the party as of the event
  being played back, while the bar already has the outcome
  (`TestPartyBarFollowsPlay` compares the formation only there).
- **Layout:** the dungeon screen fills the window like the town: header and
  party bar in view, the maze or Camp between them fitting or scrolling;
  the corridor picture shrinks to fit. Hide party is in its header and
  applies everywhere; the choice survives restarts.
- **Window title:** "Istary".
- **Tests:** `TestPartyBarFollowsPlay` holds the bar to each screen's party
  through random Camp play (reorders and trades), exploring with
  encounters, and the return to the castle; `campViews.test.js` checks the
  bar's Camp picks against the cards' former ones.
- **Checked live** at 760x532, 980x692 and 1512x945: the castle, Camp
  (inspect and reorder by the bar, hidden party and the keys), the maze,
  the app killed during the expedition with the party hidden and started
  again (resumed exactly, still hidden), Camp and the stairs back to the
  castle, and a practice encounter without a bar; the bar always matched
  Go's party and stayed fully in view, and no control was out of reach or
  covered.
- **Tests:** `gui/settings_test.go` (kept across restarts, accepted out on
  an expedition, damaged and unwritable files, roster management leaves it
  alone); `frameViews.test.js` over `frame-views.json` recorded from Go.

**Polish (Castle-0 C0-6).**
- **"← Castle" and Escape** (`Session.ReturnToCastle`, `View.Return`):
  from any town place, back to the Castle, abandoning what is unfinished
  there by the game's own steps: a character being created is discarded
  (`app.Controller.AbandonCreation`, at any step, as Cancel and Discard
  do), a join waiting for its password or another open prompt is
  cancelled, Settings is left. Nothing already done is undone; the note
  under the button says what returning abandons. At the Castle nothing
  happens; during an expedition or practice it is refused and changes
  nothing; with an unresumable expedition it works and leaves the
  expedition alone. It writes nothing and draws no random number. An open
  character sheet takes Escape first; the maze, Camp and combat keep their
  own keys.
- **Shaded controls** (tokens in `style.css`): buttons are shaded, not
  outlined: lighter on hover, darker when pressed, olive for the main
  action, muted red for Reset, Delete and Restore, flat and dim when
  disabled, a gold outline for keyboard focus, gold shading for what is
  selected (party members, roster rows).
- **The compact party bar** (above).
- **Tests:** `TestAbandonCreationAtEveryStep`; `TestReturnToCastle` (every
  place, creation at several steps, a pending join, the earlier join
  kept, the Castle unchanged), `...RefusedOutOfTown`,
  `...LeavesRecoveryAlone`. Checked live at 760x532 and 980x692: Escape at
  the Castle, from every place, while creating, with a join pending, with
  the sheet open, from Settings' reset confirmation, in Camp and in combat;
  the bar three across with no name cut short; selecting and reordering by
  the bar; the shades, hover and keyboard focus.

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
  (`lib/dungeon.js`), the messages, the actions with the terminal's keys,
  and the shared party bar (C0-5). It is a placeholder, not the Classic
  renderer.
- **Not yet:** rescue and recovery (M6b-4). Camp is in the GUI since M6b-3B
  (below), except Disband and MALOR.

**The maps (M-1).** An explored-only minimap beside the corridor and a
full map of a level (F10), drawn from one record in Go.
- **Coordinates:** `engine/place.go` is the only conversion between the
  engine's own (0-based level; X east, Y north from the south-west stairs)
  and `L:R:C` (L0 the castle, L1 the first level; R0:C0 the north-west
  corner, rows south, columns east): `L = level+1`, `R = 19-Y`, `C = X`.
  The entrance is `L1:R19:C0` facing north, checked against the maze data of
  every scenario (`MazeEntrance`). A position off the maze (#31) has no
  `L:R:C` and is never clamped.
- **What is recorded** (`engine/explore.go`, `GameState.Explore`): the
  square the party stands on (visited, and its stairs or elevator) and, in
  the maze, every wall of `Sight()`, each from the side it was seen. The
  two sides of a boundary are kept apart because the maze has one-way walls
  and doors; nothing is read from the maze that the party has not seen; a
  hidden door is a wall until seen with light; darkness shows nothing;
  teleporters, chutes and stairs show only where they lead.
- **When:** the GUI calls `app.Controller.Explore` after every dungeon,
  Camp and combat action (`combatDo`) and after resuming; the terminal never
  does. A practice expedition starts from a copy of the map and discards it.
- **Views** (`app/map_view.go`): `LocalMap()`, the 7x7 squares around the
  party, north up, continuing across the level's edge with `NorthEdge` and
  `WestEdge` marking where it runs; `Map(level)`, a whole level in its fixed
  20x20 frame, with `Wraps` where a seen way crosses the edge. Both decide
  each boundary alike: seen from one side, or alike from both, one wall;
  seen differently, each side's own wall, `OneWay`, drawn inside its square.
  Both list the `Visited` squares (the record's own, nothing more), which
  the maps shade faintly. Reading either changes nothing and draws no
  random number.
- **Stranded markers (R-3, `design/r3/RESCUE-VISIBILITY.md`):** both
  views list `Stranded`, one `MapStranded` per square where characters lie
  (place, count, each name and condition). They come from
  `engine.GameState.StrandedSites`, which reads the characters' `Stranded`
  records and the same exclusions as Inspect, never the exploration. A
  marker reveals no geometry and marks nothing explored. The LOST and
  legacy characters with no `Stranded` record are never marked. `Map`
  also offers a level where characters lie but nothing was explored, with
  its markers alone. Nothing is saved for them.
- **Coordinates (UX-1, `design/ux1/UX-1.md`):** `Controller.Position`
  is the party's L:R:C and facing, in the maze, at Camp and in combat, and
  nothing elsewhere; `Stranded` is its square's marker count. `View.Position`
  carries it to the dungeon screen's header and its "N stranded adventurers
  here" line under the view. Inspect's button is Inspect / Rescue.
- **Bridge:** `View.Minimap` on an expedition; `Session.Automap(level)`
  (read-only); `SetMinimapVisible` (a kept setting, on by default).
- **Screen:** `MapSvg.svelte` draws both (a faint floor on visited squares,
  green walls, yellow doors, stairs and elevator symbols, the white arrow; a
  symbol under the party moves to a small badge in the square's corner, and
  the minimap draws a little past its window so doors on its outer walls
  are whole); `AutomapPopup.svelte` is the full
  map in the same window. Stranded markers are red diamonds with their
  count, drawn under the arrow (a corner badge on the party's own square).
  The full map lists the level's markers beside it, and a click on a
  marker, or Enter or Space on its line, shows who lies there. F10 or Escape closes it, and no key reaches the
  maze while it is open (`lib/automap.js` `mapKey`).
- **Tests:** `engine/place_test.go` (all 4,000 squares both ways, the
  entrance, wrapping), `engine/explore_test.go`, `app/map_view_test.go`
  (minimap and full map agree over seeded walks of every level; teleports
  reveal only the destination), `gui/automap_test.go`, and
  `lib/automap.test.js` over `testdata/map-views.json` (recorded by
  `gui/map_fixture_test.go`).

**Real expeditions (M6b-1).** `RealExpedition()` takes the party itself,
in the saved game, through the same dungeon, combat and chest operations.
- **Saving:** every action is saved (atomically), whatever happened, so
  injuries, spells spent, experience, gold and items are kept. A failed
  save is in `View.Unsaved` until one succeeds; the action is never
  repeated, the next successful save writes it.
- **The end:** the stairs, or a party that can no longer go on (in combat,
  or from a pit, chute, OUCH, SOLID ROCK or poison). That party stays
  stranded on its square (R-1, `engine.StrandParty`,
  `design/r1/STRANDED-PARTIES.md`); the player is in the castle with no
  party, saved in the same action. It cannot be abandoned.
- **Stranded characters** stay in the roster with their square
  (`Character.Stranded`, L:R:C, shown as "stranded ... at L1:R5:C3"). They
  cannot join a party or be treated, renamed, deleted or equipped in town.
- **Rescue (R-2, `design/r2/RESCUE.md`):** Inspect finds the stranded
  anywhere in the party's area (the squares it could walk to through open
  walls), and P)ICK UP takes them one at a time while the party has room
  (six, the incapacitated included). A pickup is one action, saved with the
  expedition. The rescued travel with the party as they are; home, they are
  in the castle party, stranded nowhere. R-3
  (`design/r3/RESCUE-VISIBILITY.md`) marks where the stranded lie on the
  maps and shows the rescue's progress on Inspect: the party's size and
  room, who else is here, how many lie elsewhere, and the members who
  cannot act, each with a condition label (`app.ConditionLabel`).
- **A restart during an expedition** resumes it (M6b-2): every save
  carries the expedition's checkpoint (`app.Controller.Checkpoint`, the
  last safe point), and on start `Resume` puts it back exactly. An
  encounter that was under way is over, nothing more awarded; changes it
  had made outside the characters (a teleporter, an alarm) are kept. A
  saved party that cannot go on (the app closed on the defeat screen) is
  stranded at the checkpoint before anything else (R-1, D-R). A
  resumed expedition's notice is headed EXPEDITION RESUMED (`View.Title`)
  until the next action; a new one has no heading. A
  saved expedition that cannot be resumed is kept and shown
  (`View.Recovery`), with `ReturnExpeditionToCastle`.
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
terminal directory. Since UX-3A that folder is a library of six game slots
(`gui/games.go`, `design/ux3/UX-3A.md`): each `games/slot-N/` holds one
game's `roster1.json` in this format, its `backups/` and a `game.json`
record; the earlier single `roster1.json` beside them is offered for import
and never written. The legacy `~/.config/wizardry/wiz<N>/roster.json` is still
read. The format is pinned by `engine/testdata/save_roster1.golden.json`.

The file holds the roster, the castle party's names and, while a GUI
expedition is out, an optional `expedition` section
(`engine/expedition.go`; schema in `design/m6b/EXPEDITION-PERSISTENCE.md`
§13): the members by name, the checkpoint and the changed special squares.
- **No expedition:** the section is absent, and the file is byte-identical
  to before.
- **While one is out:** its members are listed only in the section, never in
  the castle party.
- **Other versions:** a section of another version is kept and written back
  unchanged; the terminal does the same with a GUI expedition.
- **Members' identity:** while one is saved, the controller refuses to
  rename or delete its members.

An optional `exploration` section (M-1, `engine/explore.go`) holds what the
player has seen, for the GUI's maps: `version`, `dungeon` (the scenario,
"1"), and per level `L` twenty rows from R0 of `visited` ('v' or '.') and
`walls` (four characters a square, N E S W: 'o' open, 'w' wall, 'd' door,
'.' unknown), with `features` by row and column. It is absent when nothing
has been recorded, so a terminal save, or a GUI save before any
exploration, is unchanged; an old save loads with an empty map. A section
of another version or scenario, or one that cannot be read, is kept and
written back unchanged, and nothing is recorded over it.

**One window per folder (M6b-2).** The GUI holds an advisory `flock` on
`istary.lock` in its save folder (`gui/lock.go`), the library folder since
UX-3A.
- **A second window:** refuses to start and changes nothing.
- **Release:** the kernel releases the lock when the process ends, a crash
  included, so it can't go stale.
- **When the folder is opened:** in `OnStartup`, not `main`, because Wails
  runs the program to generate bindings.

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
- The terminal ignores save errors as before. The GUI reports them and tells
  a failed save (`View.Unsaved`: nothing replaced) from one that replaced
  the file but could not sync the folder (`engine.NotDurableError`,
  `View.Warning`). A failed save does not undo what was done; the next
  change, or `Session.SaveNow`, writes it (M8).
- Every load works out each character's AC from what is worn (#35): an old
  save's stale AC is replaced, and the next save writes the worn one.
