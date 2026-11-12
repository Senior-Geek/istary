# UX-3A — Game lifecycle and save management

Status: implemented on `ux-3a-game-lifecycle` (2026-11-12), awaiting
acceptance. First stage of UX-3, gameplay usability consolidation.

A game slot is one complete, independent Wizardry adventure. The roster is
part of that adventure, not the adventure itself: clearing characters does
not reset the world, and a new adventure never inherits another's party,
map or place.

## 1. Audit (before the change)

Starting point: `main` at `c706363` (v0.9.0-navigation-ux), clean.

- **One game per window.** The GUI kept one save,
  `~/Library/Application Support/Istary/dev-saves/roster1.json`
  (`$ISTARY_SAVE_DIR` if set), opened on start straight into the Castle
  or, with an expedition saved, back into the maze.
- **Written by the engine.** `engine.GameState.Save` after every action
  (`Session.save`), replaced atomically (`writeFileAtomic`); the folder is
  chosen by the process-wide `engine.SetSaveDir`.
- **What the file holds:**
  - the roster, each character whole: attributes, level, XP, HP, spells,
    items (equipped, cursed, identified), gold, status, poison, age,
    password, and the `stranded` square of a character left in the maze;
  - the castle party, by name, in order;
  - `expedition` (version 1, `engine/expedition.go`): the party in the
    maze, its checkpoint (level, square, facing, light, protection, open
    prompt, open Camp screen), changed special squares, and an interrupted
    encounter or chest;
  - `exploration` (version 1, `engine/explore.go`): the automap: visited
    squares, walls, doors and features per level.
- **Beside it, not in it:** `backups/` (roster backups and restores),
  `gui-settings.json` (the window's preferences), `istary.lock` (one
  window per folder).
- **Not saved anywhere:** the town location (a game opens at the Castle),
  the stairs suggestion (maze navigation), used special squares between
  expeditions (kept in the loaded scenario until Istary closes), receipts
  and notices.
- **Migrations:** the engine reads `wiz<N>/roster.json` and old item and
  spell layouts and works out AC; sections of another version are kept and
  written back unchanged. Pinned by `engine/testdata/save_roster1*.golden.json`.

### Concerns found, and what was done

1. **`SetSaveDir` is process-wide.** Each slot is a folder holding exactly
   what the save folder held, and the engine is pointed at the open
   game's. The save format and the engine's save code are unchanged.
2. **Used special squares live in the loaded scenario,** shared by the
   window. Opening a game loads Wizardry I afresh, so one game's maze is
   never another's. (Within a game, changing games and continuing is like
   restarting Istary: what the save keeps is kept.)
3. **An encounter or chest interrupted by closing Istary** is not
   restored: by the existing rule (M6b) it is over, its unclaimed rewards
   lost, the party back in the maze where it fought, and the resume notice
   says so. UX-3A keeps that rule; it changes no combat mechanics.
4. **Reset roster** (Settings) already refused while an expedition is
   saved, and leaves the map. Stranded characters are roster entries, so it
   clears them too, as before.
5. **Nothing of a game is deleted.** Replacing moves the game aside.

## 2. Layout

The window opens a *library* folder: the same default folder (or
`$ISTARY_SAVE_DIR`).

```
dev-saves/
  istary.lock            the window's lock (one window per library)
  gui-settings.json      the window's preferences, the same in every game
  roster1.json           the earlier single save, if any: never written
  backups/               its backups: never written
  games/
    slot-1/ … slot-6/
      game.json          the slot's record: version, created, origin, import checksum
      roster1.json       the game, exactly as the engine saves it
      backups/           this game's roster backups (Settings)
  replaced/
    slot-N-<stamp>/      a replaced game, moved aside whole
```

- **A slot** is empty (no `game.json`, no `roster1.json`), a game, or
  damaged. Damaged: the save cannot be read, is missing, or the record is
  unreadable or from a newer Istary, or the folder holds no game at all (a
  new game interrupted before its first save). A damaged slot is listed with why,
  cannot be continued, and is left as it is; it can be replaced.
- **`game.json`** (`gameFormat` 1) is new; `roster1.json` is the existing
  format, so the engine's goldens and every compatibility rule still hold.
  A save that parses but cannot be loaded (an unreadable expedition section,
  say) opens nothing and is left as it is.

## 3. Behaviour

- **Start.** Istary opens on game selection, with no game open: nothing is
  loaded, resumed or saved until the player chooses.
- **New game** (an empty slot): an empty roster and party, nothing of the
  maze seen, no one stranded, no expedition, at the Castle. Its first save
  is the engine's ordinary save of an empty game.
