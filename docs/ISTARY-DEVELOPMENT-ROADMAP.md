# Istary development roadmap

Status: **design baseline, approved direction** (reviewed 2026-11-06).
Milestones are ordered by dependency, not by date. Each one ends with a
review stop. Decision references (D1–D12) are to the register in
`ISTARY-ARCHITECTURE-PROPOSAL.md` §7.

## Standing rules for every milestone

- Small, logically separated commits; behaviour-preserving refactors and rule
  changes are never in the same commit.
- The terminal game builds and its regression and golden tests pass unchanged,
  unless an approved rule change says otherwise.
- The roster save format stays compatible unless a change is approved
  (`engine/testdata/save_roster1.golden.json`).
- No rule changes except recorded Classic decisions (D1) or approved
  enhancements.
- The GUI uses only its isolated save directory; tests verify that terminal
  saves are not touched.
- No presentation layout is implemented before the presentation design
  checkpoint approves a direction (M2).
- Full verification before review:
  - **root module:** `go vet`, tests, the race detector, shuffled test
    order, and `make build`;
  - **GUI module:** `go vet`, tests, the race detector, and shuffled test
    order;
  - **frontend:** the frontend tests;
  - **app:** the app build.
- No merge, tag or push without approval.

## M0 — Existing foundation

Repository status checked on 2026-11-06:

| Work | Status |
|---|---|
| Phase 1 rename, regression tests, controller extraction (creation, Tavern), GUI prototype | On `main` |
| GUI-001 Class Requirements Guide | On `main`, in tag `v0.1.0-gui-foundation` |
| GUI-002 Roster management | On `main`, in tag `v0.1.0-gui-foundation` |
| GUI-003 Party assembly | Branch `gui-party-assembly`, 5 commits after the tag (5858bd9..68be3bd). Automated verification passed. **Subject to formal manual acceptance. Not approved, not pushed, not merged** |
| Design baseline (these documents) | Branch `design-baseline`, created from `gui-party-assembly`. It therefore contains the unaccepted GUI-003 commits; see the note below |

**Note on branches:**

- **Why the baseline contains GUI-003:** `design-baseline` was created from
  the then-current HEAD, so its history includes GUI-003.
- **What that means for merging:** merging the design baseline into `main`
  would also merge GUI-003.
- **How to avoid it:** to take the documents without GUI-003, cherry-pick
  the baseline commit onto `main` after a decision, or merge GUI-003 first.

**Remaining:**

- formal manual acceptance of GUI-003, then its push and merge decision;
- a decision on how the design baseline reaches `main`.

## M1 — Design baseline, decisions and characterization tests

**Objective:** an approved design baseline, recorded decisions, and tests
that pin current behaviour before any rules work.

- **Documents:** approve this baseline (charter, architecture proposal,
  compatibility audit, roadmap). Done when committed and accepted.
- **Decisions:** record D1–D12 as dispositioned (done in the register);
  keep D2 and D3 open.
- **Characterization tests:** read-only tests pinning today's engine
  behaviour for the priority findings:
  - XP distribution;
  - poison after returning to town;
  - Inn aging;
  - chest disarm flow;
  - shop stock persistence.
- **Investigations,** in the agreed priority (`WIZARDRY-COMPATIBILITY.md`
  §5.1):
  1. historical revision;
  2. item prices;
  3. XP distribution;
  4. poison;
  5. Inn aging;
  6. trap names.

  Each runs as a separate, reviewed task.

**Depends on:** M0 for the branch decision only.
**Acceptance:**

- **Tests:** the characterization tests are merged with no behaviour change.
- **Findings:** recorded in the compatibility decision log or the matching
  rows.
- **Decisions:** no discrepancy is decided without its revision and
  evidence.

## M1b — Save robustness

**Objective:** safe saves before the controller refactoring adds state
complexity.

- **Atomic roster writes:** write a temporary file in the same directory,
  sync it, then rename it over the roster file. The format does not change.
