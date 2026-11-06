# Istary architecture proposal

Status: **design baseline, approved direction** (reviewed 2026-11-06).

- **Current code:** described as it is at `gui-party-assembly` (68be3bd).
- **Target structure:** nothing here is implemented unless marked *current*.
- **As-built reference:** `docs/ARCHITECTURE.md` remains the description of
  the code as built.

Labels used:

- **Fact:** checked in the code.
- **Assumption.**
- **Target:** an approved architectural responsibility.
- **Open:** a design question to settle at the named milestone.

Decisions are in the register in section 7.

## 1. Current architecture (Fact)

```
cmd/istary (terminal) ─┐                         gui (separate Go module)
  tcell, timers,       │                           Wails v2 + Svelte 5
  legacy handlers      │                           Session ── View ──► Svelte
                       ▼                              │
                 app.Controller  ◄────────────────────┘
          (semantic operations + key map;
           creation, Training Grounds name/edit/delete,
           Tavern, roster rename/clear)
                       │
                       ▼
                    engine  ──►  data  ◄──  scenarios/wiz1-3 (embedded JSON)
       (GameState, TownState, CombatState, rules, save/load)
```

| Area | Current state |
|---|---|
| Engine | `engine` holds state and most rule functions. The game uses the global `math/rand` source, so game logic must run on one goroutine. `SpellTable`, the save-directory override and scenario item stock are package or shared state |
| Controller | `app.Controller` covers character creation, the Training Grounds name, password and delete flows, the Tavern, and roster rename and clear. It has no terminal dependency (`app/deps_test.go`). Everything else (Inn, Boltac's, Temple, Edge of Town, Utilities, class change, camp, maze, combat, chests) is still handled in `cmd/istary/main.go` (`legacyDispatch`), partly mixed with tcell input |
| Navigation | The controller has enter/leave operations for the Training Grounds and the Tavern. In the GUI, `Session.goTo` (`gui/location.go`) decides the sequence of those calls and settles open prompts; it is the only GUI code that changes the location. The terminal changes most locations directly in its handlers |
| Timers | Combat message pacing, Inn healing ticks, Wiz 3 story pages and the title animation live in the terminal main loop |
| GUI | `gui.Session` serialises calls with a mutex, calls controller operations, saves after each action and returns a `View`. Svelte components receive data and callbacks only. Character sheets are identified by roster index |
| Saves | `roster<N>.json` holds only `roster` and `party` (`engine/save.go` `SaveState`). There is no format version, no settings, no shop stock and no stable character identifier; party membership is stored by name. Passwords are stored in plain text. Files are written in place with `os.WriteFile`. The GUI uses `~/Library/Application Support/Istary/dev-saves` (or `$ISTARY_SAVE_DIR`) and refuses the terminal directory. Backups go to `<saveDir>/backups` |
| Tests | Root module: engine, app and terminal tests, golden files, seeded randomness, `-race`, shuffled order. GUI module: session tests. Frontend: `node:test` for panel geometry. Headless end-to-end checks were run ad hoc and are not in the repository |
| Rules documentation | `docs/RULES.md` lists hard-coded rules and where they live. `docs/WIZARDRY-COMPATIBILITY.md` compares them with the Player's Guide |

**Strengths.**

- **One rules implementation:** the GUI already uses the same controller
  operations as the terminal.
- **Isolated saves:** the GUI's development data is kept apart from the
  terminal's.
- **Deterministic tests:** they would catch accidental rule changes.

**Constraints.**

- **Rules in the terminal:** most of the game still sits in terminal input
  handlers.
- **Pacing in the main loop:** time-based behaviour belongs to the terminal
  main loop.
- **Fragile saves:** saves are not atomic and record neither which rules
  were used nor stable identities.

## 2. Target architecture

### 2.1 Layers and responsibilities (Target)

```
 Presentation (Svelte)          renders View snapshots; sends intents
        ▲ View            │ intents
 GUI session (Go, gui/)         serialises calls, saves, builds View, preferences
        ▲                 │ semantic operations (incl. navigation requests)
 Application orchestration (app/)   legal actions, transitions, queries, message queue
        ▲                 │
 Rules and engine state (engine/, data/, scenarios/)   state, rules, Ruleset, save/load

 Persistent data: roster<N>.json (format unchanged) · campaign sidecar · preferences · backups
```

The diagram shows **logical layers only**. It says nothing about window
layout.

| Layer | Owns | Must not |
|---|---|---|
| **Rules and engine state** | Game state, every rule and random draw, ruleset values, roster load/save | Know about terminals, windows, terminology or timing |
| **Application orchestration** | The sequence of each player action. Which location transitions are legal, and their consequences (e.g. a party disbanding, if D2 decides so). Read-only queries that answer "what would happen if" with the same code as the action. The queue of game messages waiting to be shown | Draw, sleep or read keys; keep a second copy of state |
| **GUI session** | One game per window, call serialisation, saving after each action, building the `View`, backups, preferences I/O | Decide rules or legal transitions; change the location except through controller operations |
| **Presentation** | Layout, terminology, panels, animation, message pacing on screen, keyboard shortcuts | Hold or compute game rules; cache game state beyond the latest `View` |
| **Preferences and optional rules** | Per-user presentation preferences; per-campaign ruleset and enhancement choices | Change rules through presentation preferences |
| **Persistent data** | Roster files (format unchanged), sidecar files, backups | Mix user preferences into roster files |

### 2.2 Controller-first extraction (Target; D5 approved)

Continue the extraction already used for creation and the Tavern, one
location at a time, in the order the roadmap gives. For each location:

- **semantic operations:** one per player action, returning `*Refusal` with
  the game's text;
- **read-only queries** where the GUI needs to explain an option before the
  player chooses it, sharing code with the action (as `JoinRefusal` does);
- **terminal key map:** the terminal calls the new operations through its key
  map, and its golden tests pass unchanged;
- **no rule changes:** behaviour differences found in the compatibility audit
  are not fixed during extraction.

### 2.3 Navigation ownership (Target)

The controller ultimately owns legal game-location transitions and their
consequences.

- **The GUI requests; the controller decides.** The GUI asks for a
  destination, such as "go to the Tavern". The controller decides whether
  the move is allowed from the current phase, location and open prompt. It
  then applies the original route and side effects, and returns a refusal
  when the move is not allowed.
- **No direct location changes.** Neither the GUI session nor the
  presentation modifies location state outside approved controller
  operations.
- **Current code stays until its milestone.** `gui.Session.goTo` and its
  prompt-settling logic remain as they are. They move into the controller
  when the town navigation is extracted (roadmap M3, with Edge of Town).
- **Open:** the shape of the navigation operation. It could be a single
  `NavigateTo(destination)` or one operation per route. It must also be
  decided how open prompts are settled and how each refusal is reported.

### 2.4 Paced steps: message queue and `Advance()` (D6 approved in principle)

Rules never sleep. Steps that the original game paced in time (combat
messages, Inn healing, story pages) are modelled as a message queue plus an
`Advance()` operation. The terminal and the GUI each decide how long to show
a message. The original T)IME delay becomes a presentation preference.

