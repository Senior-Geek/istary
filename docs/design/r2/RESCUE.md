# R-2: rescue expeditions and selective recovery

Status: **implemented on `r2-rescue-expeditions`, awaiting acceptance.**
R-1 (`../r1/STRANDED-PARTIES.md`, released in `v0.6.0-stranded`) leaves a
party that cannot go on stranded on its square. R-2 lets another party
find those characters, take them one at a time within its six places,
carry them as it explores, and bring them home. Rescue markers and other
presentation are R-3.

## 1. Sources and how sure they are

Three sources are cited, and they are kept apart below.

- **The Player's Guide** (`docs/reference/`, 1987 printing). Page 42: "you
  must be close enough to them so that you could walk to where they are
  without going through a door"; "There must be room in your party to hold
  the new members". Page 39: up to six characters in a party. Page 48: the
  dead "will be hauled around by the living adventurers".
- **Pascal (reconstructed).** Thomas W. Ewers' re-engineered Pascal of
  Wizardry I, as kept by Eric Labelle's Wizardry Proving Grounds v3.2
  project (github.com/snafaru/Wizardry.Code, commit `10184e9`; "Wizardry PG
  Coding Bugs and Issues v3.2", zimlab.com). It is rebuilt from the p-code,
  not Sir-Tech's own source, and carries the project's `WCnnn` fixes. The
  procedures below carry none that touch rescue (`SPECIALS.TEXT` lists only
  WC036, a ProDOS quit). This is the strongest evidence available, but it
  is a reconstruction.
- **Istary's choices.** Where Istary differs from the reconstruction, the
  approved R-2 decision (Q1-Q8, `WIZARDRY-COMPATIBILITY.md` §6) says so.

| Rule | Reconstructed original | Istary R-2 |
|---|---|---|
| Where the stranded are found | EXPLROOM fills the area from the party's square through OPEN walls (each square's own wall), X and Y modulo 20, on the party's level; LOOKLOST keeps those whose location is on that level and in the area | The same (`engine.RescueArea`, `GameState.FindStranded`) |
| What stops the search | Any wall that is not OPEN: walls, doors, hidden doors. Special squares are only squares | The same |
| How many listed | At most five (FOUNDLOS stops at 5) | Everyone found (Q5) |
| Picking | `GET WHO (0=EXIT) >`, one character a key, repeated; picked lines erased | The same in the terminal (Q6); one button or digit each in the GUI |
| Room | `YOU HAVE 6 - PRESS [RET]` at six | `YOU HAVE 6` (terminal), `YOUR PARTY HAS 6; NAME STAYS HERE.` (both) |
| Who can be found | Anyone with a location on the level, not in a party; LOST never | Stranded or legacy-located characters, not in the party, not on a saved expedition, not LOST |
| What a pickup changes | Location cleared, INMAZE set, record written at once | `Stranded` and the legacy fields cleared, `InMaze` kept; saved in the same action |
| Condition | Unchanged | Unchanged (no healing, no raising) |
| Castle return | CHK4WIN clears locations; members not OK leave the party | `PartyHome`: `InMaze` and `Stranded` cleared; the incapacitated stay in the castle party (Q8) |
| Wipe-out | All non-LOST made DEAD, gold halved, items may break on Luck, bodies lost for good with chance level/50 (CEMETARY BADSTUFF) | None of these (Q4; R-1 behaviour). Documented for a future Classic mode |
| SOLID ROCK square | ROCKWATE: back to level 1 at the same X and Y, in Camp, unharmed (MAZELEV -99, then NEWMAZE) | Lethal, as R-1 left it (Q3); see §6 |

## 2. The area rule

`engine.RescueArea(sc, level, x, y)` is the original's EXPLROOM as a
breadth-first search: from the party's square it steps east, west, south
and north wherever that square's own wall on that side is `open`, wrapping
at the edges, and never leaves the level.

- **One-way passages** count only the way they can be walked: the square
  being left decides, as for a step forward.
- **Doors and hidden doors** stop it, whichever side.
- **Stairs, chutes, pits, teleporters** and other special squares are
  ordinary squares to it: they neither stop it nor carry it anywhere.
- **It reveals nothing.** The player's map (`Exploration`) is untouched;
  the GUI shows no positions for the characters found.

`GameState.FindStranded` lists, in roster order:

