package app

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// atStats returns a controller at the bonus-point step with the given race,
// alignment and stats.
func atStats(t *testing.T, race engine.Race, align engine.Alignment, stats [6]int, points int) *Controller {
	t.Helper()
	c := newController(t, 1)
	beginCreation(c, "GUIDE")
	cs := c.Creation()
	cs.Race, cs.Alignment, cs.Stats, cs.BonusPoints = race, align, stats, points
	cs.Step = engine.StepStats
	return c
}

func TestClassGuideNeedsStats(t *testing.T) {
	c := newController(t, 1)
	if _, err := c.ClassGuide(); !errors.Is(err, ErrWrongState) {
		t.Errorf("no creation: %v", err)
	}
	beginCreation(c, "X")
	if _, err := c.ClassGuide(); !errors.Is(err, ErrWrongState) {
		t.Errorf("before alignment: %v", err)
	}
}

// The alignments the guide derives from the engine match the documented
// restrictions.
func TestClassGuideAlignments(t *testing.T) {
	want := map[engine.Class][]engine.Alignment{
		engine.Fighter: {engine.Good, engine.Neutral, engine.Evil},
		engine.Mage:    {engine.Good, engine.Neutral, engine.Evil},
		engine.Priest:  {engine.Good, engine.Evil},
		engine.Thief:   {engine.Neutral, engine.Evil},
		engine.Bishop:  {engine.Good, engine.Evil},
		engine.Samurai: {engine.Good, engine.Neutral},
		engine.Lord:    {engine.Good},
		engine.Ninja:   {engine.Evil},
	}
	c := atStats(t, engine.Human, engine.Good, engine.RaceBaseStats[engine.Human], 0)
	guide, _ := c.ClassGuide()
	for _, st := range guide {
		if len(st.Alignments) != len(want[st.Class]) {
			t.Errorf("%v: alignments %v, want %v", st.Class, st.Alignments, want[st.Class])
			continue
		}
		for i := range st.Alignments {
			if st.Alignments[i] != want[st.Class][i] {
				t.Errorf("%v: alignments %v, want %v", st.Class, st.Alignments, want[st.Class])
			}
		}
		if st.Minimums != engine.ClassRequirements[st.Class] {
			t.Errorf("%v: minimums %v", st.Class, st.Minimums)
		}
	}
}

func TestClassGuideExample(t *testing.T) {
	// Human good with base stats 8/8/5/8/8/9 and 9 points to spend.
	c := atStats(t, engine.Human, engine.Good, [6]int{8, 8, 5, 8, 8, 9}, 9)
	guide, _ := c.ClassGuide()
	byClass := func(cl engine.Class) ClassStatus { return guide[cl] }

	f := byClass(engine.Fighter) // STR 11: needs 3
	if f.Eligible || f.PointsNeeded != 3 || f.Shortfall[0] != 3 || !f.ReachableWithRemaining {
		t.Errorf("fighter: %+v", f)
	}
	th := byClass(engine.Thief) // not good
	if th.AlignmentOK || th.ReachableWithRemaining || th.ReachableByMoving {
		t.Errorf("thief should be blocked by alignment: %+v", th)
	}
	l := byClass(engine.Lord) // needs 7+4+7+7+6+6 = 37 points
	if l.PointsNeeded != 37 || l.ReachableWithRemaining || l.ReachableByMoving {
		t.Errorf("lord: %+v", l)
	}

	// Spending a point updates the guide immediately.
	c.SelectStat(0)
	c.AddStatPoint()
	guide, _ = c.ClassGuide()
	if g := guide[engine.Fighter]; g.PointsNeeded != 2 || g.Shortfall[0] != 2 {
		t.Errorf("after +1 STR: %+v", g)
	}
}

// Points already spent elsewhere can be moved to reach a class.
func TestClassGuideReachableByMoving(t *testing.T) {
	// Elf base 7/10/10/6/9/6, all 8 points already in LUCK, none left.
	c := atStats(t, engine.Elf, engine.Neutral, [6]int{7, 10, 10, 6, 9, 14}, 0)
	guide, _ := c.ClassGuide()
	m := guide[engine.Mage] // IQ 11: short 1
	if m.Eligible || m.ReachableWithRemaining || !m.ReachableByMoving {
		t.Errorf("mage: %+v", m)
	}
}

func TestClassGuideDoesNotChangeState(t *testing.T) {
	c := atStats(t, engine.Dwarf, engine.Evil, [6]int{12, 7, 10, 11, 5, 6}, 6)
	before := snapshot(t, c.Game.Town)
	c.ClassGuide()
	if after := snapshot(t, c.Game.Town); after != before {
		t.Errorf("ClassGuide changed state:\nbefore %s\n after %s", before, after)
	}
}

// Property test against the engine itself: for random creation states,
// Eligible matches ClassAvailability, and each reachability claim is
// confirmed or refuted by actually spending (and moving) points with the
// engine's AddStatPoint and RemoveStatPoint.
func TestClassGuideAgreesWithEngine(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	for n := 0; n < 3000; n++ {
		race := engine.Race(r.Intn(5))
		align := engine.Alignment(r.Intn(3))
		base := engine.RaceBaseStats[race]
		total := 7 + r.Intn(24)
		stats := base
		left := total
		for left > 0 && r.Intn(4) != 0 { // spend some points at random
			i := r.Intn(6)
			if stats[i] < 18 {
				stats[i]++
				left--
			}
		}
		cs := &engine.CreationState{Race: race, Alignment: align, Stats: stats, BonusPoints: left}
		guide := classGuide(cs)
		avail := cs.ClassAvailability()

		for _, st := range guide {
			if st.Eligible != avail[st.Class] {
				t.Fatalf("%v %v %v: Eligible %v, engine %v", race, align, stats, st.Eligible, avail[st.Class])
			}
			if st.Eligible != (st.AlignmentOK && st.PointsNeeded == 0) {
				t.Fatalf("%v: Eligible disagrees with alignment/shortfall: %+v", st.Class, st)
			}
			if got := reachBySpending(*cs, st.Class); got != st.ReachableWithRemaining {
				t.Fatalf("%v %v %v left %d %v: ReachableWithRemaining %v, simulated %v",
					race, align, stats, left, st.Class, st.ReachableWithRemaining, got)
			}
			if got := reachByMoving(*cs, st.Class); got != st.ReachableByMoving {
				t.Fatalf("%v %v %v left %d %v: ReachableByMoving %v, simulated %v",
					race, align, stats, left, st.Class, st.ReachableByMoving, got)
			}
		}
	}
}

// reachBySpending spends unspent points on the class's short stats with the
// engine's AddStatPoint and reports whether the class becomes available.
func reachBySpending(cs engine.CreationState, class engine.Class) bool {
	req := engine.ClassRequirements[class]
	for i := range cs.Stats {
		cs.StatCursor = i
		for cs.Stats[i] < req[i] {
			before := cs.Stats[i]
			cs.AddStatPoint()
			if cs.Stats[i] == before {
				break
			}
		}
	}
	return cs.ClassAvailability()[class]
}

// reachByMoving takes every spent point back with RemoveStatPoint, then
// spends the whole pool on the class's requirements.
func reachByMoving(cs engine.CreationState, class engine.Class) bool {
	for i := range cs.Stats {
		cs.StatCursor = i
		for {
			before := cs.Stats[i]
			cs.RemoveStatPoint()
			if cs.Stats[i] == before {
				break
			}
		}
	}
	return reachBySpending(cs, class)
}
