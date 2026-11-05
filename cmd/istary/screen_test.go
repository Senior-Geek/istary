package main

import (
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// Lightweight terminal snapshots: plain text of a few key 80x24 screens
// (no colors or styles). They guard the existing terminal layout while the
// controller is extracted; the GUI will not use them.
func TestTownScreenSnapshots(t *testing.T) {
	seed(42)
	g := newGame(t, 1)
	s := newSimScreen(t)
	d := newDriver(t, g)
	var out strings.Builder
	snap := func(name string) {
		s.Clear()
		redraw(s.Screen, g)
		out.WriteString("=== " + name + "\n" + s.text() + "\n")
	}

	snap("castle, no party")

	for _, c := range adventurers()[:3] {
		g.Town.Roster.Add(c)
		g.Town.Party.Members = append(g.Town.Party.Members, c)
	}
	d.keys("G")
	snap("tavern with party")

	d.enter().keys("ET").keys("NEWHERO").enter()
	snap("create prompt")

	d.keys("Y").enter().enter().keys("D").keys("C") // gnome, evil
	snap("creation stats")

	d.enter() // cursor to I.Q.
	spendPoints(d)
	d.esc()
	snap("creation class")

	checkGolden(t, "screens_town.golden", []byte(out.String()))
}

// The maze and combat views draw wireframe graphics, so only check that they
// render and show the party roster.
func TestMazeAndCombatScreensRender(t *testing.T) {
	seed(21)
	g, d := partyInMaze(t)
	for _, phase := range []string{"maze", "combat"} {
		if phase == "combat" {
			g.Combat = engine.NewCombat(g)
			g.Phase = engine.PhaseCombat
		}
		d.screen.Clear()
		redraw(d.screen.Screen, g)
		text := d.screen.text()
		for _, name := range []string{"FIGHTER A", "PRIEST", "MAGE"} {
			if !strings.Contains(text, name) {
				t.Errorf("%s screen is missing party member %s:\n%s", phase, name, text)
			}
		}
	}
}
