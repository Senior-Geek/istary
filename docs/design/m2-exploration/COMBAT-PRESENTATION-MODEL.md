# Combat presentation model

Status: **design exploration; decisions X25–X28 approved** (2026-11-06,
with the X26 addendum; see §10). This refines approach C from
`COMBAT-EXPERIENCE-REVIEW.md` around one principle. It changes no rule, no
engine code and no application code. The difficult cases (deaths, formation
changes, four groups, locations) are validated in
`COMBAT-DIFFICULT-CASES.md`. Labels follow the
README: Approved, Fact, Hypothesis, Alternative, Decision (X-numbers), Risk.

> **Six adventurers, one battlefield, one action at a time.**
> The player orders all six; Go resolves the round; the presentation shows
> the resolved actions one by one. It illustrates results. It adds no
> real-time fighting, no movement controls and no combat mechanics.

Future controller and event-stream needs are kept apart, in
`COMBAT-EVENT-REQUIREMENTS.md`.

## 1. The prototype

**Open `mockups/combat-refined.html` in Safari or Chrome.** It needs no
server or network. It plays the same fixture as `combat.html` (both now load
`mockups/combat-fixture.js`), so the A/B/C comparison stays valid alongside
it.

| Key | Effect |
|---|---|
| `Return` | Continue; at review, Fight |
| `1`–`6` | Choose a member (or a target group or member) |
| `F` `P` `S` `U` `D` `R` | Fight, Parry, Spell, Use, Dispel, Run |
| `Tab` / `⇧Tab`, `B`, `C`, `Esc` | Next or previous member, back, clear all, cancel |
| `Space` | Next action |
| `⇧Return` | Show the rest of the round |
| `A` | Automatic playback on or off |
| `K` | Stable camera, or stable with restrained push-in |
| `I` | Read-only character card |

**Mouse:** click figures, group labels, strip cards and menu entries. In
review, hover a card to see that member's target ringed in white.

**Toolbar:**
- **Camera during the round:** stable, or stable with restrained push-in
  (§4.4).
- **Playback:** manual or automatic, with a speed.
- **Scenery:** strict or neutral shell (X18).
- **Design notes:** the reviewer panel, with a plan of the camera and axis.

**Reviewer panel:** eleven jump buttons, one per moment in §3. Deep links use
`#jump=…&cam=fixed|push&still=1`, which is how the stills were made.

**Stills** are in `stills/combat-refined/`:

| Still | Moment |
|---|---|
| `R01-exploration` | First person; the team strip is the only HUD |
| `R02-encounter-transition` | Halfway through the rise to the battlefield view; the team fading in |
| `R03-order-entry` | Brannoc being ordered; commands on the east wall; strip as summary |
| `R04-target-choice` | Fight: which group? Both group labels pulse white |
| `R05-unavailable-command` | Pell, back line: Fight shown but unavailable, with the reason |
| `R06-order-review` | All six set; hovering Ilsa rings her HALITO target |
| `R07-adventurer-action` | Torvald attacks: gold ring, nameplate and card; white ring on the target |
| `R08-monster-action` | A creature attacks Torvald: gold group label, one figure lunges |
| `R09-round-complete-next-orders` | Round 2 orders in the same view; values updated; repeat offered |
| `R10-victory-chest` | Chest: message, actions and reports in one panel |
| `R11-return-to-exploration` | First person again |
| `K1`–`K5`, `compare-spell`, `compare-dispel` | Stable vs restrained push-in (§4.4) |
| `00-full-page-review` | The whole page, with the reviewer panel and the plan |

## 2. What changed from `combat.html` approach C

| | `combat.html` C | Refined |
|---|---|---|
| Orders on screen | Order tags floating over figures, plus a review card | One team strip: name, status, HP, order. It is the order summary and the review |
| Front and back | Pips in one row | Strip split into **Front line · 1–3** and **Back line · 4–6**; a faint floor line between the lines |
| Commands | Menu beside the actor, side chosen per actor | Menu over the east wall, level with the actor; tied by the gold ring and gold card |
| Messages | Message line at the bottom, plus a log in the card | One message area, top left: messages, prompts, Fight, playback controls, chest |
| Playback camera | Eases to each actor; reverse shots for monsters | Stable view, never cut; optional restrained push-in |
| Highlighting | Rings, tags and pulses in several styles | Gold: selected or acting. White: target. Green dashed: may be chosen |
| Values | Classic or per-event toggle | Classic numbers; figures follow the messages (X26) |
| Reviewer aid | Engine-vs-shown table | The same, plus a plan showing the axis and the camera |

