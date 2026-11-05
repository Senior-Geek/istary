package app

import "github.com/Senior-Geek/istary/engine"

// ClassStatus describes one class against the character being created. It is
// read-only guidance: every value is derived from the engine's own tables and
// rules (engine.ClassRequirements, CreationState.ClassAvailability,
// AddStatPoint, RemoveStatPoint), so it cannot disagree with them.
type ClassStatus struct {
	Class engine.Class
	// Minimums are the required STR, IQ, PIE, VIT, AGI, LCK.
	Minimums [6]int
	// Alignments lists the alignments this class allows.
	Alignments []engine.Alignment
	// AlignmentOK is true when the chosen alignment is allowed.
	AlignmentOK bool
	// Shortfall is how many points each stat is below its minimum (0 if met).
	Shortfall [6]int
	// PointsNeeded is the sum of Shortfall.
	PointsNeeded int
	// Eligible is the engine's verdict for the current stats and alignment.
	Eligible bool
	// ReachableWithRemaining is true when spending the unspent bonus points
	// can make the class eligible.
	ReachableWithRemaining bool
	// ReachableByMoving is true when the class can be made eligible by also
	// taking back points already spent on other stats.
	ReachableByMoving bool
}

// ClassGuide reports every class against the creation in progress. It is
// available once alignment is chosen (stats exist) and changes nothing.
func (c *Controller) ClassGuide() ([]ClassStatus, error) {
	cs := c.Creation()
	if cs == nil || cs.Step < engine.StepStats {
		return nil, ErrWrongState
	}
	return classGuide(cs), nil
}

func classGuide(cs *engine.CreationState) []ClassStatus {
	eligible := cs.ClassAvailability()
	allowed := alignmentsAllowed()
	ceiling, floor := statLimits(cs)

	movable := 0 // points that RemoveStatPoint could take back
	for i := range cs.Stats {
		movable += cs.Stats[i] - floor[i]
	}

	out := make([]ClassStatus, 0, 8)
	for class := engine.Fighter; class <= engine.Ninja; class++ {
		st := ClassStatus{
			Class:      class,
			Minimums:   engine.ClassRequirements[class],
			Alignments: allowed[class],
			Eligible:   eligible[class],
		}
		for _, a := range st.Alignments {
			if a == cs.Alignment {
				st.AlignmentOK = true
			}
		}
		fromBase, capped := 0, false
		for i, min := range st.Minimums {
			if short := min - cs.Stats[i]; short > 0 {
				st.Shortfall[i] = short
				st.PointsNeeded += short
			}
			if min > ceiling[i] {
				capped = true
			}
			if need := min - floor[i]; need > 0 {
				fromBase += need
			}
		}
		st.ReachableWithRemaining = st.AlignmentOK && !capped && st.PointsNeeded <= cs.BonusPoints
		st.ReachableByMoving = st.AlignmentOK && !capped && fromBase <= cs.BonusPoints+movable
		out = append(out, st)
	}
	return out
}

// alignmentsAllowed asks the engine which alignments each class accepts, by
// checking availability with every stat at the top of the range.
func alignmentsAllowed() [8][]engine.Alignment {
	var out [8][]engine.Alignment
	for a := engine.Good; a <= engine.Evil; a++ {
		probe := engine.CreationState{Alignment: a, Stats: [6]int{18, 18, 18, 18, 18, 18}}
		for class, ok := range probe.ClassAvailability() {
			if ok {
				out[class] = append(out[class], a)
			}
		}
	}
	return out
}

// statLimits finds, for each stat, the highest value AddStatPoint allows and
// the lowest value RemoveStatPoint allows, by trying them on a copy.
func statLimits(cs *engine.CreationState) (ceiling, floor [6]int) {
	for i := range cs.Stats {
		up := *cs
		up.StatCursor = i
		up.BonusPoints = 100
		for {
			before := up.Stats[i]
			up.AddStatPoint()
			if up.Stats[i] == before {
				break
			}
		}
		ceiling[i] = up.Stats[i]

		down := *cs
		down.StatCursor = i
		for {
			before := down.Stats[i]
			down.RemoveStatPoint()
			if down.Stats[i] == before {
				break
			}
		}
		floor[i] = down.Stats[i]
	}
	return ceiling, floor
}
