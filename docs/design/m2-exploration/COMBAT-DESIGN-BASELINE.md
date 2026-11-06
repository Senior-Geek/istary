# M2 combat design baseline and acceptance report

Status: **baseline for review** (2026-11-06). It consolidates the M2 combat
exploration into one reference: approved decisions, provisional assumptions,
open questions, the engine and controller work required before GUI combat,
and the next bounded prototype. Documentation only: no engine, application,
game-data or save-format change.

> **Six adventurers, one battlefield, one action at a time.**

**Sources.** The material is consolidated from these documents, which are
not rewritten; this document points into them:

| Document | What it covers |
|---|---|
| `COMBAT-EXPERIENCE-REVIEW.md` | Approaches A/B/C, X17–X24 |
| `COMBAT-PRESENTATION-MODEL.md` | The refined model, X25–X28 |
| `COMBAT-DIFFICULT-CASES.md` | Validation, Q-D1–Q-D9 |
| `COMBAT-EVENT-REQUIREMENTS.md` | E1–E15 |
| `../../WIZARDRY-COMPATIBILITY.md` | Finding #15, row F3 |

## 1. Governing principle (approved 2026-11-06)

**Preserve Wizardry's gameplay challenges; eliminate unnecessary
software-era interface and implementation quirks.**

A quirk is eliminated only if removing it changes neither what the player
must decide nor the information legitimately available. A behaviour that
changes outcomes is a rules question and follows the charter (§5.1: evidence
and the compatibility decision log), not a presentation change.

## 2. Approved decisions

### 2.1 Presentation (approved earlier, recorded in the model §10)

| ID | Decision |
|---|---|
| X25 | Stable elevated camera behind the team, monsters ahead; one viewpoint for the encounter; no reverse shots or cuts. Restrained push-in only as an optional, experimental preference |
| X26 | Figures show a result when its message reveals it, never before. Monsters visibly die; sleeping, incapacitated and dead figures show their condition. Classic round-end numbers kept. Brief, skippable death animations chosen from the revealed result and the visible look. Party addendum: paralyzed rigid, asleep slumped, dead → translucent figure, ashes distinct; no new mechanics |
| X27 | Six-member strip with front and back lines as status, order summary and review; one message area; contextual commands in a free part of the frame; gold, white and green dashed highlighting |
| X28 | Go resolves the round; one action at a time. No initiative order, future targets, remaining count or unrevealed outcomes. Gestures come from the action; skipping never discards an outcome |

### 2.2 Formation, orders and representation (approved 2026-11-06)

| ID | Decision | Classification |
|---|---|---|
| X29 | **Combat orders stay attached to character identity.** An order never passes to another member because positions change. This applies to orders, ally targets and repeated orders. | Correction of an implementation quirk. The Classic evidence agrees: the re-sort came before order entry (§3) |
| X30 | **Incapacitated characters move toward the rear by the game's formation rules.** These are the engine's status order, as `sortPartyByStatus`. | Rule unchanged |
| X31 | **Recovery does not silently rearrange the formation.** A cured member stays where the rules left them. The player has a straightforward way to restore or manually reorder the formation, without removing and re-adding characters. | Rule unchanged; restoring is an interface convenience over the existing reorder (where it may be used: Q-B2) |
| X32 | **Dead characters stay represented in the party** with a distinct translucent treatment, and paralysis looks different from death (confirms the X26 addendum). | Presentation |
| X33 | **The continuous dungeon environment and the refined hybrid presentation are kept.** That means an immersive scene, contextual order entry, and an expandable team and order review. This settles X1 and X7 (continuous environment, same corridor) for combat. It takes refined approach C (X17) as the combat direction. | Presentation |
| X34 | **Three kinds of state stay distinct.** They are authoritative engine state, displayed state (for example the Classic round snapshot), and information legitimately known to the player. The GUI holds only the last two, and the controller decides what is known. | Architecture |

## 3. Historical observations and Istary corrections

