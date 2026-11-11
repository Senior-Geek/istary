package engine

// Stranded characters (R-1). When a whole party can no longer go on, in
// combat or from a square's damage or poison, its members stay in the maze
// on the square where it happened, for a rescue expedition (decision D-D),
// instead of being carried back to the castle.
//
// The state, and its invariants:
//
//   - Character.Stranded is the one record of where a stranded character
//     is, in L:R:C (place.go). It is set only by StrandParty, always to a
//     square of a maze level (never the castle), and only for characters
//     who were in the party.
//   - A stranded character keeps InMaze set: physically out of the castle.
//     (InMaze is also set for every member of a party, in the castle too,
//     as the original's Tavern sets it; it alone does not say where anyone
//     is.)
//   - A stranded character is in no party and on no saved expedition.
//   - Condition, items, gold and experience are left exactly as they were.
//   - MazeLevel, MazeX and MazeY (the original's LOSTXYL) are never written
//     but to clear them: their level was 1-based in the original and
//     0-based here, so a 0 could mean the castle or the first level. They
//     are read only for characters an older version left out with no
//     Stranded record (Camp D)ISBAND, finding #26), exactly as before, and
//     cleared when such a character is picked up (rescue.go).
//   - Old saves have no Stranded record and none is ever made up for them.
//   - A rescue (rescue.go) clears Stranded as the character joins the
//     party; a stranded character is never in a party.

import "errors"

// PartyCanContinue is the one rule for whether a party can go on: someone
// in it can still act. ASLEEP and AFRAID members can (they recover);
// PARALYZED, STONED, DEAD, ASHED and LOST members cannot. A party with no
// one in it cannot.
func PartyCanContinue(members []*Character) bool {
	for _, m := range members {
		if m != nil && m.IsAlive() {
			return true
		}
	}
	return false
}

// IsStranded reports whether c was left in the maze by a party that could
// not go on.
func (c *Character) IsStranded() bool { return c.Stranded != nil }

// OutOfCastle reports whether c, if not in the castle party, is out of the
// castle: stranded, out on an expedition, or left out by an older version
// (InMaze, or the legacy LOSTXYL level the original's Tavern also checks).
// It is the castle's one test for "** OUT **".
func (c *Character) OutOfCastle() bool {
	return c.Stranded != nil || c.InMaze || c.MazeLevel != 0
}

// ErrNoStrandingSquare is returned when the party is not on a square of a
// maze level (a floor below the last, finding #31), so there is no square
// to leave it on. Nothing is changed.
var ErrNoStrandingSquare = errors.New("the party is not on a square of the maze")

// StrandMessage is the castle's message after a party is stranded (the
// original's line); Message2 says where the party remains.
const StrandMessage = "YOUR PARTY HAS PERISHED IN THE MAZE..."

// StrandParty ends an expedition whose party cannot go on
// (PartyCanContinue is false): each member stays on the party's square,
// with their condition as it is, and the player is back in the castle with
// no party. A LOST member is not left anywhere (LOST is for ever, as the
// Temple and Camp MALOR treat it): they leave the party and the maze.
//
// It reports the square, and false when there was no party to strand (so
// calling it again changes nothing). It refuses, changing nothing, a party
// that can still go on or that is not on a square of the maze.
func (g *GameState) StrandParty() (Place, bool, error) {
	var party []*Character
	for _, m := range g.Town.Party.Members {
		if m != nil {
			party = append(party, m)
		}
	}
	if len(party) == 0 {
		return Place{}, false, nil
	}
	if PartyCanContinue(party) {
		return Place{}, false, errors.New("the party can still go on")
	}
	here, ok := g.Place()
	if !ok {
		return Place{}, false, ErrNoStrandingSquare
	}
	for _, m := range party {
		if m.Status == Lost {
			m.InMaze, m.Stranded = false, nil
			continue
		}
		p := here
		m.InMaze, m.Stranded = true, &p
	}
	g.Town.Party.Members = nil
	g.Combat = nil
	g.Phase = PhaseTown
	g.Town.Location = Castle
	g.Town.InputMode = InputNone
	g.Town.EditChar, g.Town.TradeTarget = nil, nil
	g.Town.Message = StrandMessage
	g.Town.Message2 = "THEY REMAIN AT " + here.String() + "."
	g.MazeMessage, g.MazeMessage2 = "", ""
	g.MazeMessages, g.MazeMsgScroll, g.MazeMsgWait = nil, 0, false
	g.MazeSearchYN, g.SearchCell, g.SearchMonster = false, nil, 0
	g.MazePendingBack2Shop, g.MazePendingTryGet, g.MazeTryGetItem = false, false, 0
	g.WaitButton, g.ButtonCount = false, 0
	g.MazeInspecting, g.MazeInspectFound = false, nil
	g.ViewportMsg, g.ViewportMsg2 = "", ""
	return here, true, nil
}
