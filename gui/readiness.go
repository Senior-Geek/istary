package main

// The equipment check (UX-1) at the Edge of Town: before the party sets out
// for the maze, who has no usable weapon or no armor equipped
// (app.Controller.EquipmentCheck). It is a caution only: the page offers
// Continue into Maze or Return to Town, and setting out is never refused
// for it.

import (
	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// ReadinessView is the equipment check: each member's problems, and the
// whole as one sentence ("Equipment check: BILBO has no weapon equipped.").
// Notes is empty and Line "" when everyone who can act is ready.
type ReadinessView struct {
	Notes []app.ReadinessNote `json:"notes"`
	Line  string              `json:"line"`
}

// readiness is the check at the Edge of Town with a party and no expedition
// under way, or nil.
func (s *Session) readiness() *ReadinessView {
	g := s.ctrl.Game
	if g.Phase != engine.PhaseTown || s.location() != "edge" || len(g.Town.Party.Members) == 0 {
		return nil
	}
	notes := s.ctrl.EquipmentCheck()
	return &ReadinessView{Notes: notes, Line: app.EquipmentCheckLine(notes)}
}
