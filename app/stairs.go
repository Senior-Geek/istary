package app

// The stairs' suggested answer. The maze asks TAKE THEM (Y/N) ? on any
// staircase, as the original does, and both answers stay open; the screen
// only suggests one, by how the party came to stand there:
//
//   - off the stairs (down from the castle, or off a staircase just taken):
//     no. The party has just come that way and goes on, into the maze.
//   - by a step or a kick onto the square: yes. The party came to the
//     stairs to take them, out to the castle or on to another level.
//   - in any other way (a teleporter, a chute, MALOR, the elevator, or a
//     restart, as nothing of this is saved): yes, as before.
//
// Turning in place, Camp and an encounter on the square change nothing.

// How the party came to a square.
const (
	arrivedWalking  = iota + 1 // a step or a kick
	arrivedByStairs            // down from the castle, or off a staircase taken
)

// arrival is how the party came to the square it stood on then.
type arrival struct{ level, x, y, by int }

// arrived records how the party came to the square it stands on.
func (c *Controller) arrived(by int) {
	g := c.Game
	c.arrival = arrival{g.MazeLevel, g.PlayerX, g.PlayerY, by}
}

// stairsDefault is the suggested answer to the stairs here: ActNo when the
// party came off the stairs onto this square, otherwise ActYes.
func (c *Controller) stairsDefault() string {
	g := c.Game
	if c.arrival == (arrival{g.MazeLevel, g.PlayerX, g.PlayerY, arrivedByStairs}) {
		return ActNo
	}
	return ActYes
}