## 3. The sequence

| Moment | What the player sees | Notes |
|---|---|---|
| **A. Exploration** | First person. The strip shows the six members, status and HP. No figures. | As Classic. |
| **B. Encounter** | The camera rises from first person to the battlefield view (1.5 s). The team fades in where it stands; the monsters appear ahead in group order. Labels appear when the camera settles. | Same corridor, light and walls; no new location. |
| **C. Order entry** | Stable battlefield view. The selected member has a gold ring and a gold card; the commands appear over the east wall. Unavailable commands stay, with Go's reason. The strip fills in as orders are given. | No movement, positioning or grid. |
| **C′. Review** | When all six are set, the message area offers **Fight**, **Clear all**, and "change one: click a member or press 1–6". The strip is the review. | Changing one order directly is still X19. |
| **D. Resolution** | One action at a time in Go's resolved order (§5). | Team actions and monster actions interleave. |
| **E. Round complete** | After the last action, numbers catch up in one step (changed cards flash), fallen monsters leave, and order entry starts again in the same view. "Repeat round N's orders" is offered (X23). | No transition. See §7 for the line re-sort. |
| **F. Victory and chest** | The victory message, then the camera eases to the chest. Message, actions and reports share one panel; the strip stays for choosing who acts. After Open or Leave, the camera returns to first person and the team fades out. | Unchanged chest rules. |

## 4. Camera

### 4.1 The axis of action

**The axis is the corridor centre line** from the team towards the monsters
(x = 0 in the fixture). **Every combat camera stays east of it**, behind the
team and inside the team's square. The consequences:
- the team is always at the bottom of the frame and the monsters ahead;
- group 1 is always nearer than group 2;
- the front line always stands beyond the back line;
- nobody changes side of the screen between order entry, playback and the
  next round.

The reviewer panel's plan draws the axis, the camera side and the live
camera position, so this can be checked at any moment.

The side is a presentation constant for the encounter (Hypothesis): east
here because the fixture's door is on the west wall. A real implementation
needs a rule for picking the side (for example, away from the nearer side
wall, or always the same side). That is an open question (§9).

### 4.2 The shots

| Shot | Use | Position |
|---|---|---|
| First person | Exploration | Classic eye position |
| Battlefield | The whole encounter | Elevated, behind the team, east of the axis, inside the team's square |
| Restrained push-in | Optional, significant actions only | A short dolly forward, slight turn, same lens; still east, behind the team, inside the square |
| Chest | After victory | From the front of the team's square |

**No reverse angle and no cut** in ordinary combat. The only camera
movements are the encounter rise, the optional push-in, the move to the
chest, and the return to first person.

### 4.3 Exposure check

**Facts (fixture, measured):**
- **Combat shots:** the battlefield shot, the push-ins at both extremes and
  the chest shot expose **no** surface the team has never seen.
- **The old reverse shot:** the reverse shot in `combat.html` exposes two,
  the never-seen door south of the stairs and the niche's south wall.
- **The check:** `node mockups/leak-check.js`. It tests facing, framing and
  plan occlusion. The SVG renderer has no occlusion, so its "unbounded"
  drawing overstates exposure; that option is therefore not offered on the
  refined page.

The stable camera looks forward from inside the team's square, so it never
shows the space behind the team. That makes scenery policy X18 matter much
less: strict and neutral shell look almost the same here.

### 4.4 Stable vs restrained push-in

The same fixture and the same events in both modes:
- **Stable:** no camera movement at all.
- **Restrained:** a 0.35-unit dolly and a slight turn, for spells, dispels
  and items only.

| | Stable | Restrained push-in |
|---|---|---|
| Who is acting | Clear: gold ring, nameplate, gold card, others dimmed slightly, token lunge | The same, plus the move |
| Continuity | Perfect: nothing on screen moves except the actor | Good: same side, same lens, no cut; the near team shifts noticeably, the monsters barely |
| Reading the message | Eyes can stay on the message area; nothing to re-find | The scene shifts under the message, then shifts back on the next ordinary action |
| Information | Nothing beyond the message | Pushes in **before** the result is shown, so it must be chosen from the action kind, not the outcome (else it would announce kills) |
| Crowding | The far group label is clear of the message area | The near back line grows into the strip; labels move a little |
| Monster actions | Identical in both: monster actions never trigger a push-in | |
| Cost | None | Camera path per action kind; a reduced-motion setting must turn it off |

