# Camera and interaction

Status: **exploration, not approved.** Labels follow the README.

This document proposes how a camera-driven presentation could sit on top of
the existing engine without changing a rule, a random roll or an action
sequence.

## 1. Layered model (Hypothesis)

```
 ┌─────────────────────────────────────────────────────────────────────┐
 │ 1. Authoritative game state (Go engine)                            │
 │    position, facing, level, light, combat state, chest, roster…    │
 ├─────────────────────────────────────────────────────────────────────┤
 │ 2. Gameplay interaction context (Go controller)                    │
 │    Exploration · Combat (orders / resolution) · Chest · Camp ·     │
 │    prompts. Lists legal actions, refusals, the visible set, and    │
 │    the knowledge records (map, reports).                           │
 ╞═══════════════════════  View snapshot (JSON)  ══════════════════════╡
 │ 3. Camera target and configuration (front end, presentation)       │
 │    mode, attached player, distance/height/orbit, context framing   │
 ├─────────────────────────────────────────────────────────────────────┤
 │ 4. Visual presentation and animation (front end, renderer)         │
 │    scene built only from the View; tweens; message pacing          │
 ├─────────────────────────────────────────────────────────────────────┤
 │ 5. User input (front end → controller)                             │
 │    keys/pointer mapped per context; gameplay requests go to Go,    │
 │    camera/map/UI requests stay local                               │
 └─────────────────────────────────────────────────────────────────────┘
```

**Rules of the model:**

1. **Layers 1–2 are Go; layers 3–5 are presentation.** Data flows down only
   as the `View` snapshot (Approved: `gui.Session` returns View snapshots).
2. **The View holds only what the user may know.** The renderer cannot leak
   what it never receives.
   - **Example:** the chest's real trap type stays in layer 1 and is never
     serialised.
   - **Fact:** today `CombatState.TrapType` is a plain engine field, so a
     future combat View must deliberately exclude it.
3. **Input splits in two.**
   - **Gameplay requests** (forward, turn, kick, an order, open chest) go to
     the controller.
   - **Presentation requests** (cycle camera, zoom map, open a profile
     read-only, skip animation) never reach the engine.
4. **The presentation animates what Go decided.**
   - **Order:** a step animation starts after Go reports the new position,
     never before.
   - **Rejected moves:** if Go rejects a move (a wall: "OUCH!"), the
     presentation may play a bump, but it never shows the team moving.
5. **The camera reads layer 2's visible set; it never computes visibility
   itself** (§3).

### 1.1 Interaction contexts

These are contexts, not separate application screens. The 3D scene persists
across them.

| Context | Entered when (Go decides) | Legal gameplay input (Go lists) | Default camera (presentation) |
|---|---|---|---|
| Exploration | Team in the Maze, no prompt pending | Forward, turn, kick, camp, search square, status | User's chosen exploration camera |
| Combat: orders | Encounter begins, or a new round | Orders for the current actor, back, confirm | Elevated framing of team and groups |
| Combat: resolution | Round confirmed | Advance, show all | Framing follows message actor and target |
| Inspection (chest) | Victory with a chest | Open, Inspect, Calfo, Disarm, Leave, choose who | Close-up on the chest |
| Object interaction | Go presents a prompt tied to a feature (buttons, stairs, wall message, search) | The prompt's answers | Framing on the feature, if any |
| Camp | Team camps | Camp actions | Team at rest, or the existing panel UI |

**Rules for contexts:**

- **Transitions are owned by Go.** The front end changes context only when a
  new View says so.
- **Camera transitions follow, and never cause, a context change.**

## 2. Why camera transitions cannot change the game

**Facts that make this safe today:**

| Fact | Reference | Consequence |
|---|---|---|
| Encounters roll only after a successful move | `cmd/istary/main.go:3121–3185` (`if moved {…}`) | Camera motion is not a move. No roll can occur. |
| Light decrements once per handled maze key, not per frame | `cmd/istary/main.go:3116` | Redrawing does not spend light; the GUI must never send a "redraw" request to Go. |
| A combat round is resolved in one call | `engine/combat.go:603` `ExecuteRound` | Replay pacing cannot alter outcomes or ordering. |
| Display snapshot frozen at round start | `engine/combat.go` `DisplayPartySnap`, `DisplayAliveCounts` | Defines when team values may visibly change during replay. |
| Party order re-sorts by status each round | `engine/combat.go:610` `sortPartyByStatus` | The figures' formation must follow the View's order at each round start. |

