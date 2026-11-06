# Combat experience review

Status: **design exploration, not approved** (2026-11-06). This follows the
M2 exploration in this directory. It compares three ways to run a Classic
combat round inside the dungeon, using an interactive prototype. Labels
follow the README: Approved, Fact, Hypothesis, Alternative, Decision
(X-numbers), Risk.

The working preferences from the review brief are treated as preferences,
not approvals:

- minimal immersive exploration;
- contextual commands;
- combat in the same environment;
- Classic round-based orders;
- no change to Classic mechanics or knowledge.

## 1. The prototype

**Open `mockups/combat.html` in Safari or Chrome.** It needs no server or
network, and it is linked from the storyboard.

**Playing a round:** click the stage, then play with the keyboard or the
mouse.

| Key | Effect |
|---|---|
| `Return` | Continue, or confirm |
| `1`–`6` | Choose a member, a target group, or a target member |
| `F` `P` `S` `U` `D` `R` | Fight, Parry, Spell, Use, Dispel, Run |
| `Tab` / `⇧Tab` | Next or previous member |
| `B` | Back |
| `C` | Clear all orders (at review) |
| `Esc` | Cancel |
| `I` | Read-only character card |
| `Space` | Next message |
| `⇧Return` | Show the rest of the round |
| `A` | Toggle automatic playback |
| `V` | Change camera while exploring |
| `R` | Restart |

**Mouse:** click figures, monster groups, pips, menu entries, the review
rows, or the order tags.

**The fixture runs as follows.**

1. **The step:** the team glides from the square behind into the corridor
   square. This is the step that Go has just resolved.
2. **The encounter:** press `Return` and Go reports an encounter.
3. **Round 1:** give six orders, review them, then fight.
4. **Round 2:** play back round 1, then give orders again. Corvin's KATINO
   is now unavailable because he has no spell points left.
5. **Victory and the chest:** after victory comes the chest, then the return
   to exploration. Running away is also possible: the first attempt fails
   and later attempts succeed.

**Toolbar:**

| Control | Options |
|---|---|
| Interaction approach | A, B, or C |
| Scenery policy (§4) | Strict, neutral shell, remembered, or unbounded (the hazard demo, with leaks marked) |
| Playback | Manual or automatic, with a speed |
| Value-update timing | Classic (at the end of the round) or per event |
| Reverse shots | On or off |
| Design notes | On or off |

**Reviewer aids** sit to the right of the stage:

- **The twelve moments:** the brief's twelve steps, with the current one
  highlighted and a "go" button that jumps to each.
- **What this moment tests:** labelled notes for the current moment.
- **Engine state vs what the user sees:** Go's committed values beside the
  values on screen, with differences in red. It also shows what the engine
  holds and never sends: the real trap and the true identities of
  unidentified groups.

**Deep links:** `combat.html#jump=<moment>&ap=A|B|C&vis=…&timing=event&still=1`
opens a fixed state.

- **Moments:** `explore`, `attached:3`, `encounter`, `orders`, `actor:3`,
  `target`, `unavail`, `spell`, `review`, `play:5`, `reverse`, `round2`,
  `chest:1`, `return`, `profile`.
- **Stills:** the stills were made this way.

**Stills** are in `stills/combat/`. They capture the stage only, except
`00-full-page-review.png`.

| Prefix | What it shows |
|---|---|
| `C01`–`C14` | All twelve moments in approach C, plus the unavailable-action, spell-choice and read-only character views |
| `A1`–`A4`, `B1`–`B4` | The same moments (orders, review, playback, chest) in approaches A and B |
| `V1`–`V4` | One reverse shot under each scenery policy |

## 2. Correction to the earlier M2 exploration

**Fact.** Classic does **not** update HP, status and monster counts as each
message appears.

- **What the engine does:** it snapshots them at the start of the round
  (`engine/combat.go:634`, `DisplayPartySnap` and `DisplayAliveCounts`). It
  shows the snapshot for the whole round, then clears it after the last
  message (`cmd/istary/main.go:3623`).
- **What the user sees:** messages report kills and damage while the
  numbers stay still. The numbers jump at the end of the round.
- **Where the earlier documents went wrong:** they said the opposite:
  - `EXPERIENCE-CONCEPT.md` §5;
  - `MOCKUP-REVIEW.md`, frame 3b;
  - `CAMERA-AND-INTERACTION.md` §9.3;
  - one storyboard annotation.