- characters with a `Stranded` square on the party's level and in its area;
- characters an older version left out with no `Stranded` record, on the
  **exact** square their legacy fields name, when those fields name a
  square of the maze (Q2). Fields naming no square of the maze match
  nothing. Nothing is invented. Characters disbanded by Istary's terminal
  Camp D)ISBAND have zero fields, so they are still found only at the
  entrance (#26, deferred).

It never lists party members, LOST characters, characters not out of the
castle, or characters named by a saved expedition (this version's or
another's).

## 3. Capacity and picking

The party holds `engine.MaxPartySize` (6), the incapacitated included.
`Party.Size()` counts members, not the empty entries Camp D)ISBAND leaves
(#26).

`GameState.RecoverStranded(c)` checks everything before it changes
anything:

1. `c` is found from where the party stands (`FindStranded`), so not in
   the party already, not LOST, still stranded;
2. the party has room.

Then:

- `c` joins the end of the party (empty entries are closed up first);
- `Stranded` and the legacy fields are cleared, and `InMaze` stays set;
- nothing else changes: condition, HP, items, gold, experience, age and
  spells.

`app.Controller.PickUp(n)` takes the n-th character found (from 1); the
terminal's `GET WHO` prompt calls the same `pickUp`, so both front ends
apply one rule.

| Situation | Result | Changes |
|---|---|---|
| Room, `n` on the list | `NAME JOINS THE PARTY.`; the character leaves the list; Inspect stays open | Party, `Stranded` |
| Party of six | Refused: `YOUR PARTY HAS 6; NAME STAYS HERE.` (terminal P: `YOU HAVE 6`) | Nothing |
| `n` not on the list | `ErrInvalidChoice` (the terminal ignores the key, as the original waits) | Nothing |
| The same character again | Not on the list any more; `RecoverStranded` says `ErrNotFoundHere` | Nothing |
| Leave | Whoever was not picked stays stranded where they were | Nothing |

A full party can still Inspect and see everyone, each with the reason.

## 4. The states

| Step | Changes | Unchanged |
|---|---|---|
| **Stranded** (R-1 `StrandParty`) | `Stranded` = the square, `InMaze` set, out of the party | Everything else |
| **Picked up** (`RecoverStranded`) | `Stranded` nil, legacy fields cleared, `InMaze` set, in the party | Condition, HP, possessions, gold, experience, age, spells |
| **Carried** | Travels as a member: takes slot, takes square effects like any member (a pit hurts a carried PARALYZED character, as before) | Cannot act while incapacitated |
| **Rescuers fall** | R-1 `StrandParty` strands everyone in the party, the rescued too, at the **new** square | Each character has one record and one square |
| **Home** (`GameState.PartyHome`: the stairs up, a square that sends the party home, Camp MALOR to the castle) | `InMaze` false, `Stranded` nil | Condition and possessions; nobody healed or raised; the party stays together (Q8) |

At the castle the rescued are ordinary castle characters: the Temple treats
them (they are in the castle party), and they can be removed at the Tavern.
**A party in which no one can act cannot set out** (`ErrNoOneCanGoOn`, and
the GUI's departures say so); the incapacitated may go along with someone
who can, as in the original.

## 5. Saves and restarts

- **One pickup, one save.** In the GUI the pickup, the checkpoint (the
  expedition's party now names the rescued) and the save are one atomic
  write. Roster and expedition can never disagree: a character is either
  stranded and not on the expedition, or on it and not stranded.
- **Partial recovery** is just the state after each pickup.
- **Restarting while carrying** resumes the expedition with the rescued in
  the party; the Inspect screen, if it was open, comes back with its list.
- **Saved Inspect lists are checked again** on resume: only characters the
  restored party still finds are kept, once each, in their saved order. A
  name no longer on the roster, someone picked up since, or a duplicate is
  dropped, not a problem.
- **A contradictory save** (an expedition member still marked stranded) is
  refused by `Resume` with an `ExpeditionProblem`, changing nothing; the
  GUI shows it as any expedition that cannot be resumed.
- **A failed save** leaves the window `Unsaved` and the file at the last
  safe point. A restart before a retry finds the character still stranded
  and still listed; **SaveNow** writes the pickup once.
- **Format:** unchanged. No new field: the expedition's party list and the
  R-1 `stranded` field already say everything. Old saves load as before;
  another version's expedition is kept and written back untouched, and its
  members are never found by Inspect.
- **Practice** (Q7) copies the whole roster, so a practice rescue finds and
  takes copies; the copies are discarded with the practice game, and the
  roster, the castle party and the save are never touched.

## 6. SOLID ROCK

R-1's design record said the original "sets MAZELEV to -99 there, so its
bodies could never be found". **That was wrong.** In the reconstructed
Pascal, SPECSQAR ROCKWATE sets MAZELEV to -99 and goes to NEWMAZE, which
turns any level below 0 into level 1 and goes to XEQUIP6, the Camp the
Edge of Town's M)AZE enters (ENTMAZE sets MAZELEV to -1 the same way). X
and Y are not changed. So the original's square puts the party back on the
first level, at the same X and Y, in Camp, unharmed. Only Camp MALOR into
rock (UTILITIE ROCK) makes a party LOST.

Istary's square still kills (decision Q3), and the comment in
`app/maze.go` now says what the original does. Bodies there are found, and
picked up, from any square joined to it by open walls: the rescuers never
have to step on it (`TestSolidRockBodiesAreRecoveredFromNextDoor`). No
scenario of Wizardry I-III contains the square, so it is reached only in
tests. The difference is finding #36, for future review.

## 7. Interfaces

**GUI.** The dungeon view's `inspect` (`app.InspectView`) lists each
character found: number, name, alignment-class, level, condition, whether
they can be picked up and, if not, why; and the room left. The Inspect
panel shows one line each with a **Pick up** button (keys 1-9) or the
reason, and the pick's message in the message area. It shows no positions,
as the maze view never does. `DungeonAction("pick-up", n)` picks the n-th.

**Terminal.** Inspect lists up to 19 (a roster holds 20, one is the
rescuer), numbered, with level, alignment-class and condition, then `ROOM
IN THE PARTY FOR n`, the last message and the options. P opens `GET WHO
(0=EXIT) >`: a digit takes that character at once when nine or fewer are
listed; with more, the number is typed and [RET] takes it (Backspace
edits). 0 or [ESC] closes the prompt. It closes by itself when the party is
full or nobody is left. `testdata/screens_rescue.golden` pins the screen.

## 8. Tests

| Brief scenarios | Tests |
|---|---|
| 1-8, 10 geometry | `engine/rescue_test.go` (`TestRescueArea*`, synthetic levels and every Wizardry I level); `app/rescue_test.go` `TestInspectFindsTheStrandedInTheArea` |
| 9 SOLID ROCK | `TestSolidRockBodiesAreRecoveredFromNextDoor` |
| 11-13, 19, 20 capacity | `TestRescueCapacity` (1, 3 and 6 rescuers) |
| 14 empty entries | `TestEmptyPartyEntriesTakeNoRoom` |
| 15-18 selection | `TestPickUpOneAtATime`, `TestTerminalGetWho` |
| 21-28 state | `TestRescuedTravelAsTheyAre` |
| 29 rescuers fall | `TestRescuePartyDefeatStrandsEveryoneAnew`, `gui` `TestR2RescuersDefeatedWhileCarrying` |
| 30-34 castle | `TestTheRescuedComeHome` (stairs, the square that sends the party home, Camp MALOR), `TestTheRescuedAtTheCastle` |
| 35, 36, 39, 40 restart | `TestPartialRescueSurvivesARestart`; `gui` `TestRescueThroughTheBridge`, `TestRestartWhileCarryingTheRescued` |
| 37, 38 failed save | `gui` `TestRescueWithAFailedSave` |
| 41, 42 legacy and other versions | `TestLegacyOutCharactersAndOtherExpeditions` |
| 43 LOST | `TestLostAreNeverFound` |
| 44 practice | `gui` `TestPracticeRescueKeepsNothing` |
| Walkthrough | `gui` `TestR2Walkthrough` |

## 9. Limitations and R-3

- **Camp D)ISBAND (#26)** still records no square; its characters are found
  only at the entrance, and the GUI still refuses Disband.
- **SOLID ROCK** differs from the original (#36); no scenario uses it.
- **The original's wipe-out penalties** are not modelled (#37).
- **Camp MALOR (#25)** is unchanged and still refused in the GUI; its
  castle landing now uses `PartyHome`.
- **A pit or other square hurts carried members** who are not dead, as for
  any member; that is the existing square rule.
- **R-3** needs: where stranded parties lie, for markers (`Character.Stranded`
  is the record; the maze view deliberately gives no positions, so markers
  need their own decision on what the player may see); any presentation of
  the Inspect screen beyond the plain list.
