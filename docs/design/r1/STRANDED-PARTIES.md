# R-1: dungeon defeat and stranded parties

Status: **implemented on `r1-stranded-parties`, awaiting acceptance.**
Decision D-D (2026-11-07) said a defeated party stays in the maze for a
rescue. R-1 makes that true and keeps it true across saves and restarts.
Rescue itself (P)ICK UP, R-2) and its presentation (markers, R-3) come
later.

## 1. What happened before R-1

| Situation | Before |
|---|---|
| Whole party unable to act in combat | `EndRound` decided defeat; `LeaveCombat` put the game in the castle, the dead party still a castle party, nowhere recorded (finding #29) |
| SOLID ROCK square | Killed everyone and sent the party straight to the castle (`checkSquare`) |
| Pit, chute, OUCH square, poison killing everyone | The dead party walked on, could take the stairs home or meet encounters (finding #28) |
| One member DEAD, ASHED, PARALYZED or STONED | The fight and the walk go on (unchanged) |
| LOKTOFEIT or MALOR in combat | Set the combat to "defeat" for a moment; the round's end set it back and the fight went on (#24). Never a defeat (unchanged) |
| R)UN, the stairs home | Back to the maze, or to the castle with `InMaze` cleared (unchanged) |
| Camp MALOR into rock, the moat or a volcano | The party LOST for ever, in the castle (unchanged, #25) |
| Restart on the defeat screen | The checkpoint said "encounter interrupted"; the dead party was put back in the maze and could walk |
| `MazeLevel`, `MazeX`, `MazeY` | Never written; Tavern, Temple, the Rite and the roster view read `MazeLevel != 0` as "out", though 0 is also the first floor |

## 2. The rule for going on

`engine.PartyCanContinue` (`engine/stranded.go`) is the one rule: a party
can go on while someone in it can act (`Character.IsAlive`: OK, ASLEEP or
AFRAID). PARALYZED, STONED, DEAD, ASHED and LOST members cannot. Combat
(`allPartyDead`, `EndRound`), the maze step and `Resume` all use it. No
condition is changed to make it true: a party of the paralyzed and the
stoned is stranded paralyzed and stoned.

## 3. The stranded state

```go
type Character struct {
    // ...
    InMaze   bool   `json:"in_maze,omitempty"`
    Stranded *Place `json:"stranded,omitempty"` // L:R:C, nil when not stranded
}
```

Invariants:

- **`Stranded` is the only record** of where a stranded character is. It is
  set only by `GameState.StrandParty`, from `GameState.Place()` (the M-1
  conversion), so always a square of a maze level, never L0.
- **A stranded character keeps `InMaze` set.** Note: `InMaze` does not by
  itself mean "outside the castle". As in the original's Tavern it is also
  set for every member of a castle party; it means "in a party, or out".
- **A stranded character is in no party and on no saved expedition.**
  `StrandParty` empties the party, and the GUI drops the expedition section
  in the same save.
- **Condition, items, gold, experience, age and spells are untouched.**
- **The legacy fields are never written.** `MazeLevel`, `MazeX` and
  `MazeY` (the original's LOSTXYL) are read only for characters an older
  version left out with no `Stranded` record (Camp D)ISBAND, #26), and only
  as before.
- **`Character.OutOfCastle()`** is the castle's one test for `** OUT **`:
  `Stranded != nil || InMaze || MazeLevel != 0`. A character stranded on
  L1 has `MazeLevel` 0 and is still out.
- **LOST members are not stranded.** A LOST member of a defeated party
  leaves the party and the maze (`InMaze` cleared), as Camp MALOR's LOST
  does. LOST is for ever, and there is nothing to rescue.

Several characters can share a square. Every defeat records its own
square.

## 4. The transition

`StrandParty` refuses a party that can still go on, and a party that is
not on a square of the maze (a floor past the last, #31). In both cases it
changes nothing. Otherwise it:

1. leaves every member on the party's square, conditions as they are;
2. empties the party and ends combat and every maze prompt;
3. puts the player in the castle with "YOUR PARTY HAS PERISHED IN THE
   MAZE..." and "THEY REMAIN AT L1:R5:C3."

With no party left, a second call changes nothing.

It is called from:

| Path | Where |
|---|---|
| Combat defeat | `LeaveCombat` after `EndRound` decided defeat (a defeat with someone able to act is refused) |
| Victory after which no one can act (a chest's trap) | `LeaveCombat` |
| A step after which no one can act: pit, chute (on the square it drops to), OUCH, SOLID ROCK, poison | `mazeKey`, after the square's effect and the healing and poison roll, before the encounter check |
| A saved expedition whose party cannot go on | `Resume` (decision D-R), before anything else |

SOLID ROCK no longer sends the party to the castle. It kills everyone still
able to act, as before, and the step then strands the party on that square.

Practice expeditions strand only their copies, which are discarded.

## 5. Saves and interrupted transitions

- **Format:** one optional field, `stranded` (`{"level","row","col"}`).
  Absent, it means not stranded. Saves without stranded characters are
  byte-identical to before (the save goldens are unchanged).
- **Old saves:** no square is ever invented. A legacy out character stays
  OUT with no square, and `Inspect` finds them by the legacy fields as
  before.
- **Unknown expedition versions:** unchanged. The section is kept and
  written back untouched beside stranded characters.
- **One action, one save:** in the GUI, the fatal action, the transition
  and the save of the stranded party with no expedition section are one
  atomic write (`combatDo`).

| When the app stops | On restart |
|---|---|
| Before the deciding round is committed | The last checkpoint: the encounter is over, the party walks on (unchanged) |
| After the defeat is decided, before `LeaveCombat` (the checkpoint says "combat interrupted", the roster holds the losses) | `Resume` sees no one can go on and strands the party at the checkpoint's square, then saves. The checkpoint has no random state, so a second restart from the same file gives the same result |
| During the transition | It happens in memory, inside one action: there is nothing in between to stop at |
| After the stranded state is saved | Loaded as saved: no expedition, the characters stranded |
| A failed save of the transition | The window shows `Unsaved`, and the file keeps the last checkpoint. A restart before a retry resumes from that file, one step back with the party alive, as if the step had not happened. **SaveNow** writes the stranded state once; it repeats nothing, because the transition already happened in memory |

## 6. The castle

| Operation | Stranded character |
|---|---|
| Tavern join | `** OUT **` (`OutOfCastle`) |
| Temple (GUI and terminal) | Refused: "is OUT in the maze, not at the castle" (GUI), "*NAME* IS OUT" (terminal); nothing paid, nothing changed |
| Rename, delete (Training Grounds) | `** OUT IN THE MAZE **` (`ErrStranded`) |
| Town equipping | `ErrStranded` |
| Rite of Passage | Refused (`OutOfCastle`) |
| Inn, shops, gold pooling | Party members only, so not reachable |
| Character sheet, Training Grounds, Tavern | Show "stranded in the maze at L1:R5:C3" (`CharacterView.Stranded`) |
| Reset roster (settings) | Allowed, as for everyone. It backs up the whole roster first, and restoring that backup brings them back |

## 7. Inspect, Pick Up and R-2

- **Inspect** finds a stranded character when `Stranded` is the party's
  square (L:R:C, so it is exact at the edges and after one-way walls).
  Characters with no record are found by the legacy fields, as before.
- **P)ICK UP is refused** when Inspect found a stranded character
  (`ErrPickUpStranded`, and the terminal key does nothing). Today's P)ICK
  UP frees everyone found, even those it has no room for (#27), and never
  clears a square. It would bring stranded characters back to the castle
  without a rescue, or leave them in the party still marked stranded.

R-2 needs:

1. a P)ICK UP that takes only those who fit and leaves the rest stranded
   (#27);
2. clearing `Stranded` for those taken, and keeping `InMaze` set while they
   are in the party;
3. checkpoints and `Resume` for a party carrying the rescued;
4. what reaching the castle does for them (Temple, Tavern);
5. a rule for SOLID ROCK: a party stranded on a rock square is DEAD on a
   square that kills anyone who steps on it. The original sets MAZELEV to
   -99 there, so its bodies could never be found. R-1 keeps the square and
   the DEAD condition and does not decide whether they can be rescued;
6. Camp D)ISBAND (#26), which should then write `Stranded` too (the GUI
   still refuses it).

## 8. Limitations

- **Old binaries drop the field:** an older Istary that loads and saves a
  file with stranded characters drops `stranded`. The characters stay OUT
  (`InMaze`), but their square is lost. This is the same kind of loss as
  the expedition section (EXPEDITION-PERSISTENCE §5).
- **A floor past the last (#31):** a party there that cannot go on cannot
  be stranded, as there is no square. `LeaveCombat` refuses, and `Resume`
  reports a problem. A party there cannot move, so no hazard reaches it.
- **Camp MALOR (#25) and D)ISBAND (#26):** unchanged, and still refused in
  the GUI.
