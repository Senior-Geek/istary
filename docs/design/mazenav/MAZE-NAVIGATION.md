# Maze navigation follow-up: resumed expeditions and the stairs' suggestion

Two small GUI changes to entering and leaving the maze. No rule, save field
or terminal behaviour changes.

## Findings before the change

- **Entering goes through Camp on purpose.** EDGE OF TOWN M)AZE puts the
  party in Camp at the entrance (level 1, south-west corner, facing north),
  as the original does (guide p.40, 43; `WIZARDRY-COMPATIBILITY.md` M1).
  Leaving Camp on the stairs asks TAKE THEM (Y/N) ? at once. Unchanged.
- **No entry from the Edge of Town resumes an expedition.** A saved
  expedition resumes only when Istary starts (`gui/dungeon.go` `resume`),
  straight to its checkpoint in the maze or Camp. While one is saved, the
  Edge of Town refuses to start another (`errExpeditionSaved`). Every entry
  from the Edge of Town is a new expedition, a rescue after a stranding
  included.
- **The engine did not record how the party reached a square.** The stairs
  question comes from a step onto the stairs (`checkSquare`) or from
  leaving Camp on them (`LeaveCamp`). Taking a staircase between floors puts
  the party on the matching staircase, unasked. Teleporters, chutes, MALOR
  and the elevator can also leave it on a staircase. Turning never asks.
- **The GUI always highlighted Yes**, the first answer; only Y and N (or a
  click) answer, never Enter.
- **Leaving Camp names stairs between floors wrongly** (#39): `LeaveCamp`
  says GOING UP only for the stairs to the castle and GOING DOWN for every
  other staircase, so floor 2's stairs up read STAIRS GOING DOWN after Camp.
  A step onto the same square says GOING UP (`checkSquare` compares the
  destination with the floor). Reported, not corrected here.

## Decisions (2026-11-11)

1. **The rule is how the party arrived**, at every staircase:
   - **off the stairs** (down from the castle, or off a staircase just
     taken): suggest **No**. The party has just come that way.
   - **by a step or a kick onto the square**: suggest **Yes**. The party came
     to the stairs to take them: home from inside the maze, or on to the
     next floor.
   - **any other way** (teleporter, chute, MALOR, the elevator, a restart):
     suggest **Yes**, as before.
   Turning in place, Camp and an encounter on the square change nothing.
   Walking off and back onto the stairs makes the suggestion Yes.
2. **Kept in memory only**: the controller's `arrival` (`app/stairs.go`)
   records the square and how it was reached; it is not in the checkpoint,
   so the save format is unchanged. After a restart the suggestion is Yes.
3. **The suggestion is a highlight only.** `DungeonView.Default` names it;
   the maze screen highlights that button. Y, N or a click still answer;
   no key takes the suggestion, and nothing is done by itself.
4. **EXPEDITION RESUMED** heads the notice when Istary restarts and puts a
   saved expedition back (`View.Title`, `gui/dungeon.go` `resume`): "Your
   party has returned to the dungeon. Stranded adventurers remain at their
   recorded locations." An encounter or chest that was interrupted adds its
   sentence between the two. It is informational: no button, and it goes
   with the next action. It names no square and shows nothing the party
   has not seen. A new expedition, a rescue included, has no heading.

## Where

- `app/stairs.go`: `arrival`, `arrived`, `stairsDefault`. Set by
  `EnterMaze`, by taking a staircase between floors, and by every step or
  kick (before the square's effect, which may move the party on).
- `app/dungeon_view.go`: `DungeonView.Default`, for the stairs only.
- `gui/dungeon.go`, `gui/view.go`, `gui/session.go`: the heading.
- Frontend: `dungeon.js` `suggested`, `DungeonScreen.svelte`, `App.svelte`.

## Tests

- `app/stairs_test.go`: entering (No, through turns and Camp, Yes still
  takes the party home); returning to the entrance (Yes, through Camp, No
  still stays; the next entry is No again); floors 1 and 2 both ways
  (walked onto: Yes; arrived by: No; off and back: Yes; both answers taken);
  set down on stairs another way, and a new controller (a restart): Yes.
- `gui/mazenav_test.go`: a first expedition and a rescue expedition have no
  heading and suggest No; a restart is headed, once, with the party, its
  square, the dungeon view and the stranded unchanged, and its entrance
  then suggests Yes; returning to the entrance suggests Yes and Yes goes
  home with the stranded left where they lie.
- `gui/expedition_test.go` `restart`: every restart of the expedition walks
  now also checks the heading (and none in town).
- `gui/camp_test.go`: the Camp restart comparison allows the one thing a
  restart forgets, the suggestion, to become Yes.
- Frontend `dungeonViews.test.js`: the recorded stairs views suggest both
  answers, Enter and Space take neither, other prompts are unchanged.