- **Tests:**
  - **Save content:** the save content is byte-identical to today's output
    (the golden file).
  - **Failed writes:** a failed write leaves the previous roster intact.
  - **Platforms:** the terminal and the GUI both use the new path.

**Depends on:** M1 approval.
**Acceptance:**

- **Format and outcome:** save-format golden unchanged; terminal outcomes
  unchanged.
- **Failure handling:** failure-path tests pass.

## M2 — Presentation design checkpoint, then the main window

**Objective:** choose Istary's interface direction deliberately, then build
the application shell around existing GUI-001 to GUI-003 functions.

**Part A: presentation design checkpoint** (no production code).

- **What to compare:** alternative layouts and interaction models, using
  mockups or prototypes. The options to compare:
  - how locations, navigation and messages are shown;
  - how team information stays accessible. The candidates include an
    expanded formation, a compact strip and a hidden mode, but none is
    preferred;
  - how roster and profiles behave: fixed, floating, docked or modal;
  - visual hierarchy;
  - Classic versus Modern presentation;
  - a first view of artwork and rendering direction.
- **Stable identity:** assess stable character identifiers (architecture
  §2.5) if the chosen design keeps long-lived references to characters.
- **Approval:** the user approves one direction, recorded in the charter's
  decision record.

**Part B: implementation of the approved direction.**

- **Shell:** the main window shell and location routing from engine state.
- **Existing features:** the existing Training Grounds and Tavern features
  move into it.
- **Team information:** shown as designed.
- **Presentation:**
  - the presentation switch, if it is in scope;
  - a terminology lexicon;
  - menu bar and keyboard shortcuts.
- **Preferences:** a preferences file for presentation settings only.

**Depends on:**

- M0, with GUI-003 accepted and merged;
- M1;
- D4, D8, D10.

**Acceptance:**

- **Part A:** an approved design record.
- **Part B, existing behaviour:** every GUI-001 to GUI-003 behaviour still
  works, with its tests passing.
- **Part B, team information:** reflects joins and removals without a
  reload.
- **Part B, no rule effects:** a presentation change alters no game state. A
  test compares `View` game data across presentations.
- **Part B, window sizes:** usable at the minimum window size and in full
  screen.

## M3 — Town services and navigation in the controller

**Objective:** make the rest of the castle and all town navigation reachable
through `app.Controller`, without behaviour changes.

- **Queue design note first.** A design note for the message queue and
  `Advance()` (D6), answering the open questions in architecture §2.4:
  - the commit point;
  - what the queue holds;
  - saving mid-sequence;
  - interruption and recovery;
  - equivalence tests.

  It is reviewed before any queued extraction.
- **Controller-owned navigation** (architecture §2.3). The legal town
  transitions and their consequences move into the controller, including
  Edge of Town. `gui.Session.goTo` becomes a request to the controller, and
  GUI-003's transition tests still pass.
- **Operations and read-only queries** for:
  - the Training Grounds character menu: inspect, change class (with an
    eligibility query), set password, reroll;
  - the Adventurer's Inn, with the queue replacing the healing timer;
  - the Temple of Cant;
  - Boltac's Trading Post, with shop stock moved from `Scenario.Items` into
    game state, keeping today's behaviour.
- **Terminal key map:** the terminal calls the new operations through it.

**Depends on:** M1b, D5 and D6. Can run in parallel with M2 Part A.
**Acceptance:**

- **Terminal goldens:** pass unchanged.
- **Same state by key or operation:** keys and operations reach identical
  state under the same seed.
- **No terminal code in `app`:** `app/deps_test.go` passes.
- **Same answers:** each query returns the same result as its action.
- **Navigation:** no GUI code changes location state directly.

## M4 — Castle loop in the GUI

**Objective:** a complete town workflow:

**Recruit → Select Team → Equip → Recover → Develop**

- **Boltac's:** buy, sell, identify, uncurse.
- **Player profile actions:** equip, drop, trade and read spell books, as
  allowed for the location.
- **Inn:** rooms, paced healing, level-up results.
- **Temple:** cure and resurrect.
- **Training Grounds:** change class and set password.

