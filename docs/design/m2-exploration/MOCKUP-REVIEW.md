# Mockup review

Status: **exploration, not approved.** The mockups illustrate alternatives
for discussion. They are not production components, not a renderer choice,
and not artwork.

## 1. What was produced and why this method

- **Interactive storyboard:** `mockups/storyboard.html`. Open it directly in
  a browser; it uses no network and no build.
- **Renderer:** `mockups/scene.js`, a dependency-free projection renderer,
  about 250 lines. It draws one fixed dungeon location from any camera as
  SVG.
- **Stills:** `stills/storyboard-*.png`, full-page exports of the three
  treatments for review without a browser:
  - **A** with a floating manual map;
  - **B** with a docked automatic map;
  - **C** with no light and an overlay manual map.

**Why this method.** Hand-drawn wireframes could not keep the same location
consistent across six camera positions. A tiny projection renderer does that
cheaply, and it can also show the visibility rule working (the niche, light
on/off, the camera sandbox).

**What it is not.** It does not introduce a 3D engine into the application:
it lives under `docs/`, and nothing in `gui/` references it.

**Limitations:**

- **Placeholder art:**
  - **Figures** are capsules and **monsters** are blobs. They are deliberate
    placeholders, not proposed art and not original game art.
  - **Shading** is flat, with simple distance shading, and a few
    painter's-algorithm overlaps show.
  - These say nothing about the eventual visual quality (X16).
- **Content:** names, numbers and combat messages are illustrative. The
  message wording imitates Classic style but is not taken from a verified
  run.
- **No image generation.** No image-generation tools were used. The stills
  are screenshots of the HTML mockup.

## 2. The location (reviewer aid)

The location is a short corridor running north:

- **square 0:** the team, facing north;
- **square 1:** a door on the left and an opening on the right into a
  dead-end niche;
- **square 3:** the corridor ends in a wall.

The plan at the top of the storyboard marks the edges that the Classic view
can show from square 0. The far wall of the niche is never in that set; it
demonstrates the visible-set rule (X3). The plan is not an in-game view.

## 3. Frames

All frames show the same location, with the same team and the same light.
"Treatment" changes only the overlays and the drawing style.

| # | Activity | What it tests | Key observations |
|---|---|---|---|
| 1 | First person, exploring | Equivalence with the Classic view; HUD weight; the map in exploration | The 3D first-person view and the Classic line view (C) draw the same edges. With no light, two squares only. No compass or coordinates under Classic. |
| 2 | Attached camera on player 1 | Simulator-style camera within the rules | The same walls as frame 1 (the camera reveals nothing new). Inside the team's square the figures dominate the frame; this is the X4 trade-off. In-square formation is presentation only. |
| 3a | Encounter: orders | Continuity into combat; order entry | Same walls, door and light; no battle screen. Orders are listed from Go (back line has no Fight; no Spell for a non-caster). Unidentified group names. |
| 3b | Round resolution | Sequential replay of a resolved round | One message at a time; Ysolde's HP changes as the message shows it (snapshot rule). Replay controls are presentation. |
| 4 | Chest close-up | Contextual inspection and the four knowledge layers | The chest looks the same whatever it holds. Reports are shown as reports. The annotations separate engine knowledge (never sent), team knowledge, what is visible, and what may be attempted. |
| 5 | Back to exploration | Natural return | The camera restores the user's exploration camera; nothing moved, so the visible set is unchanged. |

**Correction (combat review).** Frame 3b says Ysolde's HP changes as the
message shows it. Under Classic, displayed values stay frozen for the whole
round and update after the last message (`engine/combat.go:634`). See
`COMBAT-EXPERIENCE-REVIEW.md` §2 and decision X21.

## 4. Interface treatments compared