**Finding (Hypothesis, to test with users).**
- **Stable is enough:** the stable camera carries the round on its own. The
  attention cues (gold ring, nameplate, gold card, slight dimming and the
  lunge) identify each actor without any camera help.
- **What the push-in adds:** some emphasis for spells, but no information.
  It costs a little steadiness and adds one more thing to tune and to turn
  off for reduced motion.
- **Proposed (X25):** stable as the default. Restrained push-in, if kept,
  is an optional preference.

## 5. Anatomy of one action

Each action block (the messages between two `ActionSeparator`s,
`engine/combat.go:3356`) is presented as one event:

1. **Identify:** the actor gets the gold ring.
   - **A member acting:** a nameplate above them, and their strip card turns
     gold.
   - **A monster acting:** its group ring and label turn gold.
   - **The target:** a white ring, and a white card or label border.
2. **Direct attention:** other figures dim slightly. The camera stays put
   (or pushes in, §4.4).
3. **Gesture (about 0.4 s):** a token gesture chosen from the **order** or
   action kind.
   - **Fight:** a lunge towards the target.
   - **Dispel:** a short step.
   - **Spell or item:** a casting glow, no step.
   - **Run:** a step back.
   - **A monster attack:** one figure of the group lunges towards its target.
4. **Result:** at the end of the gesture the effect appears and the
   remaining message lines are shown.
   - **The effects:** a hit or a miss, HALITO fire, KATINO mist, a dispel
     flash.
   - **What the figures show:** a monster the message calls killed fades; a
     monster put to sleep greys and shows a z.
5. **Advance:** on `Space`, or after a delay when automatic.
   - **Skipping:** skipping mid-gesture completes the action's result first,
     so skipping never hides an outcome.
   - **Show the rest:** applies every remaining result at once.

**What the gesture must not imply.** It is not a simulation:
- the lunge always stops short;
- a miss uses the same lunge as a hit;
- figures return to their places;
- nothing travels or collides to decide anything.

Hits, damage, targets and order all come from the resolved round.

**What playback deliberately does not show:**
- the total number of actions in the round;
- who acts next;
- any initiative list.

Go knows all of these before playback begins (initiative slots,
`engine/combat.go:521–578`; monster victims are pre-assigned,
`engine/combat.go:1016`), but Classic reveals them only one action at a
time. The counter says "action 3", not "3 of 8".

## 6. Screen layout

The positions below are illustrative (X12-style flexibility applies), but the
**roles** are the proposal:

- **The team strip:** bottom of the frame.
  - **Contents:** six cards, front line left and back line right. Each card
    shows name, class, status, HP and, in combat, the order.
  - **Roles:** in exploration it is the status bar; in combat it is the
    order summary and the review.
  - **Duplicate labels:** there is no second copy of the orders in the
    scene.
- **The message area:** top left, over the west wall and ceiling, which
  hold no figures in any combat shot.
  - **Contents:** the one place for messages, prompts and toasts, plus the
    Fight, Clear all and playback controls, and the chest panel.
  - **Why it moved:** top centre was tried first, but three-line messages
    covered the far group's label.
- **Commands:** over the east wall, level with the member being ordered.
  - **Scope:** commands only, never the summary.
  - **Why it moved:** a menu placed beside the actor covered other figures
    (Pell, behind Torvald).
- **Group labels:** above each group, giving the number, the count, the
  name (unidentified until Go says otherwise) and how many are asleep.
- **Highlight language:**
  - **Gold:** the member being ordered, or the combatant acting.
  - **White:** the target, whether a group or a member.
  - **Green dashed:** members who may be chosen (DIOS, chest "who").

## 7. Displayed information vs engine state

- **Fact.** Go resolves and commits the whole round before anything is shown
  (`engine/combat.go:603`). Classic freezes displayed HP, status and monster
  counts for the round and updates them after the last message
  (`engine/combat.go:634`).
- **Prototype: numbers.** Numbers (strip HP and status, group counts) stay
  on the round snapshot until the round has been shown, as Classic does.
- **Prototype: figures.** Figures follow the messages: a figure the message
  calls killed fades, and one put to sleep shows asleep.
  - **Why:** without this, a monster could be "killed" in the message and
    still stand, undamaged, until the round ends.
  - **What it shows early:** this shows nothing the messages have not
    already said, but it does make the numbers and figures disagree for the
    rest of the round (four creatures labelled, one faded).
  - **Decision:** this is decision X26.