All of these use the approved presentation direction.

**Depends on:** M2 Part B and M3.
**Acceptance:**

- **End-to-end script** (in the repository, run against scratch saves):
  create six players, assemble a team, equip everyone and rest one at the
  Inn.
- **Same outcomes as the terminal:** a seeded comparison of gold, items and
  HP.
- **Saves:** terminal saves untouched.

## M5 — Expedition in the controller, including recovery

**Objective:** make the Maze, Camp, combat, chests and expedition recovery
reachable through `app.Controller`.

- **Maze entry and Camp:** reorder, equip the team, inspect with spell and
  use, disband.
- **Movement:** maze movement and special squares.
- **Encounters and combat:** encounter start, plus combat orders and round
  steps through the queue.
- **Chests.**
- **Expedition end and recovery:**
  - return to the castle;
  - OUT status;
  - rescue pick-up (I)NSPECT and P)ICK UP).

  Characters left in the Maze can always be recovered with today's engine
  mechanisms.
- **Restarting an OUT party:** included only if D3 has been decided by then.
  Otherwise a placeholder entry is kept in the compatibility audit.

**Depends on:** M3.
**Acceptance:**

- **Terminal goldens:** `maze_walk`, `maze_wander` and `combat_trace`, plus
  the screen goldens, pass unchanged.
- **No timers in the controller:** combat is fully driven by `Advance()`.
- **Recovery:** controller tests show an OUT party being found and picked
  up.

## M6 — Maze, Camp and recovery in the GUI

**Objective:** explore the dungeon from the GUI without stranding
characters.

- **Rendering spike:** maze rendering from engine visibility data, using the
  technology and treatment chosen at or after the presentation checkpoint.
- **Maze and Camp:** movement and keyboard controls, status, Camp, and the
  stairs back to the castle. *UX-1 (2026-11-11): the party's L:R:C and
  facing in the maze's and Camp's header, Inspect / Rescue with a line when
  someone lies on the party's square, and an equipment check before
  setting out (`design/ux1/UX-1.md`).*
- **OUT handling:** OUT characters are shown as such in the roster, can be
  recovered by a rescue party, and disbanding behaves as in the terminal.
  *R-1 (2026-11-10): a party that can no longer go on stays stranded on its
  square, saved and shown with its L:R:C (`design/r1/STRANDED-PARTIES.md`).
  R-2 (2026-11-10): rescue expeditions find the stranded in their area
  and take them home one at a time within the six places
  (`design/r2/RESCUE.md`). R-3 (2026-11-11): the maps mark where the
  stranded lie, from their own records and revealing nothing else, and
  Inspect shows each one's condition and the rescue's progress
  (`design/r3/RESCUE-VISIBILITY.md`).*

**Depends on:** M5 and M2 Part B.
**Acceptance:**

- **Same walk:** a seeded walk visits the same squares and events as the
  terminal golden.
- **Rescue:** a rescue expedition brings OUT characters back, end to end in
  the GUI.
- **No map unless enabled:** no map is shown unless the automap enhancement
  is enabled (D9). *Amended at M-1 acceptance (2026-11-10): until the
  Classic/Enhanced switch exists (M9), the GUI's explored-only minimap and
  F10 full map stay available, the minimap on by default.*

## M7 — Combat and chests in the GUI

**Objective:** fight, loot and return.

- **Combat:** order entry for each active player, front-line restrictions,
  and paced messages.
- **Results and chests:** results, chests and trap handling.
- **Death and defeat:** death and total defeat lead into the M6 recovery
  path.

**Depends on:** M6.
**Acceptance:**

- **Same trace:** a seeded combat yields the same trace as the terminal's
  `combat_trace` golden.
- **Full expedition:** a complete expedition works in the GUI alone:
  Camp → fight → chest → return → Inn.
- **Rescue after defeat:** a defeated party can be rescued.

## M8 — Classic Wizardry I complete

**Objective:** the whole game playable in the GUI under the Classic rules as
decided by then.

