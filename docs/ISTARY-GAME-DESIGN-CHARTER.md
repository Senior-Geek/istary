# Istary — Game Design Charter

*Istary — A Classic Dungeon Adventure for macOS* (working subtitle)

Status: **design baseline, approved direction** (reviewed 2026-11-06).
Nothing here is implemented unless it says so. Companion documents:

- `WIZARDRY-COMPATIBILITY.md` (evidence);
- `ISTARY-ARCHITECTURE-PROPOSAL.md` (structure and decision register);
- `ISTARY-DEVELOPMENT-ROADMAP.md` (sequence).

This charter fixes Istary's **functional philosophy**. It deliberately does
**not** fix its graphical presentation: layout, HUD design, artwork and
rendering are open and will be chosen at a presentation design checkpoint
(section 6.3).

## 1. Identity

Istary is a native macOS team-management dungeon RPG built on classic
Wizardry I gameplay. The user manages a team of adventurers: recruits them,
picks a starting lineup, equips them, sends them into the Maze, and brings
them home to recover and develop.

Istary is:

- a polished desktop game that feels at home on macOS;
- faithful to classic Wizardry I rules by default (section 5.1);
- clearer to use than the 1981 interface, without hiding the game's
  difficulty.

Istary is not:

- a pixel-accurate Apple II reproduction or an emulator;
- a sports simulation (sports terms are a navigation aid only);
- a rewrite of the inherited Go engine;
- a product with a public release date. Distribution is kept possible, not
  scheduled.

## 2. Design principles

1. **Rules are one thing; presentation is another.** The same game rules run
   under every presentation. Changing how something looks never changes what
   it does.
2. **Classic rules are documented and evidence-based.** Neither the 1987
   Player's Guide nor the inherited engine is automatically authoritative.
   Differences are recorded in `WIZARDRY-COMPATIBILITY.md` and decided
   explicitly, never resolved silently (section 5.1).
3. **Enhancements are opt-in, named, documented and tested.** Anything that
   changes the gameplay experience (automap, extra information, difficulty)
   is an individually switchable option, and every one is off under the
   Classic ruleset.
4. **One engine.** All presentations and rulesets share one rules engine and
   one game state. There is no second implementation of any rule in the
   interface.
5. **Complete workflows before polish.** A feature is done when the user can
   finish the whole task (e.g. recruit → join → equip → enter the Maze), not
   when a screen exists.
6. **Information follows the ruleset, not the presentation.** A presentation
   may explain what the selected ruleset already lets the user know
   (requirements, refusals, consequences). It never reveals more (identities
   of unidentified items, disguised monsters, the map) unless an enabled
   enhancement allows it.
7. **The user's data is safe.** Saves are never lost silently; destructive
   actions are confirmed and backed up; development data is isolated.
8. **Presentation stays open until it is designed.** Existing components and
   framework capabilities are implementation resources, not constraints on
   the eventual visual design.

## 3. Gameplay cycle

**Recruit → Select Team → Equip → Explore → Recover → Develop → Repeat**

| Step | Where | What happens |
|---|---|---|
| Recruit | Training Grounds | Create players; inspect, change class, set password, delete |
| Select team | Gilgamesh's Tavern | Choose up to six active players; alignment restrictions apply |
| Equip | Boltac's Trading Post | Buy, sell, identify, uncurse; equip from the player profile |
| Explore | The Maze (via Edge of Town) | Move, camp, fight, open chests, map |
| Recover | Temple of Cant, Adventurer's Inn | Cure, resurrect, heal hit points, restore spells |
| Develop | Adventurer's Inn, Training Grounds | Make levels, learn spells, change class |

## 4. Terminology

Sports-team terms are used in the interface where they make navigation
clearer. Traditional fantasy names are kept for places. The code and the save
files keep the original terms (`Party`, `Roster`, `Character`). Terminology
is presentation only (decision D10).

**"Player" means an adventuring character.** In the interface, a *player* is
a member of the roster or the team, never the human using the app. The
documentation calls the human the **user**.

| Traditional term | Istary term (interface) |
|---|---|
| Party | Team |
| Party members | Active players |
| Roster | Player roster |
| Party formation / marching order | Starting lineup |
| Front rank (slots 1–3) | Front line |
| Back rank (slots 4–6) | Back line |
| Character sheet | Player profile |
| Character attributes | Player statistics |

The Player's Guide itself uses "team" freely (e.g. PG p.21, p.52–55), so this
is a change of emphasis, not of meaning.

| Place | Function the interface may explain |
|---|---|
| Training Grounds | Recruitment and Player Development |
| Gilgamesh's Tavern | Team Management |
| Boltac's Trading Post | Equipment and Supplies |
| Temple of Cant | Medical Treatment and Resurrection |
| Adventurer's Inn | Rest, Recovery and Advancement |
| Edge of Town | Departures (Maze, Training Grounds, leave game) |
| Maze | Exploration and Combat |

Rule messages from the game (e.g. `** BAD ALIGNMENT **`) keep their original
wording in Classic presentation. Modern presentation may add a plain-language
explanation beside them; it does not replace the rule that produced them.

## 5. Rules, presentation and enhancements

Three independent settings:

| Concern | Examples | Who may change it |
|---|---|---|
| **Ruleset** | stat minimums, combat, spells, prices, password on join | The ruleset only. Classic is the default |
| **Presentation** | layout, terminology, panels, fonts, art, sound, message speed | The user, at any time, with no effect on rules, difficulty, available information or saves |
| **Optional enhancements** | automap, showing monster identities, difficulty changes | The user, per campaign, each one named and documented; all off under Classic |

### 5.1 Classic rules

**Classic is a documented, historically verified Wizardry I ruleset selected
for Istary** (decision D1).

- **Current baseline:** the inherited Go engine is the *provisional
  executable baseline* until each historical discrepancy has been
  investigated and decided. Until then, existing rules are not changed.
- **Disagreements:** where the Player's Guide and the engine disagree, the
  discrepancy stays open in `WIZARDRY-COMPATIBILITY.md` until evidence
  supports a decision.
- **Decision records:** each decision records:
  - the historical program revision it refers to;
  - the evidence;
  - the decision;
  - any resulting implementation change, with tests.
- **Missing features:** features documented in the guide but missing from
  the engine (gold pooling, Divvy Gold, restarting an OUT party) are
  candidate missing Classic features, not automatically enhancements
  (decision D3).

### 5.2 Classic rules versus Classic presentation

These are independent.

- **Classic rules** are the ruleset of section 5.1.
- **Classic presentation** is an interface style that stays closer to the
  original's menus, wording and look.
- **Any combination is valid.** A user may play Classic rules with Modern
  presentation, or Classic presentation with enhancements enabled.
- **Presentation changes nothing else.** Switching presentation never changes
  game mechanics, difficulty, or the information legitimately available
  under the selected ruleset and enhancements.

### 5.3 Rules of thumb

- **Enhancement or presentation?** If a setting can change who wins a fight,
  what an item costs, what a player can do or what the user can know, it is
  an enhancement, not presentation.
- **Where settings live:**
  - presentation settings are per user;
  - enhancements are per campaign, saved with the roster they affect, so a
    roster played under changed rules is identifiable.
- **The terminal as reference:** the terminal game remains the executable
  reference for the provisional baseline until the GUI covers the whole
  game.

## 6. Interface

### 6.1 Approved functional requirements

- **Team information is readily accessible** in every location where it
  matters, including the Maze and combat.
- **The user can inspect characters and manage the team**: roster, player
  profiles, team selection, lineup and equipment, within the limits the
  ruleset sets for each location.
- **Location-specific gameplay is presented clearly**: what the current place
  offers, what is possible now and why something is refused.
- **macOS desktop interaction**: menu bar, keyboard shortcuts, full screen,
  window resizing, standard confirmation for destructive actions, system
  appearance.
- **Every action is reachable by keyboard; status is never shown by colour
  alone.**
- **Presentation never changes gameplay** (section 5.2).
- **One source of game state**: views of the same character or team never
  disagree.

### 6.2 Deferred presentation decisions

These are open and must not be fixed by implementation before the
presentation design checkpoint:

- **Layout:** layout and placement of regions, including any HUD; sidebars
  versus bottom areas.
- **Visual hierarchy and navigation style.**
- **HUD:** its design, modes, size and orientation.
- **Panels:** behaviour (fixed, floating, docked, modal) and how many may be
  open.
- **Player profiles:** their layout.
- **Visual style:** portraits, artwork, graphical style and animation.
- **Rendering:** the Maze and combat rendering technology and visual
  treatment.
- **Classic and Modern:** their visual appearance.

Candidates already discussed are recorded as **candidates only**:

- a bottom or right-hand team HUD;
- a 3×2 front-line/back-line formation;
- a compact horizontal strip;
- a hidden HUD;
- movable in-window panels (the current GUI's floating character sheet).

### 6.3 Presentation design checkpoint

Before the main-window layout is implemented (roadmap M2):

- **Evaluate alternatives:** compare alternative layouts and interaction
  models, using mockups or prototypes.
- **Approve and record:** the user approves one direction, recorded in
  section 8.
- **Build after approval:** implementation follows the approved direction.

No mockups or presentation changes are made before that checkpoint.

## 7. Scope

In scope now:

- Wizardry I (Proving Grounds of the Mad Overlord) on the inherited engine.
- The full castle, maze, camp and combat loop in the macOS GUI.
- Classic rules (as decided) with Classic and Modern presentation.

Later, by decision:

- Optional gameplay enhancements.
- Wizardry II and III scenarios in the GUI (the engine and terminal already
  support them).
- Original art, music and sound.
- Public distribution (signing, notarisation, legal review: D12).

Out of scope:

- Rule changes without an explicit, recorded decision.
- Online features, multiplayer, accounts.
- Windows and Linux GUI builds as a goal (Wails could make them possible).

## 8. Decision record

Decisions that change this charter are recorded here. The full register of
architectural decisions is in `ISTARY-ARCHITECTURE-PROPOSAL.md` §7.

| Date | Decision |
|---|---|
| 2026-11-06 | Charter drafted. |
| 2026-11-06 | Direction approved. Classic is a historically verified ruleset; the engine is provisional (D1). Classic rules and Classic presentation are independent. "Player" means an adventuring character. Presentation and graphics are deferred to a design checkpoint before M2. |