**Hazards to design out:**

1. **Rendering must not consume game randomness.**
   - **Fact:** the terminal renderer decides whether an unlit hidden door is
     drawn with `rand.Intn(6)` on the shared source (`render/dungeon.go:178`).
     Every redraw advances the game's random sequence.
   - **Risk:** a 60 fps renderer would perturb every later roll. Today, the
     number of terminal redraws already influences outcomes.
   - **Hypothesis:** the controller decides "hidden door visible this step"
     once per handled key and puts the result in the View.
   - **Decision:** whether this changes the terminal's random sequence, and
     whether that matters for Classic, is a compatibility question to log.
     It must be decided before the M5/M6 maze work. This exploration does
     not resolve it.
2. **Light must not tick for presentation actions.**
   - **Fact:** the terminal's map key passes through the light decrement.
   - **Requirement:** in the GUI, opening a map, cycling a camera or opening
     a profile is presentation and sends nothing to Go.
   - **Open:** whether non-movement keys (map, status, quick plot, time)
     tick light in Classic is a compatibility question, not resolved here.
3. **Silent changes must stay silent.**
   - **Fact:** spinners set a random facing and transfers move the team,
     without a message (`cmd/istary/main.go:3407–3435`).
   - **Requirement:** an animated turn or a camera swing would announce
     the spin, and a smooth glide would announce the teleport.
   - **Hypothesis:** the View marks each change as *perceived*, which is
     animated, or *unperceived*, which is shown as an instant cut, in the
     same way Classic simply redraws.
4. **Animation must not block or reorder input.**
   - **Hypothesis:** while a step animation plays, the next gameplay key is
     either queued (one deep) or completes the current animation instantly
     and then applies.
   - **Never:** a key is dropped silently, or two keys are sent as one
     request.
   - **Encounters during animation:** an encounter that Go reports at the
     end of a step interrupts the queue; the queued key is discarded with a
     visible cue.

## 3. Visibility

### 3.1 Four layers of knowledge (Hypothesis)

Every element shown in the dungeon is classified as follows.

| Layer | Meaning | Where it lives | Example (chest) |
|---|---|---|---|
| 1. Engine knows | True state | Go engine only | The trap type |
| 2. Team has discovered | What the team has seen, been told or reported | Go controller, as knowledge records in the View | "Pell inspected: POISON NEEDLE" (may be false); Calfo not cast |
| 3. User may see | What the selected rules and enhancements let the user see | The View | The chest, the reports, who already inspected |
| 4. User may attempt | Legal actions now | The View's action list | Open; Inspect (not Pell); Calfo (Brannoc, Corvin); Disarm; Leave |

The presentation may **explain** layers 3–4 more clearly than the original;
for example, it may say "inspection can misreport", because the rules make
that the case. It may never **derive** layer 1. The real trap type is
therefore never sent to the front end.

### 3.2 The Classic visible set (proposed binding rule, decision X3)

**Fact.** The Classic view (`render/dungeon.go:191`, `DrawMaze`) walks
forward from the team's square, up to a light distance:

- 2 squares without light;
- 5 with MILWA or LOMILWA (3 with quick plot).

At each depth it draws:

- the left and right edges, or, where a side is open, the front edge of the
  side square;
- the front edge, stopping at the first front edge that is not open.

A dark square stops the walk.

**Hypothesis.** Define the **visible set** as exactly those edges, plus the
floors and ceilings of the squares walked. Then:

- the controller computes it, per position, facing, light and level, once
  per handled key;
- every camera and renderer draws only the visible set, the team, and the
  objects Go places (opponents, chest);
- everything else renders as darkness.

**Consequences:**

