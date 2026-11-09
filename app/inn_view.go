package app

import "github.com/Senior-Geek/istary/engine"

// InnView describes the Adventurer's Inn as the player sees it now: the
// step, the guest, and every room with what a stay until healed there
// would do. It is read-only and draws no random number; a front end shows
// it and calls the operations in inn_ops.go.
type InnView struct {
	Step     string         `json:"step"`  // "who", "rooms", "healing" (a terminal stay under way) or "levelUp"
	Guest    int            `json:"guest"` // party slot (0-based) of the guest, -1 if none
	Name     string         `json:"name"`  // the guest's name
	Message  string         `json:"message"`
	Messages []string       `json:"messages"` // the level check's words, at "levelUp"
	Guests   []InnGuestView `json:"guests"`   // every party member, and whether they may stay
	Rooms    []InnRoomView  `json:"rooms"`    // with a guest: each room, quoted for them
}

// InnGuestView is a party member at "WHO WILL STAY".
type InnGuestView struct {
	Slot    int    `json:"slot"` // 0-based
	Name    string `json:"name"`
	Refusal string `json:"refusal"` // e.g. "PRIEST IS NOT OK"; "" if they may stay
}

// InnRoomView is a room, its rate, and for the guest a stay until healed
// there.
type InnRoomView struct {
	Room        int      `json:"room"` // engine.InnRoom
	Key         string   `json:"key"`  // its letter on the terminal's menu
	Name        string   `json:"name"`
	HPPerWeek   int      `json:"hpPerWeek"`
	FeePerWeek  int      `json:"feePerWeek"`
	Free        bool     `json:"free"`
	UntilHealed InnQuote `json:"untilHealed"`
}

var innStepNames = map[engine.InnStep]string{
	engine.InnWho: "who", engine.InnSelectRoom: "rooms", engine.InnHealing: "healing", engine.InnLevelUp: "levelUp",
}

// InnView describes the Inn now; ErrWrongState away from it.
func (c *Controller) InnView() (InnView, error) {
	if !c.atInn() {
		return InnView{}, ErrWrongState
	}
	town := c.Game.Town
	v := InnView{Step: innStepNames[town.InnStep], Guest: -1, Message: town.Message}
	for i, m := range town.Party.Members {
		if m == nil {
			continue
		}
		g := InnGuestView{Slot: i, Name: m.Name}
		if m.Status != engine.OK {
			g.Refusal = m.Name + " IS NOT OK"
		}
		v.Guests = append(v.Guests, g)
		if m == town.InnChar && town.InnStep != engine.InnWho {
			v.Guest, v.Name = i, m.Name
		}
	}
	ch := town.InnChar
	if ch == nil || town.InnStep == engine.InnWho {
		return v, nil
	}
	if town.InnStep == engine.InnLevelUp {
		v.Messages = append([]string(nil), town.InnMessages...)
	}
	for r, room := range innRooms {
		rv := InnRoomView{
			Room: r, Key: string(room.key), Name: room.name,
			HPPerWeek: room.hp, FeePerWeek: room.fee, Free: room.fee == 0,
			UntilHealed: innQuote(ch, engine.InnRoom(r)),
		}
		v.Rooms = append(v.Rooms, rv)
	}
	return v, nil
}

// Quote quotes the guest's stay in room, as Stay would take it, without
// changing anything.
func (c *Controller) Quote(room engine.InnRoom) (InnQuote, error) {
	town := c.Game.Town
	if !c.atInn() || town.InnChar == nil || town.InnStep == engine.InnWho {
		return InnQuote{}, ErrWrongState
	}
	if room < engine.Stables || room > engine.Royal {
		return InnQuote{}, ErrInvalidChoice
	}
	return innQuote(town.InnChar, room), nil
}
