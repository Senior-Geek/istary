package app

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// templeController is a scenario-1 controller at the castle with a random
// party (some members not OK, gold of every size) and, outside it, random
// patients: every condition, levels, vitality and age, some out in the
// maze, all from rng.
func templeController(t *testing.T, rng *rand.Rand) *Controller {
	t.Helper()
	c := newController(t, 1)
	g := c.Game
	add := func(name string, st engine.Status) *engine.Character {
		ch := engine.NewCharacter(name, engine.Race(rng.Intn(5)), engine.Class(rng.Intn(8)), engine.Alignment(rng.Intn(3)))
		ch.Status = st
		ch.Level = 1 + rng.Intn(6)
		ch.Vitality = 3 + rng.Intn(16)
		ch.MaxHP = 1 + rng.Intn(80)
		ch.HP = rng.Intn(ch.MaxHP + 1)
		if st >= engine.Dead {
			ch.HP = 0
		}
		ch.Age = (18 + rng.Intn(60)) * 52
		ch.Gold = []int{0, 50, 299, 300, 1000, 2500, 40000}[rng.Intn(7)]
		g.Town.Roster.Add(ch)
		return ch
	}
	for i := 0; i < 1+rng.Intn(6); i++ {
		st := engine.OK
		if rng.Intn(4) == 0 {
			st = engine.Status(rng.Intn(8))
		}
		g.Town.Party.Members = append(g.Town.Party.Members, add(fmt.Sprintf("M%d", i), st))
		g.Town.Party.Members[i].InMaze = true // joined (Pascal ADDPARTY)
	}
	for i := 0; i < 2+rng.Intn(6); i++ {
		ch := add(fmt.Sprintf("P%d", i), engine.Status(rng.Intn(8)))
		if rng.Intn(5) == 0 {
			ch.InMaze, ch.MazeLevel = true, rng.Intn(3)
		}
	}
	return c
}

// termTreat is a treatment as the terminal takes it, from the castle or
// from the Temple's last result: C (or RETURN), the name, RETURN, the
// payer's number, a key at the ritual.
func termTreat(c *Controller, name string, slot int) {
	town := c.Game.Town
	if town.Location == engine.Castle {
		c.HandleKey(Rune('C'))
	}
	for town.TempleStep == engine.TempleResult || town.TempleStep == engine.TempleTithe {
		c.HandleKey(Key{Kind: KeyEnter})
	}
	for _, r := range name {
		c.HandleKey(Rune(r))
	}
	c.HandleKey(Key{Kind: KeyEnter})
	c.HandleKey(Rune('1' + rune(slot)))
	c.HandleKey(Key{Kind: KeyEnter})
}

// choice is a patient the view says can be treated and a payer who can
// pay, chosen by rng; ok false if there is none.
func choice(t *testing.T, c *Controller, rng *rand.Rand) (index, slot int, ok bool) {
	v, err := c.TempleView()
	if err != nil {
		t.Fatal(err)
	}
	type pair struct{ i, s int }
	var pairs []pair
	for _, p := range v.Patients {
		if p.Refusal != "" {
			continue
		}
		for _, py := range v.Payers {
			if py.Refusal == "" && py.Gold >= p.Donation {
				pairs = append(pairs, pair{p.Index, py.Slot})
			}
		}
	}
	if len(pairs) == 0 {
		return 0, 0, false
	}
	p := pairs[rng.Intn(len(pairs))]
	return p.i, p.s, true
}

