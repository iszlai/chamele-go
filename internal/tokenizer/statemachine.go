package tokenizer

// StateFn is a state in a CodeStateMachine. It returns true when the state
// signals "done" (equivalent to returning a truthy value from a Python state method),
// which causes the machine to pop back to savedState and fire the callback.
type StateFn func(tok string) bool

// Machine is the Go port of Python's CodeStateMachine. Language readers embed
// or compose Machine values to drive their token-level state machines.
//
// The func-pointer pattern mirrors Python's `self._state = self._other_state`
// assignment directly.
type Machine struct {
	state      StateFn
	savedState StateFn
	callback   func()

	// LastToken is the previous token, readable by state functions.
	LastToken string
	// ToExit is set by Return(); the outer loop checks it after each Call.
	ToExit bool

	// BrCount is used by ReadInsideBracketsThen helpers.
	BrCount int
	// RutTokens is the accumulator used by ReadUntilThen helpers.
	RutTokens []string
}

// NewMachine creates a machine whose initial state is stateGlobal.
func NewMachine() *Machine {
	m := &Machine{}
	m.state = m.stateGlobal
	m.savedState = m.stateGlobal
	return m
}

func (m *Machine) stateGlobal(_ string) bool { return false }

// Call processes one token through the current state.
// Returns true if the machine has exited (ToExit is set).
func (m *Machine) Call(tok string) bool {
	if m.state(tok) {
		m.state = m.savedState
		if m.callback != nil {
			m.callback()
			m.callback = nil
		}
	}
	m.LastToken = tok
	return m.ToExit
}

// Next sets the current state. If tok is non-empty, it is immediately processed
// through the new state and the result of Call is returned.
func (m *Machine) Next(state StateFn, tok ...string) bool {
	m.state = state
	if len(tok) > 0 {
		return m.Call(tok[0])
	}
	return false
}

// NextIf sets state and processes tok only if tok == expected.
func (m *Machine) NextIf(state StateFn, tok, expected string) {
	if tok == expected {
		m.Next(state, tok)
	}
}

// SubState saves the current state, sets the sub-state, and optionally processes tok.
func (m *Machine) SubState(state StateFn, callback func(), tok ...string) {
	m.savedState = m.state
	m.callback = callback
	m.Next(state, tok...)
}

// Return signals that this machine is done; the next Call will return true.
func (m *Machine) Return() {
	m.ToExit = true
}

// StatemachineBeforeReturn is a hook called when the machine is about to return.
// Embed Machine and override this method to react to end-of-tokens.
func (m *Machine) StatemachineBeforeReturn() {}
