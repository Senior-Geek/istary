package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// The terminal reading a roster that holds a GUI expedition: it never
// drops or breaks it. The members are out (refused at the Tavern, not
// renamed), the castle can form and save its own party, and every save
// writes the section back unchanged.
func TestTerminalKeepsAGUIExpedition(t *testing.T) {
	engine.SetSaveDir(t.TempDir())
	defer engine.SetSaveDir("")

	gui := newGame(t, 1)
	for _, name := range []string{"OUTA", "OUTB", "HOME"} {
		c := character(name, engine.Human, engine.Fighter, engine.Neutral)
		c.InMaze = name != "HOME" // as the Tavern left them when they set out
		gui.Town.Roster.Add(c)
	}
	gui.Town.Party.Members = gui.Town.Roster.Characters[:2]
	gui.Expedition = &engine.Expedition{Version: engine.ExpeditionVersion, Party: []string{"OUTA", "OUTB"},
		Where: engine.Checkpoint{Phase: "maze", Level: 3, X: 4, Y: 5, Facing: engine.East}}
	if err := gui.Save(); err != nil {
		t.Fatal(err)
	}
	section := func() []byte {
		dir, _ := engine.SaveDir()
		raw, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
		i := bytes.Index(raw, []byte(`"expedition"`))
		if i < 0 {
			return nil
		}
		return raw[i:]
	}
	saved := section()
	if saved == nil {
		t.Fatal("no section written")
	}

	g := newGame(t, 1)
	if err := g.Load(); err != nil {
		t.Fatal(err)
	}
	if g.Town.Party.Size() != 0 || g.Expedition == nil {
		t.Fatalf("terminal loaded party %d expedition %v", g.Town.Party.Size(), g.Expedition)
	}
	d := newDriver(t, g)
	d.keys("G")
	addMember(d, "OUTA", "")
	if g.Town.Party.Size() != 0 || g.Town.Message != "** OUT **" {
		t.Fatalf("an expedition member joined: %q", g.Town.Message)
	}
	addMember(d, "HOME", "")
	if g.Town.Party.Size() != 1 {
		t.Fatalf("HOME did not join: %q", g.Town.Message)
	}
	if err := d.ctrl.RenameCharacter(0, "RENAMED"); err == nil || g.Town.Roster.Characters[0].Name != "OUTA" {
		t.Fatalf("an expedition member was renamed: %v", err)
	}
	if err := d.ctrl.Save(); err != nil { // the terminal saves after every key
		t.Fatal(err)
	}
	if section() == nil {
		t.Fatal("the terminal's save dropped the section")
	}
	again := newGame(t, 1)
	again.Load()
	if !reflect.DeepEqual(again.Expedition, gui.Expedition) || again.Town.Party.Size() != 1 ||
		again.Town.Party.Members[0].Name != "HOME" {
		t.Fatalf("after the terminal saved: expedition %v castle party %d", again.Expedition, again.Town.Party.Size())
	}
}