- **Other cameras:** an attached or elevated camera shows the same walls as
  first person, from another angle. Moving the camera never discovers a wall.
- **Through an opening:** the front wall of the side square is drawn; the
  space beyond it is not, even though a 3D view would naturally show it. The
  storyboard's niche demonstrates this.
- **Behind the team:** the squares behind the team, and the team's own back
  edge, are not in the set. A camera must not look at them (§4.2).
- **Darkness:** dark squares show nothing beyond, as in Classic. The team
  itself may still be drawn (presentation of known facts).

### 3.3 Information in the HUD and map

The same bound applies to overlays.

- **Compass, facing and coordinates:** none under Classic. The original
  discloses position and facing only through DUMAPIC; a DUMAPIC result may be
  shown when cast.
- **Light:** the visible depth already shows the light. Any explicit "light
  remaining" readout would disclose a hidden duration, so it would be an
  enhancement.
- **Monster names:** the unidentified names, as Go provides them.
- **Map:** see §6.

### 3.4 Hidden doors

**Fact.** A hidden door is drawn when the team has light, or by chance
without light (`render/dungeon.go:178`).

**Hypothesis.** The decision is made in Go per handled key and appears in the
visible set as "door" or "wall". The renderer never knows a hidden door
exists unless the set says it is visible this step. Whether the
door, once seen, remains on an automatic map is part of decision X13 and the
automap design.

## 4. Camera behaviour

### 4.1 Modes

| Mode | Description | Status |
|---|---|---|
| First person | The team's eye at the rear of its square, looking along the facing; equivalent to the Classic view | Hypothesis: always available |
| Attached third person | Follows a selected player (1–6), with adjustable distance, height and orbit, inside bounds | Hypothesis; bounds are decision X4 |
| Context framing | Automatic shots for combat orders, combat replay, chest close-up, object framing | Hypothesis |
| Elevated or overhead | A higher view of the team's surroundings | Decision X5: limited to the visible set, or only with the automap enhancement |
| Free-flying | Detached camera | **Not proposed** |

### 4.2 Attached-camera bounds (decision X4)

- **(a) Inside the team's square.** The camera may move anywhere within the
  team's own square below the ceiling, and collides with its walls.
  - **Effect:** it never looks at an unseen square behind.
  - **Cost:** the square is small, so with six figures inside it the framing
    is tight (see the sandbox).
- **(b) Trailing into the square behind.** This gives better framing, but
  the side walls of the square behind are not in the visible set.
  - **Display:** they render as darkness or fog, which reads as "unknown".
  - **Open:** whether that is acceptable is a judgement for the user.
- **(c) Free orbit within a radius.** Exposes unseen space in most
  directions. **Not recommended.**

**Common rules:**

- **What the camera follows:** a *player* (presentation choice), but the
  formation within the square is presentation only.
  - **Fact:** the rules know front line (1–3) and back line (4–6) and
    nothing else.
- **Ownership:** the attached player is a presentation preference. It
  persists in user preferences, not in the save.
- **If the attached player dies or leaves:** the camera falls back to first
  person or the next player, with no rule effect.

### 4.3 Smooth transitions for grid movement (decision X6)

| Event (Go-reported) | Perceived? | Proposed presentation |
|---|---|---|
| Forward succeeded | Yes | Short glide one square (e.g. 150–250 ms, preference); camera follows |
| Forward blocked ("OUCH!") | Yes | Small bump, sound; no displacement |
| Turn left or right | Yes | 90° rotation tween of team and camera |
| Kick through door | Yes | Door-burst cue, glide one square |
| Stairs, chute, pit, teleport, spinner | Per Go | Perceived: a fade or cut with a message. Unperceived: an instant cut, no motion |
| Encounter after a step | Yes | The step completes, then the camera reframes into combat |

Under Classic presentation, the default may be instant redraws as in the
original. Animation speed is a presentation preference and never affects
timing in the rules (Approved: rules never sleep).

**Wrapping.** Fact: levels are 20×20 and wrap (`engine/game.go:204`). The
renderer builds geometry relative to the team's square (local coordinates),
so wrapping needs no special case in the scene. Maps must handle it (§6).