| | A · Minimal immersive | B · Information-rich | C · Classic presentation |
|---|---|---|---|
| Idea | Scene first; compact team strip; one message line; contextual menus near the object | Persistent side column: full team table, message log, key hints, context panel | Line-drawn maze and figures, boxed monospace text, original letter commands |
| Strengths | Atmosphere; least occlusion; good for full screen | Everything visible at once; best for learning and for long sessions; easiest to make accessible | Familiar to Wizardry players; shows that rules and presentation are independent |
| Weaknesses | Less information visible; relies on keys and hovers | Squeezes the scene, especially with a docked map (frame 1, B + docked) | Least discoverable; the 3D cameras are less meaningful in line art |
| Combat | Order menu beside the actor; message line over the scene | Orders table, opponent list, scrolling log | Original boxes and wording |
| Chest | Action menu beside the chest; reports in the message line | Knowledge card: reports, who may act | Original chest menu |

**Observations:**

- **C is not tied to Classic rules.** It is a presentation that pairs
  naturally with them, but any treatment can run either ruleset (Approved,
  charter §5.2).
- **A and B can share one implementation** with a density setting. The
  information in each is the same View, so either is a presentation
  preference rather than a fork.
- **Placement in B is not fixed.** Its right-hand column is one placement,
  shown only to make the treatment concrete.

## 5. Navigation map alternatives (frames 1, 2 and 5)

Toolbar controls:

- **mode:** hidden, docked, floating, or overlay;
- **kind:** manual or automatic;
- **orientation:** north-up or heading-up.

| | What to look for |
|---|---|
| Manual (user-drawn) | Hand-drawn strokes and notes ("door: try K", "opening?"), a user marker labelled "me?", sheets instead of floors, drawing tools. It is deliberately imperfect: the user's map can be wrong, and the game never corrects it. |
| Automatic (enhancement) | Squares walked (stronger tint) and seen (weaker), edges seen from each step, stairs as recorded. The niche shows only its front wall. Level selector lists only visited levels. No coordinates unless DUMAPIC was cast. |
| Hidden | A key hint only. |
| Docked | A full-height panel; compare how much it squeezes treatment B. |
| Floating | An in-window panel with a title bar, like the existing floating character sheet idea. |
| Overlay | Dims the scene; the most room for annotation (DCS F10-like). |

**Limits of the map mockup:**

- **Heading-up:** the team faces north in every frame, so heading-up and
  north-up look the same; only the label changes.
- **Map history:** the automatic map's earlier squares (the stairs and the
  corridor behind) are illustrative history.

## 6. Camera sandbox

- **Controls:**
  - attach the camera to any player, or use first person;
  - change follow distance, height and orbit;
  - toggle "Keep the camera inside the team's square".
- **With the constraint on:** the camera is pushed back inside the square,
  like a camera colliding with walls.
- **With the constraint off:** the camera can move behind the team into
  space that is not in the visible set, and that space renders as nothing.
  This makes decision X4 concrete.

## 7. Management workspace and transition

- **Two wireframes:**
  - **W-A:** workspaces, browser, canvas and inspector;
  - **W-B:** the castle as a canvas with a team token.
  - **Fusion vocabulary:** both apply it (selection-driven commands,
    contextual inspector, marking menu, command search).
  - **Undecided:** neither fixes positions.
- **The transition strip:**
  - **Steps:** Prepare → Enter → Arrive → Explore → Return.
  - **Responsibilities:** the controller performs each transition, and the
    presentation only animates it.

## 8. Questions for the reviewer

1. **Continuity:** does one continuous environment (frames 1 → 5) feel right,
   or is a separate battle presentation preferred? (X1, X7)
2. **Treatments:** which treatment, or which combination, should be developed
   first? (X2)
3. **Visible set:** is the rule acceptable even though a 3D view then shows
   deliberate darkness through openings (the niche)? (X3)
4. **Camera bounds:** inside the team's square only, or trailing behind with
   darkness? (X4)
5. **Movement:** should movement be animated by default under Modern
   presentation, and instant under Classic? (X6)
6. **Close-ups:** is the chest close-up enough? (X9)
7. **Management:** W-A, W-B, or both prototyped further? (X10)
8. **Maps:** manual map always, automatic map as an enhancement; which
   placements to keep, and should the automatic map use perceived or true
   position? (X11–X13)
9. **Renderer:** should prototype P1 (WebGL2 fixture scene outside the app)
   be authorised as the next step? (X15)