// Treat is the terminal's keys: the same town, roster, party and random
// numbers after each of two treatments in a row, for 1,500 random rosters.
func TestTreatIsTheKeys(t *testing.T) {
	treated := map[string]int{}
	for s := int64(1); s <= 1500; s++ {
		ops := templeController(t, rand.New(rand.NewSource(s)))
		keys := templeController(t, rand.New(rand.NewSource(s)))
		rng := rand.New(rand.NewSource(s))
		if err := ops.GoTo(engine.Temple); err != nil {
			t.Fatal(err)
		}
		for n := 0; n < 2; n++ {
			index, slot, ok := choice(t, ops, rng)
			if !ok {
				break
			}
			name := ops.Game.Town.Roster.Characters[index].Name
			seed(s)
			r, err := ops.Treat(index, slot, false)
			if err != nil {
				t.Fatalf("seed %d: treat %s by %d: %v", s, name, slot, err)
			}
			next := rand.Int63()
			seed(s)
			termTreat(keys, name, slot)
			if next != rand.Int63() {
				t.Fatalf("seed %d: the random numbers differ", s)
			}
			if a, b := townJSON(t, ops), townJSON(t, keys); a != b {
				t.Fatalf("seed %d treatment %d of %s:\nops  %s\nkeys %s", s, n, name, a, b)
			}
			treated[r.Kind+"/"+r.StatusAfter]++
		}
	}
	// Every outcome was met: cures, raises, ashes, losses.
	for _, k := range []string{"cure/OK", "raise/OK", "raise/ASHED", "ashes/OK", "ashes/LOST"} {
		if treated[k] == 0 {
			t.Errorf("never: %s (%v)", k, treated)
		}
	}
}

// templeFixture is a fixed party and roster at the Temple.
func templeFixture(t *testing.T) *Controller {
	t.Helper()
	c := newController(t, 1)
	g := c.Game
	add := func(name string, st engine.Status, level, gold int) *engine.Character {
		ch := engine.NewCharacter(name, engine.Human, engine.Fighter, engine.Good)
		ch.Status, ch.Level, ch.Gold, ch.Vitality, ch.MaxHP, ch.Age = st, level, gold, 10, 30, 30*52
		ch.HP = 12
		if st >= engine.Dead {
			ch.HP = 0
		}
		g.Town.Roster.Add(ch)
		return ch
	}
	for _, m := range []*engine.Character{
		add("RICH", engine.OK, 1, 5000),
		add("POOR", engine.OK, 1, 100),
		add("GHOST", engine.Dead, 1, 3000), // dead, with gold
	} {
		m.InMaze = true
		g.Town.Party.Members = append(g.Town.Party.Members, m)
	}
	add("PARA", engine.Paralyzed, 3, 0)
	add("STONE", engine.Stoned, 2, 0)
	add("DEAD", engine.Dead, 2, 0)
	add("ASHY", engine.Ashed, 4, 0)
	add("LOSTY", engine.Lost, 1, 0)
	add("SLEEPY", engine.Asleep, 1, 0)
	out := add("BELOW", engine.Dead, 1, 0)
	out.InMaze, out.MazeLevel = true, 3
	add("AWAY", engine.Dead, 1, 0)
	g.Expedition = &engine.Expedition{Party: []string{"AWAY"}}
	c.GoTo(engine.Temple)
	return c
}

func patient(v TempleView, name string) TemplePatient {
	for _, p := range v.Patients {
		if p.Name == name {
			return p
		}
	}
	return TemplePatient{}
}

