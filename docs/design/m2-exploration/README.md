# M2 presentation design exploration

Status: **design exploration for review** (2026-11-06). Nothing in this
directory is approved, and nothing here changes the application, the engine,
the save format or the approved baseline documents.

This is Part A of roadmap M2 (the presentation design checkpoint, charter
§6.3). It proposes an overall experience and lays out alternatives for review.
Choosing a direction, a renderer or a layout is left to the user.

## Contents

| File | What it covers |
|---|---|
| [EXPERIENCE-CONCEPT.md](EXPERIENCE-CONCEPT.md) | The three experiences (management, exploration, combat), contextual inspection, the navigation map, and the journey from Castle to Maze and back |
| [CAMERA-AND-INTERACTION.md](CAMERA-AND-INTERACTION.md) | The layered model (state, interaction context, camera, presentation, input), visibility rules, camera behaviour, grid-movement transitions, map knowledge rules, architecture and persistence implications |
| [RENDERING-OPTIONS.md](RENDERING-OPTIONS.md) | Five rendering approaches compared, their costs and risks, and a proposed prototype sequence |
| [MOCKUP-REVIEW.md](MOCKUP-REVIEW.md) | What each mockup frame tests, how the treatments compare, and what the mockups cannot show |
| [mockups/storyboard.html](mockups/storyboard.html) | Interactive storyboard: open it in a browser. It has six frames of one location, three treatments, light on/off, four map modes, two map kinds, a camera sandbox, workspace wireframes and the transition strip |
| [mockups/scene.js](mockups/scene.js) | The throwaway projection renderer the storyboard uses (no dependencies; not application code) |
| [stills/](stills/) | PNG exports of the storyboard, one per treatment, for reviewing without a browser |
| [COMBAT-EXPERIENCE-REVIEW.md](COMBAT-EXPERIENCE-REVIEW.md) | Follow-on: three combat interaction approaches compared, scenery vs gameplay knowledge, ordering, playback, the chest, D6 questions, decisions X17–X24 and what to test next |
| [mockups/combat.html](mockups/combat.html) | Follow-on: interactive combat prototype (a full round in the dungeon, approaches A/B/C, four scenery policies, manual or automatic playback) |
| [mockups/combat-scene.js](mockups/combat-scene.js) | The prototype's throwaway renderer (extends the ideas of `scene.js`, which is unchanged) |
| [stills/combat/](stills/combat/) | Stills of the combat prototype: every moment in approach C, the key moments in A and B, and one shot per scenery policy |

## Status labels used in these documents

| Label | Meaning |
|---|---|
| **Approved** | Already approved in the design baseline (charter, architecture proposal, roadmap). Restated here; not changed. |
| **Fact** | Verified in the current code, with a file reference. |
| **Hypothesis** | A working proposal from this exploration. Not approved. |
| **Alternative** | One of several options under consideration. Not preferred unless stated. |
| **Decision** | Requires user approval. Listed in the register below. |
| **Risk / question** | To be resolved by prototyping or investigation. |

No hypothesis or alternative in this directory is a requirement.

## Approved principles carried forward (not changed)

- **One engine, one state.** One authoritative Go engine and game state; no rule is reimplemented in Svelte or in a renderer (charter §2.4).
- **Classic rules are historically verified.** The engine is the provisional baseline, and open discrepancies stay tracked in `WIZARDRY-COMPATIBILITY.md` (D1). This exploration does not resolve any of them.
- **Presentation is independent of rules.** Presentation never changes mechanics, difficulty, or what the user may know (charter §5.2).
- **Information follows the ruleset.** No presentation feature reveals information unavailable under the selected rules and enhancements (charter §2.6).
- **The automap is an enhancement.** It is off under Classic (D9).
- **The controller owns actions and transitions.** It owns every legal gameplay action and location transition (architecture §2.3).
- **The application framework stays.** Wails v2 and Svelte remain the application framework (D4).
- **Paced steps.** Paced steps use a queue and `Advance()`; rules never sleep (D6, commit semantics still open).
- **Profiles in combat.** Profiles are viewable in combat read-only, if shown at all (D8).

## Decision register for this checkpoint

Each item needs the user's approval before implementation depends on it.
"Proposed" is this exploration's suggestion, not a decision.

