// Package app is the application controller: it owns a game session and
// exposes semantic operations (choose a race, add a party member, ...) that
// any front end can call. It has no terminal dependencies; the tcell front
// end translates key presses into Key values and the controller maps those
// onto the same operations a GUI would call directly.
//
// Workflows are migrated here incrementally. HandleKey reports whether it
// handled a key so the terminal front end can fall back to its legacy
// handlers for phases that have not moved yet.
package app

import "github.com/Senior-Geek/istary/engine"

// Controller drives one game session.
type Controller struct {
	Game *engine.GameState
}

// New returns a controller for the given game session.
func New(game *engine.GameState) *Controller {
	return &Controller{Game: game}
}

// Save persists the roster and party. Front ends call it after each action,
// as the terminal loop does after every key press.
func (c *Controller) Save() error {
	return c.Game.Save()
}

// HandleKey routes a key press to a migrated workflow. It returns false when
// the current phase or prompt is still handled by the front end's legacy
// input code.
func (c *Controller) HandleKey(k Key) bool {
	switch c.Game.Phase {
	case engine.PhaseCreation:
		c.creationKey(k)
		return true
	}
	return false
}
