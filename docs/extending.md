# Writing a chamele extension

A chamele extension observes — and optionally transforms — the token stream
the analyzer feeds through each source file. Extensions can record their own
per-function metrics on `FunctionInfo.Ext`, contribute output columns, run a
post-analysis pass over every file, and print their own summary blocks.

This document walks through building one end-to-end. We'll use the (real)
`ext/exitcount` package as the running example because it covers all of the
moving parts in under 60 lines.

## The contract

```go
type Extension interface {
    Name() string

    // OrderingIndex controls pipeline position relative to the built-in
    // processors (Preprocessing → CommentCounter → LineCounter →
    // TokenCounter → ConditionCounter). Negative values run before the
    // built-ins; 0 and up run after. Default for new extensions: 1000.
    OrderingIndex() int

    // Process wraps the incoming token sequence. ctx.CurrentFunction is
    // kept up to date by the language reader's state machines, which run
    // interleaved with extensions on every token.
    Process(tokens iter.Seq[string], ctx *FileInfoBuilder) iter.Seq[string]

    // FunctionInfoColumns adds columns to the tabular / CSV output.
    FunctionInfoColumns() []ColumnSpec
}
```

Two optional interfaces exist on top of this:

- `CrossFileExtension` — adds `CrossFileProcess([]FileInformation)` for
  cross-file passes (fan-in/out, duplicate detection).
- `Printer` — adds `PrintResult(io.Writer)` for free-form summary output
  (e.g. the duplicate-block listing).

## Step 1: package skeleton

```go
// Package myext counts how many times a function does <something>.
package myext

import (
    "iter"

    "github.com/iszlai/chamele-go/chamele"
)

const Key = "my_metric"

func init() { chamele.RegisterExtension(New()) }

type ext struct{}

func New() chamele.Extension { return &ext{} }

func (e *ext) Name() string       { return "myext" }
func (e *ext) OrderingIndex() int { return 1000 }
```

The blank-import in `ext/all/all.go` is what makes your extension show up
when callers do `_ "github.com/iszlai/chamele-go/ext/all"`. Add a line for
your package there too.

## Step 2: record a per-function metric

Per-function extension data lives on `FunctionInfo.Ext` (a `map[string]any`),
not on the struct itself. Pick a stable key and stick to it.

```go
func (e *ext) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
    return func(yield func(string) bool) {
        for tok := range tokens {
            fn := ctx.CurrentFunction
            if fn.Ext == nil {
                fn.Ext = make(map[string]any)
            }
            if tok == "return" {
                n, _ := fn.Ext[Key].(int)
                fn.Ext[Key] = n + 1
            }
            if !yield(tok) {
                return
            }
        }
    }
}
```

Two rules:

