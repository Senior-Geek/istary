# ART-3E: the collapsible Actions panel

**Branch:** `art-3c-3d-castle-artwork`, after ART-3C (`940021f`) and
ART-3D (`ce37ac3`), on v0.19.0-boltac (`3b28701`). **Status:** awaiting
review.

At an illustrated place the Actions panel stands over the picture's right
side. That is where Sergeant Major Brannock Ironheel and Father Caelion
stand, so at 980×692 it covered their faces (ART-3C and ART-3D, open
point 1). The panel now collapses to its heading, so the picture can be
seen, and comes back. The picture, its crop, the display, the
information, the story box and the party bar do not move.

## Behaviour

- **Expanded (as before, and at the start of each session):** the
  Actions' heading has **Collapse** at its right (named "Collapse
  Actions", with a tooltip, `aria-expanded="true"`). Everything else in
  the panel is where and as it was.
- **Collapsed:** only the heading is drawn, with **Show Actions**
  (`aria-expanded="false"`), at the top right of the Actions column where
  the panel was: 204×46 px in place of 298×218 at the Training Grounds.
  The rest of the column is the picture and lets clicks through
  (`pointer-events: none`). The panel's body is hidden, not removed, so
  what a place holds in it (a name being typed, who tithes) is kept.
- **Keyboard:** both are ordinary buttons, reached with Tab and pressed
  with Enter or Space. The focus moves to the other one when pressed.
  Escape is unchanged: it shows hidden panels, hides a shown sheet, or
  goes back to the Castle.
- **One choice for the session (App's `PANELS`):** the same at every
  illustrated place with Actions (the Training Grounds, the Tavern, the
  Inn, Boltac's and the Temple), and kept moving between them. It does not depend on the picture setting (Full, Subdued
  or Hidden). Each place shows its own Actions when expanded. Nothing is
  saved: no setting, no save file, nothing sent to Go. The Castle's
  panel is its destinations, and the Edge of Town's Actions are its ways
  out (Enter the Maze, the Training Grounds, Settings…): neither collapses,
  so the ways out are always in view. Settings has no picture, so its
  panel is as it was.
- **While a character is being created** the Actions hold creation's
  decisions (name, password, race, alignment, points, Keep and Discard),
  so they are shown and not collapsible. After creation the choice holds
  again.
- **Operations go on:** Temple treatments (the question is in the
  information, below), Boltac's counter and the roster's sheets work
  collapsed or not, and the panel can be expanded at any time.

## Screenshots

Taken with `wails dev` on a copy of the acceptance library in which BGUI
is dead and CGUI paralysed in the party, RESERVE is in ashes, and AGUI
has 5,000 GP. The Temple's tithing list is therefore at its longest.
Headless Chrome read each page at exactly the size named, at 2× device
pixels, with Dark panels, Parchment story (closed) and Full artwork. Each
pair was taken in the same state, one before and one after Collapse.

| File | What it shows |
|---|---|
| `training-expanded-980x692.png` | The Training Grounds, Actions expanded: Brannock's face under the panel |
| `training-collapsed-980x692.png` | The same, collapsed: Brannock's face, beard and medals in view; Show Actions at the top right |
| `temple-expanded-980x692.png` | The Temple with three patients and the long tithing list: Father Caelion's eyes under the panel |
| `temple-collapsed-980x692.png` | The same, collapsed: his face in view; the patients and Treat… as they were |
| `trading-collapsed-980x692.png` | Boltac's, collapsed: the same control, reused |
| `*-1440x900.png` | The same at 1440×900 |

Measured in each pair (three points on each face: eyes, nose, mouth;
seen means the picture is the topmost thing drawn there):

| | Expanded | Collapsed |
|---|---|---|
| Brannock, 980×692 | 0 of 3 seen | 3 of 3 |
| Brannock, 1440×900 | 2 of 3 | 3 of 3 |
| Father Caelion, 980×692 | 1 of 3 | 3 of 3 |
| Father Caelion, 1440×900 | 3 of 3 | 3 of 3 |

The picture was the same size, at the same crop, in every pair
(`980x692 100% 50%` at the Training Grounds, and so on).

## Tests

- **Frontend (`actionsPanel.test.js`):** the session's choice in PANELS,
  expanded at the start, never saved; one control, with its names and
  state, only at illustrated places with Actions and not while creating;
  the body hidden, not removed; the collapsed column letting clicks
  through, nothing else in the layout changing; Settings' corrected
  text.
- **Live (`scripts/acceptance/live.mjs`, the scenes suite):**
  - **Start and keyboard:** the Actions start expanded, with Collapse
    named and its state. Enter on Collapse hides the body and Show
    Actions takes the focus; Space on Show Actions brings them back and
    Collapse takes the focus.
  - **Nothing moves:** collapsed, only the heading, at the column's top
    right. The picture, the crop, the display, the information and the
    story box do not move, and nothing of the game changes (the whole
    View is the same).
  - **Clicks and faces:** the column under the heading lets clicks
    through. Brannock's and Father Caelion's faces are in view, and every
    control is in view and uncovered.
  - **Training Grounds:** a roster row shows its sheet, and Escape hides
    it. Hide panels and Escape behave as before, the Actions staying
    collapsed.
  - **Moving between places:** kept from the Training Grounds to the
    Castle, which has no Actions, then the Temple and the Tavern. Each
    place's own Actions come back when expanded.
  - **A Temple treatment:** a dead character on the roster (the scratch
    library's save, read again by Reload from disk). Treat… asks with the
    Actions collapsed, its buttons in view; the Actions expand while it
    asks; Cancel leaves the character untreated and uncharged. The
    library is put back as it was afterwards (its SHA-256 checked).
  - **Creation:** creating a character, the Actions are not collapsible
    and the decisions are in view; declined, the Actions as chosen.
  - **Live-suite helper change:** the helper that finds every button
    (`controls()`) now skips buttons inside a `hidden` element: those in
    a collapsed panel are not drawn at all. No button was hidden that way
    before.

## Limitations

- **Collapsing is the player's choice:** at 980×692 the expanded panel
  still covers the faces, as before. Nothing moves the panel or detects
  the faces.
- **The choice is not saved:** each session starts expanded.
- **The Castle and the Edge of Town do not collapse:** their panels are
  their ways out.