func TestTempleViewAndRefusals(t *testing.T) {
	c := templeFixture(t)
	idx := func(name string) int {
		for i, ch := range c.Game.Town.Roster.Characters {
			if ch.Name == name {
				return i
			}
		}
		return -1
	}
	before := townJSON(t, c)
	seed(3)
	v, err := c.TempleView()
	next := rand.Int63()
	seed(3)
	if err != nil || next != rand.Int63() || townJSON(t, c) != before {
		t.Fatal("looking changed something")
	}
	var names []string
	for _, p := range v.Patients {
		names = append(names, p.Name)
	}
	// The well, the asleep and the healthy are not patients; the LOST are
	// listed, never treatable.
	if strings.Join(names, ",") != "GHOST,PARA,STONE,DEAD,ASHY,LOSTY,BELOW,AWAY" {
		t.Errorf("patients: %v", names)
	}
	for _, tc := range []struct {
		name, kind, refusal, funding string
		donation                     int
	}{
		{"GHOST", TempleRaise, "", "payer", 250},
		{"PARA", TempleCure, "", "payer", 300},
		{"STONE", TempleCure, "", "payer", 400},
		{"DEAD", TempleRaise, "", "payer", 500},
		{"ASHY", TempleAshes, "", "pool", 2000 * 3 / 3}, // 500 x 4 = 2000: only the party together has it
		{"LOSTY", "", "LOSTY IS LOST", "", 0},
		{"BELOW", TempleRaise, "BELOW is OUT in the maze, not at the castle; only a rescue can bring them back.", "payer", 250},
		{"AWAY", TempleRaise, "AWAY is out on an expedition, not at the castle.", "payer", 250},
	} {
		p := patient(v, tc.name)
		if p.Kind != tc.kind || p.Refusal != tc.refusal || p.Donation != tc.donation {
			t.Errorf("%s: %+v", tc.name, p)
		}
		if tc.name != "ASHY" && p.Funding != tc.funding {
			t.Errorf("%s funding: %q", tc.name, p.Funding)
		}
	}
	if v.Payers[2].Refusal != "GHOST IS NOT OK and cannot tithe." || v.Payers[0].Refusal != "" || v.PartyGold != 8100 {
		t.Errorf("payers: %+v, party gold %d", v.Payers, v.PartyGold)
	}
	// ASHY (2000): RICH has 5000, so a payer has it.
	if p := patient(v, "ASHY"); p.Funding != "payer" {
		t.Errorf("ASHY funding: %q", p.Funding)
	}
	c.Game.Town.Party.Members[0].Gold = 1000
	if v, _ := c.TempleView(); patient(v, "ASHY").Funding != "pool" {
		t.Errorf("ASHY with 1000+100 at hand and 3000 dead: %q", patient(v, "ASHY").Funding)
	}
	c.Game.Town.Party.Members[2].Gold = 0
	if v, _ := c.TempleView(); patient(v, "ASHY").Funding != "short" {
		t.Errorf("ASHY short: %q", patient(v, "ASHY").Funding)
	}
	c.Game.Town.Party.Members[0].Gold = 5000

	// Refusals: decided first, nothing changed, nothing drawn.
	for _, tc := range []struct {
		name string
		slot int
		want string
	}{
		{"LOSTY", 0, "LOSTY IS LOST"},
		{"SLEEPY", 0, "SLEEPY IS OK"},
		{"RICH", 0, "RICH IS OK"},
		{"BELOW", 0, "BELOW is OUT in the maze, not at the castle; only a rescue can bring them back."},
		{"AWAY", 0, "AWAY is out on an expedition, not at the castle."},
		{"PARA", 2, "GHOST IS NOT OK and cannot tithe."},
		{"PARA", 1, "CHEAP APOSTATES! OUT!"},
	} {
		before := townJSON(t, c)
		seed(4)
		_, err := c.Treat(idx(tc.name), tc.slot, false)
		next := rand.Int63()
		seed(4)
		var r *Refusal
		if !errors.As(err, &r) || r.Message != tc.want || townJSON(t, c) != before || next != rand.Int63() {
			t.Errorf("%s paid by %d: %v", tc.name, tc.slot, err)
		}
	}
	if _, err := c.Treat(99, 0, false); !errors.Is(err, ErrNoSuchCharacter) {
		t.Errorf("no such patient: %v", err)
	}
	if _, err := c.Treat(idx("PARA"), 5, false); !errors.Is(err, ErrNoSuchPartyMember) {
		t.Errorf("no such payer: %v", err)
	}
}