1. **Always pass tokens through.** Yield every token. The next stage in the
   pipeline (and the reader's state machines) need them.
2. **Bail out on a `false` yield.** When the consumer stops early — e.g. on
   a "GENERATED CODE" comment — `yield` returns `false`. Don't keep going.

## Step 3: expose a column

```go
func (e *ext) FunctionInfoColumns() []chamele.ColumnSpec {
    return []chamele.ColumnSpec{{
        Header: " my_metric ",
        Value: func(f *chamele.FunctionInfo) any {
            if f.Ext != nil {
                if v, ok := f.Ext[Key]; ok {
                    return v
                }
            }
            return 0
        },
    }}
}
```

## Step 4 (optional): cross-file pass

If your metric depends on knowing every function in the corpus — fan-in,
duplicate detection, dependency graphs — implement `CrossFileProcess`:

```go
func (e *ext) CrossFileProcess(files []chamele.FileInformation) []chamele.FileInformation {
    // Walk files, mutate FunctionInfo.Ext or top-level counters as needed.
    return files
}
```

`Analyze` calls it on the main goroutine after every per-file analysis has
finished, so you see the files in deterministic order.

## Step 5 (optional): summary output

```go
func (e *ext) PrintResult(w io.Writer) error {
    fmt.Fprintln(w, "==== myext summary ====")
    return nil
}
```

Implementations of `Printer` are not invoked automatically — wire them up in
your binary or test harness via type-assertion:

```go
for _, x := range chamele.RegisteredExtensions() {
    if p, ok := x.(chamele.Printer); ok {
        _ = p.PrintResult(os.Stdout)
    }
}
```

## Ordering caveats

- Extensions with `OrderingIndex < 0` run **before** the built-in processors,
  which means they see tokens *before* comments are stripped. The `cpre`
  extension does this so it can rewrite C-preprocessor lines.
- Extensions with `OrderingIndex >= 0` run after, on the cleaned token
  stream. This is the right default for almost everything.
- Inside each bucket, extensions run in the order `RegisteredExtensions()`
  returns them — currently registration order. If your extension depends on
  another running first, register it after.

## Testing

Construct a `FileAnalyzer` with only your extension to keep the test scope
narrow:

```go
a := chamele.NewFileAnalyzerWithExts([]chamele.Extension{New()})
fi := a.AnalyzeSourceCode("t.c", src, clike.NewCLikeReader())
```

See `ext/mccabe/mccabe_test.go` and `ext/nd/nd_test.go` for working
templates.

## Reference: built-in extensions

| Package              | What it measures                                | Status |
|----------------------|-------------------------------------------------|--------|
| `ext/boolcount`      | Count of boolean operators per function         | done   |
| `ext/duplicate`      | Cross-file duplicate-block detection            | done   |
| `ext/dumpcomments`   | Lists every comment encountered                 | done   |
| `ext/exitcount`      | Number of `return`/`exit` per function          | done   |
| `ext/gotocount`      | Number of `goto` per function                   | done   |
| `ext/ignoreassert`   | Drop CCN inside `assert(...)`                   | done   |
| `ext/io`             | Fan-in, fan-out, general fan-out                | done   |
| `ext/mccabe`         | Strict McCabe CCN (folds case fall-through)     | done   |
| `ext/modified`       | Modified CCN (switch/case counts as 1)          | done   |
| `ext/nd`             | Maximum nesting depth                           | done   |
| `ext/ns`             | Maximum nested control structures               | done   |
| `ext/outside`        | Adds the `*global*` pseudo-function to output   | done   |
| `ext/statementcount` | Count of statements per function                | done   |
| `ext/complextags`    | Stub — slated for deletion ([CLEANUP.md][c] §A) | stub   |
| `ext/cpre`           | Stub — slated for deletion ([CLEANUP.md][c] §A) | stub   |
| `ext/dependencycount`| Stub — slated for deletion ([CLEANUP.md][c] §A) | stub   |
| `ext/dupparams`      | Stub — slated for deletion ([CLEANUP.md][c] §A) | stub   |
| `ext/nonstrict`      | Stub — slated for deletion ([CLEANUP.md][c] §A) | stub   |
| `ext/wordcount`      | Stub — slated for deletion ([CLEANUP.md][c] §A) | stub   |

[c]: ../CLEANUP.md

## Upcoming changes — read before writing a new extension

The cleanup plan ([`CLEANUP.md`](../CLEANUP.md) Phase D) will introduce a
small but **breaking** change to the extension contract:

1. `RegisterExtension(ext)` becomes `RegisterExtensionFactory(name, func() Extension)`.
   Extensions are constructed once per `chamele.Analyze` call rather than
   living for the whole process. The current registration form keeps working
   one release as a deprecated wrapper.
2. `OrderingIndex int` becomes `Phase Phase` (`PhasePreBuiltins` /
   `PhasePostBuiltins`). Numeric ordering inside a phase is rarely needed.
3. `Printer.PrintResult` is auto-invoked by the engine; the type-assertion
   loop shown earlier on this page disappears.
4. Counter boilerplate (the `if fn.Ext == nil { fn.Ext = make(...) }; if _, ok …`
   six-liner) is replaced by `ctx.CurrentFunction.Inc(key, delta)` and
   `(*FunctionInfo).GetInt(key)`.

If you're writing a new extension *today*, follow this guide as-is — the
deprecated wrappers will keep it compiling. If you can wait for the Phase D
PR to land, you'll skip the migration step.
