package main

// The maps (M-1): the minimap beside the dungeon view, and the full map of
// an explored level (F10). Both are the controller's map views of the
// player's exploration (app/map_view.go), which the controller records
// after every action in the maze (combatDo); the page draws them and
// decides nothing. Looking at a map is not an action: it takes no turn,
// moves nobody, draws no random number and saves nothing.
//
// A practice expedition starts from the player's map, and what it shows is
// discarded with it (practiceGame).

import (
	"errors"

	"github.com/Senior-Geek/istary/app"
)

var errNoMapYet = errors.New("Nothing of the maze has been explored yet.")

// Automap returns explored level l (from 1) of the game being played (the
// practice game during practice); 0 is the party's level. It changes
// nothing.
func (s *Session) Automap(l int) (app.MapView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed() {
		return app.MapView{}, s.closedErr()
	}
	c := s.active()
	if c == nil {
		c = s.ctrl
	}
	m, err := c.Map(l)
	if errors.Is(err, app.ErrNoMap) {
		err = errNoMapYet
	}
	return m, err
}

// minimap is the minimap for the View: the squares around the party on
// an expedition (real or practice), or nil.
func (s *Session) minimap() *app.LocalMapView {
	c := s.active()
	if c == nil || !(s.expedition || s.real) {
		return nil
	}
	if v, ok := c.LocalMap(); ok {
		return &v
	}
	return nil
}
