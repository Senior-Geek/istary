# UX-3B — Castle interface consolidation

Status: implemented on `ux-3b-castle-interface` (2026-11-12), awaiting
acceptance. Second stage of UX-3. A user-interface and workflow milestone:
no rule of the game changes.

## Audit (before)

- **One frame already:** every town place was drawn in `LocationFrame`
  (Castle-0): the head, the artwork beside the destinations and then the
  place's actions, the content below, and the party bar. The body between
  the head and the party bar scrolled as one, so the artwork and the
  actions scrolled away with the content, except at Boltac's (UX-2's
  `fill`).
- **Character sheet:** a floating, draggable, resizable panel
  (`FloatingPanel`) opened by `App` for the selected roster character.
  There was no sheet button in the Actions, except the Inn's own
  "Character sheet" for its guest.
- **Equipment:** the sheet listed what was equipped and then the whole
  inventory again with Equip / Take off, so every worn item appeared
  twice.
- **Creation:** Yes / No, the name, the password, the race and alignment
  choices, "Done — choose class", the class buttons and Keep / Discard
  were all in the content area. The points and the class were two steps
  (`FinishStats`, then `ChooseClass`), on two displays.

## Concerns, and how they were met

1. **Choosing a class from the points step** needs the controller's two
   steps in one request. `Session.PickClass(class)` runs `FinishStats` and
   then `ChooseClass`, both unchanged. If the class is refused it steps
   back (`CreationBack`), so the character is at the points exactly as
   before (tested). Nothing in `app` or `engine` changed.
2. **Discard at every step** needs a way to discard without leaving.
   `Session.DiscardCreation` is the controller's `AbandonCreation`, the
   same call "← Castle" makes, without leaving the Training Grounds.
3. **Boltac's** has its own Selected Character panel (UX-2), which already
   shows the customer's equipment, and its redesign is UX-3C. To keep UX-2
   as it was accepted, the frame offers no sheet there (`noSheet`).
4. **The class step** (`FinishStats` alone) can no longer be reached from
   the page. If it were reached, it shows the same table, and its classes
   are chosen with `ChooseClass`.

## Layout

The frame (`LocationFrame.svelte`) has the head, then four areas:

| Area | Where | What |
|---|---|---|
| A, display | upper left, fixed | the place's artwork, or the embedded character sheet |
| B, actions | upper right, fixed, scrolls inside if long | the place's decisions, the Character Sheet toggle, then the destinations |
| C, information | below both, scrolls on its own | the roster, the attributes and class table, the merchandise, the rooms... |
| D, party | bottom | the party bar, with Show party / Hide party as before; C takes its room when it is hidden |

- **Top row:** A and B share a definite height (`--top`: about 16:9 of
  A's column, at most 42% of the window, 34% in short windows, 30% at
  Boltac's).
- **Page:** the page and the frame's body never scroll on a wide window.
- **Under 700px:** the areas stack and the body scrolls as one, as
  before.
- **Every place** in town is this frame: the Castle, Edge, Training
  Grounds, Tavern, Boltac's, Inn, Temple and Settings.

## Character sheet

- **Where:** embedded in A in place of the artwork. There is no window;
  `FloatingPanel` and its geometry helper are gone.
- **State:** `App` keeps whose sheet it is and whether it is shown, and
  gives both to the frame through a context (`lib/sheet.js`, `SHEET`).
- **One toggle, in B:** "Character Sheet" / "Hide Character Sheet". Hiding
  brings the artwork back, and so does Escape, before "← Castle".
- **Choosing the character:**
  - Clicking a party member (or a roster row at the Training Grounds or
    Tavern) shows theirs; clicking the same row again hides it.
  - At the Inn the sheet follows the guest; at the Temple, the member
    who tithes (a patient's name shows the patient's).
  - With nobody chosen, the toggle shows the first party member, then the
    roster's first.
- **Inside:** the header (portrait placeholder, name, race, class,
  alignment, level, status) stays in view while the pages scroll.
- **Sections:** Overview (vitals, attributes), Equipment, Spells, and the
  place's own (Party at the Tavern, Delete at the Training Grounds). Each
  is a snippet rendered in turn, so tabs can show one at a time later; no
  tabs now.
- **Offered as before:** Rename and Delete at the Training Grounds,
  Join / Leave at the Tavern.

## Equipment: Worn & Wielded / Pack

- **Two lists of one inventory:** split by Go's `equipped` flag. Each item
  is listed once, and the total is shown as "N / 8".
- **Remove** is the existing take-off (`EquipAction`, `equip: false`). It
  is disabled with Go's reason where Go refuses it (a cursed item).
- **Use** is the existing equip. It is shown only for items Go can equip
  (`kind` set). Scrolls, potions and other items have no Use, since Istary
  has no use for them in town yet. Where the character cannot equip an item
  (class, alignment, a cursed item in the way), Use is disabled with Go's
  reason.
- **Saving and refusals:** each change is saved at once, as before, and
  the notice says so. Nothing is removed, replaced or destroyed.

## Training Grounds

- **Actions:** every decision is in the Actions, under the next one in
  words:
  - Yes / No;
  - the name and password with Continue;
  - the race and alignment choices, with Back;
  - the bonus points left;
  - Keep and save / Discard;
  - Discard at every step.
- **Information:** the character so far, the attributes and the class
  table, the review, or the roster.
- **Points and class, one screen:**
  - The class table follows the points as they are spent.
  - At zero points, the classes Go allows (`classOk`) are a quiet green
    and can be chosen.
  - The others stay visible, dimmed, with Go's reasons.
  - The Actions say "Choose a highlighted class."
  - Points can be taken back and moved until a class is chosen.
  - Choosing a class goes straight to the review.
- **Outcomes:** hit points, gold and age are rolled when the character is
  kept, as before. `TestPickClassIsTheTwoStepsTogether` checks that a
  character made this way is identical to one made through the two
  steps.

## Tests

| Requirement | Tests |
|---|---|
| Shared layout across places | `layout.test.js` (every place is the frame, its four areas in order, A and B fixed, C and B scrolling); live at every place |
| Actions accessible | `layout.test.js` (B scrolls inside); live (every action button in the window at each place, both sizes) |
| Creation decisions in Actions | `layout.test.js` (each creation call and button is in the Actions, none below) |
| Unified points and class | `TestAClassIsChosenFromThePoints`, `creation.test.js` over `creation-views.json`; live |
| Class refused / points left | `TestAClassIsChosenFromThePoints` (points, stats and step unchanged), `creation.test.js` |
| Outcomes unchanged | `TestPickClassIsTheTwoStepsTogether` |
| Discard at every step | `TestDiscardingACreationStaysAtTheTrainingGrounds` (nothing saved, still at the Training Grounds) |
| Embedded sheet, one toggle, no window | `layout.test.js`, `sheet.test.js`; live |
| Worn & Wielded / Pack, capacity, Use / Remove | `equipViews.test.js` over `equip-views.json`; live round trip (Remove, then Use, saved exactly as before) |
| Boltac's unchanged | `selectedViews.test.js` (its UX-2 layout checks; the frame check now reads the new CSS) |

## Deferred

- Boltac's side-by-side redesign (UX-3C).
- Sheet tabs.
- Using consumables from the sheet.
- The Temple's confirmation stays in its own area, with its information,
  as accepted in T-3.