These two kinds of material are recorded separately. An observation states
what the original or the inherited engine does. A correction states what
Istary intends.

**Historical observations** (facts, with sources):

| # | Observation | Source |
|---|---|---|
| H1 | Classic: a member who dies is moved to the "dead-last" position *at the end of the round*; the member behind moves up and can fight | Player's Guide printed p. 48 (PDF p. 54) |
| H2 | Classic: a front-line member "killed or incapacitated", such as by KATINO, is replaced by the next member | Guide p. 55 (PDF p. 61) |
| H3 | Classic: reordering is a Camp action (R)EORDER, choosing members in order) | Guide p. 41; engine `cmd/istary/main.go` Camp `R` |
| H4 | Engine: the status sort is an exchange sort on status. Any worse status moves back (asleep, afraid, paralyzed, stoned, dead, ashed, lost). It is not stable, and an OK member never moves past another OK member, so a cured member stays back | `engine/combat.go:585`; Go probe |
| H5 | Engine: the sort runs at the start of `ExecuteRound`, after orders were entered by slot. The engine's comments place Classic's sort at the top of the round, in CUTIL, before orders | `engine/combat.go:610`, comments at `:225`, `:280`, `:634` |
| H6 | Engine: orders are stored by slot (`cs.Actions`) and not reordered with the members, so an order passes to whoever moved into the slot. Verified: an order given for one member was carried out by another | `engine/combat.go:199`; `probes/resort-probe_test.go.txt` |
| H7 | Engine (read, not run): skipped slots keep the previous round's order; ally targets are slot indices | `cmd/istary/main.go:3543`, `3812`…; `engine/combat.go:1843` |
| H8 | Engine: ASLEEP and AFRAID members are offered orders (`IsAlive`). Classic behaviour not verified | `cmd/istary/main.go:4097`, `engine/character.go:306` |

**Istary corrections** (intentional improvements, approved as principles;
not implemented):

| # | Correction | Replaces | Gameplay effect |
|---|---|---|---|
| C1 | Orders, ally targets and repeated orders are bound to the character, not the slot (X29) | H6, H7 | None intended. It restores the behaviour the guide implies, where no order could transfer |
| C2 | The formation change is applied and shown at the round boundary, before order entry, so orders and Fight eligibility are given against the lines that will act | H5 | Matches H1. The timing is a provisional assumption until confirmed in M1 (A1) |
| C3 | Stale orders never carry over; members who cannot act have no order | H7 | None intended |
| C4 | A "restore formation" convenience returns the team to the player's chosen order using the existing reorder action, where reordering is allowed | Re-adding characters, or reordering by hand each time | None, if limited to where reorder is already legal (Q-B2) |