// The keys keep the original where Treat is stricter: a patient out in the
// maze treated from the castle, a dead member tithing.
func TestTempleKeysKeepTheOriginal(t *testing.T) {
	c := templeFixture(t)
	below := c.Game.Town.Roster.Characters[9]
	below.Vitality = 18 // never fails
	c.LeaveTemple()
	termTreat(c, "BELOW", 2) // the dead GHOST pays
	if below.Status != engine.OK || c.Game.Town.Party.Members[2].Gold != 2750 {
		t.Errorf("the keys: %v, ghost %d", below.Status, c.Game.Town.Party.Members[2].Gold)
	}
}

func TestTreatReceipts(t *testing.T) {
	find := func(want func(int) bool) int64 {
		for s := int64(1); ; s++ {
			rand.Seed(s)
			if want(rand.Intn(100)) {
				return s
			}
		}
	}
	c := templeFixture(t)
	roster := c.Game.Town.Roster.Characters
	rich := c.Game.Town.Party.Members[0]
	// A cure: assured; HP unchanged; a little older.
	seed(1)
	r, err := c.Treat(3, 0, false) // PARA, 300
	if err != nil || r.Kind != TempleCure || !r.Healed || r.StatusBefore != "PARALYZE" || r.StatusAfter != "OK" || r.Donation != 300 ||
		r.PayerBefore != 5000 || r.PayerAfter != 4700 || r.HPAfter != 12 || r.AgeAfter <= r.AgeBefore || r.AgeAfter > r.AgeBefore+52 ||
		fmt.Sprint(r.Messages) != "[PARA IS WELL]" {
		t.Fatalf("cure: %+v %v", r, err)
	}
	if v, _ := c.TempleView(); v.Step != "result" {
		t.Errorf("after a treatment: %q", v.Step)
	}
	// A raise that fails: ashes, the donation kept, no older.
	seed(find(func(r int) bool { return r > 80 })) // 50 + 3x10
	r, _ = c.Treat(5, 0, false)                    // DEAD, 500
	if r.Healed || r.StatusAfter != "ASHED" || rich.Gold != 4200 || r.AgeAfter != r.AgeBefore || r.HPAfter != 0 ||
		fmt.Sprint(r.Messages) != "[DEAD NEEDS KADORTO NOW]" {
		t.Errorf("a failed raise: %+v", r)
	}
	// From ashes, a success: full HP.
	seed(find(func(r int) bool { return r <= 70 })) // 40 + 3x10
	r, _ = c.Treat(5, 0, false)                     // DEAD, now ashes: 500 x 2
	if !r.Healed || r.Kind != TempleAshes || r.Donation != 1000 || r.HPAfter != 30 || rich.Gold != 3200 {
		t.Errorf("ashes raised: %+v", r)
	}
	// From ashes, a failure: LOST, kept on the roster, never treated again.
	seed(find(func(r int) bool { return r > 70 }))
	r, _ = c.Treat(6, 0, false) // ASHY, 2000
	if r.Healed || r.StatusAfter != "LOST" || rich.Gold != 1200 || roster[6].Status != engine.Lost {
		t.Errorf("ashes lost: %+v", r)
	}
	if v, _ := c.TempleView(); patient(v, "ASHY").Refusal != "ASHY IS LOST" || patient(v, "ASHY").Kind != "" {
		t.Errorf("the LOST: %+v", patient(v, "ASHY"))
	}
	// A dead raise: 1 HP. Leaving and coming back.
	seed(find(func(r int) bool { return r <= 80 }))
	if r, _ = c.Treat(2, 0, false); !r.Healed || r.HPAfter != 1 || r.StatusAfter != "OK" { // GHOST, a party member
		t.Errorf("raised: %+v", r)
	}
	if err := c.LeaveTemple(); err != nil || c.Game.Town.Location != engine.Castle {
		t.Errorf("leave: %v", err)
	}
	if err := c.LeaveTemple(); !errors.Is(err, ErrWrongState) {
		t.Errorf("leave twice: %v", err)
	}
	if _, err := c.Treat(4, 0, false); !errors.Is(err, ErrWrongState) {
		t.Errorf("treat away from the Temple: %v", err)
	}
}