- **The reviewer table.** It shows Go's final values, the shown numbers and
  the figures' state side by side.
- **Fact (formation risk).** `ExecuteRound` sorts the team by status at the
  start of each round, so dead members go to the back
  (`engine/combat.go:610`; comment at `cmd/istary/main.go:3918`).
  - **The conflict:** "consistent positions between rounds" therefore has a
    rules exception. When a front-line member dies, someone moves up.
  - **What the presentation needs:** to show that move explicitly between
    rounds, with figures changing places and cards changing groups.
  - **What needs checking:** when the sort happens relative to order entry.
    The fixture has no deaths and does not demonstrate it.
  - **Since checked** (`COMBAT-DIFFICULT-CASES.md` §4): Classic re-sorts at
    the end of the round, before the next orders (Player's Guide p. 48).
    Go re-sorts at the start of `ExecuteRound`, after the orders were given
    by slot, so an order can pass to another member (verified with a Go
    probe). The sort also moves asleep and paralyzed members back. The
    prototype shows the change at the round's update (Classic timing).

## 8. Mechanics and information boundaries

| Requirement | How the prototype meets it |
|---|---|
| Orders for all six before resolution | Fight is offered only when six orders exist; Go's `commit` takes all six |
| Actions in resolved order, monsters included | Playback follows the fixture's resolved blocks, which interleave members and monsters (round 1: Ilsa, a creature, Torvald, a figure…) |
| Initiative not roster order | As above; no queue is shown in advance |
| Restricted actions only for eligible members | Menus and reasons come from `GO.menu`; Fight unavailable for 4–6; Dispel by class; spells by points. In the product these come from Go (`cmd/istary/main.go:3917–4005` today) |
| No leaks through the camera | Stable and push-in shots expose nothing unseen (§4.3); the camera stays inside the team's square |
| No leaks through rendering | Unidentified names until Go identifies; the chest trap and true identities stay in the pink "engine only" box; no action total or next-actor preview |
| Displayed vs authoritative | Reviewer table; Classic numbers; figures-follow-messages flagged as X26 |
| No new rules | The gesture is decoration; the push-in is chosen from the action kind; nothing is computed by the presentation |

## 9. Open questions

Questions 1–4 were taken further in `COMBAT-DIFFICULT-CASES.md`:
- **1, camera side:** east held in a corridor, a room, a junction and a dead
  end.
- **2, rooms, junctions and narrow spaces:** see §6 of that document; rooms
  raise a visibility question (Q-D3).
- **3, larger encounters:** four groups were staged; a dead end crowds them
  (Q-D5).
- **4, formation changes:** shown at the round's update, with Classic timing;
  Go's timing differs (Q-D7).

Its own open questions are Q-D1 to Q-D9.

1. **Camera side.** Which side of the axis, and who chooses: always east,
   away from the nearer wall, or a user preference? It must stay fixed for
   the whole encounter.
2. **Corridor width and junctions.** The fixture is a 1-square corridor.
   Encounters in rooms, at junctions, or with walls close behind the team
   may not leave room for the elevated camera inside the team's square.
3. **Larger encounters.** The engine works with up to four groups (the
   identification roll picks among four, `engine/combat.go:622`), and groups
   can be much larger than the fixture's. Staging and labels for four full
   groups have not been tried; nor has a group that is mostly asleep.
4. **Formation changes.** How the line re-sort (§7) is shown, and when.
5. **Figures vs numbers.** Whether figures following messages while numbers
   stay frozen is clear or confusing (X26). Test with users alongside the
   per-event option in `combat.html`.
6. **Message history.** Classic shows only the current action. Should a
   short history of the round be available (for example, on a key), and
   does it count as information Classic withholds?
7. **Monster appearance.** Placeholder blobs give no clue to identity. Real
   art must not reveal an unidentified monster's true kind (D12 also bears
   on original pictures).
8. **D6 questions.** They are unchanged (`COMBAT-EXPERIENCE-REVIEW.md` §7):
   when the round counts as committed, whether unseen actions survive a
   quit, and pause points for HAMAN.

## 10. Decisions

These continue the register after X24. **X25–X28 were approved on
2026-11-06.** What was approved is below; the options and proposals as they
were put for review follow it, for the record.

- **X25 Combat camera: approved.**
  - The stable, elevated camera behind the team, adventurers in the
    foreground and monsters ahead.
  - One viewpoint for the whole encounter.
  - No reverse shots or cinematic cuts.
  - Restrained push-in stays an optional, experimental preference, not a
    requirement.
  - The priority is the clarity of the outcome, not spectacle.
