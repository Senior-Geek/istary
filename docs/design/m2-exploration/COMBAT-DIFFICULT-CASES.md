# Combat difficult cases: validation

Status: **design validation for review** (2026-11-06). This tests the
approved combat presentation (X25–X28, `COMBAT-PRESENTATION-MODEL.md` §10)
against harder situations before any application work. It changes no rule,
no engine code, no application code, no game data and no save format. Labels
follow the README: Approved, Fact, Hypothesis, Alternative, Decision, Risk.
Open questions are listed separately in §10 and are not decisions.

> **Six adventurers, one battlefield, one action at a time.**

## 1. What to open

- **Prototype:** `mockups/combat-refined.html` in Safari or Chrome, no
  server. In the toolbar choose **Encounter → Difficult cases (4 groups)**.
  The side panel lists 15 moments; **go** jumps to each.
- **Location:** the second Encounter menu (corridor, room, junction, narrow
  dead end).
- **Review options:**
  - **Re-sort: Classic / Go today:** when the party is re-sorted (§4).
  - **Label at message:** the strip's condition label changes at the message
    instead of at round end (open question Q-D1).
- **Checks** (Node, no dependencies), in `mockups/`:
  - `node fixture-check.js`: the sort port against Go's output, Go-timing
    preview, outcomes stated by their messages, and pacing.
  - `node leak-check.js`: camera exposure for every location, and that
    monster placements stand on seen floor.
- **Go probe:** `probes/resort-probe_test.go.txt`, the scratch probe that
  established the re-sort finding, with its output. It was run from a
  temporary copy and is not part of the build.
- **Stills:** `stills/combat-difficult/` (§11).

The original encounter is unchanged and still the default. It now shows
deaths instead of fades (X26).

## 2. Approved decisions recorded

Recorded in `COMBAT-PRESENTATION-MODEL.md` §10 and the README register:

- **X25:** the stable camera.
- **X26:** results shown with their message, including the death animations
  and the party-condition addendum.
- **X27:** the layout.
- **X28:** the playback rules.

This validation keeps to them. Treatments marked **provisional** are not
approved final artwork.

## 3. The difficult-case fixture

`mockups/combat-fixture-hard.js` stands in for the Go controller. It has the
same interface as the original fixture and scripted outcomes. The team is the
same six; Brannoc is level 5 here so that he knows DIALKO.

Four groups, all unidentified at first. The "look" is presentation (§7).

| Group | Shown as | True name | Look | Size |
|---|---|---|---|---|
| 1 | SMALL HUMANOIDS | BOGLINS | humanoid | 7 (two rows) |
| 2 | SKELETAL FIGURES | BONE WALKERS | skeleton | 2 |
| 3 | WINGED CREATURES | GLOOM WINGS | flier | 3 |
| 4 | HUGE FIGURE | CAVE HULK | large | 1 |

| Round | What happens |
|---|---|
| 1 | KATINO puts five of the seven humanoids to sleep. Torvald kills a sleeper. A skeleton claws Ysolde: **YSOLDE IS PARALYZED** (front line). Her own action is skipped without a message. Corvin's rod kills three humanoids in one action. The huge figure hits Torvald for 31: **TORVALD IS SLAIN!** Brannoc's BADIOS kills a flier. |
| 2 starts | **Re-sort** (Classic timing): front line BRANNOC, PELL, ILSA; back line CORVIN, YSOLDE (paralyzed), TORVALD (dead). Ilsa, a mage, is now in the front line. Ysolde and Torvald are not offered orders. |
| 2 | Brannoc casts DIALKO: **YSOLDE IS CURED!** Corvin's KATINO puts both fliers to sleep, and they come down to the floor. Pell kills a sleeper. Groups 1 and 2 are identified (silently). |
| 3 starts | No move: Ysolde is OK again but stays in the back line, because the sort never moves an OK member past another OK member. |
| 3 | Brannoc dispels both skeletons. Pell and Ilsa kill the last humanoids. Corvin's rod kills both sleeping fliers. |
| 4 | Brannoc kills the hulk. **Victory**, no chest. Back to first person; the team explores on with Torvald still in the six. |

## 4. The party re-sort: when it happens (a finding)

