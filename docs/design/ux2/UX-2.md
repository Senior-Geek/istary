# UX-2: Boltac's Trading Post layout

Status: implemented on `ux2-boltac-layout`, awaiting acceptance.

UX-2 rearranges Boltac's for the window. It changes no price, stock,
transaction, rule or save field; the terminal is unchanged.

## 1. What was there, and what was checked first

- **The customer** was chosen at "WHO WILL ENTER", by a click on the party
  bar or, with the party hidden, a list of buttons in the Actions panel.
  The Actions panel also showed the customer's name, gold and carrying,
  and a Character sheet button that opened the floating sheet.
- **Who may shop** is the counter's own rule (`app.ShopEnter`, the
  original's p-code): any member of the party, whatever their condition.
  The castle refuses Boltac's to no party (`YOU HAVE NO PARTY!`). Boltac's
  transactions never reach a roster character outside the party.
- **Equipping outside the party.** The floating character sheet equips
  any roster character anywhere in town (M8; `EquipAction` takes a roster
  index), and the page keeps an open sheet open from place to place. So a
  sheet opened on a character outside the party (at the Training Grounds,
  say) can still equip them at Boltac's. That is a sheet's operation, not a
  transaction: no gold or item changes hands. UX-2 keeps it as it is
  (`gui` `TestSheetsOutsideThePartyAtBoltacs` records it); Boltac's own
  panel equips only the customer.
- **The town location is not saved**, so a restart begins at the castle;
  arriving at Boltac's is always a Navigate.
- **The party bar's visibility** is one setting for every place.

## 2. Arriving

`Session.navigate` to Boltac's now serves the first party member Boltac
would serve (`serveFirst`: the first slot `ShopEnter` accepts), so the
page opens on the Buy tab with a customer and no "WHO WILL ENTER" first.
It writes nothing. "WHO WILL ENTER" is still reached by Leave the counter;
then a click on the party bar serves someone again. If nobody could be
served, the Actions panel says so and offers Back to the Castle.

## 3. The layout (980x692 and up)

The location frame gains a `fill` layout (`LocationFrame.svelte`): the page
does not scroll; the artwork row is a little shorter (at most 30% of the
window's height) and the content row takes the rest.

| Area | Holds |
|---|---|
| upper left | the artwork area, kept for Boltac's picture |
| upper right, Actions | Pool party gold, Leave the counter; with nobody at the counter, a line saying so (or why nobody can be served, and Back to the Castle) |
| lower left, merchandise | the Buy, Sell, Identify and Uncurse tabs, Boltac's words or question and the last receipt, fixed; under them the list, which alone scrolls, its column heads kept in view |
| lower right, Selected character | name, alignment, class, level, condition; gold, AC, carrying (n of 8, room); the items, Equipped apart from Carried, not equipped, each with Equip or Take off; the item lists scroll on their own |
| bottom | the party bar |

At Boltac's the party bar is always shown (`partyAlways`), whatever the
Hide party setting, and its toggle is not offered there: the bar is how the
customer is chosen. Elsewhere the setting applies as before. A click on a
member serves them at once; it never opens a sheet. The Character sheet
button and the customer buttons are gone from the Actions panel.

## 4. Equipping beside the counter

The panel's Equip and Take off are the character sheet's (`lib/equip.js`,
`Session.EquipAction`), so the rules are the same: the class rule, a cursed
item stuck once worn, an item of another alignment cursing when worn, AC
worked out again, saved at once. One guard is new: equipping is refused
while Boltac waits for an answer (`equipRefusal`, "Answer Boltac's question
first."), since his question is about an item as it is worn. The panel
shows the reason instead of buttons then.

## 5. Tests

| Brief | Tests |
|---|---|
| arriving serves the first member, Buy tab | `gui/ux2_test.go` `TestArrivingServesTheFirstMember`; the shop and navigation tests check it on every arrival; `selectedViews.test.js` |
| switching customers | `TestSwitchingCustomers`; `selectedViews.test.js` |
| party only | `TestBoltacServesThePartyOnly`, `TestSheetsOutsideThePartyAtBoltacs` |
| purchases, sales, equipping, rules | `TestEquippingBesideTheCounter`, `TestEquipRulesBesideTheCounter`; every shop test |
| persistence | `TestTheCounterAfterARestart` |
| panel agrees with Boltac | `selectedViews.test.js` over the recorded views (`shop-views.json` gains each case's `customer` and one case equipping at the counter) |
| layout | `selectedViews.test.js` (no sheet button, shop actions only, the list alone scrolls, the bar always shown); the live check at 980x692 and 1440x900 |

The existing shop tests begin at "WHO WILL ENTER" as before: they first
check that arriving served the first member, then leave the counter.

## 6. Limitations

- At 980x692 the merchandise shows about four lines at once, fewer when a
  refusal wraps under its button; it scrolls.
- The artwork area is placeholder art until the artwork milestone.
