package app

import "github.com/Senior-Geek/istary/engine"

// The Inn's operations (I-1), for front ends without keys. Each one is the
// terminal's keys and heal ticks, pressed for the player, so the two cannot
// differ; arguments are checked first, and a bad one changes nothing. A
// refusal is returned as *Refusal. A stay is one operation: the room, every
// week of it and the level check after it, as the terminal's timer would
// run them; healing draws no random number, the level check does, once.

// atInn reports that the player is at the Inn.
func (c *Controller) atInn() bool {
	g := c.Game
	return g.Phase == engine.PhaseTown && g.Town.Location == engine.Inn
}

// InnQuote is what a stay in a room would do for a guest, worked out by the
// rules the stay follows (InnTick): each week heals the room's rate, to
// the maximum at most, and costs a full week, while the guest is hurt and
// can pay. Nothing is changed and no random number is drawn.
type InnQuote struct {
	Weeks      int  `json:"weeks"`      // weeks of healing the stay pays for
	Cost       int  `json:"cost"`       // their fee
	HP         int  `json:"hp"`         // hit points after the stay
	Healed     bool `json:"healed"`     // the stay ends at full hit points
	Needed     int  `json:"needed"`     // weeks to heal fully (0 healed, or the stables)
	Affordable int  `json:"affordable"` // weeks the guest's gold pays for (0 in the stables)
}

// innQuote quotes a stay in room for ch until healed or out of gold, the
// original's only stay.
func innQuote(ch *engine.Character, room engine.InnRoom) InnQuote {
	r := innRooms[room]
	q := InnQuote{HP: ch.HP}
	if r.hp > 0 {
		if ch.HP < ch.MaxHP {
			q.Needed = (ch.MaxHP - ch.HP + r.hp - 1) / r.hp
		}
		q.Affordable = ch.Gold / r.fee
	}
	q.Weeks = min(q.Needed, q.Affordable)
	q.Cost = q.Weeks * r.fee
	q.HP = min(ch.MaxHP, ch.HP+q.Weeks*r.hp)
	q.Healed = q.HP == ch.MaxHP
	return q
}

// InnStay is what a stay did: the room and weeks, the gold charged, and the
// guest before and after it, with the level check's outcome.
type InnStay struct {
	Guest        string   `json:"guest"`
	Room         int      `json:"room"` // engine.InnRoom
	RoomName     string   `json:"roomName"`
	Weeks        int      `json:"weeks"` // weeks of healing (0 in the stables)
	Charged      int      `json:"charged"`
	HPBefore     int      `json:"hpBefore"`
	HPAfter      int      `json:"hpAfter"`
	MaxHPBefore  int      `json:"maxHpBefore"`
	MaxHPAfter   int      `json:"maxHpAfter"`
	GoldBefore   int      `json:"goldBefore"`
	GoldAfter    int      `json:"goldAfter"`
	Spells       bool     `json:"spells"` // spell points came back
	MageBefore   [7]int   `json:"mageBefore"`
	MageAfter    [7]int   `json:"mageAfter"`
	PriestBefore [7]int   `json:"priestBefore"`
	PriestAfter  [7]int   `json:"priestAfter"`
	LevelBefore  int      `json:"levelBefore"`
	LevelAfter   int      `json:"levelAfter"`
	StatsBefore  [6]int   `json:"statsBefore"` // STR, I.Q., PIE, VIT, AGI, LUCK
	StatsAfter   [6]int   `json:"statsAfter"`
	AgeBefore    int      `json:"ageBefore"` // weeks
	AgeAfter     int      `json:"ageAfter"`
	Learned      []string `json:"learned"`  // spells first known after the stay
	Status       string   `json:"status"`   // after the stay, e.g. "LOST"
	Died         bool     `json:"died"`     // of old age at the level check
	Messages     []string `json:"messages"` // the Inn's words at the level check
	XPNeeded     int      `json:"xpNeeded"` // experience still needed for the next level (0 if levelled or unknown)
}