- **Fact (Classic, Player's Guide):** "Should one of your characters die in
  combat, he or she will automatically be shifted to the 'dead-last' position
  in the party at the end of the round … the party member previously in
  fourth place will move up to the number-three slot … This character will
  be able to fight" (printed p. 48, PDF p. 54).
  - **Incapacitated members too:** p. 55 (PDF p. 61) says the same of a
    front-line member "killed or incapacitated, such as getting hit with a
    KATINO spell".
- **Fact (engine comments):** the engine's comments place Classic's sort in
  DSPPARTY, run at the start of each round in CUTIL
  (`engine/combat.go:225`, `280`, `634`). CUTIL is also where orders are
  chosen (`cmd/istary/main.go:3770`, `3927`). So Classic sorts before order entry.
- **Fact (Go today):**
  - **Where the sort runs:** `sortPartyByStatus` runs at the start of
    `ExecuteRound` (`engine/combat.go:610`), after the orders have been
    entered, and at combat start (`:282`).
  - **How orders are stored:** by slot, in `cs.Actions` (`engine/combat.go:199`).
    The sort permutes the party but not `cs.Actions`.
- **Verified with a scratch Go probe** (`probes/resort-probe_test.go.txt`).
  The team was ALDA, BRAM (dead), CORA, DREW, EDDA, FINN, and CORA was given
  Run in slot 3. After the re-sort, **DREW FAILED TO RUN!**: DREW carried out
  CORA's order.
- **Read from the code, not run:**
  - **Stale orders:** members who were not offered an order keep the previous
    round's order in their slot. `cs.Actions` is only written for the actor
    being ordered and cleared by B)ack (`cmd/istary/main.go:3543`, `3812`…).
    A member who moves into such a slot carries out that old order.
  - **Ally targets:** an ally target (DIALKO, DIOS) is also a slot index, so
    it can land on a different member.
  - **Not modelled:** what Go does when a member is handed an order it could
    not have chosen (a spell it does not know).
- **Fact (sort behaviour):** the sort is an exchange sort on status, not
  stable. It moves any worse status back, not only death: asleep, afraid,
  paralyzed, stoned, dead, ashed, lost (`engine/combat.go:585`). The
  fixture's port matches Go's output on four cases (`fixture-check.js`).
  - **Consequence:** an OK member never moves forward past another OK
    member, so a cured member stays where the sort left them.
- **Risk / compatibility:** Go's timing differs from Classic and can give an
  order to the wrong member. This is a gameplay discrepancy, not a
  presentation one. Nothing was changed: engine changes were out of scope.
  - **Recommendation:** record it in `WIZARDRY-COMPATIBILITY.md` and resolve
    it in M1, with a characterization test, before the GUI depends on it.
    That is your decision; it is question Q-D7.
- **Presentation adopted in the prototype (Classic timing).** At the round's
  display update:
  - the values update and the re-sort happens in one step;
  - figures walk to their new places (1 s);
  - the strip shows the new lines at once;
  - a one-line notice names who moved, for example: *"New lines this round:
    YSOLDE (paralyzed) and TORVALD (dead) to the back; PELL, ILSA to the
    front."*

  The notice is presentation (requirement E5); Classic simply redraws the
  party. **Re-sort: Go today** shows instead what Go would do: a table of
  "given for → carried out by" at the moment of Fight.

## 5. Four groups, large groups, sleeping creatures

- **Placement rule** (`CS.layout`):
  - **Floor used:** only the floor the team can see from its square: its
    column ahead, up to the first visible wall. Placement depends on nothing
    the team has not seen.
  - **Order:** group 1 is nearest.
  - **Rows:** up to four per row; a large creature takes two places.
  - **Staggering:** narrower groups after the first stand alternately west
    and east of the centre line, so the group behind is not hidden.
  - **Short floor:** where the visible floor is short (a dead end), rows
    compress and start nearer the team.
- **Labels:**
  - **Counts:** one per group; the count stays on the Classic snapshot. Group
    1 reads "7 SMALL HUMANOIDS" for all of round 1 while five fall asleep and
    four die, and "3 SMALL HUMANOIDS · 1 asleep" after the update.
  - **Collisions:** where labels would collide with each other or with the
    message area, a small pass moves them up or to the right. Four labels
    stay readable in every location tried.
- **Sleeping creatures:**
  - **Look:** they slump grey with a *z*.
  - **Fliers:** they come down to the floor while asleep.
  - **Mixed groups:** in a group that is partly asleep, only the sleepers
    change. The fixture reports each figure, not a count.
- **Fixed during this work: occlusion.** Architecture is now always drawn
  behind figures. Figures stand in the team's column and every camera is
  inside it, so no wall can be in front of a figure. Before this, a painter's
  sort on large wall polygons hid the large creature in the dead end and
  clipped Pell's head. `combat.html` uses the same renderer and so changes
  the same way; its stored stills are unchanged.
- **Finding: crowding in a dead end.**
  - **Space:** with one square ahead, four groups (13 figures) fit into
    about 1.8 units of depth.
  - **What still reads:** figures overlap, but the labels, the group rings
    and the action highlighting still identify them (stills L3, L4).
  - **What a real renderer needs:** a crowding rule (smaller figures,
    tighter rows, or fewer figures than the count) is question Q-D5.

## 6. Locations and camera framing

The camera is unchanged (X25): elevated, behind the team, east of the axis,
inside the team's square, for the whole encounter. The east side worked in
all four locations, so the side rule stays "always east" for now.

