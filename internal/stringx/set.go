package stringx

// Set is a string set with O(1) membership. Nil is a valid empty set.
type Set map[string]struct{}

// NewSet returns a Set containing the given members.
func NewSet(members ...string) Set {
	s := make(Set, len(members))
	for _, m := range members {
		s[m] = struct{}{}
	}
	return s
}

// Has reports whether tok is in the set.
func (s Set) Has(tok string) bool {
	_, ok := s[tok]
	return ok
}

// Add inserts tok into the set.
func (s Set) Add(tok string) { s[tok] = struct{}{} }