**Open, to be designed and approved before implementation (roadmap M3):**

1. **Commit point:** when a gameplay action becomes committed. For example,
   is a combat round's outcome final when orders are confirmed, or step by
   step?
2. **What the queue holds:** whether queued messages describe outcomes that
   are already committed to game state, or carry pending state changes
   applied on `Advance()`.
3. **Saving mid-sequence:** how saving works during a partially displayed
   sequence: what is on disk while messages are still showing.
4. **Interruption:** how interruption, application closure and recovery are
   handled (e.g. quitting during combat or Inn healing, then reopening).
5. **Equivalence testing:** how deterministic tests prove that terminal and
   GUI reach equivalent outcomes under the same seed, whatever the pacing.

| Alternative | Trade-off |
|---|---|
| **Queue plus Advance (recommended)** | Deterministic and testable; both front ends pace freely; depends on the commit design above |
| Go-side timers pushing Wails events | Faithful timing, but couples rules to wall-clock time and to Wails; harder to test |
| Front end replays a full round result | Simple for display, but loses step-by-step state for team displays |

### 2.5 One state, many views; stable identity (Target)

- **One source of state:** `Session` holds the only `GameState`, and every
  call returns a full `View` snapshot.
- **Shared rendering:** every team display, roster view, player profile and
  location view renders parts of the same snapshot.
- **Front-end state:** Svelte keeps only interface state: what is open and
  where, focus, and text being typed.
- **Snapshot size:** if the `View` becomes large (maze geometry, combat
  logs), split it into named sections with a version counter rather than add
  a second state store.

**Stable character identity:**

- **Today:** profiles are referenced by roster index, and the party is saved
  by name. Index references are an implementation convenience, not an
  architectural requirement.
- **Target:** long-lived references (open profiles, selections, the campaign
  sidecar, future features) use a stable character identifier. Then
  deletion, reordering or renaming cannot make a reference point at the
  wrong character.
- **Open:** how identifiers are assigned and persisted. One option is an
  identifier in the campaign sidecar keyed to the roster; another is a new
  roster field, which would be a save-format change needing approval. Both
  need a migration path for existing rosters.
- **Timing:** assess at the milestone that first needs long-lived references
  (roadmap M2 or M3). Nothing changes now.

### 2.6 Configuration (D7 approved in principle)

**Presentation preferences**

- **Scope:** per user, stored in
  `~/Library/Application Support/Istary/preferences.json`.