- **Continue** (a game): the game as last saved: roster, party, map,
  equipment, gold and status; an expedition resumes on its square, facing
  the same way, under EXPEDITION RESUMED, as after a restart (maze
  navigation). The autosave after every action is unchanged and writes
  only the open game's slot.
- **Replace** (a game or a damaged slot): asked, then confirmed for that
  slot (`ReplaceGame`, then `ConfirmReplaceGame`); New Game on an
  occupied slot, an unasked confirmation, a cancelled one, or one for
  another slot changes nothing. Confirmed, the slot's folder moves to
  `replaced/` and a new game begins there.
- **Change game** (the Edge of Town, beside Leave the game): closes the
  open game and returns to selection. Refused anywhere else, during an
  expedition or practice, while a character is being created, and while
  the last save has failed: everything else is already saved.
- **Clear roster** is Settings' Reset roster, unchanged: in the game's own
  slot, backed up in its own `backups/`, the map kept.
- **One window per library.** The lock is on the library folder.
- **No game open.** Every action of the window is refused or shows game
  selection; none writes a game.

### Earlier save (migration)

- **Offered** when the library folder holds `roster1.json` and no slot's
  record holds its SHA-256: "A game saved by an earlier Istary", with what
  it holds, and Import into each empty slot.
- **Import:** the file is copied byte for byte into the slot, with the
  library's `backups/`; the copy is read back and compared, then opened
  (loaded and, with an expedition, resumed) before it counts. If any step
  fails, the new slot folder is removed and nothing else is touched.
- **Never touched:** the original and its backups are not written, moved or
  deleted, so an earlier Istary can still open them.
- **Once:** the slot's record holds the checksum, so a restart does not
  offer it again and nothing is imported twice. If the slot holding it is
  later replaced, the original (still as it was) is offered again.
- **Damaged:** listed with why, not offered for import, not touched.
- **The terminal game** (`~/.config/wizardry`) is unaffected.

## 4. Tests

`gui/games_test.go`, and `gui/games_fixture_test.go` with
`frontend/src/lib/games.test.js` over the recorded views:

| Requirement | Test |
|---|---|
| 1 Two independent games | `TestTwoGamesAreIndependent` |
| 2 Equipment, gold, status | `TestAGameKeepsEquipmentGoldAndStatus` |
| 3 Separate exploration | `TestChangingGamesCarriesNothingAcross`, `TestTwoGamesAreIndependent` |
| 4 Coordinates and facing | `TestAnExpeditionContinuesWhereItWasSaved` |
| 5 Stranded and rescue state | `TestTheEarlierSaveIsImportedOnce` |
| 6 Active expedition or encounter | `TestAnExpeditionContinuesWhereItWasSaved`, `TestAnInterruptedEncounterContinuesUnderTheExistingRule` |
| 7 Clear roster keeps the map | `TestClearingTheRosterKeepsTheMap` |
| 8 Replace only once confirmed | `TestAGameIsReplacedOnlyOnceConfirmed` |
| 9 Import without loss | `TestTheEarlierSaveIsImportedOnce`, `TestAnImportIsTheOriginalUntilPlayed` |
| 10 No duplicate import | `TestTheEarlierSaveIsImportedOnce`, `TestAnImportIsTheOriginalUntilPlayed` |
| 11 No cross-contamination | `TestChangingGamesCarriesNothingAcross`, `TestTwoGamesAreIndependent`, `TestSettingsAreTheWindows` |
| 12 Missing, invalid, corrupt | `TestDamagedSavesAreLeftAsTheyAre`, `TestAnEmptySlotFolderIsReplaced` |
| Autosave in the slot only | `TestTwoGamesAreIndependent` |
| Format unchanged | `TestANewGameIsAnOrdinarySave`, `TestAnImportIsTheOriginalUntilPlayed`, the engine goldens |
| Guards | `TestChangingGamesIsRefusedMidGame`, `TestNoGameOpenActsOnNothing`, `TestALibraryIsOneWindows` |

Every earlier GUI test still opens one save folder directly
(`NewSession`), as before, and is unchanged.

## 5. Deferred

- Naming a game; deleting one outright (only replacing, which keeps it).
- Clearing `replaced/`; it is never pruned.
- A Change game button at the Castle: it is at the Edge of Town, beside
  Leave the game, where the original leaves the game.
- Changing games during an expedition (it is saved, but the transition is
  not offered).
- Restoring an interrupted encounter itself (the M6b rule stands).
