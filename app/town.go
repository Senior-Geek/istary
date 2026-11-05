package app

import "github.com/Senior-Geek/istary/engine"

// GoToCastle returns the player to the castle menu, clearing location
// messages. Used by every town location's "leave" action.
func GoToCastle(town *engine.TownState) {
	town.Location = engine.Castle
	town.SelectedIndex = 0
	town.Message = ""
	town.InnMessages = nil
}

// inTown reports whether the game is at the given town location with no
// prompt open.
func (c *Controller) inTown(loc engine.TownLocation) bool {
	g := c.Game
	return g.Phase == engine.PhaseTown && g.Town.Location == loc && g.Town.InputMode == engine.InputNone
}