// InnEnter checks the party member in slot (0-based) in as the guest
// ("WHO WILL STAY"). Only a guest who is OK may stay.
func (c *Controller) InnEnter(slot int) error {
	town := c.Game.Town
	if !c.atInn() || town.InnStep != engine.InnWho {
		return ErrWrongState
	}
	if slot < 0 || slot >= MaxPartySize || slot >= len(town.Party.Members) || town.Party.Members[slot] == nil {
		return ErrNoSuchPartyMember
	}
	c.press(Rune('1' + rune(slot)))
	if town.InnStep == engine.InnWho {
		return &Refusal{Message: town.Message}
	}
	return nil
}

// innFinish runs a stay begun with the terminal's keys to its end, as its
// timer would.
func (c *Controller) innFinish() {
	for c.Game.Town.InnStep == engine.InnHealing {
		c.InnTick()
	}
}

// InnLeaveGuest ends the guest's visit: back to "WHO WILL STAY", by RETURN
// as the keys do.
func (c *Controller) InnLeaveGuest() error {
	town := c.Game.Town
	if !c.atInn() || town.InnStep == engine.InnWho {
		return ErrWrongState
	}
	c.innFinish()
	for town.InnStep != engine.InnWho {
		c.press(Key{Kind: KeyEnter})
	}
	return nil
}

// LeaveInn leaves the Inn for the castle from wherever the player is in it,
// by RETURN as the keys do (a stay under way is finished first).
func (c *Controller) LeaveInn() error {
	if !c.atInn() {
		return ErrWrongState
	}
	c.innFinish()
	for c.Game.Town.Location == engine.Inn {
		c.press(Key{Kind: KeyEnter})
	}
	return nil
}

// Stay is the guest's stay in room until healed or out of gold, as the
// original does, then the level check. From the level check's result the
// guest may stay again.
func (c *Controller) Stay(room engine.InnRoom) (*InnStay, error) {
	town := c.Game.Town
	if !c.atInn() || town.InnChar == nil || (town.InnStep != engine.InnSelectRoom && town.InnStep != engine.InnLevelUp) {
		return nil, ErrWrongState
	}
	if room < engine.Stables || room > engine.Royal {
		return nil, ErrInvalidChoice
	}
	ch := town.InnChar
	if town.InnStep == engine.InnLevelUp {
		c.press(Key{Kind: KeyEnter})
	}
	s := &InnStay{
		Guest: ch.Name, Room: int(room), RoomName: innRooms[room].name,
		HPBefore: ch.HP, MaxHPBefore: ch.MaxHP, GoldBefore: ch.Gold,
		MageBefore: ch.MageSpells, PriestBefore: ch.PriestSpells,
		LevelBefore: ch.Level, StatsBefore: stats(ch), AgeBefore: ch.Age,
	}
	known := ch.SpellKnown
	c.press(Rune(innRooms[room].key))
	for town.InnStep == engine.InnHealing {
		if town.InnHealAmt > 0 && ch.HP < ch.MaxHP && ch.Gold >= town.InnHealCost {
			s.Weeks++
		}
		c.InnTick()
	}
	s.Charged = s.GoldBefore - ch.Gold
	s.HPAfter, s.MaxHPAfter, s.GoldAfter = ch.HP, ch.MaxHP, ch.Gold
	s.MageAfter, s.PriestAfter = ch.MageSpells, ch.PriestSpells
	s.Spells = s.MageAfter != s.MageBefore || s.PriestAfter != s.PriestBefore
	s.LevelAfter, s.StatsAfter, s.AgeAfter = ch.Level, stats(ch), ch.Age
	for i, k := range ch.SpellKnown {
		if k && !known[i] && i < len(engine.SpellTable) {
			s.Learned = append(s.Learned, engine.SpellTable[i].Name)
		}
	}
	s.Status = ch.Status.String()
	s.Died = ch.Status == engine.Lost
	s.Messages = append([]string(nil), town.InnMessages...)
	if s.LevelAfter == s.LevelBefore {
		if next := engine.XPForNextLevel(ch, c.Game); next > ch.XP {
			s.XPNeeded = next - ch.XP
		}
	}
	return s, nil
}

func stats(ch *engine.Character) [6]int {
	return [6]int{ch.Strength, ch.IQ, ch.Piety, ch.Vitality, ch.Agility, ch.Luck}
}
