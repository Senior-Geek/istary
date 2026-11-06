# Combat event stream: future controller requirements

Status: **requirements for a later milestone, not implemented** (2026-11-06).
This lists what the Go controller would need to supply for the presentation
model in `COMBAT-PRESENTATION-MODEL.md` (X25–X28 approved) and its validation
in `COMBAT-DIFFICULT-CASES.md`. It is kept separate from that design so the
two can be reviewed independently. Nothing here changes the engine, the save
format or the rules.

## Architectural requirements (from the approval, 2026-11-06)

- **One source for events.** Structured combat events must come from the
  same Go code that produces the combat messages.
- **No string parsing.** The GUI must never parse message strings to
  determine combat results.
- **Unrevealed information stays in the controller.** It is not merely
  hidden in the frontend: what is not yet shown is never sent.
- **M5 first.** The M5 controller extraction is a prerequisite for
  application integration. Combat order entry still lives in the terminal
  program.

## Where things stand today (Facts)

- **Resolution:** `ExecuteRound` resolves the whole round
  (`engine/combat.go:603`).
- **Messages:** output is a list of strings (`cs.Messages`). Action blocks
  are separated by `ActionSeparator` (`engine/combat.go:3350–3359`). There is
  no structured record of who acted, on whom, or what changed.
- **Display snapshot:** captured at round start and frozen for the round
  (`DisplayAliveCounts`, `DisplayPartySnap`, `engine/combat.go:634`).
- **Hidden state:** initiative slots and monster victims are assigned
  before any action is shown (`engine/combat.go:521–578`, `1016`).
