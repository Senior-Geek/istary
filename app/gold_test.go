package app

import (
	"errors"
	"math/rand"
	"slices"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

func partyGold(c *Controller) (gold []int, total int) {
	for _, m := range c.Game.Town.Party.Members {
		gold = append(gold, m.Gold)
		total += m.Gold
	}
	return gold, total
}

// Pooling and sharing keep the party's total, draw no random number, and
// do what the guide says, for random parties in every place they are open.
func TestPoolAndDivvyGold(t *testing.T) {
	for s := int64(1); s <= 400; s++ {
		rng := rand.New(rand.NewSource(s))
		for _, place := range []engine.TownLocation{engine.Tavern, engine.Trading, engine.Inn} {
			c := innController(t, rand.New(rand.NewSource(s)))
			c.Game.Town.Location = place
			before, total := partyGold(c)
			slot := rng.Intn(len(before))
			seed(s)
			m, err := c.PoolGold(slot)
			next := rand.Int63()
			seed(s)
			if err != nil || next != rand.Int63() {
				t.Fatalf("seed %d at %v: %v", s, place, err)
			}
			after, sum := partyGold(c)
			for i, g := range after {
				if (i == slot && g != total) || (i != slot && g != 0) {
					t.Fatalf("seed %d: pooled to %d: %v -> %v", s, slot, before, after)
				}
			}
			if sum != total || !slices.Equal(m.Before, before) || !slices.Equal(m.After, after) || m.Total != total || m.To != slot {
				t.Fatalf("seed %d: %+v", s, m)
			}
			if place != engine.Tavern {
				if _, err := c.DivvyGold(); !errors.Is(err, ErrWrongState) {
					t.Fatalf("divvy at %v: %v", place, err)
				}
				continue
			}
			c = innController(t, rand.New(rand.NewSource(s)))
			c.Game.Town.Location = engine.Tavern
			m, err = c.DivvyGold()
			after, sum = partyGold(c)
			n := len(after)
			for i, g := range after {
				want := total / n
				if i < total%n {
					want++
				}
				if g != want {
					t.Fatalf("seed %d: divvy %v -> %v", s, before, after)
				}
			}
			if err != nil || sum != total || m.To != -1 || !slices.Equal(m.After, after) {
				t.Fatalf("seed %d: divvy %+v %v", s, m, err)
			}
		}
	}
}

func TestGoldRefusals(t *testing.T) {
	c := innFixture(t)
	if _, err := c.PoolGold(0); !errors.Is(err, ErrWrongState) {
		t.Errorf("pool at the castle: %v", err)
	}
	if _, err := c.DivvyGold(); !errors.Is(err, ErrWrongState) {
		t.Errorf("divvy at the castle: %v", err)
	}
	c.GoTo(engine.Inn)
	if _, err := c.PoolGold(3); !errors.Is(err, ErrNoSuchPartyMember) {
		t.Errorf("pool to no one: %v", err)
	}
	c.LeaveInn()
	c.GoTo(engine.Tavern)
	c.Game.Town.InputMode = engine.InputAddMember
	if _, err := c.DivvyGold(); !errors.Is(err, ErrWrongState) {
		t.Errorf("divvy with a prompt open: %v", err)
	}
	c.Game.Town.InputMode = engine.InputNone
	c.Game.Town.Party.Members = nil
	if _, err := c.DivvyGold(); !errors.Is(err, ErrWrongState) {
		t.Errorf("divvy with no party: %v", err)
	}
}
