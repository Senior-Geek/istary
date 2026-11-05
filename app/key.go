package app

// KeyKind identifies a key independently of any terminal library.
type KeyKind int

const (
	KeyOther KeyKind = iota
	KeyRune
	KeyEnter
	KeyEscape
	KeyBackspace
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
)

// Key is one key press. Rune is set when Kind is KeyRune.
type Key struct {
	Kind KeyKind
	Rune rune
}

// Rune returns a printable-character key.
func Rune(r rune) Key { return Key{Kind: KeyRune, Rune: r} }