- **How they were fixed:** each has a correction note added beside the
  original text. Nothing was removed except the storyboard annotation,
  which now states the correction.

**What this means.**

- **Per-event updates are an option, not Classic.** The prototype offers
  both timings (decision X21).
- **They reveal nothing new:** they show what the messages already say,
  earlier in time.
- **They need data the engine does not produce today:** structured events
  that pair each message with the value changes it reports (§7).

## 3. Facts the prototype is built on

| Fact | Source | Used for |
|---|---|---|
| Fight is offered only to party slots 1–3 | `cmd/istary/main.go:3917` (`app.FrontRowSize` = 3, `app/tavern.go:19`) | Back-line restriction |
| Dispel: Priest always; Bishop above level 3; Lord above level 8 | `cmd/istary/main.go:3984–4005` | Corvin (Bishop level 3) cannot dispel |
| Use with no equipped item that has a spell power counts as Parry | `cmd/istary/main.go:3958–3982` | "Use · nothing usable; counts as Parry" |
| Spell is a typed name, offered to casters | `cmd/istary/main.go:3940`; `engine/magic.go:318` | Spell list with type-ahead |
| One living group: Fight and Dispel ask no target question | `cmd/istary/main.go:3927–3938` | Target step skipped in round 2 |
| B)ack at the confirm prompt clears **all** orders and restarts from member 1 | `cmd/istary/main.go:3531–3546` | Decision X19 |
| The round is resolved whole by `ExecuteRound` before any message shows | `engine/combat.go:603` | The engine-versus-display table |
| Monster identification is rolled per round, inside resolution | `engine/combat.go:622` | Names change at the start of round 2 |
| Messages are shown one action block at a time (separated by `ActionSeparator`) | `engine/combat.go:3356`; `cmd/istary/main.go:3603–3620` | One playback step = one action block |
| Messages advance on a timer only; no key skips them | `cmd/istary/main.go:3559` | Decision X20 |
| Party members whose condition is dead or worse sort to the back each round | `cmd/istary/main.go:3918` (comment), `engine/combat.go:610` | Lines are re-derived from Go every round |
| Chest: one inspection per member; an inspection can report the wrong trap | `engine/combat.go:258`, `3201` | Reports shown as reports |
| HAMAN/MAHAMAN can ask for a choice in the middle of playback (Wizardry 2/3) | `cmd/istary/main.go:3550` | A D6 question (§7) |

**Fact, with architecture consequence.** All the order-entry legality above
lives in the terminal program (`handleCombatChoose`), not in `engine` or
`app`. As with maze movement, the M5 controller extraction must move it
before any GUI combat can exist.

## 4. Camera and visibility

### 4.1 Scenery visibility vs gameplay knowledge

**Hypothesis.** This refines decision X3.

- **What X3 proposed:** draw only the Classic visible set, whatever the
  camera.
- **Why that breaks down:** it works for forward-looking cameras.
  - **The problem:** any camera that looks back at the team (an attached
    camera, a reverse shot in combat) shows space the Classic view never
    drew.
  - **What strict X3 draws there:** black void.
- **Proposed split:**

