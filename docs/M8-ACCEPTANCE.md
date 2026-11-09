# M8 acceptance — Boltac's Trading Post

Branch `m8-boltac`, reviewed through `70191a0` (23 commits on `main`
86e6b1c). Accepted on 2026-11-09.

"M8" here is the working milestone for Boltac's. In
`ISTARY-DEVELOPMENT-ROADMAP.md`, Boltac's belongs to M3/M4; the roadmap's
own M8 ("Classic Wizardry I complete") is not delivered by this branch.

## Features delivered

- **Stock per game** (M8-1): Boltac's stock belongs to each game, not to the
  scenario; reloading resets it, practice keeps its own (#12 kept).
- **Controller** (M8-2): the terminal shop moved intact into `app`; every
  answer (buy, sell, identify, uncurse) is decided in one place.
- **Bridge** (M8-3): `ShopAction` and `View.Shop`; each transaction is saved
  on its own, and a failed save is reported, never hidden.
- **Trading Post screen** (M8-4): Buy, Sell, Identify and Uncurse tabs;
  customers chosen and switched from the party bar.
- **#34 curses stick:** a cursed item, or one of another alignment, sticks
  when worn; only Boltac's uncurse (half the price) removes it, and he
  will not buy it.
- **#35 AC is what is worn:** worked out on every equip, sale, uncurse,
  drop and load.
- **B7:** Boltac asks before buying an equipped item (Sell it / No, keep
  it; Escape is No).
- **Town equipping** from the character sheet, by the equip walk's rules.
- **Try saving again** after a failed save, repeating nothing.

## Tests performed

**Live, in the Wails app** (`wails dev`, driven by clicks through Chrome
DevTools), at 760x532, 980x692 and 1512x945. Every check passed at every
size:

1. *Create:* a fresh roster of six (two fighters, priest, thief, two mages)
   made in the Training Grounds; party formed at the Tavern; each member
   bought weapons and armour at Boltac's (customer switched by the party
   bar, and back); everything equipped from the sheet, AC checked against
   the scenario's values; Leave the game quit the app.
2. *Trade* (after a restart; the save gave one fighter gold and the other
   an unidentified helm while closed): the helm identified and worn (AC
   down by 1); EVIL CHAIN + 2 bought and worn by a good fighter, which
   stuck (CURSED notice, Take off disabled, leather armour blocked);
   Boltac refused to buy it, then uncursed it for 4,000 GP (AC
   recomputed); selling the worn long sword: No kept it, Escape kept it,
   Sell it sold it for 12 GP and AC updated. A failed save (save folder
   made read-only) kept the purchase on screen, marked unsaved; Leave the
   game was refused; Try saving again failed while read-only and repeated
   nothing, then saved once it was writable (gold and items on disk
   matched, once).
3. *Final* (after a restart): roster, equipment, gold and AC identical to
   before leaving; every AC equal to what is worn.
4. *Expedition:* Castle, Tavern, the maze, Camp (party and ACs match the
   roster), inspecting a member, out by the stairs; roster unchanged.

**Gate at `70191a0`:** root and GUI vet, tests, shuffled tests and race
(app race 31 min, GUI race 10 min); frontend 80 pass, 0 fail; every golden
and fixture regenerated (`-update`, `ISTARY_UPDATE_FIXTURES=1`) with no
difference; `make build`; Wails production build signed and verified;
working tree clean. Each M8 completion commit had passed on its own
(root, GUI, frontend) on 2026-11-09.

## Known limitations

- **Closing the window with a failed save** (found in acceptance) was
  fixed before release: the close button, ⌘Q and Quit ask first (Keep
  playing, Try saving again, Quit without saving); see `ARCHITECTURE.md`.
- Deferred by decision: further impairment from misaligned items, bishop
  identification failure, restrictions on dead characters' equipment,
  Pool Gold (#4).
- Kept by decision: stock resets on reload (#12); unidentified items sell
  for 1 GP (B6); Boltac's sell-question wording is Istary's own.
- The live harness sees one console error per page load, from Wails' dev
  `ipc.js` in an external browser (`reading 'nodes'`); it predates M8 and
  is not in Istary's code.

## Regressions and unresolved defects

None found.

## Screenshots

`docs/design/m2-exploration/stills/gui-m8-acceptance/`, in workflow order:
party formed, bought, sheet equipped, identified, cursed and stuck, cursed
sale refused, uncursed, sell question, sold worn, not saved, leave refused,
saved again, after restart, Camp.

## Recommendation

**Accept.** Every workflow check passed in the app at three sizes, the full
gate is green, and the one finding (closing the window while unsaved) was
fixed before release. Accepted by the owner on 2026-11-09.