- **Team re-sort:** the team is re-sorted by status at the start of
  `ExecuteRound` (`engine/combat.go:610`), after the orders were entered by
  slot. Orders (`cs.Actions`, `engine/combat.go:199`) are not reordered with
  the members, so an order can pass to another member. A Go probe confirmed
  this (`probes/resort-probe_test.go.txt`).
  - **Classic:** it re-sorts at the end of the round, before the next orders
    (Player's Guide p. 48).
  - **What moves:** any worse status moves back, not only death.
  - **Details:** `COMBAT-DIFFICULT-CASES.md` §4.
- **Order entry:** order menus, legality and target questions are in
  `handleCombatChoose` (`cmd/istary/main.go:3917–4005`).

## Requirements

**Status in the controller (M5a/M5b, 2026-11-06):**
- **E3 and E4, M5a:** order entry is in `app/combat.go`.
- **E4, M5b:** `Commit`, `Advance`, `EndRound` and `ReturnToOrders` are in
  `app/combat_round.go`. `AdvanceAll` is not built; the terminal has no
  "show the rest".
- **E10:** the HAMAN boon prompt pauses `Advance` (`ChooseBoon`).
- **E2 and E3, M5c:** `CombatView()` in `app/combat_view.go`. The round-start
  display is frozen during playback, including identification, and the order
  entry view is included.
- **E11:** this holds for the controller's playback operations
  (`RevealedEvents`), not for the engine state, which stays public. See
  `docs/ARCHITECTURE.md`, "Information boundary".

**E1 spiked (2026-11-06):** `engine/combat_events.go` on the local branch
`spike/p1-combat-events`, not merged. It covers E1's actor, kind, target,
lines and effects, with silent blocks redacted. Results are in
`P1-COMBAT-ENGINE-SPIKE.md`.

**Minimum for the model** (without these the model cannot be built
without parsing strings):

| ID | Requirement | Why the presentation needs it |
|---|---|---|
| E1 | **One structured event per action block, produced by the same code that writes its messages.** Fields: actor (member slot, or group and member index), action kind as performed, target (group, member, or none), the message lines, and the changes the block reports: HP, member condition (paralyzed, asleep, dead, ashed, cured), and for monsters which members died or fell asleep (per member, not only counts; several deaths in one block are common). | Identify the actor, place the gold and white rings, choose the gesture, fade or sleep figures at the right moment. Same code path, so the event and the message cannot disagree. |
| E2 | **Display state as View fields.** The round snapshot (counts, HP, status) and the index of the last action shown. | Numbers stay frozen (Classic) without Svelte keeping its own copy; survives a reload (D6, question 6). |
| E3 | **Order-entry View.** For each member: the order menu with available or not and the reason, spells with points and reasons, target kind, line (front or back) for this round, and the current order. Plus the round number. | The command menu, unavailable reasons, front/back grouping in the strip. |
| E4 | **Commands.** `SetOrder(character, order)` keyed by character identity, never by slot (X29), `ClearOrders`, `Commit`, `Advance`, `AdvanceAll`. Formation commands: reorder, and restore the preferred formation, where allowed (X31, baseline Q-B2). | Order entry, review edits (X19), playback. |
| E5 | **Formation change.** Who moved between lines at the re-sort, with each member's condition, reported before the next order entry. Depends on the re-sort happening before order entry, as in Classic. Go today sorts after it (open question Q-D7). | Show a line change explicitly at the round's update: figures walk, the strip regroups (`COMBAT-DIFFICULT-CASES.md` §4). |

**Needed for the full sequence:**

| ID | Requirement | Why |
|---|---|---|
| E6 | **Identification event.** Groups identified this round, reported before the first action (it is not a message). | Group names update at round start. |
| E7 | **Encounter start.** Groups in order, with counts and unidentified names; the team and lines. | The encounter transition and labels. Monster positions are presentation, not rules. |
| E8 | **Round result.** Continue, victory, fled, or party lost; chest present or not. | Round completion and the move to the chest. |
| E9 | **Chest View.** Available actions and who may do each; the team's reports. | Chest panel (unchanged from the combat review). |
| E10 | **Pause points.** An event that stops playback for a controller command (HAMAN and MAHAMAN, `cmd/istary/main.go:3550`). | Playback must wait, not run ahead. |
| E13 | **Visible appearance per group** in the encounter start (E7): the look the team can see (humanoid, skeleton, slime, flier, large…), the same before and after identification. It must not carry the true identity. | Figures and death styles are chosen from it (X26). |
| E14 | **Explored geometry only.** The renderer receives the explored surfaces from the controller, never the level. | In rooms, a renderer holding the true level would expose surfaces from any camera (`COMBAT-DIFFICULT-CASES.md` §6). |
| E15 | **Who can act**, per member, with the reason (paralyzed, dead…), in the order-entry View (E3). | "Cannot act · paralyzed" in the strip; no menu for that member. |

**Must not be sent before it is shown** (information boundary):

| ID | Rule |
|---|---|
| E11 | No initiative order, no count of remaining actions, no monster victims, no true identities of unidentified groups, no chest trap. Events are released to the View one at a time as `Advance` is called, or held in the controller and exposed only up to the shown index. |
| E12 | An event's outcome fields must not be readable before the event is shown. The presentation chooses gestures and any camera move from the action kind (X28). |

The engine and controller work these require, in order, is listed in
`COMBAT-DESIGN-BASELINE.md` §6.

## Open points for the controller design

These are not decided here.

- **Event storage:** whether events are stored with the round (D6, questions
  1–2: commit point, queue persistence) or rebuilt from it.
- **Monster member index (E1):** whether the index of the acting member is a
  rules fact (Classic tracks members within a group) or a presentation
  choice.
- **Event granularity:** whether a block with several hits (a fighter
  striking twice) is one event or several. The model only needs one.
- **The re-sort and order entry (E5):** checked against Classic
  (`COMBAT-DIFFICULT-CASES.md` §4). E5 assumes Classic timing. If Go's timing
  is kept, the GUI could not show the new lines before orders are given;
  that is question Q-D7, a gameplay question for M1.
- **Pacing preference:** where the T)IME delay lives (`cmd/istary/main.go:152`)
  and whether a playback-speed preference replaces it in Modern
  presentation.
