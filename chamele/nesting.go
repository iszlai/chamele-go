package chamele

// Namespace represents a named nesting level (e.g. class, namespace).
type Namespace struct {
	Name string
	Type string
}

// NestingStack tracks nesting levels during file analysis.
type NestingStack struct {
	stack []Namespace
}

// Push adds a new nesting level.
func (ns *NestingStack) Push(n Namespace) { ns.stack = append(ns.stack, n) }

// Pop removes the top nesting level and returns it.
func (ns *NestingStack) Pop() (Namespace, bool) {
	if len(ns.stack) == 0 {
		return Namespace{}, false
	}
	top := ns.stack[len(ns.stack)-1]
	ns.stack = ns.stack[:len(ns.stack)-1]
	return top, true
}

// Depth returns the current nesting depth.
func (ns *NestingStack) Depth() int { return len(ns.stack) }
