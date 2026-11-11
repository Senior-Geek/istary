package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// The terminal's Inspect screen (R-2): everyone found in the party's area,
// numbered; the room left; P)ICK UP's "GET WHO (0=EXIT) >", one character a
// key; the message of the last pick; YOU HAVE 6 for a full party.
func TestRescueScreenSnapshots(t *testing.T) {
	seed(43)
	g := newGame(t, 1)
	d := newDriverWithScreen(t, g)
	s := d.screen
	var out strings.Builder
	snap := func(name string) {
		s.Clear()
		redraw(s.Screen, g)
		out.WriteString("=== " + name + "\n" + s.text() + "\n")
	}
	for _, c := range adventurers()[:4] {
		c.InMaze = true
		g.Town.Roster.Add(c)
		g.Town.Party.Members = append(g.Town.Party.Members, c)
	}
	here, _ := engine.PlaceOf(len(g.Scenario.Mazes.Levels), 0, 0, 0)
	for i, st := range []engine.Status{engine.Dead, engine.Paralyzed, engine.Ashed} {
		c := character(fmt.Sprintf("FALLEN%d", i+1), engine.Human, engine.Fighter, engine.Neutral)
		c.Status, c.Level, c.InMaze = st, 3+i, true
		p := here
		c.Stranded = &p
		g.Town.Roster.Add(c)
	}
	if err := d.ctrl.EnterMaze(); err != nil {
		t.Fatal(err)
	}
	d.ctrl.LeaveCamp()
	d.ctrl.Answer(false)
	g.MazeMessage, g.MazeMessage2 = "", ""

	d.keys("I")
	snap("inspect, three found")
	d.keys("P")
	snap("get who")
	d.keys("9")
	snap("not on the list")
	d.keys("2")
	snap("FALLEN2 picked up")
	d.keys("1")
	snap("party full: the prompt closes")
	d.keys("P")
	snap("you have 6")
	d.keys("L")
	if g.MazeInspecting {
		t.Fatal("still inspecting")
	}
	checkGolden(t, "screens_rescue.golden", []byte(out.String()))
}