C1–C3 change engine behaviour. Under charter §5.1 each needs a compatibility
decision record and tests before it is implemented. They are logged in
`WIZARDRY-COMPATIBILITY.md` (finding #15, decision log).

## 4. Provisional assumptions

These are working assumptions; each may change after review in the browser
or an M1 finding.

| # | Assumption | Basis |
|---|---|---|
| A1 | The re-sort timing is the end of the round, before the next order entry (C2) | H1, H5 |
| A2 | The camera stays on the east side of the axis, inside the team's square | Held in a corridor, junction, dead end and room (difficult cases §6) |
| A3 | Monsters are placed on the visible floor only; group 1 nearest; narrow groups staggered; rows compress in short spaces | Difficult cases §5 |
| A4 | Death styles and timings: humanoid 0.9 s, skeleton 1.0, slime 1.1, flier 0.9, large 1.2, member 1.1. Several deaths start 120 ms apart | Difficult cases §7–8; playback is not slowed |
| A5 | Remains stay until the round's display update | Difficult cases §7 |
| A6 | Party treatments: paralyzed with a cold tint and pale rim; asleep slumped with a *z*; dead as a dashed translucent figure; ashes as a mound with motes | Placeholder art; final art not chosen |
| A7 | A one-line formation notice at the round boundary | Difficult cases §4 |
| A8 | The strip label changes at round end (Classic), like the numbers | Default of Q-D1 |
| A9 | "Expandable" review: the strip is the compact review. An expanded view is borrowed from approach B's structured review and not yet built in the refined page | X33; combat review §3 |
| A10 | A monster's look comes from its visible appearance and is the same identified or not | Requirement E13 |

## 5. Unresolved questions

| # | Question | Source |
|---|---|---|
| Q-B1 | Confirm the re-sort timing (A1) in M1, with a characterization test and historical evidence | H1, H5 |
| Q-B2 | Where may "restore formation" or a manual reorder be used? Camp only, as Classic? Also outside Camp between encounters? Never during combat without an enhancement, since reordering mid-combat would change who can be hit? | X31, charter §5.3 |
| Q-B3 | Is the player's preferred formation remembered? If across sessions, it is a save-format change that needs approval (M1b). If not, only for the expedition | X31 |
| Q-D1 | Strip condition label: round end or at the message | Difficult cases §10 |
| Q-D2 | Keep the formation notice? | Difficult cases §10 |
| Q-D3 | Rooms: Classic visible set only, or a remembered or line-of-sight policy (relates to X3) | Difficult cases §6 |
| Q-D5 | Crowding: staging four large groups in one visible square | Difficult cases §5 |
| Q-D6 | Remains until the update, or fade sooner | Difficult cases §7 |
| Q-D8 | Should ASLEEP and AFRAID members be offered orders (H8)? | Difficult cases §10 |
| — | D6 commit semantics: when a round counts as committed; whether unseen actions survive a quit; pause points | Combat review §7 |
| — | Monster art must not reveal an unidentified monster's identity (D12 for original pictures) | Model §9 Q7 |

Questions the approvals settle:
- **Q-D7** (Go's timing) is settled by X29 and C1–C2.
- **Q-D9** (repeat follows the member) is settled by X29.
- **Q-D4** (camera side) became assumption A2.

## 6. Engine and controller work required before GUI combat

In dependency order. None of it is started; each item is bounded and testable
without a GUI.

1. **M1 characterization tests (observe only).**
   - Pin today's behaviour: the re-sort timing, slot-bound orders, stale
     orders, slot-bound ally targets, and orders offered to asleep members.
   - Golden traces must stay as they are.
   - This gives the baseline that C1–C3 will deliberately change.
2. **Compatibility decision records for C1–C3.**
   - Record the evidence (H1–H7), the decision and the tests in
     `WIZARDRY-COMPATIBILITY.md` §6.
   - Golden traces that change because of C1–C3 are re-recorded only with
     that decision.
3. **Engine corrections** (after 1–2):
   - **Identity:** orders and ally targets are keyed by a stable character
     identity (architecture §2.5).
   - **Timing:** the status sort runs at the round boundary, before order
     entry.
   - **Stale orders:** orders are cleared each round.
   - **Classic behaviour:** for orders that did not transfer, Classic
     behaviour is unchanged.
4. **Formation actions in the controller** (M5):
   - **Reorder:** the existing Camp reorder.
   - **Restore formation:** where Q-B2 allows it.
   - **The preferred order:** where Q-B3 stores it.
5. **M5 combat controller extraction:**
   - **Legality:** move order legality from `handleCombatChoose`
     (`cmd/istary/main.go:3917–4005`) into `app.Controller`.
   - **Order-entry View:** with reasons, who can act, and the lines (E3,
     E15).
   - **Commands:** `SetOrder(character, order)`, `ClearOrders`, `Commit`,
     `Advance`, `AdvanceAll` (E4).
   - **Timers:** no timers in the controller.
6. **Structured combat events (E1)**, emitted by the same code that writes
   each message block.
   - **Contents:** actor, action kind, target, lines, HP and condition
     changes, and per-member monster deaths and sleep.
   - **Formation change (E5):** reported before order entry.
   - **Also:** identification (E6), encounter start with each group's
     visible look (E7, E13), round result (E8), pause points (E10).
7. **Display state and release control.**
   - **Display state:** the round snapshot and the shown index are View
     fields (E2).
   - **Release:** events are released one at a time; nothing unrevealed
     leaves the controller (E11, E12, X34).
8. **Explored geometry for the renderer (E14).** The controller supplies
   explored surfaces only; the renderer never receives the level.

Items 1–3 change or protect rules. Items 4–8 expose existing rules through
the controller. GUI combat (M7) depends on all of them. The renderer spike
(M6) depends on item 8.

## 7. Acceptance report

| Criterion | Result | Evidence |
|---|---|---|
| One coherent combat sequence from exploration to return | Met | `combat-refined.html`, original and difficult encounters; stills `combat-refined/`, `combat-difficult/` |
| Stable camera, no leaks through camera or rendering | Met for a corridor, junction and dead end. In rooms, nothing unseen is drawn; a true-geometry renderer would expose surfaces (Q-D3, E14) | `leak-check.js` |
| One action at a time, no early disclosure | Met | `fixture-check.js` (every outcome stated by its message); headless keyboard runs (20 + 8 checks, no script errors) |
| Deaths, paralysis, cure, ghost representation | Met as provisional treatments | Stills D03–D09, G1–G3, S1–S2 |
| Formation changes at the correct point | Shown with Classic timing; Go differs (H5–H6) and needs C1–C3 | Difficult cases §4; Go probe |
| Four groups, large and sleeping creatures | Met; crowding in a dead end open (Q-D5) | Stills D01–D02, D10–D12, L1–L4 |
| Pacing with death animations | Met: no lengthening at normal speed, +0.1 s once at fast | `fixture-check.js` |
| Displayed vs authoritative vs known | Shown side by side in the reviewer panel; principle approved (X34) | Truth table in the prototype |
| Engine or app untouched | Met | No changes outside `docs/` on the design branches |

**Verdict.** The M2 combat **presentation** design is accepted as a baseline
for implementation planning, subject to two kinds of open item:
- the provisional assumptions in §4;
- the open questions in §5.

GUI combat **implementation** is not ready to start. It is blocked by the
engine and controller work in §6. The order-identity correction (C1) and the
event stream (E1) are on its critical path.

## 8. Recommended next bounded technical prototype

*Update:* P1a and P1b were carried out on `spike/p1-combat-events`, and P1c
(corrections C1–C3) on `spike/p1c-order-binding`; both integrated into
`main` locally (2026-11-06); see
`P1-COMBAT-ENGINE-SPIKE.md`.

**P1: Combat order identity and event stream: a headless Go spike.**

- **Purpose:** retire the two largest implementation risks before any
  renderer or GUI work:
  - slot-bound orders (H6, H7);
  - string-only combat output (E1).
- **Where:** a throwaway branch. Engine and controller changes stay there
  until the compatibility decisions for C1–C3 are approved; nothing merges
  without that.
- **Scope:**
  1. Characterization tests that pin today's behaviour (§6 item 1).
  2. A minimal structured event for each message block in `ExecuteRound`,
     produced by the same code path, behind no new interface beyond a slice
     on `CombatState`.
     - **Coverage:** fight, spell, parry, run, dispel, monster melee,
       death, paralysis and cure.
     - **Golden test:** a seeded encounter where each event's lines equal the
       messages of its block.
  3. Orders keyed by character with the sort at the round boundary (C1–C3).
     A seeded test reproduces the probe case and shows the order staying
     with its member.
- **Exit criteria:**
  - **Existing goldens:** the existing goldens pass, or every difference is
    explained by C1–C3.
  - **Events and messages:** events and messages agree in every test.
  - **Hidden information:** no hidden field is needed to build the
    presentation's events.
  - **Size:** the spike is small enough to throw away.
- **Out of scope:** the renderer, Wails/Svelte, the controller View (M5),
  the save format, any other rules change.
- **Why this first:** the presentation is validated. The open risk is
  whether Go can supply what the presentation needs without reparsing
  strings or changing outcomes.