| Tier | What it contains | Rule |
|---|---|---|
| **Gameplay knowledge** | Doors, hidden doors, openings, monsters, chests, traps, special squares, stairs, anything that distinguishes one square from another | **Binding.** Only what the Classic rules let the team perceive now (the visible set and Go's messages). No camera can add to it. |
| **Scenery** | Geometry that helps the user read space: floor under the team, the team's own square, atmosphere | **Policy (X18).** Must never carry level data that is not already gameplay knowledge. |

**The four policies, all live in the prototype:**

| Policy | What is drawn outside the visible set | Leaks? | Look |
|---|---|---|---|
| **Strict** | Nothing | No | Black void behind the team in reverse shots. It reads as a stage in darkness. Honest, but abrupt in an attached view. |
| **Neutral shell** | A floor that fades into fog behind the team, and a fog curtain on the team square's unseen edge. Generated from the camera and the team's square only; reads no level data. | No, by construction | Soft and continuous. It cannot show a wall, a door or an opening that is not known, because it never knows. |
| **Remembered** | Surfaces that were in a view earlier on the way here (an exploration record), tinted. Neutral shell beyond. | Not beyond what the user saw, but it is a memory aid | The most natural. The corridor behind looks like the corridor the team walked. |
| **Unbounded** | All real geometry near the camera | **Yes.** 15 surfaces in the fixture, including the door behind the stairs the team never faced and the room beyond the side door | Hazard demonstration only. Not proposed. |

**Trade-offs.**

- **Neutral shell** is the only non-strict policy whose safety does not
  depend on bookkeeping.
- **Remembered** draws only what was shown, but it is a memory aid. Under
  Classic the user remembers, not the game. That makes it close to the
  exploration record behind the automatic map (D9, X11, X13). It should be
  decided with that enhancement, not separately.
- **Silent changes:** remembered scenery must follow the perceived position
  (X13). Otherwise, after a teleport, the corridor "behind" would be drawn
  from the old place, or the new true place would leak.
- **The hard rule (proposed):** whatever the policy, the renderer receives
  only:
  - the gameplay-knowledge set;
  - an optional remembered set, both from Go's View;
  - nothing else.

  Unbounded must be impossible by construction, not merely switched off.

### 4.2 Camera shots used

| Shot | Where the camera is | Purpose |
|---|---|---|
| First person | Team's eye | Exploration (matches the Classic view) |
| Attached | Inside the team's square, behind a chosen member | Exploration alternative (`V`, then `1`–`6`) |
| Combat framing | Back of the team's square, elevated, below the ceiling | Encounter, orders and review. Same corridor, same light |
| Event focus | Back of the team's square, eased towards actor and target | Party actions during playback |
| Reverse | In the visible corridor beyond the opponents, looking back | Monster actions during playback (toggle) |
| Chest close-up | Front of the team's square, looking at the chest | Treasure |

**Observations.**

- **The X4 rule held up.** "Camera inside the team's square" worked for
  every party-side shot. The front row is behind the camera in the chest
  close-up, which unclutters it.
- **Reverse shots break that rule.** They put the camera in the opponents'
  space.
  - **No new gameplay knowledge:** that space is inside the visible set.
  - **But scenery is needed:** they are the shot that makes the scenery
    policy matter (V1–V4).
  - **Pros:** reverse shots make monster attacks legible and lively.
  - **Cons:** they add camera motion every message, and might be tiring.
    The toggle exists to judge that (decision X22).
- **Encounter transition:** a 1.3-second ease from first person to combat
  framing kept the sense of place. The banner is the only "battle screen"
  moment.
- **Approach B's panel shortens the scene.** The prototype widens the field
  of view to keep the team in frame. Focus shots were not re-tuned for B,
  and they crop. Any B design needs its own camera set.

### 4.3 Unresolved camera and compatibility questions

- **Remembered scenery:** is it Classic-permitted presentation, or part of
  the exploration-record enhancement? (D9, X18)
- **Ceilings:** should an elevated combat camera ever rise above wall
  height?
  - **Prototype:** no. The camera stays below the ceiling, which keeps
    walls opaque.
  - **Hazard:** above-wall cameras would see into neighbouring squares;
    that is the same leak class as unbounded.
- **Hidden doors:** under strict or shell, are hidden doors drawn only when
  the Classic rules would show them? The render-RNG question in the README,
  risk 1, is still open.
- **Light:** light radius still bounds the knowledge tier. Whether
  scenery-shell fog should vary with light is an art question (X16).

## 5. The three interaction approaches

All three use the same fixture, the same cameras and the same orders. Only
the placement of commands and information changes.

### A · Contextual overlay

- **Orders:** the order menu floats beside the member being ordered.
  - **Group labels** float over the monster groups.
  - **Order tags** float over each ordered member.
  - **The team strip** at the bottom shows each order.
- **Review:** the tags grow to show names, with a one-line prompt.
- **Playback:** a message bubble sits above the actor or target.
- **Chest:** the menu sits beside the chest, and reports float near it.

| Strengths | Weaknesses |
|---|---|
| Most immersive: the scene stays full size and the eye stays on the action | Review is weakest: six floating tags are harder to check than a list |
| Direct link between a member or group and its commands | Overlays move with the camera, so they hide during camera moves |
| Lowest visual weight | Bubbles over moving targets can be hard to follow at speed |
| | Menus beside front-line members can cover other figures |

### B · Temporary tactical workspace

- **The panel:** a docked panel slides up when the encounter starts and
  leaves when exploration resumes.
- **Panel content:** a team table with orders, actions with reasons, targets
  and opponents, and during playback a message log.
- **The scene:** it stays visible above the panel.

| Strengths | Weaknesses |
|---|---|
| Everything is in one predictable place; the fastest to scan and to learn | The scene shrinks by about a third, so the dungeon becomes a backdrop |
| Best for review: the whole round is a table | Cameras must be designed around the panel (see §4.2) |
| Easiest for keyboard-only and assistive technology | Least "inside the dungeon"; closest to a conventional battle screen |
| The full log is always visible | The panel's appearance is an abrupt mode change |

### C · Hybrid (initial preferred candidate)

- **During selection:** a contextual menu beside the member, as in A.
- **When all six orders are given:** a tactical summary card appears, with
  six rows (click to change), the opponents, and Fight, Back and Clear all.
- **During playback:**
  - the card shows the opponents, a short log and the playback controls;
  - a message line sits over the scene.
- **Chest:** a contextual menu, plus a card titled "What the team has been
  told".
- **After the encounter:** the card leaves.

| Strengths | Weaknesses |
|---|---|
| Keeps A's immersion while choosing, and gives B's clarity when checking and following results | Two places to look during playback (message line and card) |
| The card appears only when there is something to confirm or follow | The card occludes the right side of the scene, where reverse shots put the action |
| Order editing is natural: click a row | More states to design and test than A or B |

### Comparison

| Concern | A | B | C |
|---|---|---|---|
| Sense of place | Best | Weakest | Good |
| Speed of giving six orders | Good | Best | Good |
| Confidence before committing | Weakest | Best | Good |
| Following playback | Fair | Best (log) | Good |
| Camera freedom | Best | Constrained | Good |
| Accessibility (keyboard, VoiceOver) | Hardest | Easiest | Medium |
| Implementation cost | Medium (anchoring) | Lowest | Highest (both) |

**Hypothesis.** C remains the best candidate. B is worth keeping as an
"information-dense" alternative for accessibility, with A as its immersive
counterpart. Two adjustments to C emerged:

1. **Place the card by the shot.** Put it on the side away from the current
   focus, or let it collapse to one line during reverse shots.
2. **One message location during playback.** Use either the line or the
   card's log, not both at full weight.

## 6. Interaction details

### 6.1 Ordering

- **Selecting members:** any of the six can be selected at any time:
  - with `1`–`6` or `Tab`;
  - by clicking the figure, the pip or the table row.

  After an order is given, selection moves to the next member without one.
- **Depth:**
  - **At most three levels:** action → spell → target.
  - **Usually less:** Parry and Run are one key. Fight against the only
    group is one key.
  - **No menu trees:** there are none.
- **Targets:**
  - **Groups:** the rings and labels pulse when a group is expected; press a
    number or click.
  - **Members:** for a spell such as DIOS, the members' rings and pips go
    green.
  - **Dead groups:** they cannot be chosen; the reason is given.
- **Unavailable actions:** stay visible, dimmed, with the reason underneath
  (decision X24).
  - **Pressing one** shows a sentence explaining why, for example:
    - "Back line (4–6) cannot fight";
    - "Bishops dispel from level 4 (Corvin is level 3)";
    - "No level 1 spell points left";
    - "CALFO is cast on a chest, not in combat".
  - **Use with nothing usable** is shown as available, with "counts as
    Parry", because that is what Go does.
- **Changing an earlier order** (decision X19):
  - **Fact:** Classic offers only B)ack at the confirm prompt, which clears
    everything.
  - **What the prototype adds:** a direct change to one member's order (click
    the tag or the row, or press the number), plus B for the previous member
    and C to clear all.
  - **Hypothesis:** this is outcome-equivalent. Go receives a complete set
    of six orders either way.
