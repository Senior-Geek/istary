# Experience concept

Status: **exploration, not approved.** Labels follow the README:
Approved, Fact, Hypothesis, Alternative, Decision (X-numbers), Risk.

## 1. The idea in one paragraph

**Hypothesis.** Istary has two halves.

- **Castle.** A calm, direct-manipulation management workspace in the spirit
  of a professional macOS tool. The user selects players, items and places,
  and acts on them in context.
- **Maze.** One continuous dungeon environment in which exploration, combat
  and close inspection all happen. The camera reframes for each activity, and
  the place stays the same.

Under both halves, the rules are Classic Wizardry I as verified, and nothing
the user sees exceeds what the team could know.

## 2. Approved constraints that shape everything

| Constraint | Source | Consequence for the experience |
|---|---|---|
| One engine, one state | Approved, charter §2.4 | Every view (team strip, profile, map, 3D scene) is a projection of the same Go state. Nothing in the presentation decides a rule. |
| Information follows the ruleset | Approved, charter §2.6 | The presentation's "knowledge" is bounded by the team's. Camera, map and close-ups all obey the same bound. |
| Presentation independent of rules | Approved, charter §5.2 | Every treatment (A, B, C) and every camera can be swapped without changing an outcome. |
| Controller owns actions and transitions | Approved, architecture §2.3 | The presentation asks; Go decides; the presentation animates what Go decided. |
| Paced steps via queue and `Advance()` | Approved in principle, D6 | Combat and other sequences are already resolved when shown; pacing is presentation. |
| Automap is an enhancement | Approved, D9 | Under Classic, the only map is one the user makes. |

## 3. Experience A: the management workspace (Fusion-inspired)

**Reference.** Autodesk Fusion's interaction philosophy, not its appearance:

- **Workspaces** that change the available commands (Design, Render,
  Manufacture), comparable to Castle locations.
- **A browser** of the objects in the document, comparable to the roster,
  the team, shop stock and inventories.
- **Selection first, then command**, with commands that act on the selection.
- **Contextual inspectors and command dialogs** that show only what applies.
- **Marking menus** (radial context menus on right-click) and a command
  search, so the user never walks a menu tree.
- **Direct manipulation on a canvas**, with immediate visual feedback.

**What it replaces.** The original's nested menu sequences:

- "Tavern → A)dd → type name → password";
- "Trading Post → who will enter? → B)uy → page → number → confirm".

These become object-oriented actions: select Ysolde, select the Long Sword,
choose *Buy for Ysolde*. The rules are still checked in Go; Svelte shows the
answer.

**Hypothesis: the interaction vocabulary.**

| Fusion idea | Istary use | Rules boundary |
|---|---|---|
| Workspace switcher | Castle locations (Training Grounds, Tavern, Trading Post, Temple, Inn, Edge of Town) | Switching location is a controller transition; refused when the rules refuse it |
| Browser tree | Team (front line, back line), roster with filters, shop stock | Lists only what the location lets the user see (e.g. unidentified items stay unidentified) |
| Selection-driven commands | Commands enabled by the current selection: Buy for…, Identify, Uncurse, Join, Remove, Cure | Enabled state and refusal reasons come from Go |
| Contextual inspector | Profile of the selected player; detail of the selected item | Read-only where the location requires it (e.g. in combat, D8) |
| Marking menu / ⌘K command search | Fast access to the same commands by pointer or keyboard | Same commands, same checks |
| Preview in command dialogs | Shows consequences the rules already disclose (price, "usable by", alignment refusal) | Never previews hidden values (e.g. effects of an unidentified item) |

**Alternatives** (decision X10; wireframes in the storyboard):

- **W-A: workspaces, browser and inspector.** A conventional pro-app frame
  with location tabs, a browser, a canvas for the location's objects, and an
  inspector for the selection.
  - **Strengths:** familiar, dense, keyboard-friendly.
  - **Risk:** it can feel like a spreadsheet.
- **W-B: the castle as a canvas.** Locations are objects on one surface, and
  the team is a token the user moves between them. A location panel explains
  what the place offers.
  - **Strengths:** spatial and characterful.
  - **Risk:** less dense; it needs art to work.

