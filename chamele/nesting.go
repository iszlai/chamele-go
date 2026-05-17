package chamele

import "strings"

// nestingKind identifies which variant a nestingItem holds.
type nestingKind uint8

const (
	nkBare      nestingKind = iota
	nkNamespace             // named scope: class, package, module
	nkFunction              // a FunctionInfo that opens a new nesting level
)

// nestingItem is the stack element used by NestingStack. It is an unexported
// discriminated union rather than an interface so that FunctionInfo does not
// need to implement extra methods.
type nestingItem struct {
	kind nestingKind
	name string        // valid when kind == nkNamespace
	fn   *FunctionInfo // valid when kind == nkFunction
}

var emptyNesting = nestingItem{kind: nkBare}

func newNamespaceNesting(name string) nestingItem {
	return nestingItem{kind: nkNamespace, name: name}
}

func newFunctionNesting(f *FunctionInfo) nestingItem {
	return nestingItem{kind: nkFunction, fn: f}
}

func (n nestingItem) nameInSpace() string {
	switch n.kind {
	case nkNamespace:
		if n.name != "" {
			return n.name + "::"
		}
		return ""
	case nkFunction:
		return n.fn.Name + "."
	default:
		return ""
	}
}

// Namespace is a named scope (class, struct, module, package).
type Namespace struct {
	Name string
}

// NestingStack tracks the stack of scopes entered during file analysis.
// FileInfoBuilder embeds *NestingStack so callers can call these methods directly.
type NestingStack struct {
	stack           []nestingItem
	pendingFunction *FunctionInfo
}

// WithNamespace returns name qualified by all enclosing scope names.
// Mirrors Python's NestingStack.with_namespace.
func (ns *NestingStack) WithNamespace(name string) string {
	var b strings.Builder
	for _, item := range ns.stack {
		b.WriteString(item.nameInSpace())
	}
	b.WriteString(name)
	return b.String()
}

// AddBareNesting pushes an anonymous nesting level (e.g. a bare block {}).
func (ns *NestingStack) AddBareNesting() {
	ns.stack = append(ns.stack, ns.createNesting())
}

// AddNamespace pushes a named scope (class, namespace token, etc.).
func (ns *NestingStack) AddNamespace(name string) {
	ns.pendingFunction = nil
	ns.stack = append(ns.stack, newNamespaceNesting(name))
}

// StartNewFunctionNesting registers f as the pending nesting entry for the
// next AddBareNesting call.
func (ns *NestingStack) StartNewFunctionNesting(f *FunctionInfo) {
	ns.pendingFunction = f
}

func (ns *NestingStack) createNesting() nestingItem {
	tmp := ns.pendingFunction
	ns.pendingFunction = nil
	if tmp != nil {
		return newFunctionNesting(tmp)
	}
	return emptyNesting
}

// PopNesting removes and returns the top nesting entry.
// Returns nil if the stack is empty.
func (ns *NestingStack) PopNesting() *nestingItem {
	ns.pendingFunction = nil
	if len(ns.stack) == 0 {
		return nil
	}
	top := ns.stack[len(ns.stack)-1]
	ns.stack = ns.stack[:len(ns.stack)-1]
	return &top
}

// CurrentNestingLevel returns the current stack depth.
func (ns *NestingStack) CurrentNestingLevel() int { return len(ns.stack) }

// LastFunction returns the innermost FunctionInfo on the stack, or nil.
func (ns *NestingStack) LastFunction() *FunctionInfo {
	for i := len(ns.stack) - 1; i >= 0; i-- {
		if ns.stack[i].kind == nkFunction {
			return ns.stack[i].fn
		}
	}
	return nil
}
