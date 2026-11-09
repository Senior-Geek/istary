package app

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// navController returns a controller at the castle, with a one-member party
// when party is set.
func navController(t *testing.T, party bool) *Controller {
	t.Helper()
	c := newController(t, 1)
	if party {
		ch := engine.NewCharacter("WALKER", engine.Human, engine.Fighter, engine.Good)
		c.Game.Town.Roster.Add(ch)
		c.Game.Town.Party.Members = append(c.Game.Town.Party.Members, ch)
	}
	return c
}

func exitsTo(c *Controller) []engine.TownLocation {
	var out []engine.TownLocation
	for _, e := range c.Exits() {
		out = append(out, e.To)
	}
	return out
}

// Every location's exits, in menu order, and the moves along them.
func TestTownExits(t *testing.T) {
	c := navController(t, true)
	want := map[engine.TownLocation][]engine.TownLocation{
		engine.Castle:     {engine.Tavern, engine.Inn, engine.Trading, engine.Temple, engine.EdgeOfTown},
		engine.EdgeOfTown: {engine.Training, engine.Castle},
		engine.Training:   {engine.Castle},
		engine.Tavern:     {engine.Castle},
		engine.Trading:    {engine.Castle},
	}
	for _, path := range [][]engine.TownLocation{
		{engine.Tavern, engine.Castle},
		{engine.EdgeOfTown, engine.Training, engine.Castle},
		{engine.EdgeOfTown, engine.Castle},
		{engine.Trading, engine.Castle},
	} {
		for _, to := range path {
			if got := exitsTo(c); !slices.Equal(got, want[c.Game.Town.Location]) {
				t.Fatalf("exits from %v = %v", c.Game.Town.Location, got)
			}
			if err := c.GoTo(to); err != nil || c.Game.Town.Location != to {
				t.Fatalf("go to %v: %v, at %v", to, err, c.Game.Town.Location)
			}
		}
	}
}

// Going anywhere that is not an exit, or with a prompt open, or out of town,
// changes nothing.
func TestGoToRefusals(t *testing.T) {
	c := navController(t, true)
	all := []engine.TownLocation{engine.Castle, engine.Tavern, engine.Inn, engine.Trading, engine.Temple, engine.EdgeOfTown, engine.Training}
	for _, from := range []engine.TownLocation{engine.Castle, engine.EdgeOfTown, engine.Training, engine.Tavern, engine.Trading} {
		for _, to := range all {
			c.Game.Town.Location = from
			if slices.Contains(exitsTo(c), to) {
				continue
			}
			before := *c.Game.Town
			if err := c.GoTo(to); !errors.Is(err, ErrWrongState) || !reflect.DeepEqual(*c.Game.Town, before) {
				t.Errorf("%v to %v: %v", from, to, err)
			}
		}
	}

	c.Game.Town.Location = engine.Castle
	c.Game.Town.InputMode = engine.InputTavernPassword
	if err := c.GoTo(engine.Tavern); !errors.Is(err, ErrWrongState) || c.Game.Town.Location != engine.Castle {
		t.Errorf("with a prompt open: %v", err)
	}
	c.Game.Town.InputMode = engine.InputNone
	for _, phase := range []engine.Phase{engine.PhaseCamp, engine.PhaseMaze, engine.PhaseCombat, engine.PhaseCreation} {
		c.Game.Phase = phase
		if err := c.GoTo(engine.Tavern); !errors.Is(err, ErrWrongState) || c.Game.Town.Location != engine.Castle {
			t.Errorf("in %v: %v", phase, err)
		}
	}
}

// The Inn, Boltac's and the Temple refuse a castle with no party, as the
// castle menu does. With a party GoTo enters Boltac's; the Inn and the
// Temple are still the terminal's, and going there changes nothing.
func TestGoToShopsAndServices(t *testing.T) {
	for _, loc := range []engine.TownLocation{engine.Inn, engine.Trading, engine.Temple} {
		c := navController(t, false)
		c.Game.Town.Message = "OLD"
		var r *Refusal
		if err := c.GoTo(loc); !errors.As(err, &r) || r.Message != "YOU HAVE NO PARTY!" ||
			c.Game.Town.Message != r.Message || c.Game.Town.Location != engine.Castle {
			t.Errorf("%v with no party: %v, message %q", loc, err, c.Game.Town.Message)
		}
		if e := c.Exits(); e[slices.Index(exitsTo(c), loc)].Refusal != "YOU HAVE NO PARTY!" {
			t.Errorf("%v exit with no party: %+v", loc, e)
		}

		c = navController(t, true)
		c.Game.Town.Message = "OLD"
		before := *c.Game.Town
		if e := c.Exits(); e[slices.Index(exitsTo(c), loc)].Refusal != "" {
			t.Errorf("%v exit with a party: %+v", loc, e)
		}
		if loc == engine.Trading { // Boltac's is the controller's (M8-2)
			if err := c.GoTo(loc); err != nil || c.Game.Town.Location != engine.Trading || c.Game.Town.ShopStep != engine.ShopWho || c.Game.Town.Message != "" {
				t.Errorf("Boltac's with a party: %v", err)
			}
		} else if loc == engine.Inn { // the Inn is the controller's (I-1)
			if err := c.GoTo(loc); err != nil || c.Game.Town.Location != engine.Inn || c.Game.Town.InnStep != engine.InnWho || c.Game.Town.Message != "" {
				t.Errorf("the Inn with a party: %v", err)
			}
			if err := c.GoTo(engine.Castle); err != nil || c.Game.Town.Location != engine.Castle {
				t.Errorf("back from the Inn: %v", err)
			}
		} else if loc == engine.Temple { // the Temple is the controller's (T-1)
			if err := c.GoTo(loc); err != nil || c.Game.Town.Location != engine.Temple || c.Game.Town.TempleStep != engine.TempleWho ||
				c.Game.Town.InputMode != engine.InputTempleHelp || c.Game.Town.Message != "" {
				t.Errorf("the Temple with a party: %v", err)
			}
			if err := c.GoTo(engine.Castle); err != nil || c.Game.Town.Location != engine.Castle || c.Game.Town.InputMode != engine.InputNone {
				t.Errorf("back from the Temple: %v", err)
			}
		} else {
			t.Errorf("%v: an exit not tested", loc)
			_ = before
		}
	}
}

func TestEdgeOfTownPreconditions(t *testing.T) {
	c := navController(t, false)
	if err := c.LeaveEdgeOfTown(); !errors.Is(err, ErrWrongState) {
		t.Errorf("leave from the castle: %v", err)
	}
	if err := c.EnterEdgeOfTown(); err != nil || c.Game.Town.Location != engine.EdgeOfTown {
		t.Fatalf("enter: %v", err)
	}
	if err := c.EnterEdgeOfTown(); !errors.Is(err, ErrWrongState) {
		t.Errorf("enter twice: %v", err)
	}
	if err := c.LeaveEdgeOfTown(); err != nil || c.Game.Town.Location != engine.Castle {
		t.Errorf("leave: %v", err)
	}
}
