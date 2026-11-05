package app

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// JoinRefusal must give exactly the outcome the real join produces, for
// random rosters and parties in Wizardry I and III.
func TestJoinRefusalMatchesAddMember(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	// The Tavern only reads scenario data, so load each scenario once.
	scenarios := map[int]*engine.GameState{1: newController(t, 1).Game, 3: newController(t, 3).Game}
	for n := 0; n < 3000; n++ {
		num := []int{1, 3}[r.Intn(2)]
		g := engine.New(scenarios[num].Scenario)
		g.Title, g.Phase = nil, engine.PhaseTown
		c := New(g)
		town := c.Game.Town
		size := 4 + r.Intn(8)
		for i := 0; i < size; i++ {
			ch := member(fmt.Sprintf("C%02d", i), engine.Alignment(r.Intn(3)))
			if r.Intn(8) == 0 {
				ch.Status = engine.Lost
			}
			if r.Intn(6) == 0 {
				ch.InMaze = true
			}
			if r.Intn(10) == 0 {
				ch.MazeLevel = 1 + r.Intn(3)
			}
			ch.IsLegacy = r.Intn(2) == 0
			town.Roster.Add(ch)
		}
		// A random party, built directly (it may break alignment on purpose).
		for _, ch := range town.Roster.Characters {
			if r.Intn(3) == 0 && town.Party.Size() < MaxPartySize {
				town.Party.Members = append(town.Party.Members, ch)
			}
		}
		if err := c.EnterTavern(); err != nil {
			t.Fatal(err)
		}
		for _, ch := range town.Roster.Characters {
			want := c.JoinRefusal(ch)
			needs, err := c.AddMember(ch.Name)
			got := ""
			var ref *Refusal
			switch {
			case needs:
				c.CancelTavernPassword()
			case errors.As(err, &ref):
				got = ref.Message
			case err != nil:
				got = err.Error() // ErrPartyFull carries the game's text
			}
			if got != want {
				t.Fatalf("wiz%d %s (status %v inMaze %v level %d legacy %v align %v, party %d %v): JoinRefusal %q, AddMember %q",
					num, ch.Name, ch.Status, ch.InMaze, ch.MazeLevel, ch.IsLegacy, ch.Alignment,
					town.Party.Size(), PartyAlignment(town), want, got)
			}
			if town.InputMode != engine.InputNone {
				t.Fatalf("AddMember left prompt %v open", town.InputMode)
			}
		}
	}
}

func TestPartyMembershipHelpers(t *testing.T) {
	c := atTavern(t, 1, member("A", engine.Good), member("B", engine.Good))
	a, b := c.Game.Town.Roster.Characters[0], c.Game.Town.Roster.Characters[1]
	c.AddMember("A")
	c.SubmitTavernPassword("")
	if !InParty(c.Game.Town, a) || InParty(c.Game.Town, b) || PartySlot(c.Game.Town, a) != 0 || PartySlot(c.Game.Town, b) != -1 {
		t.Error("membership helpers disagree with the party list")
	}
	// Membership comes from the party list, not InMaze: a character out in
	// the dungeon has InMaze set too but is not in the party.
	b.InMaze = true
	if InParty(c.Game.Town, b) {
		t.Error("InMaze alone must not count as party membership")
	}
	if c.JoinRefusal(b) != "** OUT **" || c.JoinRefusal(a) != "A IS ALREADY IN THE PARTY!" {
		t.Errorf("refusals: a=%q b=%q", c.JoinRefusal(a), c.JoinRefusal(b))
	}
}

func TestCloseNamePrompt(t *testing.T) {
	c := atTraining(t, 1)
	keys(t, c, "abc")
	if err := c.CloseNamePrompt(); err != nil || c.Game.Town.InputMode != engine.InputNone || c.Game.Town.InputBuf != "" {
		t.Fatalf("close: %v mode=%v buf=%q", err, c.Game.Town.InputMode, c.Game.Town.InputBuf)
	}
	if err := c.CloseNamePrompt(); !errors.Is(err, ErrWrongState) {
		t.Errorf("close twice: %v", err)
	}
	// ESC at the prompt reaches the same state.
	k := atTraining(t, 1)
	keys(t, k, "abc")
	press(t, k, KeyEscape)
	if a, b := snapshot(t, k.Game.Town), snapshot(t, c.Game.Town); a != b {
		t.Errorf("ESC and CloseNamePrompt differ:\nkey %s\nop  %s", a, b)
	}
}