- **Review:** collected orders are always visible (A: tags; B: table;
  C: card). Commit is `Return` or **Fight**.
- **Next round:** orders start empty, as in Classic. "Repeat last round's
  orders" is an alternative (decision X23).
  - **What it does:** fills the orders that are still valid and asks again
    for the rest.
  - **Example:** Corvin's KATINO is gone in round 2.

### 6.2 Character information

- **Opening it:** `I`, the ⓘ on a pip or row, or `1`–`6` during playback.
- **What it shows:** a read-only card with HP, AC, status, line, spells left
  and items (D8). Camp actions are not offered, and the card says so.
- **Values during playback:** the card shows the displayed values, not Go's
  final ones. Final values would reveal results, including spell points,
  before their messages. This is a View rule the controller must support.

### 6.3 Keyboard and mouse parity

Every action in the prototype can be done with either the keyboard or the
mouse.

- **Keys:** keyboard letters follow Classic (`F`, `P`, `S`, `U`, `D`, `R`;
  `O`, `I`, `C`, `D`, `L` at the chest).
- **Keyboard focus:** it is not yet handled to an accessible standard; see
  §10.

## 7. Playback

- **Unit of playback:** one action block, the messages between two
  `ActionSeparator`s, as in the terminal.
- **Manual:** `Space` or **Next**.
- **Automatic:** a timed advance with a speed choice; `A` toggles it.
- **Show rest:** `⇧Return` shows the whole round at once.
- **Classic pacing:** timer-only, with no skip (decision X20).
- **What each event shows:**
  - **Camera:** it eases to the actor and target, or to a reverse shot for
    monsters.
  - **Effect:** a sword slash, HALITO fire, KATINO mist, a dispel flash, a
    heal glow, or a miss.
  - **Message:** the message itself.
