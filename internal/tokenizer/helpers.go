package tokenizer

// ReadInsideBracketsThen returns a StateFn that tracks matching brackets with
// a locally-scoped counter. When endState is non-nil, body is called for every
// token including the brackets; otherwise body is called only when the brackets
// balance (count reaches 0). On balancing, m transitions to endState (if non-nil).
//
// Each call creates a fresh closure with its own counter, so nested uses of
// ReadInsideBracketsThen are independent.
func ReadInsideBracketsThen(m *Machine, open, close string, endState StateFn, body func(tok string)) StateFn {
	count := 0
	return func(tok string) bool {
		switch tok {
		case open:
			count++
		case close:
			count--
		}
		if count == 0 || endState != nil {
			body(tok)
		}
		if count == 0 && endState != nil {
			m.state = endState
		}
		return false
	}
}

// ReadUntilThen returns a StateFn that collects tokens into a local buffer until
// one of stops is encountered, then calls body with the stop token and collected
// list. The buffer is reset after each call to body.
func ReadUntilThen(stops []string, body func(stop string, collected []string)) StateFn {
	var buf []string
	return func(tok string) bool {
		for _, s := range stops {
			if tok == s {
				body(tok, buf)
				buf = nil
				return false
			}
		}
		buf = append(buf, tok)
		return false
	}
}
