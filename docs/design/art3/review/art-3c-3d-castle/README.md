# ART-3C and ART-3D: the Training Grounds and the Temple, as integrated

**Branch:** `art-3c-3d-castle-artwork`, on v0.19.0-boltac (`3b28701`).
**Status:** awaiting review. Design notes and the approved stories:
`docs/design/art3/training-grounds/README.md` and
`docs/design/art3/temple/README.md`.

Taken with `wails dev` on a copy of the acceptance library. For the
Temple, the copy has BGUI dead and CGUI paralysed in the party, RESERVE
in ashes on the roster, and 5,000 GP for AGUI. Headless Chrome read the
pages at exactly 1440×900 or 980×692, at 2× device pixels, with Dark
panels, Parchment story and Full artwork.

| File | What it shows |
|---|---|
| `training-arrival-story-open-980x692.png` | The first visit: Brannock's story open, the Actions (Create a character) and the roster; his face is under the Actions panel at this size |
| `training-roster-1440x900.png` | The story closed: Brannock beside the Actions panel, the castle gate, the recruits at practice, the roster |
| `training-creating-980x692.png` | Creating a character: the bonus points and the class table; no story box and no Story while creating |
| `training-panels-hidden-980x692.png` | Hide panels at the smaller window: Brannock whole |
| `temple-arrival-story-open-1440x900.png` | The first visit: Father Caelion's story open beside the stained glass; Father Caelion, the patients, tithing |
| `temple-ask-raise-980x692.png` | Raising BGUI: the risk (ashes), the outcomes, the donation |
| `temple-receipt-980x692.png` | CGUI cured of paralysis ("Saved"); with a shorter tithing list, Father Caelion's face is in view |

## Choosing the crops

Both principal figures stand at the right of their pictures, where the
Actions panel is. Both windows are narrower than 16:9, so a picture fills
the window's height and its crop moves it only sideways: by at most 160 px
at 1440×900 and 250 px at 980×692. Crops at 50%, 75% and 100% were
measured at both sizes. In every state below, for each crop, the measure
was which panel, if any, is drawn over each part of the picture, and
whether every button and field is in view and uncovered.

- **Training Grounds:** at 100% Brannock's eyes, beard, medals and the
  hand on his axe are in view at 1440×900, with the Actions panel at his
  cheek. At 50% and 75% his eyes and hair are under it too. At 980×692
  his face is under the panel at every crop. Chosen: 100%.
- **Temple:** his face is in view at every crop at 1440×900. At 980×692
  his eyes are under the panel at every crop while it lists the members
  who cannot tithe. The top of his staff is in view only at 75% (it
  leaves the window at 50%, and is under the panel at 100%). Chosen: 75%.

States measured: Training Grounds: arrival with the story open, story
closed, a character's sheet, creating (the points), creating after
asking for the story, panels hidden. Temple: arrival with the story open,
story closed, the question before raising the dead, the receipt for
curing paralysis, panels hidden. In all of them, at every crop and both
sizes, no control was covered or clipped.

## Checked live (`scripts/acceptance/live.mjs`)

The layout, scenes, nav and startup suites, at both sizes:

- **The pictures:** loaded 1920×1080, filling the window, cropped to cover
  it, seen through the display, its panels near-opaque.
- **Every control:** in view and uncovered.
- **Hide panels and Escape:** Hide panels leaves the picture alone, with
  Show panels; Escape brings the panels back.
- **Hide party:** gives its room to the panels.
- **Window sizes:** larger and smaller windows, and all 27 appearances
  (text contrast, opacity, nothing moved).
- **The stories:**
  - each place's own, open on its first visit, titled, with no challenge,
    covering no control;
  - once closed, not opened again on return; Story opens it again.
- **Creating a character:** no story box and no Story; every control in
  view; declined, the roster and the story box again.

Since every place in town now has a picture, Settings is the place
without one that the suites check, in place of the Temple. That includes
the layout suite's Hide party check, which had used the Training Grounds.
At an illustrated place the information is only as tall as it needs, so
hiding the party gives its room to the display and actions instead. The
scenes suite checks that rule at every illustrated place.

## Open points

1. **At 980×692 the Actions panel covers the principal figures' faces:**
   Brannock's always, and Father Caelion's eyes while the tithing list is
   long. No crop avoids it; Hide panels or a larger window shows them.
   Showing them would mean changing the frame's layout. (ART-3E: the
   Actions now collapse to their heading, and the faces are in view:
   `../art-3e-actions/`.)
2. **The Temple's sentence on what it treats was corrected**, as approved:
   "Petrification may be reversed" in place of "Poison may be cleansed".
   The Temple does not cure poison.
3. **Two quotation marks were added to the Training Grounds' text:** one
   opening Brannock's first speech, one closing his last. No word is
   changed.
4. **Settings' Appearance section still says** "For the places with
   pictures, the Castle and the Edge of Town." This was already out of
   date before this work, and is not changed here. (Corrected in ART-3E.)
