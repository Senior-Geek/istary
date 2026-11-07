# M6b: expedition persistence and recovery (design)

Status: **M6b-2 implemented** (sections 12 and 13 record what was built
and where it differs from the proposal below). Written with M6b-0 (atomic saves), on
`m6b-persistent-expeditions`. Decisions D-A to D-F were approved on 2026-11-07
(`WIZARDRY-COMPATIBILITY.md` decision log). D-G, D-H, W, L and P were approved
in principle the same day, with the safeguards in section 11; R1 to R3 are
still open.

## 1. What the engine does today

- **The save** (`engine/save.go`): the roster, plus the names of the party
  in the castle, in formation order. Nothing else is saved: no position, no
  floor, no combat. Since M6b-0 the file is replaced atomically.
- **When it saves:** the terminal saves after every key and the GUI after
  every action, mid-combat included.
- **After a restart:** the party is in the castle with whatever it had at the
  last save. This is the free escape D-A removes.
- **`InMaze`:** set when a character joins a party at the Tavern, cleared
  when the character leaves it. In the castle it means "in a party".
- **`MazeLevel`, `MazeX`, `MazeY`:** exist on every character but are never
  written (finding #26). The Tavern refuses anyone with `InMaze` set or a
  non-zero `MazeLevel` (`** OUT **`).
- **I)NSPECT** lists every roster character with `InMaze` set whose
  position matches the party's square and who is neither in the party nor
  LOST. **P)ICK UP** takes them along (but see #27).
- **State that lives only in memory:**
  - the fight zones (`FightMap`), rebuilt when a floor is entered;
  - light, protection and anti-magic;
  - a pending chest alarm;
  - the maze prompts;
  - special-square counters. These are changed in the loaded scenario data,
    e.g. one-time messages, items and special encounters, which become
    normal squares once used up (`app/maze.go`, `SqSpclEnctr`). A restart
    resets them all.
- **Encounters:** an encounter starts at the end of a step, after the
  square's effect and the healing and poison roll. A fight room is cleared
  (`ClearFightRoom`) in that same step, before combat begins.
- **Rewards:** experience, gold and items are all given in one call,
  `FinalizeChest`, inside one action: when the chest is opened or left, or
  at victory when there is no chest. Before that call, the only lasting
  effects of combat are on characters: damage, status, spent spells, items
  and gold taken by traps, and LOKTOFEIT (#24).

## 2. Save schema: the optional `expedition` section

The section is added to the existing roster file. It is absent whenever no
expedition is under way, so a save outside an expedition stays byte-identical
to today's (the save goldens enforce this).

```jsonc
{
  "roster": [ ... ],               // unchanged
  "party": [ ... ],                // unchanged meaning: the party in the CASTLE.
                                   // Empty while the expedition party is out.
  "expedition": {                  // optional; absent = nobody exploring
    "version": 1,
    "party": ["ALDRIC", "..."],    // the party in the maze, formation order
    "where": {                     // the last safe point (section 3)
      "level": 3, "x": 7, "y": 12, "facing": 1,
      "light": 0, "protect": 0, "fizzles": 0, "quickPlot": false,
      "chestAlarm": 0,
      "fightMap": ["0010...", "..."],  // 20 rows of 20 '0'/'1'
      "phase": "maze",             // "maze" | "camp"
      "prompt": null               // an open maze prompt (section 3), or null
    },
    "interrupted": null,           // "combat" | "chest" while one is unresolved
    "squares": [                   // special squares changed on this expedition
      { "level": 2, "x": 4, "y": 9, "type": 0, "count": 0, "spclMonster": 0 }
    ]
  }
}
```

**Why the expedition party is not in the top-level `party` list:**

- **Older front ends:** today's terminal and older GUI builds would load
  those names into the castle party and could take the same characters into
  the maze a second time. With the list moved, an older build sees the
  members as `** OUT **` (`InMaze` is still set) and cannot use them.
- **Rescue:** a castle party (for a rescue) and an expedition can then exist
  at the same time, as D-D needs.

**Older saves** have no section and load as today.

**Unknown `version`:** the file is left unchanged and the GUI refuses to
start the expedition, saying why. Nothing is discarded.

**What `squares` and `prompt` cover:**

- `squares` records changed special squares as they were at the last safe
  point. Without it, a restart in the middle of an expedition would reset
  one-time items and encounters while the party is still beside them.
  (Today's resets on restart outside an expedition are left alone; see
  decision W.)
- `prompt` holds only plain data:
  - the message pages and how far they have been read;
  - search Y/N, with the square found again from the party's position and the
    original monster number;
  - a pending return to the castle or item gift;
  - the buttons;
  - the stairs question;
  - the Inspect screen, whose finds are re-listed on resume.

  The terminal-only map overlay and delay entry are not saved.

## 3. Safe points and transactions

A **safe point** is a moment when the game is waiting for the player, the
current step has finished completely, and no combat or chest is open:

1. in the maze with no prompt open;
2. in the maze at a maze prompt (messages, search, buttons, stairs,
   Inspect). These are pauses inside a step: the step's effects have already
   happened, and saving the prompt lets the step finish exactly as it would
   have;
3. at the Camp menu (Camp's sub-screens resume at the Camp menu; see below);
4. **the encounter checkpoint:** the moment an encounter begins, without the
   combat itself. The step that led to the encounter is complete, including
   the fight room being cleared, the alarm reset and that step's random
   draws.

**One action, one save.** After each player action, the GUI saves the whole
file once, atomically: the roster as it is now, and the expedition as of its
latest safe point. So the roster and the expedition always come from the
same moment and can never disagree. A crash during a save leaves the
previous file (M6b-0), so at most the last action is lost.

**During combat or a chest:**

- The save keeps `where` at the encounter checkpoint, with `interrupted` set.
- The roster keeps being saved after every action, so losses are committed as
  they happen.
- When the encounter ends with victory and the party is back in the maze,
  the next save clears `interrupted` and moves `where` to the new safe point.

**Camp sub-screens:**

- Equip, trade, spells and the MALOR screen resume at the Camp menu. Their
  effects reach the roster only when they happen.
- MALOR deducts the spell before its screen opens, so an interrupted MALOR
  screen loses the spell and moves nobody. This is the same rule as D-B:
  resources are kept as spent.

## 4. Committed losses and unresolved rewards (D-B, interim)

When the app restarts with `interrupted` set:

- **Position and expedition state:** the party is put back at the
  encounter checkpoint.
- **Characters:** come back exactly as saved. Damage, deaths, status,
  poison, spent spells, and items and gold lost to traps or LOKTOFEIT all
  stay. They are already in the roster.
- **Rewards:** experience, gold and items are given only if `FinalizeChest`
  had already run before the interruption. In that case they are already in
  the roster and are kept, and that encounter can never award them again.
  Otherwise nothing is awarded.
- **Effects outside the characters:** an alarm triggered by the unresolved
  chest, and light, protection or a MALOR position change from spells cast
  in that combat, return to their checkpoint values. The spell stays spent.
  This is conservative: nothing is gained twice, and a little can be lost.
- **The player is told,** for example: "The expedition was interrupted
  during combat. The party recovers where the encounter began; the
  encounter's unclaimed rewards are lost."

**Why the encounter checkpoint, not the step before it:** the step before
it still has the fight room's flag set. Resuming there would let the party
step back in and win the same fight's rewards again. At the encounter
checkpoint the room is already cleared, so the fight cannot be repeated. A
new random encounter on later steps is ordinary play.

This is an **interim** rule. Exact resumption needs the combat and chest
state (groups, orders, playback position, chest stage) saved as a section
of its own; this is recorded in the decision log.

## 5. Ways a save could duplicate rewards or lose a party, and the guards

| Risk | Guard |
|---|---|
| Reward given twice after a restart | Rewards come from one action (`FinalizeChest`) and are saved with it; the encounter is never re-entered (section 4) |
| A one-time item or encounter used twice by restarting | `squares` keeps changed squares for the expedition (decision W covers restarts outside one) |
| The same characters in the castle party and the expedition | The expedition party is stored only in `expedition.party`, and its members keep `InMaze` set (Tavern: `** OUT **`) |
| An expedition member renamed, deleted or reset while out | The GUI refuses Training Grounds edits, deletion, Reset and Tavern removal for expedition members, and Reset while any expedition exists. Restoring a backup reloads everything from that file |
| A file that doesn't match (a name missing from the roster, a character LOST or in the castle party, a position off the map, an unknown version) | On load, validate and **never discard**. Show the problem and offer only "return the party to the castle" (today's outcome), after an automatic backup |
| An older binary drops the section on its next save | It can't be prevented. The members stay `** OUT **` and none of their data changes, but their position is lost. Documented; the GUI warns before the first expedition is saved |
| Two copies of the app saving the same file | The second save overwrites the first. Proposal: a lock file in the save folder for the GUI (decision L) |
| A crash during a save | Atomic replacement (M6b-0). Stale `.tmp-` files are never read; proposal: remove them on the next successful save |

## 6. Defeated and abandoned parties, and rescue (D-D; M6b-4 design)

**Proposed:** use the engine's own per-character fields, which the existing
I)NSPECT and P)ICK UP already read. No new save structure is needed.

