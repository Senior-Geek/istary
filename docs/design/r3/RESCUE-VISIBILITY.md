# R-3: rescue visibility and feedback

Status: implemented on `r3-rescue-visibility`, awaiting acceptance.

R-3 shows the player what R-1 and R-2 already record: where stranded
characters lie, who they are and how they are, and how a rescue is going.
It is presentation only. The stranded state (R-1,
`design/r1/STRANDED-PARTIES.md`) and the rescue rules (R-2,
`design/r2/RESCUE.md`) are unchanged: no new saved field, no new rule for
finding, picking up, defeat or return.

The original has no map and no markers; the maps are an Istary
enhancement (M-1, decision D9), and so are these markers. The Classic maze
view (`DungeonView`) and the terminal still give no positions.

## 1. Where the markers come from

One rule, `engine.GameState.StrandedSites` (`engine/rescue.go`), lists
the squares where characters lie, each with its characters in roster
order, sorted by level, row and column. A character is placed when:

- their `Stranded` record names a square of a maze level. Only
  `StrandParty` writes it, when the expedition that left them there ended,
  so the square is one the player's own expedition reached; and
- they are someone `FindStranded` could find: out of the castle
  (`InMaze`), not LOST, not in the party, not on a saved expedition
  (another version's included). Both rules share `outInTheMaze`, so a
  marker and Inspect never disagree about who lies where
  (`TestMarkersAndInspectAgree`).

Never placed:

- **LOST characters.** LOST is for ever; they are not recoverable bodies.
- **Characters with no `Stranded` record** (left out by an older version,
  finding #26). Their legacy fields cannot say for certain which level they
  mean (`engine/stranded.go`: a 0 could be the castle or the first level),
  so no square is claimed for them, even when the fields name one. Inspect
  still finds them on that very square, as in R-2.
- A `Stranded` record that names the castle or no square of the maze.

Nothing about markers is saved. They are worked out from the roster each
time a map is drawn, so after a pickup, a save, a restart or a second
defeat they are exactly what the characters' records say.

## 2. What the map may reveal

A marker reveals its square's place (`L1:R11:C5`) and who lies there. It
reveals nothing else:

- no floor, wall, door, stairs, trap or feature on or around the square;
- no square is marked explored, by the maps or by Inspect's search: the
  exploration record (`engine/explore.go`) is written only by
  `GameState.Explore`, which the maps and `FindStranded` never call;
- `RescueArea` is used only to find characters, never to draw.

A marked square the party has not explored is drawn as a marker on an
empty square. A level with markers but nothing explored can be chosen on
the full map (its level button), and shows its markers alone. Other
levels' markers appear only when their level is chosen; the minimap shows
only its own window of the party's level.

## 3. One marker a square

- **Maps** (`app/map_view.go`): `MapView.Stranded` (the whole level) and
  `LocalMapView.Stranded` (the 7x7 window, placed by the same window
  arithmetic as the walls, wrapping at the edges). Each `MapStranded` has
  the square in the map's own frame, the place, the count, and each
  character's name, class, level and condition (`CharacterNote`).
- **Drawing** (`MapSvg.svelte`, `lib/automap.js` `marker`): a red diamond
  at the square's centre, with the count inside when more than one lie
  there (a dot when one). It is not a feature symbol and not the party's
  white arrow. The arrow is drawn after the markers, so it is never
  covered; when the party stands on a marked square the marker moves to a
  small badge in the corner ahead and to the party's left. A feature's
  badge takes the corner ahead-right, and the arrow's tail fills the
  corners behind, so the three never overlap. On the full map the markers
  are drawn 1.4 times larger, so the count can be read.
- **F10 interaction** (`AutomapPopup.svelte`): beside the map, a list of
  the level's markers, one button each (`L1:R11:C5 — 3 stranded`).
  Clicking a marker or choosing its line shows the place, the count, and
  each character with their condition. Tab, Enter and Space work the
  list, as they do the map's other buttons; no other key reaches the maze
  while the map is open. The minimap has no interaction.
- **Partial recovery:** a pickup takes the character out of the roster's
  stranded state (R-2), so the count falls, the line goes from the
  details, and the marker goes when no one is left. Those not picked up
  keep their marker.
- **A second defeat** strands the rescuers and everyone they carry on the
  square where it happens (R-1's `StrandParty`, unchanged), so a new marker
  appears there; those left behind keep the old one.

## 4. Conditions

`app.ConditionLabel` names a condition for the maps and Inspect. These
are the original's status words, spelled out in full: PARALYZED (not
PARALYZE), STONED, DEAD and ASHES (not ASHED), and also OK, ASLEEP, AFRAID
and LOST. `FoundView.Condition` and `CharacterNote.Condition` carry the
label. `FoundView.Status` keeps the Classic word for compatibility.

`Condition.svelte` and `lib/condition.js` show the word itself with a mark
and colour: ≈ PARALYZED (dashed), ▲ STONED (double), ✝ DEAD and ∴ ASHES
(dotted). The word is always shown, so nothing depends on colour alone.

## 5. Progress on Inspect

`InspectView` (`app/dungeon_view.go`) gains:

| Field | Shown as |
|---|---|
| `Size`, `Room` | `PARTY 5 OF 6 · ROOM FOR 1` |
| `Found` (unchanged) | the list, each with their condition |
| `Picked` (the message is a pickup's) | `5 MORE ARE HERE.` after a pickup, `NO ONE ELSE IS HERE.` when none are left; otherwise `7 CHARACTERS ARE HERE.` or `LOOKING... NO ONE IS HERE.` |
| `Elsewhere` | `3 MORE LIE ELSEWHERE (MAP, F10).`: known to lie in the maze but not found from here |
| `Carried` | `In the party, unable to act: BGUI ≈ PARALYZED, NEAR1 ▲ STONED` |

The confirmation after each pickup is R-2's message, `NAME JOINS THE
PARTY.`. Every number is read from the game as it stands, so it is the
same after a restart. None of it gives a position.

**Recovered or not, the game does not say.** Nothing records whether a
member was picked up or started out with the party; the expedition saves
only who is in it now. So the view does not count "recovered" members.
It shows only what is authoritative: who in the party cannot act, each with
their condition. A rescued character keeps their condition (R-2), so
nothing suggests they were healed or raised.

## 6. Stranded, recovered and LOST

| State | Record | Marker | Inspect |
|---|---|---|---|
| Stranded | `Stranded` set, `InMaze` | on their square | found from their area |
| Left out by an older version | no `Stranded`, legacy fields | none | found on the exact square |
| Picked up (recovered) | `Stranded` cleared, in the party | none | in `Carried` if they cannot act |
| Home | `InMaze` false, in the castle party | none | — |
| LOST | `Status` LOST | none | never found |
| Practice copy | the practice game's copy | on the practice map only | the copy |

## 7. Persistence and compatibility

- There is no new save field; the map save format and the exploration
  section are unchanged.
- Markers follow pickups, saves and restarts because they come from the
  roster (`gui` `TestMarkersSurviveARestart`).
- Practice works on copies of the roster (R-2, Q7), so its pickups move
  only the practice map's markers; the player's map, read from the real
  game, is unchanged during and after practice
  (`TestPracticeLeavesTheMarkers`).
- Saves of an unknown version are handled as before: another version's
  expedition members are neither found nor marked.

## 8. Tests

| Brief | Tests |
|---|---|
| 1, 2, 3, 16 | `app/markers_test.go` `TestStrandedMarkers`; `lib/automap.test.js` (one marker a square, the count) |
| 4 | `TestStrandedMarkersByLevel`; `automap.test.js` (other levels) |
| 5 | `TestMarkersRevealNoGeometry`; `automap.test.js` (a marker shows nothing around it) |
| 6, 7 | `TestNoMarkerWithoutAKnownSquare`; `automap.test.js` (the LOST and uncertain never marked) |
| 8, 9 | `TestPickupUpdatesTheMarkers`; `gui` `TestMarkersSurviveARestart` |
| 10 | `TestSecondDefeatMovesTheMarkers` |
| 11 | `gui` `TestMarkersSurviveARestart`, `TestR3Walkthrough` |
| 12 | `gui` `TestPracticeLeavesTheMarkers` |
| 13 | `TestPickupUpdatesTheMarkers`, `TestMarkersAndInspectAgree`; `automap.test.js` (the two maps agree); `gui` `minimapAgrees` |
| 14 | `TestPartyOnAMarkedSquare`; `automap.test.js` (a badge clear of the arrow and the feature badge; the arrow drawn last) |
| 15, 17 | live check at 980x692 in the dev app (mouse on a marker; Tab, Enter and Space on the list; level choice; Escape; Inspect's keys and buttons; nothing clipped or covered) |
| 16 | `lib/condition.test.js`, `TestInspectProgress` |
| 18 | `TestLookingChangesNothing`; every R-2 test unchanged |
| progress | `TestInspectProgress`; `condition.test.js` (the lines) |
| walkthrough | `gui` `TestR3Walkthrough` (x6, x3, a restart, x3, gone) |

The recorded fixtures were extended, not rewritten. The old cases are
unchanged apart from the new empty fields. `map-views.json` gains two
cases with markers (`gui/map_fixture_test.go`). `dungeon-views.json` gains
the progress fields.

## 9. Limitations

- **Legacy characters are not marked** (section 1). Camp D)ISBAND (#26)
  and recovery of invalid legacy locations stay separate compatibility
  work.
- **The party bar keeps the Classic status words** (PARALYZE, ASHED),
  as everywhere outside the maps and Inspect. Changing them would change
  every screen's party bar and is not part of R-3.
- **The terminal is unchanged.** It has no maps. Its Inspect is the
  Classic screen, and its map overlay (DUMAPIC) gives no markers.
- **"Recovered" is not counted** (section 5).
- **Inspect's list** shows about five lines at 980x692 and scrolls for
  more. When Inspect is open, the corridor view shrinks to make room.