Neither fixes sidebar, inspector or toolbar positions. Both satisfy the
approved functional requirements (charter §6.1): team information is
accessible, players can be inspected, the location's choices and refusals are
clear, and the app behaves like a Mac app.

## 4. Experience B: dungeon exploration (simulator-inspired)

**Reference.** A flight simulator's camera system: first-person cockpit,
external view attached to the aircraft with adjustable distance, height and
angle, and quick view cycling. It does **not** include MSFS's free-flying
drone.

**Hypothesis:**

- **The same place, more ways to see it.** The dungeon is drawn as a
  continuous 3D environment. The user chooses a viewpoint:
  - **first person:** the team's eye, matching the Classic view;
  - **attached third person:** following a selected player, with adjustable
    distance, height and orbit, inside bounds (decision X4);
  - **elevated framing** for combat and inspection, which the context sets
    automatically.
- **The camera never changes the game.**
  - It is chosen independently of gameplay movement.
  - It never moves the team, turns it, spends light or triggers an
    encounter.
  - It never draws more than the Classic visible set (decision X3; see
    `CAMERA-AND-INTERACTION.md` §3).
- **Grid movement stays grid movement.**
  - "Forward" is still one square and "turn" is still 90°. Encounters, light
    and square effects are decided by Go on each step, exactly as now.
  - Presentation may animate the step or turn after Go has resolved it, or
    show it instantly (decision X6).
  - Continuous free movement is **not** proposed.

**Team information while exploring.**

- **Approved:** team information must stay accessible.
- **Not fixed:** the form. The candidates are the compact strip (treatment
  A), the information-rich table (treatment B), the Classic roster box
  (treatment C), or hidden.
- **Profiles:** a profile opens read-only from the team display or with keys
  1–6. Actions that the rules restrict to Camp, such as equip or reorder,
  stay in Camp.

## 5. Experience C: combat (traditional, round-based)

**Hypothesis:** combat happens where the team is.

1. **Encounter.** When Go starts an encounter, the camera eases from the
   exploration view to an elevated framing of the team and the opponents in
   the same corridor, with the same light. No separate battle screen
   (decision X7).
2. **Orders.** The user gives every member an order before anything happens,
   as in Classic: Fight, Parry, Spell, Use, Dispel, Run.
   - **Go supplies:** which orders each member may give (front or back line,
     known spells, usable items) and the valid targets.
   - **Selecting:** the user selects the actor and then the target group, by
     pointer on the scene, by keys, or from a list.
   - **Changing orders:** "Back" revisits earlier orders until the round is
     confirmed.
3. **Resolution.** Go resolves the whole round at once (Fact:
   `engine/combat.go:603`, `ExecuteRound`). The presentation steps through
   the resulting messages in initiative order:
   - each step may briefly focus the camera on actor and target;
   - it shows damage, spell effects and status changes as the message says
     they happened.
   - **Team values:** HP, status and monster counts change on screen only
     when the message that caused them is shown (Fact: the engine's display
     snapshot, `DisplayPartySnap`).
   - **Not real time:** sequential presentation is not real time, and it is
     not alternating individual turns.
4. **Next round or end.** Another round returns to orders. Victory leads to
   the chest, if any, then back to exploration. Defeat or fleeing follows
   Go's result.

**Monster identity.**

- Groups are named as Go names them: unidentified names until the per-round
  identification roll succeeds (Fact: `engine/combat.go:622`).
- Monster models or pictures must not disclose identity. An unidentified
  group uses an appearance that does not give away the true monster. Whether
  each unidentified name gets its own look is part of the art decision
  (X16).

**Staging.**

- **Fact:** Wizardry combat has no positions beyond the team's lines and the
  order of monster groups.
- **Hypothesis:** staging places groups in group order ahead of the team.
  This is purely presentation.
- **Prohibition:** staging must never suggest range, cover or flanking.

## 6. Contextual object interaction

**Hypothesis:** inspection is a closer camera in the same scene, with the
actions the rules allow.