- **Contents:** presentation style, message speed, layout state.
- **Safety:** they never change rules and are safe to reset.

**Ruleset**

- **Shape:** an `engine.Ruleset` value held by `GameState`. Its `Classic`
  preset reproduces the provisional baseline exactly.
- **Each option:**
  - has a stable identifier;
  - has a documentation entry in `docs/RULES.md`;
  - has a test proving that Classic is unchanged with the option off.
- **Rule decisions:** a historical rules decision (D1) that changes Classic
  itself is recorded as a decision with evidence, not as an option.

**Campaign file**

- **Location:** a sidecar `campaign.json` beside the roster file.
- **Contents:** scenario, ruleset identifier and enabled enhancements, plus a
  format version.
- **Format compatibility:** the roster format does not change, and the
  terminal ignores the sidecar.

All three are introduced only when the first option or enhancement requires
them.

| Alternative for rules options | Trade-off |
|---|---|
| **Ruleset value in the engine (recommended)** | One implementation; options explicit and testable |
| Options checked in the GUI | Duplicates rules in the interface; breaks charter principle 4 |
| Build tags or separate binaries | Can't switch per campaign; doubles the test matrix |
| Rules in scenario JSON | Good for data-like values (prices, tables), poor for behaviour |

| Alternative for campaign options | Trade-off |
|---|---|
| **Sidecar file (recommended)** | Roster format and terminal untouched |
| New fields in `roster<N>.json` | One file, but changes the pinned save format |
| Preferences only | Options would follow the user, not the roster they affect |

### 2.7 Persistence

- **Roster format:** keep `roster<N>.json` compatible, pinned by
  `engine/testdata/save_roster1.golden.json`.
- **Atomic writes, early:** write to a temporary file in the same directory,
  sync, then rename. The format stays the same. This is scheduled before
  substantial controller refactoring (roadmap M1b), so later state
  complexity is built on safe saves.
- **Backups:** keep the backup-before-destructive-action rule.
- **Before any public release:** decide on password storage (plain text
  today), a save-format version, and the user-visible save location.

## 3. Classic rules baseline (D1 amended)

- **Definition:** Classic is a **documented, historically verified Wizardry I
  ruleset selected for Istary**. Neither the 1987 Player's Guide nor the
  inherited engine is automatically authoritative.
- **Provisional baseline:** the current engine is the *provisional executable
  baseline*. Golden and characterization tests pin it, so any change is
  deliberate.
- **Open discrepancies:** they stay open in `WIZARDRY-COMPATIBILITY.md` until
  evidence supports a decision.
- **Decision records:** each decision records the historical revision, the
  evidence, the decision and the implementation change. Decisions go in the
  compatibility decision log (`WIZARDRY-COMPATIBILITY.md` §6).
- **Implementation:** an adopted rule change is made as a separate,
  behaviour-changing commit with tests. It updates the golden files only
  for the decided behaviour.

## 4. Wails v2 evaluation (D4 approved: retain v2 for now)

| Need | Wails v2 (2.17.0, current) | Assessment |
|---|---|---|
| A single main window containing all game views | One native window; any in-window arrangement is possible in the web view | Sufficient |
| Several native windows | Not supported in v2 | Needs v3 or native code, if a chosen design requires it |
| macOS menu bar, shortcuts, full screen, resize | Application menu and accelerators supported; full screen and zoom work since `mac.Options{}` was set (*current*) | Sufficient |
| Drawing the maze and combat | Any web technology (canvas, SVG, WebGL, CSS) from engine data | Sufficient; technology choice deferred |
| Background pacing | Pull model (`Advance()`), optional Wails events | Sufficient |
| Distribution (signing, notarisation) | Wails build plus Apple tools | Not tested |

Wails capabilities and the existing Svelte components are implementation
resources. They do not constrain the presentation design. If the design
chosen at the presentation checkpoint needs native multi-window, re-evaluate
Wails v3 at that point; this document does not assume its current status.

## 5. Interface architecture (functional structure only)

This section describes **what the interface must connect to**, not how it
looks. Layout, HUD design, panel behaviour, artwork and rendering are
deferred to the presentation design checkpoint (charter §6.3, roadmap M2).

