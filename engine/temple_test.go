package engine

import (
	"fmt"
	"math/rand"
	"testing"
)

// Characterization of the Temple of Cant's rules (T-0): TempleHeal and
// TempleDonation as they are, before the Temple moves into the controller.
// Each outcome is replayed from the random numbers it should draw, so the
// thresholds, their boundaries (a roll equal to the threshold succeeds) and
// the numbers drawn are all pinned.

// templeReplay is what TempleHeal should do for a patient in status with
// vitality vit, from seed: the status, HP and age after, and the next
// random number after the treatment.
func templeReplay(seed int64, status Status, vit, hp, maxHP, age int) (Status, int, int, int64, int) {
	rand.Seed(seed)
	roll := -1
	switch status {
	case Paralyzed, Stoned:
		age += rand.Intn(52) + 1
		status = OK
	case Dead, Ashed:
		roll = rand.Intn(100)
		threshold := 50 + 3*vit
		if status == Ashed {
			threshold = 40 + 3*vit
		}
		switch {
		case roll > threshold && status == Dead:
			status = Ashed
		case roll > threshold:
			status = Lost
		default:
			if status == Dead {
				hp = 1
			} else {
				hp = maxHP
			}
			status = OK
			age += rand.Intn(52) + 1
		}
	}
	return status, hp, age, rand.Int63(), roll
}

func TestTempleHealReplays(t *testing.T) {
	boundary := map[Status]int{}
	for _, status := range []Status{OK, Asleep, Afraid, Paralyzed, Stoned, Dead, Ashed, Lost} {
		for _, vit := range []int{3, 7, 10, 14, 15, 16, 18} {
			for seed := int64(1); seed <= 400; seed++ {
				c := &Character{Name: "P", Status: status, Vitality: vit, HP: 0, MaxHP: 30, Age: 20 * 52}
				if status == Paralyzed || status == Stoned {
					c.HP = 12
				}
				wantStatus, wantHP, wantAge, wantNext, roll := templeReplay(seed, status, vit, c.HP, c.MaxHP, c.Age)
				threshold := map[Status]int{Dead: 50 + 3*vit, Ashed: 40 + 3*vit}[status]
				if roll == threshold && roll >= 0 {
					boundary[status]++
				}
				rand.Seed(seed)
				msgs, ok := TempleHeal(c)
				where := fmt.Sprintf("%v VIT %d seed %d", status, vit, seed)
				if c.Status != wantStatus || c.HP != wantHP || c.Age != wantAge || rand.Int63() != wantNext {
					t.Fatalf("%s: got %v HP %d age %d", where, c.Status, c.HP, c.Age)
				}
				switch {
				case status != Paralyzed && status != Stoned && status != Dead && status != Ashed:
					if ok || msgs != nil {
						t.Fatalf("%s: treated: %v %q", where, ok, msgs)
					}
				case wantStatus == OK:
					if !ok || len(msgs) != 1 || msgs[0] != "P IS WELL" {
						t.Fatalf("%s: %v %q", where, ok, msgs)
					}
				case wantStatus == Ashed:
					if ok || len(msgs) != 1 || msgs[0] != "P NEEDS KADORTO NOW" {
						t.Fatalf("%s: %v %q", where, ok, msgs)
					}
				case wantStatus == Lost:
					if ok || len(msgs) != 1 || msgs[0] != "P WILL BE BURIED" {
						t.Fatalf("%s: %v %q", where, ok, msgs)
					}
				}
			}
		}
	}
	// The boundary (a roll equal to the threshold) was met, and succeeded.
	if boundary[Dead] == 0 || boundary[Ashed] == 0 {
		t.Errorf("the boundary roll was never drawn: %v", boundary)
	}
}

// The formulas at a glance, from fixed rolls: dead fails only above
// 50+3*VIT, ashes only above 40+3*VIT; a raise from the dead leaves 1 HP,
// from ashes full HP; paralysis and stone always heal, HP unchanged.
func TestTempleHealOutcomes(t *testing.T) {
	find := func(want func(int) bool) int64 {
		for s := int64(1); ; s++ {
			rand.Seed(s)
			if want(rand.Intn(100)) {
				return s
			}
		}
	}
	for _, tc := range []struct {
		status    Status
		vit, roll int
		want      Status
		hp        int
	}{
		{Dead, 10, 80, OK, 1},    // 50+30: the boundary succeeds
		{Dead, 10, 81, Ashed, 0}, // just above fails
		{Ashed, 10, 70, OK, 30},  // 40+30: the boundary succeeds
		{Ashed, 10, 71, Lost, 0}, // just above fails
		{Dead, 18, 99, OK, 1},    // 50+54 > 99: never fails
		{Ashed, 18, 95, Lost, 0}, // 40+54 = 94
	} {
		seed := find(func(r int) bool { return r == tc.roll })
		c := &Character{Name: "P", Status: tc.status, Vitality: tc.vit, MaxHP: 30, Age: 1000}
		rand.Seed(seed)
		TempleHeal(c)
		if c.Status != tc.want || c.HP != tc.hp || (tc.want == OK) != (c.Age > 1000 && c.Age <= 1052) || (tc.want != OK && c.Age != 1000) {
			t.Errorf("%v VIT %d roll %d: %v HP %d age %d", tc.status, tc.vit, tc.roll, c.Status, c.HP, c.Age)
		}
	}
	for _, st := range []Status{Paralyzed, Stoned} {
		c := &Character{Name: "P", Status: st, HP: 7, MaxHP: 30, Age: 1000}
		rand.Seed(1)
		if _, ok := TempleHeal(c); !ok || c.Status != OK || c.HP != 7 || c.Age <= 1000 || c.Age > 1052 {
			t.Errorf("%v: %v HP %d age %d", st, c.Status, c.HP, c.Age)
		}
	}
}

// Donations: the base for the condition times the patient's level; no
// donation for a patient the Temple does not treat.
func TestTempleDonation(t *testing.T) {
	base := map[Status]int{Paralyzed: 100, Stoned: 200, Dead: 250, Ashed: 500}
	for _, st := range []Status{OK, Asleep, Afraid, Paralyzed, Stoned, Dead, Ashed, Lost} {
		for _, level := range []int{1, 2, 5, 13, 40} {
			if got := TempleDonation(st, level); got != base[st]*level {
				t.Errorf("%v level %d: %d", st, level, got)
			}
		}
	}
}
