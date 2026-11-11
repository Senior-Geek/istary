# UX-1: maze coordinates, rescue discoverability and equipment warnings

Status: implemented on `ux1-coordinates-warnings`, awaiting acceptance.

UX-1 shows the player what the game already knows: where the party
stands, that someone lies on its square, and who in the party is not
equipped. It changes no rule, no combat, no equipment eligibility, no
rescue mechanic and no save field. The terminal is unchanged.

## 1. Coordinates

`app.Controller.Position` (`app/position.go`) gives the party's square in
canonical L:R:C (`engine.Place`, the maps' and the rescue records' own
form) and its facing (N, E, S, W), in the maze, at Camp and in combat. It
reports nothing in town, and nothing when the party is not on a square of a
maze level (a level below the last, finding #31): no coordinates are made
up. Reading it changes nothing; it does not explore.

The convention is M-1's: L0 the castle, L1 the first level; R0:C0 the
north-west corner, rows growing southward, columns eastward; the facing
kept apart. The entrance is L1:R19:C0, facing north.

`View.Position` carries it (`gui/view.go`). It is not part of
`DungeonView`, which, like the Classic maze screen, gives no position: the
readout is an Istary enhancement, as the maps are (D9). The dungeon
screen's header shows it beside the title, `L1:R19:C0 · N`, in the maze
and, larger, at Camp; its tooltip spells it out ("Level 1, row 19,
column 0, facing north"). Off the maze's squares it says "position
unknown". It sits in the header, so it covers neither the view, the
minimap, the messages nor the party.

## 2. Inspect / Rescue

The maze's Inspect button is now **Inspect / Rescue**. The `I` key, the
action (`inspect`) and what it does are unchanged; it is offered whether
or not anyone lies here, as before.

`PositionView.Stranded` counts the characters on the party's very square,
as that square's map marker counts them (`StrandedSites`, R-3): the same
records, the same exclusions (the LOST, the legacy with no certain square,
the party, other expeditions). When it is not zero, a line under the view
says `2 stranded adventurers here — Inspect / Rescue [I]`. It tells nothing
the minimap's marker under the party does not, and falls with each pickup.
Inspect itself still searches the party's whole area (R-2).

## 3. The equipment check

`app.Controller.EquipmentCheck` (`app/readiness.go`) lists, for each party
member who can act, what they lack. `View.Readiness` carries it at the Edge
of Town with a party. Pressing Enter the maze then shows

    Equipment check: B1 has no weapon equipped and has no armor equipped;
    B2 has no armor equipped; B3 has no weapon equipped.

with **Continue into maze** (sets out as before) and **Return to town**
(goes to the castle). Nothing is bought, equipped or moved, and setting out
is never refused for it. A party with nothing to report sets out at once.

The rules are the game's own:

| Question | Answer, and where it comes from |
|---|---|
| Equipped? | `Possession.Equipped`. An item carried but not equipped does not count. |
| Usable? | equipping's class rule, `canEquip` (`Item.ClassUse`). Alignment decides only whether an item curses when worn (#34), not who may use it. |
| A weapon? | an item of the weapon category (`TypeID` 0). Any usable weapon will do (a staff as well as a sword); combat swings the equipped one, or fights unarmed for 1d2. |
| Protection? | armor, a shield, a helmet or gauntlets (`TypeID` 1-4), any one of them. |
| Expected of whom? | a class the scenario offers some protection that not every class may wear. In the original that is every class but the Mage, whose only armor is ROBES: an unarmored Mage is never cautioned. |
| Ninja | a Ninja with nothing equipped is fighting unarmed on purpose (combat's WC030 initiative bonus applies only then): not cautioned. A Ninja who wears something is checked like anyone. |
| Who? | members who can act (`IsAlive`); the incapacitated carried home do not fight. |

An equipped item that the character's class may not use (an older save, a
change of class) is not counted, and the check says so: `has a weapon
equipped that a PRIEST cannot use`, `has only armor equipped that a THIEF
cannot use`. The class rule is the only usability rule the game has, so
this is reliable; nothing else is guessed.

## 4. Tests

| Brief | Tests |
|---|---|
| coordinates after moves and turns | `app/position_test.go` `TestPositionConvention`, `TestPositionFollowsMovesAndTurns`; `gui/ux1_test.go` `TestCoordinatesOnTheScreen`; `lib/position.test.js` |
| Camp and maze agree | `TestPositionInCampIsTheMaze`, `TestCoordinatesOnTheScreen` |
| no invented coordinates | `TestNoPositionOffTheMaze`; `position.test.js` |
| Inspect / Rescue, its key | `position.test.js`; every Inspect test unchanged |
| stranded indication after pickup | `TestPositionStrandedHere`, `gui` `TestStrandedHereFollowsPickups` |
| no weapon; owned but not equipped; no false warnings; class allowances | `app/readiness_test.go` `TestEquipmentCheck`, `TestEquipmentCheckByClass`, `TestEquipmentCheckLine` |
| continue or return, nothing changed | `TestEquipmentCheckChangesNothing`, `gui` `TestEquipmentCheckAtTheEdge`, `TestEquipmentCheckSkipsTheIncapacitated` |
| saves compatible | no save field added; `TestEquipmentCheckAtTheEdge` (reading the views writes nothing); every save test unchanged |

## 5. Limitations

- The check is offered for a real expedition only. Practice sets out at
  once, as before.
- The terminal shows no coordinates outside DUMAPIC and has no check.
- Alignment-restricted items (the EVIL and NEUT armor, for instance) are
  not treated as unusable by others: the game lets anyone equip them, and
  only curses some when worn.
