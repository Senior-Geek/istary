# Rendering options

Status: **exploration, not approved.** No renderer is selected (decision
X15). This document compares approaches, so that a prototype can be chosen.

## 1. What any renderer must do

These requirements come from the approved principles and
`CAMERA-AND-INTERACTION.md`.

1. **Draw from the View only.**
   - **What it draws:** the scene is built from the controller's View: the
     visible set, the team, the opponents and the chest.
   - **No rules:** the renderer has no access to level data or engine state,
     and holds no rules.
2. **Use no game randomness.** Visual variety, such as torch flicker, uses
   its own source, and none of it affects the engine.
3. **Keep the camera separate** from game state (attached, first person,
   context framing), and keep its transitions free of side effects.
4. **Pace sequences without changing them.** Pacing comes from the queue and
   `Advance()` (D6). Animation speed is a preference.
5. **Run in the current app** without a wholesale framework migration.
   - **Approved:** keep Wails v2 and Svelte (D4).
   - **When to revisit:** only on strong evidence.
6. **Be power-aware on a laptop.**
   - **Render on demand:** render while something moves; when idle, show a
     still frame.
7. **Support every treatment.** It must support immersive, information-rich
   and Classic line treatments as presentation switches.

## 2. The current platform (Fact)

- **Framework:**
  - **Wails** v2.17.0 on macOS hosts the Svelte front end in a WKWebView, in
    one native window.
  - **Frontend:** Svelte 5.57.2 and Vite 8.3.5, with no 3D dependency
    today (`gui/frontend/package.json`).
- **Existing renderers:** the terminal app draws the maze with an exact port
  of the Apple II DRAWMAZE routine into an 82×79 bitmap
  (`render/dungeon.go`). Monster pictures exist as scenario data (Apple II
  hi-res bytes and half-block art; `data/types.go` `MonsterPic`).
- **Wails v2 limits:**
  - **No native views:** v2 has no supported way to add native views
    beside the web view.
  - **One window:** multiple native windows need Wails v3 or custom code
    (architecture §4).

## 3. Options

### R1. WebGL2 in the existing web view (three.js or similar)

- **What:** a 3D scene in a `<canvas>` inside the Svelte app, drawn with
  WebGL2 through a library such as three.js. Svelte bindings such as Threlte
  exist, but the scene could also be plain three.js driven from a Svelte
  component.
- **Fit:**
  - same window, same Svelte state;
  - HUD and panels are ordinary HTML over the canvas;
  - no Go or Wails changes beyond new View content.
- **Capability:** ample for a grid dungeon with low to moderate polygon
  counts, dynamic torch lighting, simple shadows, fog, figure models (glTF),
  and particle effects for spells.
- **Costs:**
  - **Bundle:** about 150–200 KB gzipped for three.js core, more with
    loaders and post-processing.
  - **Learning:** a 3D scene-graph learning curve.
  - **Art:** asset pipeline for models and textures.
- **Risks:**
  - WKWebView WebGL performance and power on battery (to be measured);
  - GPU context loss on sleep or display change, which must be handled;
  - text in 3D is awkward, so keep text in HTML.
- **Classic presentation:** line rendering can reproduce the wireframe look
  (edges only on black) from the same visible set.

### R2. WebGPU in the existing web view

- **What:** as R1, but on WebGPU. For example, three.js's WebGPU renderer
  falls back to WebGL2 when WebGPU is not available.
- **Fit:** same as R1.
- **Gain:** a more modern API, compute shaders and better CPU overhead.
  Nothing here needs those yet.
- **Risks:**
  - **Availability:** whether WebGPU is enabled in a Wails v2 WKWebView on
    the macOS versions Istary targets needs a prototype check; do not
    assume it.
  - **Maturity:** the WebGPU renderer paths are newer than WebGL2's.
- **Assessment:** treat as a variant of R1. Write to a library that can use
  either, and choose by measurement.

### R3. 2D Canvas or SVG "pseudo-3D" (extend the Classic projection)

- **What:** draw the corridor with 2D projection (like DRAWMAZE, or the
  storyboard's `scene.js`), with filled and textured wall polygons and
  billboard sprites for figures and monsters.
- **Fit:** trivial in the current stack, with no dependency. It is
  deterministic, and very cheap on power when idle.
- **Capability:**
  - **Strong:** first person, which can be a faithful Classic line
    presentation.
  - **Weak:** attached third-person and elevated cameras. There is no depth
    buffer, so the painter's-algorithm sorting artefacts show (the
    storyboard has a few). Lighting is limited, and so are smooth camera
    moves around figures.
- **Costs:** low for Classic presentation; it rises steeply as the views
  become more ambitious.
