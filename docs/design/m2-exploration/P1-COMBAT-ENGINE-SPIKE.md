# P1a/P1b: combat engine characterization and event spike

Status: **experiment for review** (2026-11-06). This is a local branch,
`spike/p1-combat-events`, not merged. It follows
`COMBAT-DESIGN-BASELINE.md` §8, P1a and P1b only. It makes no rule change,
no save-format change and no GUI change, and it does not implement
corrections C1–C3.

## 1. What changed

| File | Change |
|---|---|
| `engine/combat_events.go` | New. Typed events and the recorder (about 300 lines, mostly comments and types) |
| `engine/combat.go` | 21 lines added and 2 changed (`git diff --numstat`: 23 in, 2 out): hooks in `addMessage` and `endAction`; where a party member or monster begins an action; monster branches (spell, breath, call for help); each melee attack's target; round end; the log reset per round. One unexported field on `CombatState` |
| `engine/combat_characterization_test.go` | New. P1a engine tests |
| `engine/combat_events_test.go` | New. No state change in silent blocks |
| `cmd/istary/combat_characterization_test.go` | New. P1a terminal tests |
| `cmd/istary/combat_events_test.go`, `testdata/combat_events.golden` | New. Events against messages, and the golden |

No existing test or golden changed. `combat_trace.golden` (three seeded
fights, every message and roll), `maze_walk`, `maze_wander` and the screen
goldens pass unchanged. Messages, their order and random-number use are
therefore unchanged.

## 2. P1a: current engine behaviour (observed, pinned by tests)

These are observations of this engine, not historical claims. The
historical column cites the Player's Guide where it speaks.

| # | Observed (test) | Historical |
|---|---|---|
| 1 | **Which statuses move back:** the status sort moves every worse status back: asleep, afraid, paralyzed, stoned, dead, ashed, lost (`TestCharResortStatusOrder`) | Guide p.48 (dead), p.55 ("killed or incapacitated"): consistent |
| 2 | **The sort is not stable:** two paralyzed members can swap order. A cured member never moves forward past an OK member (same test) | Not stated |
| 3 | **Timing:** a status change between rounds moves no one until the next `ExecuteRound`. That call re-sorts first, after that round's orders were entered. The display snapshot is taken after the sort (`TestCharResortTiming`) | Guide p.48: the dead move "at the end of the round". **Differs** |
| 4 | **Orders follow slots:** orders stay with their slot. The order given for slot 3 is carried out by whoever moves into slot 3 (`TestCharOrdersStayWithSlots`) | Not possible under the guide's timing. **Differs** |
| 5 | **Stale orders:** nothing clears orders between rounds. A slot whose member was skipped keeps last round's order, and the member who moves into it carries it out (`TestCharStaleOrderCarriedOut`, `TestCharRoundCompletion`) | Not stated |
| 6 | **Ally targets follow slots:** they are slot indices. DIALKO aimed at a paralyzed member lands on whoever is in that slot after the sort: "EDDA IS NOT HELPED!" (`TestCharAllyTargetStaysWithSlot`) | Not stated |
| 7 | **Who is offered orders:** OK, ASLEEP and AFRAID members are offered orders; PARALYZE, STONED and the dead are skipped (`TestCharWhoIsOfferedAnOrder`) | Not verified (Q-D8) |
| 8 | **Orders for incapacitated members:** a paralyzed or dead member with an order is skipped silently. An asleep or afraid member's Parry is carried out. An asleep caster's spell is "DISRUPTED" (`TestCharOrdersForIncapacitated`) | Not verified |
| 9 | **Message blocks:** each action ends with a separator. A Use with nothing usable writes an empty block. The surprise line has no separator of its own and opens the first monster's block (`TestCharMessageBlocks`) | Not stated |
| 10 | **Round completion:** one timer tick per block. After the last block the snapshot is cleared, groups are compacted, the round advances and order entry restarts, with the old orders still in place (`TestCharRoundCompletion`) | Guide: messages paced by T)IME |

**Further defects and ambiguities found** (recorded, not fixed):

- **D-1: silent monster misses.** A monster's melee writes one block per
  attack, and a miss writes no line, so the block is empty. The terminal
  shows each as a blank pause. In 448 seeded rounds there were 1,236 silent
  blocks. Whether Classic showed misses is not established.
- **D-2: other empty blocks.** These also produce blank pauses: a Use with
  nothing usable (tested), and, from the code, a party spell that fizzles
  (`executeSpell` ends its block and `executePartySlot` ends another).
- **D-3: the surprise line.** It shares the first monster's block.
  `** THE PARTY HAS PERISHED **` and the experience message are written
  without a separator.
- **D-4: one monster without a target stops the slot.** In
  `executeMonsterSlot`, when a monster has no target, `return` ends the slot
  for every remaining monster, not just that one (read, not tested; only
  reachable when no party member can be targeted).
- **D-5: round logic in the terminal.** The round transition (compaction,
  round number, order-entry restart, clearing the snapshot) lives in the
  terminal (`advanceCombatMessages`, `cmd/istary/main.go:3596`), not the
  engine. This matters for P1c (§5).

## 3. P1b: structured events

**Interface** (`engine/combat_events.go`). `CombatState.Events()` returns one
`CombatEvent` per message block for the current `Messages`:

