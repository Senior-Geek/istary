package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// innController is a scenario-1 controller at the castle with a random
// party: classes, statuses, hit points, gold, experience, age, attributes
// and spent spells, all from rng.
func innController(t *testing.T, rng *rand.Rand) *Controller {
	t.Helper()
	c := newController(t, 1)
	g := c.Game
	for i := 0; i < 1+rng.Intn(6); i++ {
		ch := engine.NewCharacter(fmt.Sprintf("M%d", i), engine.Race(rng.Intn(5)), engine.Class(rng.Intn(8)), engine.Alignment(rng.Intn(3)))
		if rng.Intn(5) == 0 {
			ch.Status = engine.Status(1 + rng.Intn(7))
		}
		ch.MaxHP = 1 + rng.Intn(120)
		ch.HP = rng.Intn(ch.MaxHP + 1)
		ch.Gold = []int{0, 9, 10, 49, 120, 777, 5000, 100000}[rng.Intn(8)]
		ch.Level = 1 + rng.Intn(4)
		ch.MaxLevAC = ch.Level
		ch.XP = []int{0, 999, 1000, 1724, 3000, 9000}[rng.Intn(6)]
		ch.Age = (18 + rng.Intn(130)) * 52
		ch.Strength, ch.IQ, ch.Piety = 3+rng.Intn(16), 3+rng.Intn(16), 3+rng.Intn(16)
		ch.Vitality, ch.Agility, ch.Luck = 3+rng.Intn(16), 3+rng.Intn(16), 3+rng.Intn(16)
		for l := 0; l < 7; l++ {
			ch.MaxMageSpells[l], ch.MaxPriestSpells[l] = rng.Intn(4), rng.Intn(4)
			ch.MageSpells[l], ch.PriestSpells[l] = rng.Intn(ch.MaxMageSpells[l]+1), rng.Intn(ch.MaxPriestSpells[l]+1)
		}
		g.Town.Roster.Add(ch)
		g.Town.Party.Members = append(g.Town.Party.Members, ch)
	}
	return c
}

func innTwins(t *testing.T, seed int64) (*Controller, *Controller) {
	return innController(t, rand.New(rand.NewSource(seed))), innController(t, rand.New(rand.NewSource(seed)))
}

