package app

import "errors"

// Errors returned by semantic operations. The terminal key map only calls an
// operation when its precondition holds, so these are mainly for GUI callers.
var (
	ErrWrongState        = errors.New("operation not available in the current state")
	ErrEmptyName         = errors.New("name is empty")
	ErrPasswordMismatch  = errors.New("passwords do not match")
	ErrPointsRemaining   = errors.New("bonus points remain to be spent")
	ErrNoClassAvailable  = errors.New("no class is available with these statistics")
	ErrClassUnavailable  = errors.New("class is not available with these statistics")
	ErrInvalidChoice     = errors.New("invalid choice")
	ErrPartyFull         = errors.New("** PARTY IS FULL **")
	ErrPartyEmpty        = errors.New("party is empty")
	ErrNoSuchPartyMember = errors.New("no party member in that slot")
)

// Refusal is returned when the game refuses an action with an in-game
// message, such as "** BAD ALIGNMENT **". The same text is shown in
// Town.Message.
type Refusal struct{ Message string }

func (r *Refusal) Error() string { return r.Message }

// ErrOnExpedition refuses renaming or deleting a character who is out on a
// saved expedition: the expedition refers to its members by name, and their
// roster entry is their only record.
var ErrOnExpedition = &Refusal{Message: "** OUT ON AN EXPEDITION **"}

// ErrStranded refuses castle actions (renaming, deleting, equipping) on a
// character stranded in the maze by a defeated party (R-1): they are not at
// the castle.
var ErrStranded = &Refusal{Message: "** OUT IN THE MAZE **"}

// ErrNoOneCanGoOn refuses setting out with a party in which no one can act
// (all DEAD, ASHED, PARALYZED or STONED, as a rescued party may come back):
// it would be stranded at the entrance. Taking the incapacitated along with
// someone who can act is allowed, as the original allows it.
var ErrNoOneCanGoOn = &Refusal{Message: "NO ONE IN THE PARTY CAN GO ON."}