- **Treasure chests** (Classic, after combat; Fact: `engine/combat.go:2938–3300`):
  - **Actions:** Open, Inspect, Calfo, Disarm, Leave, with "who acts"
    chosen per action.
  - **Close-up:** the camera moves close to the chest.
  - **What the team knows:** a record of *reports*. For example, an Inspect
    result may be wrong; Fact: on a failed inspection the engine shows a
    random trap name.
  - **The chest's look:** never varies with its real trap.
- **Doors:**
  - **Classic Wizardry I doors** have no locks or traps. Kicking or walking
    through is the whole interaction (Fact: `engine/game.go:221`).
  - **A door close-up** may only frame what is already visible.
  - **Hidden doors** follow the Classic visibility rule; see
    `CAMERA-AND-INTERACTION.md` §3.4.
- **Other features:**
  - **Stairs, messages on walls and button panels** may be framed when Go
    presents them. The framing adds no information or actions.
  - **Searching the square** for lost characters (I)nspect) is a Go action.
    The camera may look down at the floor while it runs.

**Locked or trapped doors** would be a new mechanic. They are not Classic and
are not proposed (decision X9).

## 7. Navigation map

**Reference.** The DCS World F10 map and kneeboard. The map is always
reachable without leaving the cockpit, as a full overlay or a smaller panel.
The user can zoom, choose the orientation, see their own position, add marks
and annotations, and switch pages.

**Two kinds of map, kept distinct:**

| | Manual map | Automatic exploration map |
|---|---|---|
| Rules status | **Presentation tool.** It records only what the user draws, like graph paper, so it is valid under Classic | **Gameplay enhancement** (D9), off under Classic |
| Content | Walls, doors, notes, stairs and markers placed by the user; may be wrong | Squares visited and edges seen, recorded by Go from the visible set at each step |
| Position marker | The user's own marker. Optionally, a DUMAPIC result can be offered for the user to place | Depends on decision X13: perceived movement or true position |
| Floors | Sheets the user creates and names | Levels the team has visited |
| Persistence | Per campaign (decision X14) | Per campaign, with the enhancement recorded (D7 sidecar) |

**Presentation alternatives** (decision X12, all in the storyboard):

- **hidden**, with one key to show it;
- **docked** side panel;
- **floating** in-window panel;
- **full overlay** that dims the view.

None is approved. In every mode the map is open during exploration, with no
change of gameplay location and no game time spent.

**Controls explored:**

- zoom;
- north-up or heading-up. Under Classic, heading-up is only possible for the
  manual map relative to the user's own marker, since the game does not
  disclose facing outside DUMAPIC;
- current position;
- annotations;
- floor selection.

**Never revealed:**

- geometry the team has not seen;
- hidden doors not yet seen;
- dark squares' contents;
- special squares the team did not experience;
- the true position after a teleport or spinner, unless X13 chooses the
  stronger option.

## 8. From management to adventure and back

**Hypothesis:** the journey, sketched in the storyboard's transition strip.

1. **Prepare.** At Edge of Town, a readiness summary from Go: members,
   lineup, refusals such as a dead member, and gold. Nothing is fixed
   automatically.
2. **Enter.** The user chooses Maze. The controller makes the transition, and
   the presentation animates it, for example a short camera move from the
   Castle to the stairs. The animation can be skipped and never delays the
   rules.
3. **Switch controls.**
   - **Exploration context:** the input context changes to exploration
     (movement keys, Camp, camera, map). Management shortcuts that make no
     sense in the Maze are disabled, with a reason.
4. **Character access while exploring.** Profiles are read-only in the Maze.
   Camp offers the actions the rules allow there: equip, reorder, cast,
   inspect.
5. **Return.** Taking the stairs out, a warp or an escape spell returns the
   team to the Castle through the controller. The workspace reopens at the
   location Go reports, with the user's workspace layout restored.

**Not fixed:**

- HUD placement;
- team layout;
- window arrangement;
- whether the Castle and the Maze share one window layout or switch layouts.

## 9. What this concept does not decide

- **Renderer:** see `RENDERING-OPTIONS.md`; X15.
- **Art direction:** X16.
- **Classic compatibility questions:** for example XP sharing, Disarm with
  trap-name entry, and light ticking on non-movement keys. These stay in
  `WIZARDRY-COMPATIBILITY.md`.
- **D6 commit semantics:** the combat replay works under either answer,
  because it shows results already decided.