// townJSON is everything the Inn can change: the town, party and roster
// included.
func townJSON(t *testing.T, c *Controller) string {
	t.Helper()
	raw, err := json.Marshal(c.Game.Town)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// okSlot is a party member who may stay, chosen by rng, or -1.
func okSlot(c *Controller, rng *rand.Rand) int {
	var ok []int
	for i, m := range c.Game.Town.Party.Members {
		if m.Status == engine.OK {
			ok = append(ok, i)
		}
	}
	if len(ok) == 0 {
		return -1
	}
	return ok[rng.Intn(len(ok))]
}

// termStay is a stay as the terminal takes it: the room's letter, then the
// timer's ticks until the level check is shown.
func termStay(c *Controller, room engine.InnRoom) {
	c.HandleKey(Rune(innRooms[room].key))
	for c.Game.Town.InnStep == engine.InnHealing {
		c.InnTick()
	}
}

// Stay is the terminal's keys and ticks: the same town, party and random
// numbers afterwards, for two stays in a row with random rooms.
func TestInnStayIsTheKeys(t *testing.T) {
	for s := int64(1); s <= 1500; s++ {
		ops, keys := innTwins(t, s)
		rng := rand.New(rand.NewSource(s))
		slot := okSlot(ops, rng)
		if slot < 0 {
			continue
		}
		if err := ops.GoTo(engine.Inn); err != nil {
			t.Fatal(err)
		}
		if err := ops.InnEnter(slot); err != nil {
			t.Fatalf("seed %d: enter %d: %v", s, slot, err)
		}
		keys.HandleKey(Rune('A'))
		keys.HandleKey(Rune('1' + rune(slot)))
		for stay := 0; stay < 2; stay++ {
			room := engine.InnRoom(rng.Intn(5))
			seed(s)
			if _, err := ops.Stay(room); err != nil {
				t.Fatalf("seed %d: stay: %v", s, err)
			}
			next := rand.Int63()
			seed(s)
			if stay > 0 {
				keys.HandleKey(Key{Kind: KeyEnter})
			}
			termStay(keys, room)
			if next != rand.Int63() {
				t.Fatalf("seed %d stay %d: the random numbers differ", s, stay)
			}
			if a, b := townJSON(t, ops), townJSON(t, keys); a != b {
				t.Fatalf("seed %d stay %d in room %d:\nops  %s\nkeys %s", s, stay, room, a, b)
			}
		}
	}
}

// A quote is what the stay then does, and what InnView shows for the room.
func TestInnQuoteIsTheStay(t *testing.T) {
	for s := int64(1); s <= 600; s++ {
		for room := engine.Stables; room <= engine.Royal; room++ {
			c := innController(t, rand.New(rand.NewSource(s)))
			slot := okSlot(c, rand.New(rand.NewSource(s)))
			if slot < 0 {
				break
			}
			c.GoTo(engine.Inn)
			c.InnEnter(slot)
			q, err := c.Quote(room)
			if err != nil {
				t.Fatal(err)
			}
			v, _ := c.InnView()
			if v.Rooms[room].UntilHealed != q {
				t.Fatalf("seed %d room %d: view %+v, quote %+v", s, room, v.Rooms[room].UntilHealed, q)
			}
			st, err := c.Stay(room)
			if err != nil {
				t.Fatal(err)
			}
			// The level check after the weeks changes only the maximum,
			// unless the guest dies of old age (HP 0).
			if st.Weeks != q.Weeks || st.Charged != q.Cost || (st.HPAfter != q.HP && !st.Died) {
				t.Fatalf("seed %d room %d: quote %+v, stay %d weeks %d GP HP %d", s, room, q, st.Weeks, st.Charged, st.HPAfter)
			}
		}
	}
}

// Looking at the Inn changes nothing and draws no random number.
func TestInnViewIsReadOnly(t *testing.T) {
	for s := int64(1); s <= 200; s++ {
		c := innController(t, rand.New(rand.NewSource(s)))
		c.GoTo(engine.Inn)
		if slot := okSlot(c, rand.New(rand.NewSource(s))); slot >= 0 {
			c.InnEnter(slot)
		}
		before := townJSON(t, c)
		seed(s)
		c.InnView()
		for room := engine.Stables; room <= engine.Royal; room++ {
			c.Quote(room)
		}
		next := rand.Int63()
		seed(s)
		if next != rand.Int63() || townJSON(t, c) != before {
			t.Fatalf("seed %d: looking changed something", s)
		}
	}
}

func innFixture(t *testing.T) *Controller {
	t.Helper()
	c := newController(t, 1)
	for _, ch := range []*engine.Character{
		engine.NewCharacter("ARNE", engine.Dwarf, engine.Fighter, engine.Good),
		engine.NewCharacter("PRIEST", engine.Human, engine.Priest, engine.Good),
		engine.NewCharacter("MAGE", engine.Elf, engine.Mage, engine.Good),
	} {
		ch.HP, ch.MaxHP, ch.Age = 3, 20, 20*52
		ch.Strength, ch.IQ, ch.Piety, ch.Vitality, ch.Agility, ch.Luck = 15, 15, 15, 12, 10, 10
		c.Game.Town.Roster.Add(ch)
		c.Game.Town.Party.Members = append(c.Game.Town.Party.Members, ch)
	}
	c.Game.Town.Party.Members[1].Status = engine.Dead
	return c
}

func TestInnOperations(t *testing.T) {
	c := innFixture(t)
	arne, mage := c.Game.Town.Party.Members[0], c.Game.Town.Party.Members[2]
	if _, err := c.Stay(engine.Cot); !errors.Is(err, ErrWrongState) {
		t.Errorf("a stay away from the Inn: %v", err)
	}
	if _, err := c.InnView(); !errors.Is(err, ErrWrongState) {
		t.Errorf("the view away from the Inn: %v", err)
	}
	c.GoTo(engine.Inn)
	v, _ := c.InnView()
	if v.Step != "who" || v.Guest != -1 || len(v.Guests) != 3 || v.Guests[1].Refusal != "PRIEST IS NOT OK" || len(v.Rooms) != 0 {
		t.Fatalf("who will stay: %+v", v)
	}
	var r *Refusal
	if err := c.InnEnter(1); !errors.As(err, &r) || r.Message != "PRIEST IS NOT OK" || c.Game.Town.InnStep != engine.InnWho {
		t.Errorf("the dead priest: %v", err)
	}
	if err := c.InnEnter(5); !errors.Is(err, ErrNoSuchPartyMember) {
		t.Errorf("no such member: %v", err)
	}
	if _, err := c.Stay(engine.Cot); !errors.Is(err, ErrWrongState) {
		t.Errorf("a stay with no guest: %v", err)
	}
	if err := c.InnEnter(0); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Stay(engine.InnRoom(5)); !errors.Is(err, ErrInvalidChoice) {
		t.Errorf("room 5: %v", err)
	}
	v, _ = c.InnView()
	want := []string{"THE STABLES", "COTS", "ECONOMY ROOMS", "MERCHANT SUITES", "ROYAL SUITES"}
	for i, rv := range v.Rooms {
		if rv.Name != want[i] || rv.HPPerWeek != engine.InnRoomQuality[i] || rv.FeePerWeek != engine.InnRoomCosts[i] || rv.Free != (i == 0) {
			t.Errorf("room %d: %+v", i, rv)
		}
	}
	// No gold: a paid room heals nothing, as the stables.
	if q := v.Rooms[engine.Economy].UntilHealed; q.Weeks != 0 || q.Needed != 6 || q.Affordable != 0 || q.HP != 3 {
		t.Errorf("economy with no gold: %+v", q)
	}
	arne.Gold, arne.XP = 1000, 1000
	if q, _ := c.Quote(engine.Economy); q != (InnQuote{Weeks: 6, Cost: 300, HP: 20, Healed: true, Needed: 6, Affordable: 20}) {
		t.Errorf("economy: %+v", q)
	}
	arne.Gold = 120
	if q, _ := c.Quote(engine.Economy); q != (InnQuote{Weeks: 2, Cost: 100, HP: 9, Needed: 6, Affordable: 2}) {
		t.Errorf("economy, short of gold: %+v", q)
	}
	arne.Gold = 1000
	seed(1)
	st, err := c.Stay(engine.Economy)
	if err != nil {
		t.Fatal(err)
	}
	if st.Weeks != 6 || st.Charged != 300 || st.HPAfter != 20 || st.LevelAfter != 2 || st.Messages[0] != "YOU MADE A LEVEL!" || st.XPNeeded != 0 || st.RoomName != "ECONOMY ROOMS" {
		t.Errorf("the stay: %+v", st)
	}
	if v, _ := c.InnView(); v.Step != "levelUp" || v.Guest != 0 || v.Messages[0] != "YOU MADE A LEVEL!" {
		t.Errorf("after the stay: %+v", v)
	}
	// Again, from the result: the stables, and experience still needed.
	st, err = c.Stay(engine.Stables)
	if err != nil || st.Weeks != 0 || st.Charged != 0 || st.LevelAfter != 2 || st.XPNeeded != 724 {
		t.Errorf("the stables after: %+v %v", st, err)
	}
	// A mage's spells come back and a new level teaches spells.
	if err := c.InnLeaveGuest(); err != nil || c.Game.Town.InnStep != engine.InnWho {
		t.Fatalf("leave the guest: %v", err)
	}
	mage.MaxMageSpells, mage.MageSpells, mage.XP = [7]int{2}, [7]int{}, 1100
	c.InnEnter(2)
	seed(2)
	st, err = c.Stay(engine.Stables)
	if err != nil || !st.Spells || st.MageAfter[0] == 0 || st.LevelAfter != 2 {
		t.Errorf("the mage: %+v %v", st, err)
	}
	if err := c.LeaveInn(); err != nil || c.Game.Town.Location != engine.Castle {
		t.Errorf("leave the Inn: %v", err)
	}
	if err := c.LeaveInn(); !errors.Is(err, ErrWrongState) {
		t.Errorf("leave twice: %v", err)
	}
}

func TestInnDeathOfOldAge(t *testing.T) {
	c := innFixture(t)
	old := c.Game.Town.Party.Members[0]
	old.Age, old.Vitality, old.XP, old.HP, old.MaxHP = 200*52, 3, 1000, 10, 10
	c.GoTo(engine.Inn)
	c.InnEnter(0)
	for s := int64(1); ; s++ {
		snap := *old
		seed(s)
		st, err := c.Stay(engine.Stables)
		if err != nil {
			t.Fatal(err)
		}
		if st.Died {
			if st.Status != "LOST" || old.Status != engine.Lost || st.Messages[len(st.Messages)-1] != "** YOU HAVE DIED OF OLD AGE **" {
				t.Errorf("died: %+v", st)
			}
			return
		}
		if s > 50 {
			t.Fatal("no seed brought death of old age")
		}
		*old = snap
		c.press(Key{Kind: KeyEnter})
	}
}
