package app

import (
	"errors"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

func TestClearRoster(t *testing.T) {
	c := atTavern(t, 1, member("A", engine.Good), member("B", engine.Good))
	c.AddMember("A")
	c.SubmitTavernPassword("")
	c.BeginAddMember() // leave a prompt open
	if err := c.ClearRoster(); err != nil {
		t.Fatal(err)
	}
	town := c.Game.Town
	if len(town.Roster.Characters) != 0 || town.Party.Size() != 0 || town.InputMode != engine.InputNone {
		t.Errorf("after clear: roster=%d party=%d mode=%v", len(town.Roster.Characters), town.Party.Size(), town.InputMode)
	}
	beginCreation(c, "X")
	if err := c.ClearRoster(); !errors.Is(err, ErrWrongState) {
		t.Errorf("clear during creation: %v", err)
	}
}

func TestRenameCharacter(t *testing.T) {
	c := newController(t, 1)
	c.Game.Town.Roster.Add(member("ALPHA", engine.Good))
	c.Game.Town.Roster.Add(member("BETA", engine.Good))
	refused := func(err error) string {
		var r *Refusal
		if errors.As(err, &r) {
			return r.Message
		}
		return ""
	}

	if err := c.RenameCharacter(0, "  gamma "); err != nil || c.Game.Town.Roster.Characters[0].Name != "GAMMA" {
		t.Fatalf("rename: %v name=%q", err, c.Game.Town.Roster.Characters[0].Name)
	}
	if err := c.RenameCharacter(0, "beta"); refused(err) != "NAME ALREADY IN USE" {
		t.Errorf("duplicate: %v", err)
	}
	if err := c.RenameCharacter(0, "gamma"); refused(err) != "NAME ALREADY IN USE" {
		t.Errorf("same name (Utilities rule counts itself): %v", err)
	}
	if err := c.RenameCharacter(1, "   "); refused(err) != "NAME CANNOT BE EMPTY" {
		t.Errorf("empty: %v", err)
	}
	if err := c.RenameCharacter(5, "X"); !errors.Is(err, ErrNoSuchCharacter) {
		t.Errorf("bad index: %v", err)
	}
	beginCreation(c, "X")
	if err := c.RenameCharacter(0, "DELTA"); !errors.Is(err, ErrWrongState) {
		t.Errorf("during creation: %v", err)
	}
}

func TestValidateName(t *testing.T) {
	for name, want := range map[string]error{
		"Sam Gamgee":       nil,
		"o'brien-2":        nil,
		"ABCDEFGHIJKLMNO":  nil,
		"   ":              ErrEmptyName,
		"ABCDEFGHIJKLMNOP": ErrInvalidName, // 16 characters
		"Bad!Name":         ErrInvalidName,
		"Ünïcode":          ErrInvalidName,
	} {
		if err := ValidateName(name); !errors.Is(err, want) && !(err == nil && want == nil) {
			t.Errorf("ValidateName(%q) = %v, want %v", name, err, want)
		}
	}
}
