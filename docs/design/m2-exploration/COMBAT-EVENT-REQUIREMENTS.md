# Combat event stream: future controller requirements

Status: **requirements for a later milestone, not a design decision and not
implemented** (2026-11-06). This lists what the Go controller would need to
supply for the presentation model in `COMBAT-PRESENTATION-MODEL.md`. It is
kept separate from that design so the two can be reviewed independently.
Nothing here changes the engine, the save format or the rules. Combat order
entry currently lives in the terminal program and needs the M5 controller
extraction first.

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
- **Team re-sort:** the team is re-sorted by status at the start of each
  round (`engine/combat.go:610`).
- **Order entry:** order menus, legality and target questions are in
  `handleCombatChoose` (`cmd/istary/main.go:3917–4005`).

## Requirements

**Minimum for the model** (without these the model cannot be built
without parsing strings):

| ID | Requirement | Why the presentation needs it |
|---|---|---|
| E1 | **One structured event per action block, produced by the same code that writes its messages.** Fields: actor (member slot, or group and member index), action kind as performed, target (group, member, or none), the message lines, and the changes the block reports (HP, status, alive count, asleep count). | Identify the actor, place the gold and white rings, choose the gesture, fade or sleep figures at the right moment. Same code path, so the event and the message cannot disagree. |
| E2 | **Display state as View fields.** The round snapshot (counts, HP, status) and the index of the last action shown. | Numbers stay frozen (Classic) without Svelte keeping its own copy; survives a reload (D6, question 6). |
| E3 | **Order-entry View.** For each member: the order menu with available or not and the reason, spells with points and reasons, target kind, line (front or back) for this round, and the current order. Plus the round number. | The command menu, unavailable reasons, front/back grouping in the strip. |
| E4 | **Commands.** `SetOrder(member, order)`, `ClearOrders`, `Commit`, `Advance`, `AdvanceAll`. | Order entry, review edits (X19), playback. |
| E5 | **Formation change.** Who moved between lines at the round-start re-sort, reported before the next order entry. | Show a line change explicitly between rounds instead of figures silently swapping. |

**Needed for the full sequence:**

| ID | Requirement | Why |
|---|---|---|
| E6 | **Identification event.** Groups identified this round, reported before the first action (it is not a message). | Group names update at round start. |
| E7 | **Encounter start.** Groups in order, with counts and unidentified names; the team and lines. | The encounter transition and labels. Monster positions are presentation, not rules. |
| E8 | **Round result.** Continue, victory, fled, or party lost; chest present or not. | Round completion and the move to the chest. |
| E9 | **Chest View.** Available actions and who may do each; the team's reports. | Chest panel (unchanged from the combat review). |
| E10 | **Pause points.** An event that stops playback for a controller command (HAMAN and MAHAMAN, `cmd/istary/main.go:3550`). | Playback must wait, not run ahead. |

**Must not be sent before it is shown** (information boundary):

| ID | Rule |
|---|---|
| E11 | No initiative order, no count of remaining actions, no monster victims, no true identities of unidentified groups, no chest trap. Events are released to the View one at a time as `Advance` is called, or held in the controller and exposed only up to the shown index. |
| E12 | An event's outcome fields must not be readable before the event is shown. The presentation chooses gestures and any camera move from the action kind (X28). |

## Open points for the controller design

These are not decided here.

- **Event storage:** whether events are stored with the round (D6, questions
  1–2: commit point, queue persistence) or rebuilt from it.
- **Monster member index (E1):** whether the index of the acting member is a
  rules fact (Classic tracks members within a group) or a presentation
  choice.
- **Event granularity:** whether a block with several hits (a fighter
  striking twice) is one event or several. The model only needs one.
- **The re-sort and order entry (E5):** how the round-start re-sort
  interacts with order entry, which needs verifying against Classic before
  E5 is specified.
- **Pacing preference:** where the T)IME delay lives (`cmd/istary/main.go:152`)
  and whether a playback-speed preference replaces it in Modern
  presentation.