// Pool Gold at the Temple brings the party's gold to a payer (p.33).
func TestTemplePoolThenTreat(t *testing.T) {
	c := templeFixture(t)
	c.Game.Town.Party.Members[0].Gold = 1000
	if v, _ := c.TempleView(); patient(v, "ASHY").Funding != "pool" {
		t.Fatalf("ASHY: %+v", patient(v, "ASHY"))
	}
	if _, err := c.PoolGold(0); err != nil {
		t.Fatal(err)
	}
	if v, _ := c.TempleView(); patient(v, "ASHY").Funding != "payer" || v.Payers[0].Gold != 4100 {
		t.Errorf("after pooling: %+v", v.Payers)
	}
	if _, err := c.Treat(6, 0, false); err != nil {
		t.Errorf("treat after pooling: %v", err)
	}
}

// When the payer is short and the party together is not, the party's gold
// goes into the fund (p.33): Treat with party is Pool Gold to the payer and
// then the keys, the same town and random numbers, for random rosters.
func TestTreatPartyContributes(t *testing.T) {
	met := map[string]int{}
	for s := int64(1); s <= 1500; s++ {
		ops := templeController(t, rand.New(rand.NewSource(s)))
		keys := templeController(t, rand.New(rand.NewSource(s)))
		ops.GoTo(engine.Temple)
		v, _ := ops.TempleView()
		var index, slot = -1, -1
		for _, p := range v.Patients {
			for _, py := range v.Payers {
				if q, _ := ops.TempleQuote(p.Index, py.Slot); q.Mode == "party" && index < 0 {
					index, slot = p.Index, py.Slot
				}
			}
		}
		if index < 0 {
			continue
		}
		q, _ := ops.TempleQuote(index, slot)
		shares := 0
		for _, sh := range q.Shares {
			shares += sh.Gold
		}
		if q.PayerGold >= q.Donation || q.PartyGold < q.Donation || q.PayerGold+shares != q.PartyGold {
			t.Fatalf("seed %d: quote %+v", s, q)
		}
		// Without the party: refused, nothing changed, nothing drawn.
		before := townJSON(t, ops)
		seed(s)
		_, err := ops.Treat(index, slot, false)
		next := rand.Int63()
		seed(s)
		if r := new(Refusal); !errors.As(err, &r) || r.Message != "CHEAP APOSTATES! OUT!" || townJSON(t, ops) != before || next != rand.Int63() {
			t.Fatalf("seed %d: a short payer without the party: %v", s, err)
		}
		seed(s)
		r, err := ops.Treat(index, slot, true)
		if err != nil {
			t.Fatalf("seed %d: %v", s, err)
		}
		next = rand.Int63()
		name := ops.Game.Town.Roster.Characters[index].Name
		keys.HandleKey(Rune('C')) // the keys: pool to the payer, then treat
		if _, err := keys.PoolGold(slot); err != nil {
			t.Fatal(err)
		}
		seed(s)
		termTreat(keys, name, slot)
		if next != rand.Int63() || townJSON(t, ops) != townJSON(t, keys) {
			t.Fatalf("seed %d: the party's contribution is not Pool Gold and the keys", s)
		}
		if r.Party == nil || r.Party.Total != q.PartyGold || r.PayerAfter != q.PartyGold-q.Donation || len(r.Shares) != len(q.Shares) {
			t.Fatalf("seed %d: receipt %+v", s, r)
		}
		met[r.Kind+"/"+r.StatusAfter]++
	}
	if len(met) < 3 {
		t.Errorf("too few outcomes with the party's gold: %v", met)
	}
}