- **Value timing** (decision X21):
  - **Classic:** values on screen stay frozen and update after the last
    message. The engine table shows the gap growing during the round.
  - **Per event:** each block applies its changes as it shows. Pips flash.
    Monster figures leave when the message says they died.
- **Authoritative vs sequential:** the engine table makes the distinction
  visible.
  - **Go:** the round is committed in Go as soon as **Fight** is pressed.
  - **Presentation:** it is only part-way through showing it.
  - **Shared values:** nothing the presentation shows comes from the final
    state until playback ends, except group names. Names update at round
    start, as identification is not part of the frozen snapshot.

### D6 questions (unresolved; not solved here)

1. **Commit point.** Is the round committed to the saved game when Go
   resolves it, or when playback finishes?
   - **If when resolved:** quitting mid-playback loses the unseen messages
     unless they are stored.
   - **If when playback finishes:** the engine needs a "resolved but
     unshown" state.
2. **Queue persistence.** Does the event queue persist, so that a restored
   game can replay unseen messages? If so, it becomes part of the D7
   sidecar or save design.
3. **Structured events.** Per-event values, camera focus and effects need
   structured events: actor, target, kind, value changes.
   - **Today:** the engine produces strings only.
   - **Divergence risk:** events must come from the same code path as the
     messages so that they cannot disagree. Should resolution emit both?
4. **Input during playback.** HAMAN and MAHAMAN (Wizardry 2/3) stop
   playback for a choice (`cmd/istary/main.go:3550`). The queue must be able
   to pause for a controller command.
5. **Saving in combat.** Can a game be saved or quit during combat at all
   under Classic, and what does the user see on restore?
6. **Display state as a View field.** The display snapshot (plus per-event
   progress, if X21 allows) must be a View field. Svelte must not keep its
   own copy of "what has been shown", or a reload would desynchronise it.
7. **The delay setting.** The Classic T)IME delay is a game option today
   (`cmd/istary/main.go:152`). Where does the playback speed live, game or
   user preference?

## 8. Treasure inspection

- **Camera:** the close-up from the front of the team's square.
- **Actions:**
  - **The choices:** Open, Inspect, Calfo, Disarm and Leave, then "who".
  - **Highlighting:** candidate members are highlighted.
  - **Who can act:** Go decides. For example, someone who has inspected
    already cannot inspect again, and CALFO needs spell points.
- **Knowledge layers:**
  - **Engine:** the real trap, shown in the pink box, never sent.
  - **Team:** the reports, with who said what and how.
  - **What the user sees:** the reports, labelled "reports, not facts".
  - **What the user may attempt:** what Go offers.
  - **Disagreeing reports:** the fixture includes two. Pell says POISON
    NEEDLE and Ysolde says GAS BOMB.
- **Disarm:** the prototype asks which trap, offering the reported names or
  typed entry.
  - **Compatibility question:** the engine's Disarm has no trap-name entry.
    Whether Classic asks for one is open and is not resolved here.
- **Return:** after Open or Leave, the camera eases back to the user's own
  exploration camera; the combat interface has gone.

## 9. Implications for the controller (no implementation proposed)

For any of A, B or C, Go would need to supply:

- **For each member, during orders:**
  - the order menu (key, label, available, reason, note);
  - spells, with points left and reason;
  - target kind;
  - line membership, re-derived every round.
