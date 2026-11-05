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
