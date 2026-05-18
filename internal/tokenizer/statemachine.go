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
}

// NewMachine creates a machine whose initial state is a no-op stateGlobal.
// Call SetInitialState immediately after to point it at the real global state.
func NewMachine() *Machine {
	m := &Machine{}
	m.state = m.stateGlobal
	m.savedState = m.stateGlobal
	return m
}

// SetInitialState sets the machine's current and saved state. Call this once
// after creating the machine with the language reader's real stateGlobal method.
func (m *Machine) SetInitialState(s StateFn) {
	m.state = s
	m.savedState = s
}

func (m *Machine) stateGlobal(_ string) bool { return false }

// Call processes one token through the current state.
func (m *Machine) Call(tok string) {
	if m.state(tok) {
		m.state = m.savedState
		if m.callback != nil {
			m.callback()
			m.callback = nil
		}
	}
}

// Next sets the current state. If tok is non-empty, it is immediately processed
// through the new state.
func (m *Machine) Next(state StateFn, tok ...string) {
	m.state = state
	if len(tok) > 0 {
		m.Call(tok[0])
	}
}
