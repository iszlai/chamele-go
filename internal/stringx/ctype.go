package stringx

// IsAlpha reports whether b is an ASCII letter.
func IsAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// IsHSpace reports whether s is a non-empty string consisting only of
// horizontal whitespace ([ \t\r]). Vertical whitespace (\n, \f, \v) and
// the empty string return false.
func IsHSpace(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\r' {
			return false
		}
	}
	return true
}