**When a party is defeated** (all dead in combat, SOLID ROCK, or, if
approved, a dead party after a step; see #28):

1. Each member's position is written into their own `MazeLevel`, `MazeX`
   and `MazeY`.
2. `InMaze` stays set and statuses are unchanged (dead, ashed and so on).
3. The members leave the party, and the expedition section is removed.
4. The player is in the castle with no party, and the fallen characters show
   as OUT on the roster, with the floor (B1F and so on) where they fell.

**A rescue:**

1. Form a party from characters in the castle.
2. Walk to the square and I)NSPECT, which lists the fallen.
3. P)ICK UP takes them along while the party has room (six members).
4. Back in the castle, as now, the stairs clear `InMaze` for the whole party.
5. The rescued characters' position fields are cleared at that moment, so
   the Tavern stops refusing them.
6. The Temple then deals with the dead.

**Disband** (#26) would write the position the same way, so disbanded
characters are found where they were left. Today they are only found at the
entrance.

**Interactions that need deciding:**

- **#27, room in the party:** P)ICK UP should take only those who fit and
  leave the rest in the maze.
- **#28, a dead party after a step:** should it become a defeated party at
  that square?
- **Rescuing the rescuers:** a second defeat on the same square leaves both
  parties there. The guide's rule that the square must be reachable without
  doors (M4) is not investigated yet.

Each of #26, #27 and #28 is an engine change and goes through its own
proposal (D-C). D-D itself changes `LeaveCombat` and the SOLID ROCK path.

## 7. The terminal

- **Until it is migrated:**
  - `engine.Load` reads the `expedition` section into the game state, and
    `Save` writes it back unchanged, so the terminal never silently drops a
    GUI expedition.
  - On start, the terminal shows a line saying an expedition is under way
    in the Istary app.
  - The members are not in the castle party and are refused at the Tavern,
    as today's OUT characters are.
  - The terminal's own play does not change: it never writes the section,
    and quitting in its own maze still puts its party in the castle. Every
    golden stays unchanged.
- **Later** (separate approval): the terminal resumes through the
  controller (Edge of Town M offers to resume) and writes safe points
  itself. At that point the terminal's quit-in-the-maze behaviour changes.
- **The two front ends never share a save folder** (the GUI refuses the
  terminal's), so neither can overwrite the other's expedition by accident.

## 8. Restarting the application

1. Load the file (atomic: it is the old or the new version, never a
   mixture).
2. Validate the section (section 5). If it fails, show the recovery screen.
3. If `interrupted` is set, apply section 4 and show the notice.
4. Open the dungeon screen at the restored safe point: Camp, the maze, or the
   open prompt.
5. The random sequence starts afresh, as today. Random-number state is not
   saved, and Classic play doesn't depend on it.

The practice expedition and practice encounter keep their separate copy of
the game and are never saved. While a real expedition is under way, the GUI
offers practice only with characters who are not in it.

## 9. Proposed milestone split (refined)

- **M6b-1:** the GUI drives the real game through the dungeon bridge.
  - Saves follow the current rules: no expedition section yet.
  - A restart in the maze still puts the party in the castle; this is
    stated in the GUI.
  - Defeat sends the party to the castle (the engine's current behaviour,
    until M6b-4).
  - Expedition members are protected from roster edits.
  - Tests:
    - real walks against the terminal's keys;
    - every action saves;
    - roster edits are refused.
- **M6b-2:** the `expedition` section, safe points, D-B, validation, the
  recovery screen, and the terminal keeping the section intact. Tests:
  - save, reload and compare at every kind of safe point, over seeded random
    walks;
  - interruption at each kind of step (maze, each prompt, Camp, combat at
    every phase, the chest at every stage);
  - each guard in the section 5 table;
  - files from older builds (with and without the section);
  - the terminal round trip.
- **M6b-3 to M6b-5:** as approved. M6b-4 needs decisions R1 to R3 below.

## 10. Needs approval (as proposed; see section 11 for the outcome)

- **D-G:** resume an interrupted encounter at the **encounter checkpoint**
  (the party on the encounter square, the room cleared), rather than the
  step before it. This is what prevents the fight, and its rewards, from
  being repeated.
- **D-H:** a safe point may hold an open maze prompt, which is resumed
  exactly as it was (section 3, point 2).
- **W:** keep changed special squares in the expedition section. Outside an
  expedition, restarts keep resetting them as today. Persisting them
  permanently would be a separate question about Classic.
- **L:** a lock file so only one copy of the GUI saves to a folder.
- **P:** while an expedition is under way, the expedition party is stored
  only in `expedition.party`, and the top-level `party` holds the castle
  party.
- **R1–R3 (for M6b-4):** the rescue representation using the
  per-character position fields; clearing the position when rescued
  characters reach the castle; and the treatment of #26 to #28 as part of
  D-D or as separate proposals.

## 11. Approved in principle (2026-11-07), with safeguards

- **D-G:** an interrupted encounter resumes on the square where it began,
  which the party had already entered. Nothing is replayed: not the
  encounter, not the square's effects, not the rewards. Character losses and
  spent resources stay committed.
- **D-H:** an open maze prompt may be saved and restored exactly.
- **W:** changed special squares persist for the expedition.
- **L:** one writer per save folder, through a lock. Stale-lock recovery
  (after a crash, or a lock left by another machine or user) must be
  designed and tested before the lock is implemented.
- **P:** expedition membership is stored apart from the castle party list,
  but each character has exactly **one** authoritative record: the roster
  entry. `expedition.party` holds names that refer to roster entries, never
  copies of characters.

R1 to R3 (rescue) remain open for M6b-4.

## 12. Implemented so far

- **M6b-0:** atomic saves.
- **M6b-1:** real expeditions in the GUI, saved after every action, with
  the engine's current defeat.
- **M6b-2:** the expedition section, checkpoints and resume, the
  single-writer lock (section 13).
- **M6b-3A:** Camp in the controller; Camp checkpoints keep the open Camp
  screen (section 13).

## 13. M6b-2 as built

**Schema** (`engine/expedition.go`; `"party"` beside it is the castle
party, empty while the GUI's party is out):

```jsonc
"expedition": {
  "version": 1,
  "party": ["TORVALD", "..."],            // names; the roster entry is the record
  "where": {
    "phase": "maze",                        // or "camp"
    "level": 0, "x": 5, "y": 0, "facing": 1,
    "light": 0, "protect": 0, "fizzles": 0,
    "quickPlot": false, "chestAlarm": 0,    // omitted when false / 0
    "fightMap": ["0000...", ...],           // 20 strings of 20, indexed [x][y]
    // what the maze screen shows and any open prompt (omitted when empty):
    "message": "", "message2": "", "viewport": "", "viewport2": "",
    "messages": [], "msgScroll": 0, "msgWait": false,
    "searchYN": false, "searchHere": false, "searchMonster": 0,
    "back2Shop": false, "tryGet": false, "tryGetItem": 0,
    "waitButton": false, "buttonCount": 0,
    "inspecting": false, "inspectFound": [],  // names
    "campMessage": ""
  },
  "interrupted": "combat",                  // or "chest"; omitted when none
  "squares": [ { "level": 0, "x": 13, "y": 5, "type": "spclenctr", "count": 77, "spclMonster": 77 } ]
}
```

- **Migration:** none needed. Old saves have no section and load as
  before; ordinary saves stay byte-identical (save goldens unchanged).
- **Other versions:** a section of another version is kept and written
  back unchanged by every front end. The GUI shows it and won't resume it.
- **Unreadable section:** a load error (the GUI's fatal banner), never
  silently dropped.
- **Squares:** every square whose type, count or special monster differs
  from the scenario as loaded. On resume, all squares are reset to the
  scenario as loaded, then these are applied.

**Checkpoints** (`app/expedition.go`): taken after every real-expedition
action, in the same atomic write as the roster.
- **Safe points:**
  - in the maze, with or without an open maze prompt (D-H);
  - in Camp, resumed at its menu;
  - during an encounter.
- **During an encounter:** the checkpoint is the game as it stands without
  the encounter, marked interrupted, rather than frozen where it began.
  - Committed changes outside the characters therefore survive a restart:
    a chest's teleporter, an alarm trap, combat MALOR's move.
  - Until such a change happens, it is exactly where the encounter began:
    the square already entered, its fight room cleared (D-G).
- **On resume after an interruption:**
  - the encounter is over, with its messages cleared as leaving combat does;
  - nothing is replayed;
  - a notice says its unclaimed rewards are lost;
  - the new state is saved at once.
- **Exactness, tested:** over 30 seeds with play saved, reloaded and
  resumed after every safe point, the state matched uninterrupted play at
  every step, including the random-number sequence. The runs covered every
  kind of safe point, Murphy's ghost search and the elevator.
- **Floors:** any non-negative floor is accepted. The B4F–B9F elevator's H
  and I buttons put the party on floors that do not exist (finding #31).
  The game is restored as the engine left it, not "rescued".
- **Problems:** a save that does not match the roster or the maze is
  reported (`ExpeditionProblem`) and left untouched.
  - Try resuming again: `ReloadRoster`.
  - Return its party to the castle: backs the file up first, all members
    or none.
  - While it is saved, a new expedition and Reset are refused.
- **Identity:** while an expedition is saved, renaming or deleting its
  members is refused in every front end (`ErrOnExpedition`).

**Saves:** the GUI tells three outcomes apart.
- **Durable:** the save succeeded.
- **Not replaced:** `View.Unsaved`. The file keeps the previous checkpoint,
  so a restart resumes there; the action is never replayed.
- **Replaced but the folder not synced:** `engine.NotDurableError`, shown
  as `View.Warning`. Saved, but it may not survive power loss.

**Lock (L):** an advisory `flock` on `istary.lock` in the save folder,
held while the window runs.
- **A second window:** refuses to start, names the holder and changes
  nothing; Try again works once the first has closed.
- **Release:** on quitting (`OnShutdown`), and by the kernel when the
  process ends in any way, so a crash leaves no lock to clear. Tested with
  a child process killed by SIGKILL, and live by killing the app.
- **Folders that can't be locked:** the window runs and warns, rather than
  strand the player.
- **Opened at window start:** the folder is opened in `OnStartup`, not
  `main`, because Wails also runs the program to generate bindings. The
  live check found that run resuming and saving, which consumed an
  interrupted encounter.
- **Not callable from the page:** `start` and `shutdown` are unexported, so
  the page's bindings can't reach them.

**The terminal:**
- It keeps a GUI expedition and writes it back.
- Its members are out: the Tavern refuses them, and they can't be renamed
  or deleted.
- Its own play and saves are unchanged, and the goldens are unchanged.
- Builds of the terminal from before M6b-2 would drop the section.

**Camp screens (M6b-3A).** `where.camp` records Camp's screen by name
(`"menu"`, `"inspect"`, `"equip"`, `"drop"`, `"trade"`, `"tradeGold"`,
`"tradeItem"`, `"spellBooks"`, `"spellList"`, `"castSpell"`, `"spellTarget"`,
`"useItem"`, `"dumapic"`, `"malor"`, `"reorder"`) and every Camp field:
- the character inspected and the trade partner (by name), typed text;
- the equip category and choices, and the party-wide equip progress;
- the reorder so far, the spell book shown, the spell waiting for its
  target, and MALOR's displacement.

`Resume` restores them exactly, like an open maze prompt (D-H). A Camp test
saved, reloaded and resumed after every key and followed uninterrupted play
exactly, at every screen.

**Known gaps:**
- **Escape by restart:** quitting during an encounter ends it without its
  rewards. This is D-B by design, and exact combat and chest resumption is
  still needed. Until then a player can escape any fight by restarting.
- **Special squares reset outside expeditions:** the counters still reset
  when the app restarts with no expedition saved, so a one-time item or
  encounter can be had again by finishing an expedition, restarting and
  going back. This was already true in the terminal, and W covers only
  expeditions; closing it means keeping the squares outside expeditions too,
  which needs a decision.
- **Duplicate names:** Import and Transfer in the terminal can add a
  character with a member's name. Resume then reports a problem rather than
  guess.