- **Orders state:** current orders and the round number.
- **Display values:** the display state (snapshot, or progress if per
  event), separate from final state.
- **Messages:** message blocks, and structured events if X21 or X22 need
  them (§7).
- **Chest:** the action list with "who may", and the team's reports. Never
  the trap.
- **Commands:**
  - `SetOrder(member, order)` (X19);
  - `ClearOrders`;
  - `Commit`;
  - `Advance`, and possibly `AdvanceAll`;
  - `Chest(action, member, trapName?)`.

**Svelte would hold only presentation state:**

- which member is being ordered;
- camera;
- playback speed;
- which card is open.

## 10. Prototype limitations

- **Not rules.** `GO.commit` is a scripted stand-in.
  - **What is fixed:** initiative order, hit and kill outcomes, and monster
    damage.
  - **What is missing:** deaths, re-sorting of the lines, silence, and
    monster spells.
  - **How it is labelled:** the code says so in its header.
- **Messages:** wording is illustrative and not verified against a Classic
  run. Monster names are invented (BOGLIN, BONE WALKER).
- **One encounter, one location, fixed light.** No map, no sound.
- **Placeholder art.** Capsules, blobs and flat shading, with painter's-sort
  artefacts. This is not an art direction (X16).
- **Camera tweens** interpolate position and target linearly. Overlays hide
  while the camera moves.
- **Accessibility is not built:**
  - no VoiceOver live region for messages;
  - focus order not managed;
  - no reduced-motion design beyond skipping tweens.
- **Performance and power** of SVG are not representative of a WebGL
  renderer.

## 11. Decisions for review

These continue the README register.

| ID | Question | Options | Proposed |
|---|---|---|---|
| X17 | Combat interaction approach | A contextual; B tactical panel; C hybrid; user choice | C as the baseline; keep B as a dense, accessible alternative |
| X18 | Scenery outside the Classic view | Strict; neutral shell; remembered; (unbounded is rejected) | Gameplay-knowledge tier binding (narrows X3); neutral shell by default; strict in Classic presentation; remembered only if decided together with the exploration record (D9, X13) |
| X19 | Changing an earlier order | Classic clear-all only; direct change of one order; both | Both: direct change plus "Clear all"; needs a `SetOrder` controller command |
| X20 | Playback pacing | Classic timer-only; manual; automatic with speed; "show rest" | Automatic at Classic pace by default; manual and show-rest as preferences |
| X21 | When displayed values change | Classic end of round; per event | Test both with users; Classic presentation keeps end of round; per event needs structured events (D6) |
| X22 | Camera during playback | Fixed combat framing; focus on actor; also reverse shots | Focus on actor; reverse shots only with an approved X18 policy; a setting to reduce camera motion |
| X23 | Repeat last round's orders | None; explicit command; automatic | Explicit command only; invalid orders asked again |
| X24 | Unavailable actions | Hide; show dimmed with the reason | Show with the reason supplied by Go |

## 12. Recommendation: what to test next

1. **Walkthrough sessions.** Sit two or three people (including one Wizardry
   veteran and one newcomer) at the prototype with approach C, then A and B.
   - **Measure:**
     - time to give six orders;
     - wrong or unavailable presses;
     - whether they notice and understand the reasons;
     - whether they trust the review step.
   - **Ask** whether the frozen Classic values (X21) confuse them.
2. **A harder fixture.** Add a second encounter with:
   - a member who dies and re-sorts into the back line;
   - a sleeping or silenced caster;
   - four groups;
   - a monster spell.

   This tests the order menu and playback where Classic is least forgiving.
3. **Engine reading, no code.** Map where `ExecuteRound` emits messages and
   where it changes state.
   - **Purpose:** establish whether structured events (§7, question 3) can
     come from the same code path. This decides how feasible X21 per-event
     timing and X22 focus shots are.
4. **Renderer prototype P1** (`RENDERING-OPTIONS.md` §5).
   - **Build:** the combat shots (wide, focus, reverse, chest) in WebGL2,
     from a hand-written View fixture.
   - **Judge:** the neutral shell and remembered policies with real depth
     and lighting. It replaces the painter's-sort artefacts.
5. **Accessibility pass on approach C:**
   - keyboard-only completion;
   - a VoiceOver live region for messages;
   - reduced motion with no camera moves.

Nothing in this review is approved. No renderer, layout or approach is
selected.
