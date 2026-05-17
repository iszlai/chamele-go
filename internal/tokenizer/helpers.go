package tokenizer

// ReadInsideBracketsThen returns a StateFn that tracks matching brackets using
// m.BrCount. When an end state is provided, body is called for every token
// inside the brackets; without one, body is called only on the closing bracket.
// When the brackets balance, m.state is set to endState (if non-nil).
//
// Port of CodeStateMachine.read_inside_brackets_then.
func ReadInsideBracketsThen(m *Machine, open, close string, endState StateFn, body func(tok string)) StateFn {
	return func(tok string) bool {
		switch tok {
		case open:
			m.BrCount++
		case close:
			m.BrCount--
		}
		if m.BrCount == 0 || endState != nil {
			body(tok)
		}
		if m.BrCount == 0 && endState != nil {
			m.state = endState
		}
		return false
	}
}

// ReadUntilThen returns a StateFn that collects tokens into m.RutTokens until
// one of stops is encountered, then calls body with the stop token and the
// collected list.
//
// Port of CodeStateMachine.read_until_then.
func ReadUntilThen(m *Machine, stops []string, body func(stop string, collected []string)) StateFn {
	return func(tok string) bool {
		for _, s := range stops {
			if tok == s {
				body(tok, m.RutTokens)
				m.RutTokens = nil
				return false
			}
		}
		m.RutTokens = append(m.RutTokens, tok)
		return false
	}
}
