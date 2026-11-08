package main

import (
	"errors"
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
	"github.com/gdamore/tcell/v2"
)

// Town navigation for front ends without keys (app.Controller.GoTo) must
// make exactly the change the terminal's menu key makes.

// terminalMoves is every move between town locations on the terminal's
// menus: the castle, the Edge of Town, and [RETURN] out of the Training
// Grounds and the Tavern.
var terminalMoves = []struct {
	from engine.TownLocation
	key  *tcell.EventKey
	to   engine.TownLocation
}{
	{engine.Castle, runeKey('G'), engine.Tavern},
	{engine.Castle, runeKey('A'), engine.Inn},
	{engine.Castle, runeKey('B'), engine.Trading},
	{engine.Castle, runeKey('C'), engine.Temple},
	{engine.Castle, runeKey('E'), engine.EdgeOfTown},
	{engine.EdgeOfTown, runeKey('T'), engine.Training},
	{engine.EdgeOfTown, runeKey('C'), engine.Castle},
	{engine.Training, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), engine.Castle},
	{engine.Tavern, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), engine.Castle},
}

func runeKey(r rune) *tcell.EventKey { return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone) }

// navGame is a scenario-1 game at from, with a one-member party when party
// is set and a message left on the screen.
func navGame(t *testing.T, party bool, from engine.TownLocation) *engine.GameState {
	t.Helper()
	g := newGame(t, 1)
	if party {
		c := character("WALKER", engine.Human, engine.Fighter, engine.Good)
		g.Town.Roster.Add(c)
		g.Town.Party.Members = append(g.Town.Party.Members, c)
	}
	g.Town.Location = from
	g.Town.Message = "EARLIER MESSAGE"
	return g
}

// Each move by the controller leaves the game exactly as its key does, draws
// no random number, and refuses exactly when the key refuses.
func TestGoToIsTheTerminalKey(t *testing.T) {
	for _, party := range []bool{false, true} {
		for _, m := range terminalMoves {
			byKey, byOp := navGame(t, party, m.from), navGame(t, party, m.from)
			untouched := *byOp.Town

			seed(7)
			newDriver(t, byKey).dispatch(m.key)
			keyNext := rand.Int63()
			seed(7)
			err := app.New(byOp).GoTo(m.to)
			opNext := rand.Int63()
			seed(7)
			if fresh := rand.Int63(); keyNext != fresh || opNext != fresh {
				t.Errorf("party %v %v to %v: a random number was drawn", party, m.from, m.to)
			}

			var r *app.Refusal
			switch {
			case errors.Is(err, app.ErrLocationNotMigrated):
				// The Inn, Boltac's and the Temple with a party: the terminal
				// opens them; the controller does not, and changes nothing.
				if !party || byKey.Town.Location != m.to || !reflect.DeepEqual(*byOp.Town, untouched) {
					t.Errorf("party %v %v to %v: not migrated, but the key went to %v", party, m.from, m.to, byKey.Town.Location)
				}
				continue
			case errors.As(err, &r):
				if byKey.Town.Location != m.from || byKey.Town.Message != r.Message {
					t.Errorf("party %v %v to %v: refused %q, the key went to %v with %q", party, m.from, m.to, r.Message, byKey.Town.Location, byKey.Town.Message)
				}
			case err != nil:
				t.Fatalf("party %v %v to %v: %v", party, m.from, m.to, err)
			case byKey.Town.Location != m.to:
				t.Errorf("party %v %v to %v: the key went to %v", party, m.from, m.to, byKey.Town.Location)
			}
			if byKey.Phase != byOp.Phase || !reflect.DeepEqual(*byKey.Town, *byOp.Town) {
				t.Errorf("party %v %v to %v: the game differs\n key %+v\n op  %+v", party, m.from, m.to, *byKey.Town, *byOp.Town)
			}
		}
	}
}

// The controller's exits are exactly the terminal's moves, in menu order.
func TestExitsAreTheTerminalMenus(t *testing.T) {
	for _, from := range []engine.TownLocation{engine.Castle, engine.EdgeOfTown, engine.Training, engine.Tavern} {
		var want, got []engine.TownLocation
		for _, m := range terminalMoves {
			if m.from == from {
				want = append(want, m.to)
			}
		}
		for _, e := range app.New(navGame(t, true, from)).Exits() {
			got = append(got, e.To)
		}
		if !slices.Equal(got, want) {
			t.Errorf("exits from %v: %v, the terminal's %v", from, got, want)
		}
	}
}

// No other key on these menus moves to another town location: the list of
// moves above is complete.
func TestNoOtherKeyMoves(t *testing.T) {
	var keys []*tcell.EventKey
	for r := ' '; r <= '~'; r++ {
		keys = append(keys, runeKey(r))
	}
	keys = append(keys, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	for _, party := range []bool{false, true} {
		for _, from := range []engine.TownLocation{engine.Castle, engine.EdgeOfTown, engine.Training, engine.Tavern} {
			for _, k := range keys {
				g := navGame(t, party, from)
				d := newDriver(t, g)
				d.dispatch(k)
				if g.Phase != engine.PhaseTown || g.Town.Location == from {
					continue // the maze, a prompt, quitting, or nothing
				}
				listed := false
				for _, m := range terminalMoves {
					listed = listed || (m.from == from && m.to == g.Town.Location)
				}
				if !listed {
					t.Errorf("party %v: key %q at %v goes to %v, which is not a listed move", party, k.Name(), from, g.Town.Location)
				}
			}
		}
	}
}
