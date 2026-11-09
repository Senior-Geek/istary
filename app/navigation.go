package app

import (
	"slices"

	"github.com/Senior-Geek/istary/engine"
)

// Town navigation (Castle-0): moving between the castle's locations as the
// terminal's menus do, for front ends that show locations instead of
// reading keys. The terminal still reads the castle menu (G, A, B, C, E) and
// the Edge of Town's T and C itself (cmd/istary handleTownInput); these
// operations make the same changes, and cmd/istary's navigation tests hold
// them to that key for key. Entering the maze (Edge of Town M) is
// EnterMaze. Nothing here is saved: the town location is not part of the
// save.

// townExits lists where each location leads, in menu order: the castle
// menu, the Edge of Town (Player's Guide p.36), and L)EAVE back to the
// castle from the Training Grounds (p.22) and the Tavern, and RETURN back
// from Boltac's, the Inn and the Temple.
var townExits = map[engine.TownLocation][]engine.TownLocation{
	engine.Castle:     {engine.Tavern, engine.Inn, engine.Trading, engine.Temple, engine.EdgeOfTown},
	engine.EdgeOfTown: {engine.Training, engine.Castle},
	engine.Training:   {engine.Castle},
	engine.Tavern:     {engine.Castle},
	engine.Trading:    {engine.Castle},
	engine.Inn:        {engine.Castle},
	engine.Temple:     {engine.Castle},
}

// noParty is the castle's answer when the Inn, Boltac's or the Temple is
// chosen with no party.
const noParty = "YOU HAVE NO PARTY!"

// Exit is one way out of the current town location.
type Exit struct {
	To      engine.TownLocation
	Refusal string // the game's answer if going there now is refused; "" if open
}

// Exits lists the ways out of the current town location, in menu order.
// Every location's screens are the controller's now.
func (c *Controller) Exits() []Exit {
	town := c.Game.Town
	var out []Exit
	for _, to := range townExits[town.Location] {
		out = append(out, Exit{To: to, Refusal: entryRefusal(town, to)})
	}
	return out
}

// entryRefusal is the castle's refusal of going to loc now, or "".
func entryRefusal(town *engine.TownState, loc engine.TownLocation) string {
	switch loc {
	case engine.Inn, engine.Trading, engine.Temple:
		if town.Party.Size() == 0 {
			return noParty
		}
	}
	return ""
}

// GoTo moves to one of Exits, as its key on the terminal's menu does (for
// Boltac's, B; out of it, RETURN at "WHO WILL ENTER"; the Inn's are A and
// RETURN at "WHO WILL STAY"; the Temple's are C and RETURN with nothing
// typed). A
// refused move leaves the refusal in Town.Message, as the key does, and
// returns it as a *Refusal.
func (c *Controller) GoTo(to engine.TownLocation) error {
	g := c.Game
	if g.Phase == engine.PhaseTown && g.Town.Location == engine.Temple && to == engine.Castle {
		return c.LeaveTemple() // its name prompt is always open
	}
	if g.Phase != engine.PhaseTown || g.Town.InputMode != engine.InputNone {
		return ErrWrongState
	}
	town := g.Town
	if !slices.Contains(townExits[town.Location], to) {
		return ErrWrongState
	}
	switch town.Location {
	case engine.Training: // [RETURN]
		return c.LeaveTrainingGrounds()
	case engine.Tavern: // [RETURN]
		return c.LeaveTavern()
	case engine.Trading: // [RETURN] at "WHO WILL ENTER"; deeper in, LeaveShop
		if town.ShopStep != engine.ShopWho {
			return ErrWrongState
		}
		shopReturn(town)
		return nil
	case engine.Inn: // [RETURN] at "WHO WILL STAY"; deeper in, LeaveInn
		if town.InnStep != engine.InnWho {
			return ErrWrongState
		}
		innReturn(town)
		return nil
	}
	msg := entryRefusal(town, to)
	// The castle and the Edge of Town clear the message on every key.
	town.Message = ""
	if msg != "" {
		town.Message = msg
		return &Refusal{Message: msg}
	}
	switch to {
	case engine.Trading:
		c.castleShopKey()
		return nil
	case engine.Inn:
		c.castleInnKey()
		return nil
	case engine.Temple:
		c.castleTempleKey()
		return nil
	case engine.Tavern:
		return c.EnterTavern()
	case engine.EdgeOfTown:
		return c.EnterEdgeOfTown()
	case engine.Training:
		return c.EnterTrainingGrounds()
	}
	return c.LeaveEdgeOfTown()
}

// EnterEdgeOfTown goes from the castle to the Edge of Town (castle E).
func (c *Controller) EnterEdgeOfTown() error {
	if !c.inTown(engine.Castle) {
		return ErrWrongState
	}
	c.Game.Town.Location = engine.EdgeOfTown
	return nil
}

// LeaveEdgeOfTown returns from the Edge of Town to the castle (Edge of
// Town C).
func (c *Controller) LeaveEdgeOfTown() error {
	if !c.inTown(engine.EdgeOfTown) {
		return ErrWrongState
	}
	GoToCastle(c.Game.Town)
	return nil
}