| Functional element | Data and operations it uses | Fixed now | Deferred |
|---|---|---|---|
| Current location | Engine phase and location from `View`; navigation requests to the controller (§2.3) | The view follows engine state; navigation goes through the controller | How locations and navigation are shown |
| Team information | `View` party data: lineup order, front line (slots 1–3), HP, status | Readily accessible wherever it matters, including the Maze and combat | Form, position, size, modes (candidates: expanded formation, compact strip, hidden) |
| Roster and player profiles | `View` roster data; profile actions enabled from controller state per location (PG p.29) | Inspect and manage from the interface; one source of state | Fixed, floating, docked or modal; how many open; profile layout |
| Equipment | Shop operations; equip operations per player and, in camp, for the team | Full equip workflow | Presentation of slots and items |
| Maze | Engine visibility data for position and facing; visibility rules stay in Go | No map unless the automap enhancement is enabled (D9) | Rendering technology and visual treatment |
| Combat | Combat state, order operations, message queue (§2.4) | Original options and front-line limits; profiles read-only during combat (D8, subject to presentation design) | How orders, monsters and messages are shown |
| Confirmations and passwords | Controller prompts and refusals | Destructive actions are confirmed | Visual form |

## 6. Migration considerations

- **Terminal:** the terminal stays the executable reference for every
  location not yet migrated, and its golden tests stay the guard for every
  extraction.
- **Shared-data mutation:** `Scenario.Items[i].Stock` is changed in place by
  the shop. It should move into game state when Boltac's is extracted,
  keeping current behaviour.
- **Training Grounds and the party:** GUI-001 to GUI-003 show the Training
  Grounds and the party together, and the engine keeps the party when the
  Training Grounds are visited. D2 is deferred, so this stays as built. A
  later decision to disband could change that GUI behaviour.
- **Navigation:** moving `goTo` into the controller (§2.3) must keep
  GUI-003's tested transitions unchanged.
- **Stable identity:** stable identifiers (§2.5) need a migration plan before
  any format change.
- **Scenarios:** Wizardry II and III exist in the engine. The GUI stays
  Wizardry I only until a decision says otherwise.
- **Tests:** keep the headless end-to-end checks as a script in the
  repository once a presentation direction is approved.

## 7. Decision register

| ID | Decision | Disposition | Approved now | Deferred detail |
|---|---|---|---|---|
| D1 | Classic rules baseline | **Amended** (2026-11-06) | Classic is a historically verified ruleset; the current engine is the provisional executable baseline; no rules change now | Each discrepancy's decision, with revision, evidence and implementation change |
| D2 | Training Grounds disbands the party (PG p.24; engine does not) | **Deferred** pending historical evidence | Keep engine behaviour and GUI-003 as built | The rule itself |
| D3 | Gold pooling, Divvy Gold, Temple pooling, restart an OUT party | **Candidate missing Classic features; investigate** | Not automatically enhancements | Whether and how they join Classic |
| D4 | Wails v2 | **Approved:** retain for now | Stay on v2 | Re-evaluate only if an approved design needs native multi-window |
| D5 | Controller-first extraction | **Approved** | Each location is extracted before its GUI; terminal goldens unchanged | Order within milestones |
| D6 | Message queue and `Advance()` | **Approved in principle** | Rules never sleep; front ends pace | Commit semantics, saving mid-sequence, interruption and recovery, equivalence tests (§2.4) |
| D7 | Ruleset value and campaign sidecar | **Approved in principle** | The approach | Implement only when the first option requires it |
| D8 | Profiles viewable during combat, read-only | **Approved in principle** | No actions from profiles in combat | Whether and how they are shown, decided at the presentation checkpoint |
| D9 | Automap | **Approved:** it is a gameplay enhancement | Not shown in Classic; terminal map overlay left as is | When enhancements are built |
| D10 | Terminology | **Approved:** presentation only | Code and saves keep original names | Final wording |
| D11 | Reference PDF | **Approved:** excluded from Git | `.gitignore` excludes `docs/reference/` | — |
| D12 | Legal review | **Approved:** before public distribution | No distribution without it | Scope and timing |

Further decisions recorded on 2026-11-06:

- **Presentation:** presentation and graphics are deferred to a design
  checkpoint before M2.
- **Navigation:** the controller owns navigation (§2.3).
- **Stable identity:** stable character identity is the target (§2.5).
- **Atomic saves:** atomic save writes are scheduled early (§2.7).

## 8. Risks

| Risk | Mitigation |
|---|---|
| Extraction changes behaviour unnoticed | Golden and characterization tests before and after each extraction; refactors and rule changes in separate commits |
| Queue semantics chosen late cause rework | Design note on D6 commit semantics, reviewed before the first queued extraction (M3) |
| Interrupted writes corrupt a roster | Atomic writes in M1b, before controller refactoring |
| Index-based references point at the wrong character | Stable identifiers assessed before long-lived references are added |
| `View` grows too large or slow | Sectioned `View`; measure before optimising |
| The maze renderer is hard to move out of `render` | Technology spike after the presentation checkpoint (M6), using engine visibility data |
| Global `math/rand` and package state block concurrency | Keep one game per process; no goroutines in rules |
| Historical questions stall work | Extraction does not depend on them; discrepancies stay open and documented |
| Existing components shape the design by default | Presentation checkpoint compares alternatives before M2 implementation |
