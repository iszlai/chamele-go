package tokenizer

// BraceTracker counts `{` and `}` and remembers the brace depths at which
// each currently-open function body began. Language readers that drive a
// hand-rolled state machine embed (or own) a BraceTracker and forward `{`
// and `}` tokens to OnOpen / OnClose. When stepping into a function's body,
// they call EnterFunction; the next `}` that returns to that recorded depth
// runs the close callback and closes the function.
//
// This replaces ~10 lines of duplicated code in every reader that uses the
// pattern (golang, kotlin, scala, rust, javascript, swift, zig, solidity,
// php, perl, ruby).
type BraceTracker struct {
	depth int
	fns   []int
}

// Depth returns the current brace depth.
func (b *BraceTracker) Depth() int { return b.depth }

// OnOpen records a `{`. Call it from the state machine's `{` case in the
// state(s) that should observe nesting (i.e. don't double-count when the
// `{` opens a function body — see EnterFunction below).
func (b *BraceTracker) OnOpen() { b.depth++ }

// OnClose records a `}`. If the close brings depth back to the recorded
// depth of the innermost open function, close is invoked and that
// function's record is popped.
func (b *BraceTracker) OnClose(close func()) {
	b.depth--
	if n := len(b.fns); n > 0 && b.depth == b.fns[n-1] {
		close()
		b.fns = b.fns[:n-1]
	}
}

// EnterFunction records that the next `{` already consumed opens a
// function body — its matching `}` should fire the close callback passed
// to OnClose. EnterFunction itself increments depth (treating the `{` as
// already seen), so callers do NOT call OnOpen for that brace.
func (b *BraceTracker) EnterFunction() {
	b.fns = append(b.fns, b.depth)
	b.depth++
}
