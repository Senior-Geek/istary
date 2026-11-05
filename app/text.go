package app

// Text-entry rules shared by the name and password prompts. Limits and
// allowed characters match the original prompts.

// MaxNameLength is the longest name or password the prompts accept.
const MaxNameLength = 15

// appendNameChar applies one typed character to a character-name buffer:
// at most 15 characters, letters uppercased, only A-Z, 0-9, space, - and '.
func appendNameChar(buf string, ch rune) string {
	if len(buf) >= MaxNameLength {
		return buf
	}
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}
	if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == ' ' || ch == '-' || ch == '\'' {
		return buf + string(ch)
	}
	return buf
}

// appendPasswordChar applies one typed character to a password buffer:
// at most 15 characters, letters uppercased, anything else kept.
func appendPasswordChar(buf string, ch rune) string {
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}
	if len(buf) < MaxNameLength {
		return buf + string(ch)
	}
	return buf
}

// NormalizeName returns s as the name prompt would have accepted it typed
// one character at a time.
func NormalizeName(s string) string {
	buf := ""
	for _, ch := range s {
		buf = appendNameChar(buf, ch)
	}
	return buf
}

// NormalizePassword returns s as the password prompts would have accepted it.
func NormalizePassword(s string) string {
	buf := ""
	for _, ch := range s {
		buf = appendPasswordChar(buf, ch)
	}
	return buf
}

func backspace(s string) string {
	if len(s) > 0 {
		return s[:len(s)-1]
	}
	return s
}

// upper maps a-z to A-Z, leaving other runes alone.
func upper(ch rune) rune {
	if ch >= 'a' && ch <= 'z' {
		return ch - 32
	}
	return ch
}