| Field | Meaning |
|---|---|
| `Kind` | `party-fight`, `party-spell`, `party-use`, `party-parry`, `party-run`, `party-dispel`, `monster-melee`, `monster-spell`, `monster-breath`, `monster-call-for-help`, or `narration` (no actor: surprise, the party perishing, rewards) |
| `Actor` | `CombatRef`: side; a party slot, or a monster's group and member index; the name the messages use (an unidentified monster keeps its unidentified name) |
| `Target` | As ordered: a monster group, or an ally slot named after the member now in it. For monster melee, the member attacked |
| `Lines` | The block's messages, exactly as in `Messages` |
| `Start`, `End` | `Lines == Messages[Start:End]` |
| `Part` | Numbers the blocks of one action (a monster's second attack is part 2) |
| `Silent` | The block has no lines. Actor, target and effects are withheld |
| `Effects` | `CombatEffect` per participant changed during the block. For a party member: HP before and after, and status before and after. For a monster: status only |
| `Merged` | Further actions noted in an open block. Expected 0, and 0 in every test |

**How it is produced.**
- **Same calls as the messages:** events are recorded by the same calls that
  write messages. A block opens at its first message or where an action
  begins, and closes at `endAction`.
- **Actor and target:** noted at the point where an action starts, not read
  from text.
- **Effects:** state at the block's opening compared with state at its
  close. They capture what the action changed.
- **What it does not do:** no message is parsed, no random number is drawn,
  and messages are written as before.

**Information boundaries.**
- **Never exposed:** monster IDs, monster HP, initiative, the remaining
  action count, and the victim of an action whose block says nothing.
  - **Silent blocks:** they are redacted.
  - **Monster spell targets:** they are never set, because the pre-assigned
    victim is not shown when a spell fizzles or is silenced.
- **Authoritative, not displayed:** events carry authoritative changes. When
  to show them (Classic freezes the display until round end) is for the
  controller and presentation.
- **Release is the controller's job:** `Events()` returns the whole round at
  once, as `Messages` does. Releasing events one at a time (E11) is
  therefore the controller's job (M5), not the engine's.

**Evidence that messages and events agree.**

- **`TestCombatEventsMatchMessages`:** 60 seeded encounters played through
  the terminal; 409 rounds and 4,314 events checked at every round. It
  checks that:
  - the events rebuild `Messages` exactly, block for block;
  - each party action is attributed to the member in that slot and named in
    its block;
  - every party change is about a member its own block names;
  - only known narration lines are unattributed;
  - no block merges two actions.
- **`TestCombatEventsMatchMessagesWithSpells`:** 80 seeds driven at the
  engine with spells (KATINO, HALITO, DIOS, BADIOS), dispel and use.
  - **Events seen:** 2,129 monster-melee, 1,003 fight, 562 spell, 255 parry,
    125 dispel, 49 use, and 13 narration (the party perishing).
  - **Result:** same checks, no failures.
- **`combat_events.golden`:** the events of seeds 21–23 in readable form,
  next to `combat_trace.golden` for the same fights.
- **`TestCombatEventsSilentBlocksChangeNothing`:** 448 rounds and 1,236
  silent blocks, with no state change in any. So the redaction hides nothing
  a message would show.
- **Unchanged goldens:** the existing goldens pass unchanged, so messages and
  random-number use are identical.

**Not exercised.** The B1F seeds include no monster spell, breath or call for
help, and Run was not reached. Those kinds are noted at the right branch but
untested.

## 4. Did the spike need a larger refactor?

No.
- **The message path:** messages already pass through two functions, so
  recording was local.
- **What needed care:** two engine structures needed care rather than a
  refactor: multi-block actions and silent blocks (D-1).
- **Where a refactor will be needed:** the round transition living in the
  terminal (D-5). That is P1c and M5 scope, not P1b.

## 5. Risks and open questions

- **Q-P1: misses in Classic.** Did Classic show a monster's miss? If so, the
  silent blocks are a fidelity gap; if not, a GUI should show a pause, not a
  gesture (D-1).
- **Q-P2: monster member index.** The member index is a position for this
  round only; groups are compacted at round end. A stable per-monster
  identity would need engine work (E1 open point).
- **Q-P3: effects per block.** Effects are per block, not per hit. A
  multi-hit fight block reports the net change. Enough for the presentation
  model, which shows one block at a time.
- **Q-P4: where events live.** Events live on `CombatState` and are dropped
  when `Messages` is cleared (including by the terminal). If the controller
  needs them across a reload, D6 commit semantics apply.
- **Q-D8 (asleep and afraid members offered orders)** remains open, now
  pinned by a test.

## 6. Is P1c ready for approval?

**Yes, with its scope stated.**
- **Pinned behaviour:** P1a pins every behaviour C1–C3 would change (rows
  3–6), so the change will be deliberate and visible.
- **Evidence:** the historical evidence (guide p.48 and p.55) supports the
  timing.

Points to settle when approving:

1. **Where the re-sort moves (C2).**
   - **Why not at the end of `ExecuteRound`:** the display snapshot and the
     messages are slot-indexed for the round being shown.
   - **The natural place:** the round transition, which today is in the
     terminal (D-5). P1c should move that transition into an engine method
     that the terminal calls. This is a small change to `cmd/istary`, with
     the terminal behaving the same apart from C1–C3.
2. **What identifies a character (C1).** Within a combat, the `*Character`
   is enough. Stable identifiers (architecture §2.5) can follow in M5.
3. **Goldens:** `combat_trace.golden` changes only if its seeds kill a member
   mid-fight. Any change must be explained by C1–C3.
4. **What is out of scope:**
   - Q-D8 (asleep and afraid members offered orders);
   - the silent misses (Q-P1);
   - the controller (M5).