`node leak-check.js` checks two things.

1. **Exposure.** For each shot it counts the never-seen surfaces that a
   renderer holding the true level would expose, with occlusion. That count
   is compared with a true-geometry first-person view from the team's
   square.
2. **Placement.** It checks whether every monster placement stands on seen
   floor.

| Location | Battlefield and push-in shots | Chest close-up | First person | Placements off seen floor |
|---|---|---|---|---|
| Corridor | 0 | 0 | 0 | 0 |
| Junction (crossing one square ahead) | 0 | 0 | 0 | 0 |
| Narrow dead end | 0 | 0 | 0 | 0 |
| Room (3 × 3, team inside) | 11–12 exposed; **5–6 beyond first person** | 1 (0 beyond) | 6 | 0 |

- **Finding: rooms.** Classic's visible set (the DRAWMAZE walk) does not
  include the floors of open side squares.
  - **First person:** even a true-geometry first-person view in a room shows
    six surfaces outside it.
  - **The elevated camera:** it adds the side squares of the team's own row
    and the far side walls of the next row (which reveal the room's width).
  - **Framings tried:** a narrower or more forward camera reduces this. Only
    a camera inside the formation removes it, which puts the back line behind
    the lens.
- **What the prototype does:** it draws only the visible set (Strict or
  Neutral shell). Rooms therefore show a lit lane with darkness at the sides
  (stills L1, L5), and nothing unexplored is drawn with any camera. The
  check's counts describe a future renderer, not the prototype.
- **Requirement:** a renderer that holds the real level could expose these
  surfaces. The controller must therefore give the renderer explored
  geometry only (requirement E14). Whether rooms may show more than Classic
  (a "remembered" or line-of-sight policy) is question Q-D3.

## 7. Conditions and deaths (provisional treatments)

All selected from the **revealed result** and the creature's **visible
look**, never from hidden engine data. All begin at the message's impact
moment.

**Party** (X26 addendum). The strip's label remains the authoritative
condition.

| Condition | Figure | Strip |
|---|---|---|
| Paralyzed | Upright and recognisable; flat, cooler colour; thin pale rim; takes no part | PARALYZE in pale blue; "cannot act · paralyzed" |
| Asleep / unconscious | Slumped to half height, head dropped, *z*; never like death | ASLEEP |
| Dying | Staggers back, falls (about 1.1 s), then the fallen body gives way to… | (label at round end) |
| Dead | …a translucent, dashed figure that stays in its place: the member is still with the team | DEAD; card subdued and dashed; "cannot act · dead" |
| Ashes | A low grey mound with a few faint motes | ASHED |

The translucent figure is a visual convention only: no spirit, no mechanic.
The dead member stays in the six (the fixture checks this). The team explores
on, and returning for resurrection is the player's choice.

**Monsters:**

| Look | Death style | Time |
|---|---|---|
| Humanoid | Staggers back, collapses | 0.9 s |
| Skeleton | Falls apart: skull, ribs and legs drop and scatter | 1.0 s |
| Slime | Flattens, spreads and darkens into a puddle | 1.1 s |
| Flier | Tumbles from its height to the floor | 0.9 s |
| Large | Sways, drops to its knees, falls forward, dust | 1.2 s |

- **Several deaths in one action:** they start 120 ms apart (still D04).
- **Remains:** they stay until the round's display update, when Go compacts
  the groups (`engine/combat.go:92`). Survivors keep their places.
- **Skipping:**
  - **Next during an animation** completes it at once, with the outcome kept.
  - **Show the rest** applies every remaining outcome in its final state.
  - The keyboard test checks both.
- **Look and identity:**
  - **The rule:** the look is what the team can already see; an unidentified
    group has the same look as when identified.
  - **What the look must not do:** be derived from the true monster in a way
    the shown appearance does not already reveal.
  - **Art:** real art must follow the same rule (model §9 question 7).

## 8. Playback pacing with death animations

From `node fixture-check.js`, automatic playback with the default orders. An
action holds for its reading time, or until a death it reveals has finished,
whichever is longer.

| Round | Actions | Deaths | Normal speed: without deaths / with | Fast speed: without / with | Actions lengthened (fast) |
|---|---|---|---|---|---|
| 1 | 12 | 6 | 34.4 s / 34.4 s | 19.1 s / 19.1 s | 0 |
| 2 | 8 | 1 | 21.9 s / 21.9 s | 12.2 s / 12.2 s | 0 |
| 3 | 6 | 6 | 17.4 s / 17.4 s | 9.7 s / 9.7 s | 0 |
| 4 | 4 | 1 | 11.2 s / 11.2 s | 6.2 s / 6.3 s | 1 |

- **Finding: deaths do not slow playback.** The 0.7–1.2 s styles run inside
  the message's reading time. Only at fast speed did one action grow, by
  0.1 s.
- **Finding: messages set the round's length.** A 12-action round takes about
  34 s at normal speed. Round length comes from reading time, not animation.
  Manual playback and *Show the rest* remain the answer.
- **Manual play:** a quick Next completes the death at once, so no outcome is
  lost and nothing waits for animation.

## 9. Information boundaries (checked)

| Rule | How the validation meets it |
|---|---|
| No result before its message | `fixture-check.js`: every reported outcome is stated by its own block. The keyboard test confirms that the paralysis and death begin only at the message. |
| Gestures and styles from the action and the look | The lunge comes from the order; the death style comes from the visible look; the push-in comes from the action kind (X25, X28). |
| Incapacitated members take no part | Not offered orders (as `findNextActor`); no gesture; their skipped actions produce no event. |
| Classic numbers | HP, status labels and counts change only at the round update (Q-D1 option aside). |
| No hidden information | True names, the order of remaining actions and future targets stay in the pink "engine only" box; the Go-timing table appears only under the reviewer option. |
| Explored space only | Placements on seen floor; the renderer is given the visible set only (§6). |
| No new rules | The fixture's outcomes are scripted; the one engine behaviour reproduced (the sort) is a port checked against Go. |

## 10. Open questions (not decisions)

- **Q-D1. Strip condition label timing.** The addendum asks for "the party
  strip's condition label authoritative" and also for Classic's round-end
  numbers.
  - **Default:** the label changes at round end, like the numbers (Classic).
  - **Option:** **Label at message** changes it when the message reveals it,
    as the figure does (still D15).
  - **Choice:** this is your call.
- **Q-D2. Formation notice.** Keep the one-line notice at round start, or rely
  on the figures walking and the strip regrouping?
- **Q-D3. Rooms.** Draw only Classic's visible set (darkness at the sides, as
  now), or allow a remembered or line-of-sight policy for rooms? This needs a
  rule for what counts as explored.