### 4.4 Combat framing

- **Opponent placement.** Monster groups (up to four) are placed ahead of the
  team in group order. This is staging only; it implies no distance.
- **Framing by phase:**
  - **orders:** a wide elevated shot that keeps every member and every group
    readable;
  - **resolution:** per-message focus, as a short ease toward actor and
    target, then back. Each focus is skippable.
- **The scene stays in the corridor.** Combat uses the same visible set as
  exploration. An encounter in a narrow corridor stays in that corridor; no
  arena is invented (decision X7).
- **Identity on screen:** the team's on-screen order follows the View's order
  at each round start, including the status re-sort.

### 4.5 Inspection framing

- **The approach.** The camera eases to a close shot of the chest, with the
  acting player (when chosen) beside it.
- **The look.** Same model and same lighting whatever the chest contains.
  Trap animations play only when Go reports a trap firing, and then show the
  trap Go names in its message.
- **Leaving.** Leaving, opening or completing returns the camera to the
  exploration camera the user had before the encounter.

## 5. Input

| Input | Context | Goes to |
|---|---|---|
| Arrow keys / WASD equivalents, K, C, I | Exploration | Controller (gameplay) |
| V (cycle camera), [ and ] (distance), drag (orbit) | Exploration | Presentation only |
| M (map), map zoom, pan, annotate | Exploration (and Camp) | Presentation only, plus the map store for manual maps (§6) |
| 1–6 (profile, read-only) | Exploration, Combat | Presentation only (D8) |
| Order keys (F, P, S, U, D, R), group number, click a group | Combat orders | Controller |
| Return / Shift-Return | Combat resolution | Presentation (advance and pacing) and, when the queue is empty, the controller (`Advance()`) |
| O, I, C, D, L, choose who | Chest | Controller |

**Rules for input:**

- **Fixed keys:** every gameplay action is keyboard-reachable (Approved).
- **Rebinding:** key bindings are a presentation preference. The Classic
  letters stay available.
- **Mouse and trackpad** select targets in the scene. A click on a monster
  group is translated to the same "group N" request a key would make.

## 6. Navigation map: knowledge rules

### 6.1 Manual map (presentation tool; valid under Classic)

- **Content.** Only user input: walls, doors, notes, markers and sheets.
  - **Never from Go:** the game never writes into it, corrects it or snaps
    it to the level.
- **Helpers.** Optional, and they must not leak:
  - a grid;
  - a "place DUMAPIC result" button that appears only after DUMAPIC is cast
    and copies the displayed coordinates;
  - wrap-aware editing (20×20 torus) is acceptable, because the level size is
    common knowledge.
- **No input from position.** It never auto-follows the team. Moving the
  user's marker is a user action, possibly bound to a single key per step,
  but never inferred from Go's position.

### 6.2 Automatic exploration map (enhancement, D9)

- **Recording.**
  - **What:** at each handled step, the controller adds the current visible
    set (edges seen, squares walked) to a per-level exploration record.
  - **Not from level data:** the map is drawn from that record only, never
    from the level data. Hidden doors appear only once seen; dark squares
    record nothing beyond.
- **Position (decision X13).**
  - **Perceived movement (proposed):** the record follows the team's own
    sense of movement. After an unperceived teleport or spin, the new
    observations go to a new disconnected fragment, until a DUMAPIC fix or
    a recognised landmark reconnects them.
  - **True position:** a stronger enhancement that reveals spinners and
    teleports. If offered, it is a separate named option.
- **Mid-flight rule changes.** Recording happens even when the enhancement is
  off. This avoids the problem of a map that cannot reconstruct the past when
  the enhancement is enabled mid-campaign. Whether such recording is
  allowed, and whether a campaign that ever enabled it is marked, falls under
  D7.

### 6.3 Presentation alternatives (decision X12)

