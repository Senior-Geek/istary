package app

import "github.com/Senior-Geek/istary/engine"

// TempleView describes the Temple of Cant now: the patients it has
// something to say about, with their donations and whether each can be
// treated and paid for, and who may tithe. It is read-only and draws no
// random number; it never shows a chance of success (the Temple's rule is
// engine.TempleHeal; a front end words the risk by Kind).
type TempleView struct {
	Step      string          `json:"step"` // "who", "tithe", "ritual" or "result" (the terminal's steps)
	Message   string          `json:"message"`
	Patients  []TemplePatient `json:"patients"`
	Payers    []TemplePayer   `json:"payers"`
	PartyGold int             `json:"partyGold"` // all the party's gold, which Pool Gold can bring to one payer
}

// TemplePatient is a roster character who needs the Temple, or is LOST.
type TemplePatient struct {
	Index    int    `json:"index"` // roster index
	Name     string `json:"name"`
	Class    string `json:"class"`
	Level    int    `json:"level"`
	Status   string `json:"status"`
	InParty  bool   `json:"inParty"`
	Kind     string `json:"kind"`     // cure, raise or ashes; "" for the LOST
	Donation int    `json:"donation"` // 0 for the LOST
	Refusal  string `json:"refusal"`  // why not treatable now ("" if it can be)
	// Funding says whether the donation can be met: "payer" (a member who
	// may tithe has it), "pool" (only the party's gold together has it:
	// Pool Gold to a payer first) or "short" (not even together).
	Funding string `json:"funding"`
}

// TemplePayer is a party member, their gold, and why they may not tithe.
type TemplePayer struct {
	Slot    int    `json:"slot"` // 0-based
	Name    string `json:"name"`
	Gold    int    `json:"gold"`
	Refusal string `json:"refusal"` // not OK; "" if they may tithe (the donation is checked by patient)
}

var templeStepNames = map[engine.TempleStep]string{
	engine.TempleWho: "who", engine.TempleStatus: "who", engine.TempleTithe: "tithe",
	engine.TempleRitual: "ritual", engine.TempleResult: "result",
}

// TempleView describes the Temple now; ErrWrongState away from it.
func (c *Controller) TempleView() (TempleView, error) {
	if !c.atTemple() {
		return TempleView{}, ErrWrongState
	}
	town := c.Game.Town
	v := TempleView{Step: templeStepNames[town.TempleStep], Message: town.Message}
	best := 0 // the most gold one member who may tithe has
	for i, m := range town.Party.Members {
		if m == nil {
			continue
		}
		p := TemplePayer{Slot: i, Name: m.Name, Gold: m.Gold}
		if m.Status != engine.OK {
			p.Refusal = payerRefusal(m)
		} else {
			best = max(best, m.Gold)
		}
		v.PartyGold += m.Gold
		v.Payers = append(v.Payers, p)
	}
	for i, ch := range town.Roster.Characters {
		if ch == nil {
			continue
		}
		kind := templeKind(ch.Status)
		if kind == "" && ch.Status != engine.Lost {
			continue // nothing for the Temple to do
		}
		p := TemplePatient{
			Index: i, Name: ch.Name, Class: ch.Class.String(), Level: ch.Level, Status: ch.Status.String(),
			InParty: InParty(town, ch), Kind: kind, Donation: engine.TempleDonation(ch.Status, ch.Level),
			Refusal: c.patientRefusal(ch),
		}
		switch {
		case kind == "":
		case best >= p.Donation:
			p.Funding = "payer"
		case v.PartyGold >= p.Donation && anyOK(town):
			p.Funding = "pool"
		default:
			p.Funding = "short"
		}
		v.Patients = append(v.Patients, p)
	}
	return v, nil
}

// anyOK reports whether some party member may tithe.
func anyOK(town *engine.TownState) bool {
	for _, m := range town.Party.Members {
		if m != nil && m.Status == engine.OK {
			return true
		}
	}
	return false
}
