package app

// PositionView is where the party stands, for the maze screen's and Camp's
// coordinate readout (UX-1): the canonical L:R:C (engine.Place) and the
// facing, from the same source as the maps and the rescue records. It is
// not part of DungeonView, which, like the Classic maze screen, gives no
// position.
type PositionView struct {
	Place  string `json:"place"` // "L1:R19:C0"
	Level  int    `json:"level"`
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Facing string `json:"facing"` // N, E, S or W
	// Stranded is how many characters lie on this very square, as its map
	// marker counts them (StrandedSites): nothing the maps do not show.
	Stranded int `json:"stranded"`
}

// Position returns where the party stands, or false when it is not on a
// square of the maze (in town, or carried off the maze's levels): then
// there are no coordinates to show, and none is made up. It reads the game
// and changes nothing.
func (c *Controller) Position() (PositionView, bool) {
	g := c.Game
	here, ok := partyPlace(g)
	if !ok {
		return PositionView{}, false
	}
	v := PositionView{Place: here.String(), Level: here.Level, Row: here.Row, Col: here.Col, Facing: g.Facing.Letter()}
	for _, s := range g.StrandedSites() {
		if s.Place == here {
			v.Stranded = len(s.Characters)
		}
	}
	return v, true
}