- **X26 Combat results: approved.**
  - **When:** figures respond when the message reveals the outcome, never
    before.
  - **Monster deaths:** a defeated monster visibly dies rather than
    disappearing.
  - **Conditions:** sleeping, incapacitated and defeated figures show their
    revealed condition.
  - **Numbers:** Classic's round-end updates are kept for now.
  - **Death animations:** they may have some theatrical personality.
    - **Styles:** humanoids stagger and collapse, skeletons fall apart,
      slimes dissolve, fliers tumble, and large monsters collapse more
      substantially.
    - **Status:** these are illustrative categories, not mechanics or final
      art.
    - **Timing:** brief (about 0.7–1.2 s) and skippable without losing the
      outcome.
    - **Selection:** from the revealed result and the creature's
      presentation type only.
  - **Addendum (party):**
    - **Paralyzed:** solid, rigid, subtly distinct.
    - **Asleep or unconscious:** slumped, distinguishable from death.
    - **Dead:** a brief death, then a subdued, translucent figure. This is a
      convention that the dead member stays with the expedition, not a
      spirit or a mechanic.
    - **Ashes:** a distinct subdued representation.
    - **The roster:** dead and incapacitated members stay in the six. Death
      does not end the expedition; the player decides whether to go back.
    - **No new mechanics:** no corpse-carrying or retrieval mechanics.
    - **The strip:** its condition label stays authoritative.
    - **In combat:** incapacitated members must not appear to take part.
    - **Status:** the treatments are provisional until reviewed in the
      browser.
- **X27 Layout: approved.**
  - The six-member strip, with the front and back lines clearly separated,
    serves as status, order summary and review.
  - One message area.
  - Contextual commands in an unobstructed part of the battlefield.
  - Gold for the selected or acting figure, white for the target, green
    dashed for an eligible choice.
  - Exact positions remain adjustable.
- **X28 Playback: approved.**
  - **What is shown:** Go resolves the round; the presentation shows one
    action at a time. It never shows initiative order, future targets, the
    remaining action count or unrevealed outcomes.
  - **What drives the animation:** gestures come from the action, not its
    result. Effects and deaths appear only with their result.
  - **Playback control:** manual and automatic playback stay; skipping never
    discards an outcome.

Further combat decisions X29–X34 (formation, orders, representation, state)
were approved later the same day and are consolidated, with the governing
principle, in `COMBAT-DESIGN-BASELINE.md`.

As proposed for review:

| ID | Question | Options | Proposed |
|---|---|---|---|
| X25 | Combat camera | Stable battlefield view only; stable + restrained push-in for significant actions; focus per action with reverse shots (`combat.html`) | Stable by default, on one side of the axis for the whole encounter; restrained push-in only as an optional preference, chosen from the action kind; no reverse shots. Supersedes the X22 proposal |
| X26 | What changes on screen during a Classic round | Everything frozen until round end; figures follow messages, numbers frozen; everything per action | Figures follow messages, numbers frozen; test against the other two |
| X27 | Combat layout roles | Strip as order summary and review, one message area, commands in a free part of the frame; separate review card (`combat.html` C); bottom panel (B) | Strip + one message area + commands, positions not fixed |
| X28 | Playback disclosure and gesture rules | Show action totals and a queue; show neither | No totals, no next-actor preview; gestures and camera chosen from the action, never from its outcome; results appear with their message |

## 11. Limitations

- **The fixture:**
  - **Same scripted round as `combat.html`:** no deaths, no line changes,
    no monster spells.
  - **`sub`:** monster blocks gained a `sub` field (which member of the
    group acts) for the gesture; nothing else in the fixture changed.
- **Placeholder art.** Capsules and blobs.
  - **Painter's-sort artefacts:** for example, Pell's head clipped by the
    west wall.
  - **Formation:** the in-square formation was moved forward slightly
    (presentation only) so both lines fit the battlefield view.
- **One corridor, one camera side, one encounter size.** See §9, questions
  1–3.
- **Not built:**
  - **Accessibility:** no VoiceOver live region, no reduced-motion design
    beyond skipping tweens.
  - **Sound.**
  - **Keyboard focus:** no management.
- **Headless testing.** Headless browser testing cannot run animation
  frames. The flow was tested with simulated frames: keyboard orders,
  legality refusals, editing, playback, show-rest, repeat, automatic
  playback, the chest and the return. There were no script errors.