- **Q-D4. Camera side.** "Always east" held in all four locations. Revisit
  only if real maps produce a case it cannot handle.
- **Q-D5. Crowding.** How to stage up to four large groups in one visible
  square: smaller figures, tighter rows, or figures standing for the count?
- **Q-D6. Remains.** Keep remains until the round's update (as now), or fade
  them sooner?
- **Q-D7. Compatibility.** Should the Go re-sort timing (§4) be logged as a
  discrepancy and fixed in M1 before GUI combat? It changes gameplay, so it
  is outside this assignment.
- **Q-D8. Asleep members.** `findNextActor` offers orders to ASLEEP and AFRAID
  members (`IsAlive`, `engine/character.go:306`). Whether Classic did is not
  verified; the fixture has no asleep member.
- **Q-D9. Repeat orders.** After a re-sort, *Repeat* follows each member, not
  the slot, and Go still decides legality. Confirm this is the intended
  behaviour.

## 11. Stills (`stills/combat-difficult/`)

| Still | Moment |
|---|---|
| 00 | Full page: round 2 order entry, with the reviewer panel |
| D01 | Four groups, one large, in the corridor; order entry |
| D02 | KATINO: five of seven fall asleep |
| D03 | A front-line member is paralyzed |
| D04 | Three die in one action (mid-animation) |
| D05 | A member is slain (collapsing) |
| D06 | …and stays with the team (translucent figure) |
| D07 | Round end: new values and new lines, figures walking |
| D08 | Round 2 order entry: two cannot act; Ilsa in the front line |
| D09 | The paralysis is cured; the card still reads PARALYZE (Classic) |
| D10 | Fliers fall asleep and land |
| D11 | Skeletons fall apart |
| D12 | A large collapse |
| D13 | Victory; exploring on with the dead member in the strip |
| D14 | Re-sort at Go's timing today: orders follow slots |
| D15 | Variant: the strip label changes at the message (Q-D1) |
| L1–L3 | Room, junction and dead end at order entry |
| L4 | Dead end: three die in one action |
| L5 | Room: a member is slain |
| G1–G3 | Condition and death-style gallery, start, middle and end |
| S1 | Sheet: a member's death, progress 0 → 1 |
| S2 | Sheet: the five monster death styles, progress 0 → 1 |

## 12. Limitations

- **Scripted outcomes.** Monster turns and party results are scripted for one
  encounter. Only the re-sort reproduces engine behaviour.
- **Placeholder art.** Capsules and simple shapes. The death styles show
  timing and character, not final art.
- **One elevation and one field of view.** As approved; no per-location
  camera.
- **Headless tests.** Animation frames were simulated. A keyboard run of the
  whole difficult encounter (20 checks) and of the original encounter
  (8 checks) passed with no script errors. Each jump was also loaded with no
  console errors.
- **Not built:** accessibility for the animations (reduced motion skips them
  to their end state) and sound.