| ID | Question | Options | Proposed |
|---|---|---|---|
| X1 | Overall experience direction | One continuous dungeon environment for exploration, combat and inspection (this proposal) vs a separate battle screen | Continuous environment |
| X2 | Interface treatment(s) to develop | A minimal immersive; B information-rich; C Classic presentation; a user-switchable combination | B as the Modern default, A as a mode, and C as the Classic presentation |
| X3 | Visible-set rule | Presentation draws only the Classic DRAWMAZE visible set, from the team's square and facing, whatever the camera | Adopt as a binding rule for every camera and renderer |
| X4 | Attached camera bounds | (a) inside the team's square only; (b) may trail into the square behind, with unseen geometry rendered as darkness; (c) free orbit within a radius | (a) by default; prototype (b) |
| X5 | Elevated and overhead views | None under Classic; elevated views limited to the visible set; overhead as part of an enhancement | Elevated view limited to the visible set; overhead only with the automap enhancement |
| X6 | Movement presentation | Instant (Classic); short animated step/turn; user preference | User preference, with animation off in Classic presentation by default |
| X7 | Combat framing | Same corridor with a reframed camera (this proposal) vs a staged "arena" in the same art style | Same corridor |
| X8 | Combat replay control | Auto-advance with a speed preference; manual advance; both | Both (Return to advance, a speed preference, Shift-Return to show the rest) |
| X9 | Close-up inspection scope | Chests only (Classic has no door traps or locks); also doors and special squares as purely visual close-ups | Chests; other close-ups only as visual framing with no new information or actions |
| X10 | Management model | W-A workspaces, browser and inspector; W-B castle as a canvas; a hybrid | Prototype both at low fidelity before choosing |
| X11 | Navigation map kinds | Manual map only under Classic; automatic map as an enhancement; both available | Manual map always available; automatic map only as the D9 enhancement |
| X12 | Navigation map placement | Hidden; docked; floating; full overlay; several, chosen by the user | All four as user choices; default undecided |
| X13 | Auto-map position source | True engine position (reveals spinners and teleports) vs the team's perceived movement (dead reckoning) | Perceived movement, with true position as a separate stronger option |
| X14 | Map and camera persistence | Where manual maps, exploration records and camera preferences are stored | Manual maps and exploration records in a per-campaign sidecar; camera in user preferences |
| X15 | Renderer for the prototype | WebGL2 via three.js; WebGPU; Canvas 2D/SVG; native Metal/RealityKit; an external engine | Prototype WebGL2 in the existing webview; no final choice now |
| X16 | Art direction | Placeholder geometric; stylised low-poly; painted; Classic line art | Out of scope here; needs its own checkpoint (D12 bears on reusing original monster pictures) |
| X17–X24 | Combat interaction approach, scenery policy, order changes, playback pacing, value-update timing, playback camera, repeat orders, unavailable actions | See `COMBAT-EXPERIENCE-REVIEW.md` §11 | See that section |

## Correction (combat review)

The first version of this exploration said that, in Classic, HP and monster
counts change on screen as each combat message appears. That is wrong:
- **What the engine does:** it freezes the displayed values for the whole
  round and updates them after the last message (`engine/combat.go:634`).
- **Where corrected:** correction notes were added beside the affected text
  in `EXPERIENCE-CONCEPT.md`, `MOCKUP-REVIEW.md` and
  `CAMERA-AND-INTERACTION.md`, and to the storyboard annotation.
- **Details:** see `COMBAT-EXPERIENCE-REVIEW.md` §2.

## Principal risks and questions for prototyping

These are detailed in the documents above.

1. **Rendering consumes randomness.**
   - **Fact:** the terminal renderer draws hidden doors with `rand.Intn(6)` on the shared game random source (`render/dungeon.go:178`).
   - **Risk:** a GUI renderer redraws far more often, so it must never call into game randomness.
   - **Open:** where the "is this hidden door visible now" fact is decided is an open compatibility question.
2. **Maze logic lives in the terminal.**
   - **Fact:** maze movement, light decrement, encounter rolls and square effects live in the terminal program (`cmd/istary/main.go:2693–3200`), not in `engine` or `app`.
   - **Consequence:** any GUI maze depends on the M5 controller extraction.
3. **Light ticks on keystrokes.**
   - **Fact:** light decrements on every maze keystroke that reaches the end of the handler, including the terminal's map key (`cmd/istary/main.go:3116`).
   - **Risk:** opening the GUI map must not cost light. Whether the terminal behaviour is Classic is a compatibility question to log, not to resolve here.
4. **Silent state changes.**
   - **Fact:** spinners and teleport squares change facing or position silently (`cmd/istary/main.go:3407–3435`).
   - **Requirement:** camera and map transitions must not reveal them, for example by animating a rotation.
5. **Levels wrap.**
   - **Fact:** levels are 20×20 and wrap (`engine/game.go:204`).
   - **Requirement:** maps, and any camera that can see more than one square, must handle the wrap.
6. **Performance and power** of 3D in WKWebView on battery, and whether WebGPU is available inside a Wails v2 webview.
7. **Art cost.** A 3D or stylised dungeon needs original art. Reusing original monster pictures needs the D12 legal review before any distribution.

## How to review

1. **Open the storyboard.** Open `mockups/storyboard.html` in Safari or Chrome; it needs no server or network.
2. **Switch the treatments.** Use the toolbar to switch treatment A, B or C, light, map mode and map kind. Turn the design annotations off to see a frame as a user would.
3. **Try the sandbox.** Use the camera sandbox to judge attached-camera framing and decision X4.
4. **Read the documents.** Read `MOCKUP-REVIEW.md` alongside the frames, then the decision register above.
5. **Try the combat prototype.** Open `mockups/combat.html` and play a round in approach C, then A and B. Read `COMBAT-EXPERIENCE-REVIEW.md` alongside it.