func TestTempleQuoteModes(t *testing.T) {
	c := templeFixture(t) // RICH 5000, POOR 100, GHOST (dead) 3000
	idx := func(name string) int {
		for i, ch := range c.Game.Town.Roster.Characters {
			if ch.Name == name {
				return i
			}
		}
		return -1
	}
	m := c.Game.Town.Party.Members
	before := townJSON(t, c)
	seed(2)
	q, err := c.TempleQuote(idx("ASHY"), 1) // 2000 by POOR (100): the party
	next := rand.Int63()
	seed(2)
	if err != nil || next != rand.Int63() || townJSON(t, c) != before {
		t.Fatal("quoting changed something")
	}
	want := TempleQuote{Patient: "ASHY", Donation: 2000, Payer: "POOR", PayerGold: 100, PartyGold: 8100, Mode: "party",
		Shares: []TempleShare{{0, "RICH", 5000}, {2, "GHOST", 3000}}}
	if fmt.Sprint(q) != fmt.Sprint(want) {
		t.Errorf("party:\n%+v\n%+v", q, want)
	}
	if q, _ := c.TempleQuote(idx("ASHY"), 0); q.Mode != "payer" || q.Shares != nil {
		t.Errorf("payer: %+v", q)
	}
	if q, _ := c.TempleQuote(idx("ASHY"), 2); q.Refusal != "GHOST IS NOT OK and cannot tithe." || q.Mode != "" {
		t.Errorf("a dead payer: %+v", q)
	}
	if q, _ := c.TempleQuote(idx("LOSTY"), 0); q.Refusal != "LOSTY IS LOST" {
		t.Errorf("the LOST: %+v", q)
	}
	// The dead GHOST's gold counts: without it the party is short.
	m[0].Gold, m[2].Gold = 1000, 900
	if q, _ := c.TempleQuote(idx("ASHY"), 1); q.Mode != "party" || q.PartyGold != 2000 {
		t.Errorf("exactly enough with the dead's gold: %+v", q)
	}
	m[2].Gold = 0
	before = townJSON(t, c)
	q, _ = c.TempleQuote(idx("ASHY"), 1)
	_, err = c.Treat(idx("ASHY"), 1, true)
	if q.Refusal != "CHEAP APOSTATES! OUT!" || !errors.As(err, new(*Refusal)) || townJSON(t, c) != before {
		t.Errorf("the party short: %+v %v", q, err)
	}
	// A payer who has the donation pays alone, party or not.
	m[0].Gold = 5000
	r, err := c.Treat(idx("PARA"), 0, true)
	if err != nil || r.Party != nil || r.PayerAfter != 4700 || m[1].Gold != 100 {
		t.Errorf("paid alone: %+v %v", r, err)
	}
}

// The party's gold pays for a raise that fails as for one that succeeds:
// the donation is kept, the rest stays with the payer.
func TestTreatPartyReceipts(t *testing.T) {
	find := func(want func(int) bool) int64 {
		for s := int64(1); ; s++ {
			rand.Seed(s)
			if want(rand.Intn(100)) {
				return s
			}
		}
	}
	c := templeFixture(t)
	m := c.Game.Town.Party.Members
	seed(find(func(r int) bool { return r > 80 })) // DEAD (VIT 10): fails into ashes
	r, err := c.Treat(5, 1, true)                  // POOR (100) pays 500 with the party
	if err != nil || r.Healed || r.StatusAfter != "ASHED" || r.PayerBefore != 100 || r.PayerAfter != 7600 ||
		m[0].Gold != 0 || m[2].Gold != 0 || fmt.Sprint(r.Shares) != "[{0 RICH 5000} {2 GHOST 3000}]" {
		t.Errorf("a failed raise with the party: %+v %v", r, err)
	}
	seed(find(func(r int) bool { return r <= 70 }))
	if r, _ = c.Treat(5, 1, true); !r.Healed || r.Party != nil || r.PayerAfter != 6600 {
		t.Errorf("ashes raised by the payer alone now: %+v", r)
	}
}