| Mode | Strengths | Risks |
|---|---|---|
| Hidden (key to show) | Immersive; nothing on screen | Discoverability |
| Docked side panel | Always visible; no occlusion | Takes width; squeezes the view (visible in treatment B + docked) |
| Floating panel | User positions it; reuses the existing in-window floating panel component idea | Occludes the scene; needs size and position memory |
| Full overlay | Room for annotation and multiple floors; DCS F10-like | Hides the scene; must not pause or alter game time (it does not: the game is turn-based) |

**Controls explored:**

- zoom;
- north-up or heading-up. Heading-up for the automatic map follows perceived
  facing; for the manual map it follows the user's marker;
- current position;
- annotations: notes and pins, on both kinds;
- floor selection: only visited levels, or the user's own sheets.

## 7. Architecture and persistence implications

None of this is approved. These are the consequences that would follow.

1. **Controller extraction first.**
   - **Fact:** maze movement, encounter rolls, light and square effects are
     in `cmd/istary/main.go`.
   - **Need:** the roadmap's M5 (expedition in the controller) is a hard
     prerequisite for any GUI maze or camera work. Terminal goldens must stay
     unchanged.
2. **New View content** (controller-produced). It would need:
   - **the visible set** (edges, floors, the dark stop) for the current
     square and facing, with hidden-door visibility already decided;
   - **change annotations:** perceived or unperceived, for animation;
   - **context and legal actions**, for exploration, combat orders, combat
     resolution, chest and prompts;
   - **knowledge records:** chest reports, monster identification state,
     the exploration record (if recorded), and DUMAPIC results;
   - **exclusions:** the true trap type and the unidentified monster
     identity must be absent from the View, and unit tests should assert
     their absence.
3. **The D6 queue and `Advance()`.** These must carry combat messages with
   the state deltas each message reveals: the engine's display snapshot
   already approximates this. The D6 commit-semantics design (still open)
   must account for "round already resolved, display not yet complete" when
   saving or quitting mid-replay.
   - **Correction (combat review):** the display snapshot does not
     approximate per-message deltas. It freezes values for the whole round.
     Per-message deltas would be new structured events; see
     `COMBAT-EXPERIENCE-REVIEW.md` §7.
4. **Stable identity.**
   - **Need:** the camera's attached player and map annotations that name
     players need stable character identity (architecture §2.5), not
     roster or party index. Party order changes in combat.
5. **Persistence** (decision X14).

   | Data | Proposed home | Notes |
   |---|---|---|
   | Camera mode, distances, attached-player preference, animation speed, map placement | User preferences (presentation) | Never in saves; no rule effect |
   | Manual maps (sheets, strokes, notes, markers) | Per-campaign sidecar next to the roster (D7 approach) | Atomic writes (M1b); versioned format; backups. Not in the engine save format |
   | Exploration record (automatic map) | Per-campaign sidecar | Mark the campaign if the enhancement was ever enabled (D7) |
   | Combat replay position | Not persisted | On reopen, show the resolved state (pending D6) |

   - **Not in saves:** none of these changes the existing save format. The
     sidecar must tolerate a roster restored from backup (stale map data is
     acceptable; corrupt data must not block loading).
   - **Whose map:** whether maps belong to the campaign, the team or the
     user is an open question. In the original, maps belonged to the human
     player, so per campaign is proposed.
6. **Determinism and tests.**
   - **Same outcomes in every mode:** tests should prove that the same
     inputs give the same engine states whatever the camera, animation or
     map mode is. This extends the approved "View game data identical across
     presentations" test.
   - **Characterise first:** any change to how hidden-door visibility
     consumes randomness must be characterised before it is changed (M1).

## 8. Open questions for prototyping

1. Do figures at a plausible scale fit an attached camera bounded by the
   team's square (X4a), or does the camera need option (b)?
2. Is per-message camera focus in combat pleasant at typical message counts
   (10–20 per round), or should focus be limited to notable events?
3. Does an instant cut on unperceived changes feel like a glitch to users
   who are used to animations? Would a uniform short fade for all
   transitions avoid leaking information while feeling deliberate?
4. How should the scene present a dark square (nothing visible ahead) so it
   reads as "darkness", not as a rendering failure?
5. Is the manual map pleasant enough with a mouse and trackpad to be used,
   compared with graph paper?
