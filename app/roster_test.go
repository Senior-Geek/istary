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