- **Remaining features:** Utilities equivalents (rename, backup and
  restore), and any D3 features adopted into Classic.
- **Compatibility audit:** close or explicitly defer every "Not
  investigated" row.
- **Playtest:** a guided playtest against the Hawkwind walkthrough
  (PG p.51–61).

**Depends on:** M4 and M7.
**Acceptance:**

- **Playtest:** levels 1–3 can be played to completion in the GUI.
- **Compatibility matrix:** current, with the decision log up to date.
- **Release:** a local pre-release tag on approval.

## M9 — Optional rules framework

**Objective:** make it possible to offer enhancements safely.

- **Ruleset and campaign file:** `engine.Ruleset` with the `Classic` preset,
  plus the campaign sidecar (D7).
- **Settings:** campaign settings, each option documented in
  `docs/RULES.md`.
- **First enhancements,** by decision. Candidates:
  - automap (D9, amended at M-1): the switch gives the minimap and the full
    map each its own setting; Classic defaults both off, Enhanced both on,
    and the player may turn either on or off;
  - showing monster identities.

**Depends on:** M8 (preferred) or M4 for the framework alone.
**Acceptance:**

- **Classic unchanged:** with all options off, every golden and seeded test
  is unchanged.
- **Options tested:** each option has its own tests.
- **Campaign file:** versioned and ignored by the terminal.

## M10 — Presentation polish (can overlap M6–M9)

- **Art, sound and animation:** as set by the approved presentation
  direction, plus any later presentation checkpoints.
- **Accessibility:** keyboard-only play, VoiceOver labels and colour
  independence.

**Acceptance:**

- **Accessibility:** a checklist review.
- **Rules:** no game-state differences.

## M11 — Distribution readiness (deferred; not scheduled)

- **Legal:** a legal review (D12).
- **Saves:** save location, password storage and save format version.
- **Mac app:** code signing, notarisation and an updater decision.

**Acceptance:** to be defined when distribution is approved.

## Dependency summary

```
M0 ─► M1 ─► M1b ─┬─► M3 ─┬──────────────► M4 ─┐
     M2 Part A ──┤       └─► M5 ─► M6 ─► M7 ──┼─► M8 ─► M9
                 └─► M2 Part B ───────────────┘
M2 Part A can start after M1 · M10 overlaps M6–M9 · M11 deferred
```

## Backlog (not scheduled)

Known limitations kept on purpose, each to be proposed and approved on its
own before any work.

- **Terminal Inn, early checkout.** The guide's SPACE checks out early
  (PG p.34); the terminal's stay always runs until healed or out of gold.
  The GUI has Choose weeks (row N2).
- **Terminal Pool Gold and Divvy Gold.** The controller has both
  (`app/gold.go`); the terminal's menus offer neither (finding #4).
- **Terminal saving at the Inn and on Ctrl-C.** The terminal saves after
  each key; a stay and its level check run on its timer, so they are saved
  at the next key, and Ctrl-C quits without saving. Quitting in between
  loses the stay and allows its level check to be rolled again. The GUI
  saves a stay and its level check as one action.
- **Terminal Inn quirks.** The terminal still lets a guest who died of old
  age stay on, and takes a paid room for a hurt guest who cannot pay a
  week, healing nothing (as the original); the GUI refuses both.
- **Pool Gold elsewhere.** Pool Gold from a character's inspect screen
  (PG p.28) is in the controller but on no GUI screen.
- **Inn aging.** No source establishes the rate (finding #11).
- **Terminal Temple.** The terminal keeps the original: the patient found
  anywhere on the roster, even out in the maze; any party member tithes,
  dead or not; paying and the treatment are two keys, each saved after
  it. The GUI treats only characters at the castle, takes an OK payer, and
  makes the donation and the treatment one action and one save.
- **Temple pooling (finding #5).** The GUI follows the guide (p.33): a
  short payer's donation is met from the whole party's gold, asked first.
  The terminal still refuses, as the engine's p-code citation does; which
  the original program did is unverified.