- **Assessment:**
  - **Classic presentation:** an excellent candidate.
  - **The simulator-style camera:** insufficient on its own.

### R4. Native Metal, RealityKit or SceneKit view in the same application

- **What:** a native 3D view, written in Swift or Objective-C, composed with
  the web view in the same window, with Go passing it the View.
- **Fit:** poor with Wails v2.
  - **No composition API:** there is no API for composing native views.
    Doing it means custom cgo and Objective-C code that reaches into the
    window's view hierarchy, with layering, input routing and
    focus problems between the native view and the web view.
  - **Wails v3:** may ease this; that would be a framework migration
    (D4).
- **Capability:** the best performance and power profile, with the full
  Apple toolchain.
  - **Framework choice:** Apple's newer native 3D work centres on
    RealityKit. SceneKit should be checked for its current support status
    before any reliance.
- **Costs:**
  - **Language:** a second UI language (Swift) and a bridge layer.
  - **Tooling:** harder builds and testing.
  - **Loss:** HTML overlays can no longer simply sit on the scene.
- **Assessment:** only justified if R1/R2 measurements fail on performance
  or power. Not recommended now.

### R5. External game engine (e.g. Godot), with Go rules as a library or process

- **What:** move the presentation to a game engine, keeping the Go engine
  behind a C ABI (cgo c-shared) or a local process protocol.
- **Fit:** a wholesale front-end migration.
  - **Lost:** the existing Svelte GUI-001 to GUI-003 work.
  - **Rebuilt:** macOS conventions (menus, documents, accessibility) must be
    rebuilt inside a game UI.
- **Capability:** the best tooling for 3D content, animation and effects.
- **Costs and risks:**
  - **Migration:** a rewrite of the management workspace.
  - **Platform:** weaker native macOS feel.
  - **Two runtimes:** two of them, plus a bridge.
  - **Licensing and distribution** review (D12 scope).
- **Assessment:** no evidence justifies it. **Not recommended.**

## 4. Comparison

| | R1 WebGL2 | R2 WebGPU | R3 2D canvas/SVG | R4 Native view | R5 Game engine |
|---|---|---|---|---|---|
| Fits Wails v2 + Svelte unchanged | Yes | Yes (if enabled) | Yes | No (custom bridge) | No (migration) |
| First person | Good | Good | Good | Excellent | Excellent |
| Attached / elevated cameras | Good | Good | Weak | Excellent | Excellent |
| Classic line presentation | Good | Good | Excellent | Good | Good |
| HTML HUD over scene | Native | Native | Native | Hard | Rebuild |
| Power when idle (on-demand render) | Good | Good | Excellent | Excellent | Fair |
| New dependencies | One library | One library | None | Swift toolchain + bridge | Whole engine |
| Engineering cost | Medium | Medium | Low → high with ambition | High | Very high |
| Main risk | Performance or power unknown in WKWebView | Availability | Ceiling on ambition | Integration with v2 | Migration |

## 5. Proposed prototyping sequence (for approval; nothing started)

Each step is a separate, approved task. None changes the application until
step P4.

- **P1. Fixture scene, outside the app.**
  - **Build:** a standalone page in a scratch location (not `gui/`) that
    renders the storyboard's location from a hand-written View fixture with
    R1 (WebGL2) and the Classic line treatment.
  - **Covers:** first person, attached camera (X4 a/b), combat framing, and
    the chest close-up.
  - **Purpose:** decide X4–X7 on a real 3D scene.
- **P2. Measurements in a Wails v2 web view.**
  - **Build:** the same page inside a throwaway Wails app.
  - **Measure:**
    - frame time and idle power;
    - context loss on sleep and wake;
    - full-screen behaviour;
    - whether WebGPU is available (R2).
  - **Report:** numbers, not impressions.
- **P3. Art-direction study** (X16):
  - **Content:** two or three looks for walls, figures and monsters at low
    cost.
  - **Constraint:** monster appearance must not reveal identity.
- **P4. Integration**, only after the M5 controller extraction:
  - **Build:** the controller supplies the visible set and contexts; the
    GUI renders them.
  - **Tests:** prove identical engine states across cameras and treatments.

**Exit criteria for choosing a renderer:**

- smooth step and turn animation on the target Macs;
- render-on-demand idle cost near zero;
- no information beyond the visible set in any camera mode;
- the HUD is legible in all treatments.

## 6. Assets and legal note

- **Original monster pictures:** the scenario data includes them. They could
  appear as billboards or in Classic presentation.
- **Distribution:** any use in a distributed build is subject to the
  approved legal review (D12).
- **3D content:** original models and textures would be new work. Their cost
  should be estimated in P3 before committing to a fully 3D look.
