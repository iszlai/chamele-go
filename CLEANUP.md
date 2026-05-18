# chamele-go — Cleanup & Idiomatic-Go Refactor Plan

Status: **landed (A–H)** · Owner: arch-cleanup branch · Target: incremental, behaviour-preserving

This document is the output of a top-to-bottom code review of the repository
as it stood at `7e15b85`. Phases A–H landed on the `arch-cleanup` branch in
May 2026. Items flagged "deferred" / "optional" inside individual phase
sections remain open as follow-up PRs.

It is meant to be read alongside [`PLAN.md`](./PLAN.md) (which tracks the
*porting* roadmap) — `CLEANUP.md` is the *quality* roadmap.

> Every recommendation here is justified with a file:line reference. Where a
> change touches public API (anything outside `internal/`) the phase is
> flagged **breaking** and routed through a deprecation step.

---

## 0. Executive summary

The port is functionally complete (27 readers, 19 extensions, 7 output
formats, BDD harness, parity test). What it lacks is the second pass an
idiomatic-Go codebase needs:

| Smell | Where | Impact |
|---|---|---|
| Dead code from an abandoned `BaseReader` abstraction | `internal/tokenizer/reader.go` | Confuses extension authors; 100 LOC of fiction |
| Dead helpers on `tokenizer.Machine` (`SubState`, `NextIf`, `Return`, `BrCount`, `RutTokens`) | `internal/tokenizer/statemachine.go` | Same |
| Brace-depth + funcDepths boilerplate copy-pasted across **11 readers** | `languages/{golang,kotlin,scala,rust,javascript,swift,zig,solidity,php,perl,ruby}/*.go` | ~150 LOC of duplication, drift risk |
| `isAlpha` / `isSpace` / `isHSpace` redefined in 10–13 files | `languages/**`, `chamele/processors.go` | Easy to forget when adding the 28th language |
| Three independent metric-name → value switches | `chamele/{warnings,output_scheme}.go`, `output/tabular.go`, `output/checkstyle.go` | Bugs likely when a metric is added |
| Six stub extensions registered as if real | `ext/{wordcount,dupparams,cpre,dependencycount,complextags,nonstrict}` | False contract — users see them in `--extension` listings, get nothing |
| Singleton extensions (`var ext = &dupExt{}`) carry state across `Analyze()` calls | `ext/duplicate/duplicate.go:38`, `ext/io/io.go`, `ext/boolcount/boolcount.go` | Library callers calling `Analyze` twice get wrong numbers |
| `chamele.Options` half-wired: `IgnoreWarnings` never read, `WithLanguages`/`WithThreads`/`WithExclude`/`WithWhitelist` silently ignored by `AnalyzeFile` | `chamele/options.go:8`, `chamele/analyze.go:25` | Caller astonishment |
| Custom extensions cannot be passed to `Analyze` — everyone uses the global registry | `chamele/analyze.go:64` | Bad for testability and library embedding |
| `ND` extension stores its result on both `fn.MaxNestingDepth` and `fn.Ext["max_nesting_depth"]` | `ext/nd/nd.go:127–129` | Inconsistent with peers |
| Six known metrics live on `FunctionInfo` as fields; the rest in `FunctionInfo.Ext map[string]any` with `.(int)` type assertions everywhere | `chamele/funcinfo.go:11`, every extension | Stringly-typed dispatch |
| `output/tabular.go` shadows the Go 1.21 builtin `max` | `output/tabular.go:171` | Lints clean today, papercut tomorrow |
| `chamele/walk.go` does per-file `filepath.Rel` on every gitignore frame and trims frames with `rel[:2] == ".."` | `chamele/walk.go:50–57` | Fragile and quadratic-ish on deep trees |

None of these are correctness blockers individually (the parity test
guards behaviour). But every new language reader, every new metric, every
new output format pays the duplication tax. The fixes below pay it down in
small, reviewable PRs.

---

## 1. Architectural targets

The goal is **fewer concepts, used consistently**.

```
┌────────────────────────────────────────────────────────────────────┐
│                      analysis engine (chamele)                     │
│                                                                    │
│   Reader  ──► Tokenize ──► [Preprocess → Comment → Line → Token    │
│                              → Condition] ──► Reader.RunTokens     │
│                                       ▲                            │
│                                       └─ Extensions (interleaved)  │
│                                                                    │
│   Output:  FileInformation + Functions, with cross-file pass       │
└────────────────────────────────────────────────────────────────────┘
                              ▲
                              │
        ┌─────────────────────┴─────────────────────┐
        │                                           │
   languages/                                    ext/
   (one Reader per language;                  (one Extension per metric,
    composable state machines)                 with explicit lifecycle)
        │                                           │
        └────────────► internal/tokenizer ◄─────────┘
                       (regex + state machine + brace-tracker + helpers)
                                  │
                                  ▼
                       internal/stringx
                       (isAlpha, isSpace, ReadFile, BOM, LF normalise)
```

The arrows are present today. What's missing is **shared scaffolding** at the
two yellow spots (extension lifecycle, reader-state-machine composition).

---

## 2. Findings by package

Each finding has: **where**, **why it's bad**, **what to do**.

### 2.1 `internal/tokenizer/`

#### F-1. `BaseReader` is dead.
**Where:** `internal/tokenizer/reader.go:25–82`.
**Why:** Only `CLikeReader` embeds it (`languages/clike/clike.go:20`) and never
reads any field of it. `ParallelStates`, `Conditions`, `ControlFlowKeywords`,
`LogicalOperators`, `CaseKeywords`, `TernaryOperators` are populated by
`NewBaseReader()` and never accessed. `RunParallelStates`, `EOF`, `FilterTokens`,
and the standalone `Conditions` / `DefaultConditions` types are unreferenced
outside of this file.
**Do:** Delete `BaseReader`, `NewBaseReader`, `RunParallelStates`, `EOF`,
`FilterTokens`, `Conditions`, `DefaultConditions`. Remove the `BaseReader`
embed from `CLikeReader`. Move `isSpace` (line 97) to `internal/stringx`.

#### F-2. `Machine` carries unused state.
**Where:** `internal/tokenizer/statemachine.go:23–27`, `:63–88`.
**Why:** `BrCount`, `RutTokens`, `ToExit` are public fields that no caller ever
sets or reads. The helpers `ReadInsideBracketsThen` and `ReadUntilThen`
(helpers.go) use their own local closures instead.
`SubState`, `NextIf`, `Return`, `LastToken` are equally unused outside the
tokenizer package.
**Do:** Trim `Machine` to `{state, savedState, callback}` plus `Call` /
`Next` / `SetInitialState`. `LastToken` is rebuildable from the caller side
(the Go reader already does this, `golang.go:62`); delete the field.

#### F-3. `StatemachineBeforeReturn` is a misleading no-op.
**Where:** `internal/tokenizer/statemachine.go:92`, called at
`languages/r/r.go:52` and `languages/clike/clike.go:142`.
**Why:** The method is defined on `*Machine` and does nothing. Subclassing
in Go via "override this method on your wrapper struct" doesn't work the
way the comment suggests — the override on `rMachine.StatemachineBeforeReturn`
(`r.go:148`) is never invoked because `m` has type `*tokenizer.Machine`.
**Do:** Delete the method and both call sites. If end-of-stream hooks are
needed, add an explicit `OnEnd(func())` to `Machine`.

#### F-4. `tokenizer.go` is fine; leave it.
**Where:** `internal/tokenizer/tokenizer.go`.
**Why:** This is the load-bearing code (regex assembly, macro accumulation).
Minor: the leading-`|` stripping on line 58–60 is a workaround for a
calling convention; document it as the convention.

---

### 2.2 `languages/`

#### F-5. Brace-depth tracking is copy-pasted 11 times.
**Where:** `golang.go:56–88`, `kotlin.go:60–88`, `scala.go:52–80`,
`rust.go:51–84`, `javascript.go:60–137` (mixed in with name-resolution),
`swift.go:55–92`, `zig.go:49–78`, `solidity.go:44–77`,
`php.go:117–146`, `perl.go:76–112`, `ruby.go:60–97`.

The pattern is always:
```go
type fooMachine struct {
    ...
    braceDepth int
    funcDepths []int
}

case "{":
    s.braceDepth++
case "}":
    s.braceDepth--
    if len(s.funcDepths) > 0 && s.braceDepth == s.funcDepths[len(s.funcDepths)-1] {
        s.ctx.EndOfFunction()
        s.funcDepths = s.funcDepths[:len(s.funcDepths)-1]
    }
```
plus the matching push at `stateEnteringImpl`.

**Do:** Extract to `internal/tokenizer.BraceTracker`:

```go
type BraceTracker struct {
    depth int
    fns   []int
}

func (b *BraceTracker) OnOpen()                              { b.depth++ }
func (b *BraceTracker) OnClose(end func()) {
    b.depth--
    if n := len(b.fns); n > 0 && b.depth == b.fns[n-1] {
        end()
        b.fns = b.fns[:n-1]
    }
}
func (b *BraceTracker) EnterFunction()                       { b.fns = append(b.fns, b.depth); b.depth++ }
func (b *BraceTracker) Depth() int                           { return b.depth }
```

Each reader embeds one. Saves ~10 LOC per reader (× 11 readers ≈ 110 LOC)
and gives a single place to add a future "leaked function on EOF" pass.

#### F-6. Function-detection skeleton is copy-pasted across simple-keyword readers.
**Where:** Zig, Kotlin, Scala, Swift, Solidity, PHP. Every one of them has the
shape:

```
stateGlobal       — switch on keyword → PushNewFunction("") → stateFunctionName
stateFunctionName — name? "(" ? "<" → handle, default → AddToFunctionName, next
stateExpectFunctionDec — wait for "(" → stateFunctionDec
stateFunctionDec  — ReadInsideBracketsThen("(", ")") → push to params
stateExpectFunctionImpl — wait for "{" → stateEnteringImpl
stateEnteringImpl — record depth, ++depth, return to global
```

**Do:** Introduce a `BraceKeywordReader` in `languages/clike/` (or a new
`languages/braceish/` shared package):

```go
type BraceKeywordReader struct {
    Keyword          string                       // "func", "fn", "fun", "def" …
    Conditions       map[string]struct{}
    LangAddition     string                       // regex addition passed to tokenizer
    PostNameSkippers []SkipRule                   // generics "<...>" etc.
    ParamFilter      func(tok string) bool        // tokens that count as params
    AnonymousName    string                       // "(anonymous)" or ""
}
```

A single state machine drives all of them. Per-language quirks live in the
config struct, not in copy-pasted state machines. Saves ~600 LOC across
those six readers. Reader implementations shrink to 20–40 lines.

The "weird" readers (Go's receiver / member-function split, JS's
LHS-name resolution, Rust's deferred PushNewFunction, Perl's package
qualification) keep their bespoke machines but reuse `BraceTracker`.

#### F-7. `isAlpha` / `isSpace` / `isHSpace` redefined everywhere.
**Where:** 10 files define `isAlpha`, 13 files define `isSpace` or `isHSpace`.
Implementations are textually identical except `golang.go` and `perl.go`
which both use `strings.TrimLeft` style — same semantics.
**Do:** Move all three to `internal/stringx`:
```go
func IsAlpha(b byte) bool         // ASCII letter
func IsHSpace(s string) bool      // non-empty, all of [ \t\r]
func IsLeadingSpace(s string) bool // == IsHSpace
```
Drop the per-file copies.

#### F-8. `GetConditions()` returns ad-hoc map literals.
**Where:** every reader.
**Why:** Slight duplication (C-like sets reappear in clike, java, scala,
solidity), and `Has(tok)` is open-coded as `_, ok := conds[tok]; ok`.
**Do:** A `stringset` type in `internal/stringx`:
```go
type Set map[string]struct{}
func NewSet(s ...string) Set { /* … */ }
func (s Set) Has(tok string) bool { _, ok := s[tok]; return ok }
```
plus a `tokenizer.CLikeConditions` constant for the share-this-one cases.
Each reader's `GetConditions()` becomes a single literal.

#### F-9. `BaseReader` embed in `CLikeReader` is misleading.
**Where:** `languages/clike/clike.go:20`.
**Why:** As established in F-1, the embed contributes nothing. But readers
that say `*clike.CLikeReader` (javascript, java, kotlin, scala, swift,
solidity, php, ttcn) inherit the dead methods, which is confusing for
anyone reading the code.
**Do:** Remove the embed (covered by F-1).

#### F-10. Python-family indentation logic duplicated.
**Where:** `python.go:56–121`, `gdscript.go:43–105`. The `pythonIndents` /
`gdIndents` types are byte-for-byte equivalent.
**Do:** Move `Indents` + `Preprocess` to a shared `languages/indented/`
package. Python and GDScript embed it and provide the comment-prefix.

#### F-11. R's broken `StatemachineBeforeReturn`.
**Where:** `languages/r/r.go:52`, `:148`.
**Why:** Calls the no-op base method, not `rMachine`'s implementation
(see F-3). If r's state machine has end-of-file cleanup that needs to run,
this hides the bug.
**Do:** Switch r to an explicit `defer m.flush()` pattern (or use the
`OnEnd` hook proposed in F-3).

---

### 2.3 `chamele/` (the engine)

#### F-12. Three analysis entry points with diverging behaviour.
**Where:**
- `analyze.go:23` `AnalyzeFile(path, opts...)` — applies options *but* throws them away (`_ = applyOptions(opts)`), reads via `stringx.ReadFile`, builds a `NewFileAnalyzer()` (so global extensions only).
- `analyze.go:47` `analyzeFiles` — applies options for threads, ignores `Languages`/`Whitelist`, builds one shared `NewFileAnalyzer()`.
- `analyzer.go:32` `FileAnalyzer.AnalyzeFile` — does its own read, picks reader, writes its own error to stderr.

Three near-duplicate read paths. Users who pass `WithLanguages("go")` to
`Analyze` get filtering at the walk stage; users who call `AnalyzeFile`
single get no filtering, no whitelist, no custom-extension hook.

**Do:** Collapse to one Engine type:

```go
type Engine struct {
    procs []Processor
    exts  []Extension
    opts  Options
}

func New(opts ...Option) *Engine
func (e *Engine) AnalyzePaths(paths []string) ([]FileInformation, error)
func (e *Engine) AnalyzeFile(path string) (*FileInformation, error)
func (e *Engine) AnalyzeSource(path string, src []byte, r languages.Reader) *FileInformation
```

Free functions `Analyze`, `AnalyzeFile`, `AnalyzeFiles` become thin wrappers
that build an Engine, so the v0 API stays.

#### F-13. `Options` fields silently dropped.
**Where:** `chamele/options.go:8` (`IgnoreWarnings`), `chamele/analyze.go:25`
(`_ = applyOptions(opts)`), `chamele/analyze.go:50` (`Languages` and `Whitelist`
go unused inside `analyzeFiles`).
**Do:** With F-12 done, every field is read from `e.opts` or removed:
- `IgnoreWarnings` is a CLI concern — move to the cmd package and delete
  from `Options`.
- `Whitelist` is an output concern (already used in `output/tabular.go:19`);
  delete the chamele option, keep only the output path. CLI passes the
  string directly.
- `Languages` and `Exclude` filtering happens in walk.go — wire them
  through the Engine properly.

#### F-14. Custom extensions can't be injected into the public `Analyze`.
**Where:** `chamele/analyze.go:64` always calls `NewFileAnalyzer()` which
uses the global registry.
**Why:** Library callers who want to test a single extension can't —
they have to import `ext/all`, which pulls in 19 extensions.
**Do:** `WithExtensions(...Extension) Option`. When set, overrides the
global registry. (Covered by F-12.)

#### F-15. Singleton extensions hold cross-call state.
**Where:**
- `ext/duplicate/duplicate.go:38` `var ext = &dupExt{}`
- `ext/io/io.go` no singleton but the CrossFileProcess writes back to `FunctionInfo` pointers, so within a process the data is fine — but `ioStructures` / `ioPunct` package-level maps are fine; the issue is `boolcount.totalBool` / `totalToken` counters survive an `Analyze` run.
- `ext/boolcount/boolcount.go:14–17` counters are instance-scoped but the
  extension instance is registered globally for the lifetime of the
  process.

**Do:** Switch from "extensions are long-lived singletons" to
"extensions are factories":

```go
type ExtensionFactory func() Extension
func RegisterExtensionFactory(name string, f ExtensionFactory)
```

`Engine.New(...)` instantiates each factory once per run. Singletons that
*need* persistence (none of the current ones do) can opt in via a
documented pattern. This also makes `Engine` reusable from the same
program.

#### F-16. Three places re-implement metric-name dispatch.
**Where:**
- `chamele/warnings.go:44` `metricValue`
- `chamele/output_scheme.go:81` `functionFieldStr`
- `output/tabular.go:155` `metricVal`
- `output/checkstyle.go:22` references `metricVal`

Each one switches on a string and returns the right field. Adding a new
metric requires editing all three.

**Do:** A single registry on `FunctionInfo`:

```go
type Metric struct {
    Name    string
    Aliases []string
    Get     func(*FunctionInfo) int
    Format  func(*FunctionInfo) string   // for tabular alignment
}

var Metrics = struct {
    NLOC, CCN, TokenCount, ParameterCount, Length, MaxNestingDepth Metric
}{ /* … */ }

func MetricByName(name string) (Metric, bool)
```

All three switches become `m, _ := MetricByName(t.Metric); val := m.Get(fn)`.

#### F-17. `OutputScheme` is stringly-typed reflection-lite.
**Where:** `chamele/output_scheme.go:31` builds `ColumnItem` whose `Value`
field is "the FunctionInfo field name", then `functionFieldStr` switches on
that name to format it.
**Do:** Replace `ColumnItem.Value string` with `ColumnItem.Get func(*FunctionInfo) string`.
Built-in columns get their getters from F-16's registry. Extension columns
already define `Value func(*FunctionInfo) any` (`extension.go:11`) — unify
the two so `ColumnSpec` and `ColumnItem` are the same type.

#### F-18. `FileInformation.WordCount` is dead.
**Where:** `chamele/fileinfo.go:9`. Set by no one, read by no one.
**Do:** Delete the field. If the `wordcount` extension (currently a stub)
ever ships, it stores on `Ext` like every other extension.

#### F-19. `FileInfoBuilder.Newline` is dead.
**Where:** `chamele/builder.go:16`, `:31`, `:80`. Written but never read.
**Do:** Delete the field and its writes.

#### F-20. `Namespace` struct in `nesting.go` is dead.
**Where:** `chamele/nesting.go:48–50`.
**Do:** Delete.

#### F-21. `ND` extension stores its result twice.
**Where:** `ext/nd/nd.go:127` writes to `fn.MaxNestingDepth`, `:129` also writes
to `fn.Ext["max_nesting_depth"]`.
**Why:** Inconsistent with peers. `FanIn/FanOut` from the `io` extension
live on the struct as fields. `exit_count` from `exitcount` lives on `Ext`.
There's no rule.
**Do:** Pick a rule and apply it:
> Metrics consumed by built-in code (warnings filter, sorting, tabular
> headers) live as **fields on `FunctionInfo`**. Extension-private state
> lives on `Ext`.

That keeps the type assertions out of the hot path. Audit: `MaxNestingDepth`,
`FanIn`, `FanOut`, `GeneralFanOut`, `CyclomaticComplexity`, `NLOC`,
`TokenCount` stay as fields. `exit_count`, `goto_count`, `statement_count`,
`max_nested_structures`, `duplicate_tokens`, `io_tokens`, `nd_depth` /
`nd_in_cond` / `nd_cond_depth` / `nd_log_added` / `nd_prev_else` move to `Ext`
or remain there.

Also: ND's `keyDepth`, `keyMax`, `keyPrevElse`, `keyInCond`, `keyCondDep`,
`keyLogAdded` are six map keys storing what should be a single per-function
struct on `Ext`. One key, one struct value.

#### F-22. Walk gitignore frame trimming is fragile.
**Where:** `chamele/walk.go:50–57`. The frame is popped when the relative
path starts with `..`, computed via `filepath.Rel` per file per frame.
**Why:** Quadratic in tree depth. The string-prefix check
`rel[:2] == ".."` panics on a 1-character `rel`. Mac/Windows path
separators add a `\\` case.
**Do:** Track the directory tree via `filepath.WalkDir`'s natural
parent-pointer (we already have `path` — compare directory components
instead of rebuilding relative paths). Or use `len(absPath) <
len(frame.absDir)` as the "I've left the subtree" test.

#### F-23. `md5File` reads the file twice on every walk.
**Where:** `chamele/walk.go:96` — to compute the dedup hash chamele reads
the file, then `analyzer.AnalyzeFile` reads it again to tokenise.
**Do:** Cache the bytes from the dedup pass and pass them through, or
fingerprint by `(size, mtime, dev, inode)` from `os.Stat`.

#### F-24. `analyzeFiles` channel for one-shot index hand-out is overkill.
**Where:** `chamele/analyze.go:58–62`.
**Do:** Use `sync/atomic.Int64` as a counter. Or — since the list is
known up-front — split it into `workers` chunks and give one chunk to each
goroutine. Removes the channel allocation.

---

### 2.4 `ext/`

#### F-25. Six stub extensions register but do nothing.
**Where:** `ext/wordcount/wordcount.go`, `ext/dupparams/dupparams.go`,
`ext/cpre/cpre.go`, `ext/dependencycount/dependencycount.go`,
`ext/complextags/complextags.go`, `ext/nonstrict/nonstrict.go`.
Each is ~22 lines of "return tokens unchanged".
**Why bad:** Users who pass `--extension wordcount` see no output and no
error. The stubs are listed in `ext/all/all.go` as real.
**Do:** Two options:
1. **Delete the packages.** Remove from `ext/all`. When implemented, add
   them back. (Matches what `PLAN.md` says is the v1.0 goal anyway.)
2. **Keep but exclude from `ext/all`.** Move to `ext/incomplete/` with a
   doc comment saying they're stubs.

Recommend (1) — the porting plan tracks them in Appendix C.

#### F-26. Extension boilerplate is heavy.
**Where:** every ext package. Six lines just to read/write an `int` on
`fn.Ext`:
```go
fn := ctx.CurrentFunction
if fn.Ext == nil {
    fn.Ext = make(map[string]any)
}
if _, ok := fn.Ext[key]; !ok {
    fn.Ext[key] = 0
}
if tok == "goto" {
    fn.Ext[key] = fn.Ext[key].(int) + 1
}
```
**Do:** Helpers on `FileInfoBuilder` or `FunctionInfo`:
```go
func (f *FunctionInfo) Counter(key string) *int  // returns pointer, creates if missing
func (f *FunctionInfo) GetInt(key string) int
func (f *FunctionInfo) Inc(key string, delta int)
```
The whole `gotocount.go` body becomes:
```go
if tok == "goto" { ctx.CurrentFunction.Inc(key, 1) }
```
Same for `exitcount`, `statementcount`, `boolcount`. ~80 LOC removed.

#### F-27. Pre/post extension split via `OrderingIndex < 0` is hidden magic.
**Where:** `chamele/analyzer.go:51–67`.
**Do:** Replace with an explicit enum:
```go
type Phase int
const (
    PhasePreBuiltins  Phase = iota   // raw tokens
    PhasePostBuiltins                // after comment/line/token/condition
)
func (e Extension) Phase() Phase
```
Numeric ordering inside a phase stays via a separate `Priority` field if
ever needed. Reads better than `-1 == before`.

#### F-28. `Printer` is not auto-invoked.
**Where:** `chamele/extension.go:36–39`.
**Why:** Documented as "not invoked automatically — wire up in your
binary". `cmd/chamele/main.go` doesn't wire it. Users who add a `boolcount`
column get no `boolcount` summary unless they implement the loop themselves.
**Do:** Have the Engine invoke `Printer` implementations at end of run,
ordered by registration. If the user wants tabular-only output, pass an
`io.Discard` writer (or a `NoPrint` option).

---

### 2.5 `output/`

#### F-29. `IsEmpty()` filter is open-coded 5 times.
**Where:** every `output/*.go`. Same `for i := range files { fi := &files[i]; if fi.IsEmpty() { continue } ...`.
**Do:** Helper iterator:
```go
func eachFunction(files []chamele.FileInformation, fn func(*chamele.FileInformation, *chamele.FunctionInfo)) {
    for i := range files {
        fi := &files[i]
        if fi.IsEmpty() { continue }
        for _, f := range fi.Functions { fn(fi, f) }
    }
}
```

#### F-30. `output/tabular.go` shadows the Go 1.21 builtin `max`.
**Where:** `output/tabular.go:171`. Also `output/xml.go:67` calls `max`
which currently resolves to the package-local one.
**Do:** Delete the helper; use the builtin.

#### F-31. `metricVal` lives in `output/tabular.go` but is also called from `output/checkstyle.go`.
**Where:** `output/checkstyle.go:22`.
**Why:** Cross-file dependency on an unexported helper is fragile when one
of them moves.
**Do:** Covered by F-16 — both call `chamele.MetricByName(...)`.

#### F-32. Output formatters don't share a common interface.
**Where:** `cmd/chamele/main.go:172–192` is a giant `switch` on output flag.
**Do:** Optional but nice — define `output.Formatter`:
```go
type Formatter interface {
    Render(w io.Writer, files []chamele.FileInformation, opts FormatOptions) (warningCount int, err error)
}
```
and a registry. The CLI then iterates the flag-to-formatter map. Also
makes `--output-file foo.checkstyle` infer-by-extension cleaner.

#### F-33. CSV header is gated on `--verbose`.
**Where:** `output/csv.go:13`. Python lizard always emits headers; chamele
emits them only with `-V`. Minor parity concern.
**Do:** Confirm in `test/parity` and align.

---

### 2.6 `features/`

#### F-34. Seven near-identical step definitions for "a $LANG file containing:".
**Where:** `features/steps/source_steps.go:16–56`.
**Do:** Single parameterised step:
```gherkin
Given a "Go" file containing:
```
```go
sc.Step(`^a "([^"]+)" file containing:$`, func(lang string, src *godog.DocString) error {
    w.Lang = strings.ToLower(lang)
    w.SourceCode = src.Content
    return nil
})
```
Cuts six step regs to one. Then `langExt` can collapse into a single map.

#### F-35. World state reset hardcodes Lang = "go".
**Where:** `features/steps/world.go:21`.
**Do:** Reset to `""`; require the test to call "a $LANG file containing"
first. Avoids silent miscategorisation.

---

### 2.7 `cmd/chamele/`

#### F-36. `cliFlags` struct has 25 fields and the switch on output flag is unstructured.
**Where:** `cmd/chamele/main.go:27–50`, `:172–192`.
**Why:** Cobra encourages a struct, but the switch keeps growing per
formatter.
**Do:** Pair `--output-file foo.ext` with a formatter registry (F-32),
fold the format flags (`--xml`, `--csv`, `--html`, `--checkstyle`, …) into
one `--format` flag with a `--format=xml` default-from-extension. Keep
the old flags as deprecated aliases for one release.

#### F-37. `os.Exit(1)` from inside `run`.
**Where:** `cmd/chamele/main.go:196`.
**Do:** Return a sentinel `errWarningGate`, let `main` translate to exit
code. Plays nicer with tests.

---

## 3. Phased remediation plan

Phases are ordered by **ROI/risk ratio** — each one stands alone and is
landable in a single PR.

> **Status (2026-05-18):** all eight phases landed on the `arch-cleanup`
> branch. Each phase below is annotated with its commit. Follow-up work
> still open is noted at the end of the relevant phase.

### Phase A — dead code & lint hygiene (~1 PR, ~1 day) — **DONE** (`19d9d1d`)

Findings: **F-1, F-2, F-3, F-18, F-19, F-20, F-25, F-30, F-11**.

Safe deletions only. No public-API impact except `chamele.Options.IgnoreWarnings`
which has no external callers (verify by grep before deleting).

Done when:
- `go vet ./...` clean.
- `staticcheck` reports zero unused exports.
- Stub extensions removed from `ext/all`.
- `output/tabular.go` uses the builtin `max`.
- `BENCHMARKS.md` re-run shows no regression (smoke).

### Phase B — extract shared helpers (~1 PR, ~1 day) — **DONE** (`c115a1f`)

Findings: **F-7, F-8, F-29**.

Mechanical refactor. Move `isAlpha`/`isSpace`/`isHSpace` to `internal/stringx`
behind exported wrappers; add `stringx.Set`; add `output/iter.go` with the
`eachFunction` helper. Update call sites.

Done when:
- Every `func isAlpha` / `func isSpace` / `func isHSpace` is deleted from
  `languages/`, `chamele/`, `ext/`.
- BDD + parity green.

### Phase C — metric registry (~1 PR, ~1 day) — **DONE** (`c83d329`)

Findings: **F-16, F-17, F-21**.

Introduce `chamele.Metric` + `chamele.MetricByName`. Refactor
`warnings.go`, `output_scheme.go`, `tabular.go`, `checkstyle.go`. Unify
`ColumnSpec` and `ColumnItem`. Fix ND double-write.

Done when:
- Adding a new built-in metric touches **one** file, not four.
- ND test still passes.

### Phase D — extension lifecycle & helpers (~1 PR, ~2 days) — **DONE** (`aac6a69`)

Findings: **F-15, F-26, F-27, F-28**.

This one is **breaking** for extension authors:
- `RegisterExtension` becomes `RegisterExtensionFactory`.
- `OrderingIndex` becomes `Phase`.
- `Printer` is now auto-invoked.

Keep `RegisterExtension` as a deprecated wrapper that creates a factory
returning the given instance.

Add `FunctionInfo.Inc` / `Counter` helpers. Refactor `gotocount`,
`exitcount`, `statementcount`, `boolcount`.

Done when:
- Extensions are constructed per-Analyze.
- Running `Analyze` twice in one process gives independent metrics.
- `cmd/chamele` invokes Printer summaries automatically.

### Phase E — Engine unification (~1 PR, ~2 days) — **DONE** (`7c0e76f`)

Findings: **F-12, F-13, F-14, F-24**.

The big one. Introduce `chamele.Engine`. Free functions become wrappers.
Options actually drive the engine. `WithExtensions` is real.
`IgnoreWarnings` moves to CLI.

Done when:
- Three read paths collapse to one (`Engine.AnalyzeFile`).
- Library example: "analyze with just the `mccabe` extension" is a 5-liner.

### Phase F — language reader scaffolding (~2 PRs, ~3 days) — **PARTIAL** (`5b0b7b0`, `35809d8`)

PR 1 (BraceTracker) and PR 2 (Indents) landed. F-6 (keyword-driven
BraceKeywordReader for the six simple readers) is deferred — BraceTracker
absorbed most of F-5's duplication, so the remaining ROI is smaller than
originally scoped. Re-evaluate when a 12th language with the same shape
shows up.

Findings: **F-5, F-6, F-9, F-10**.

PR 1: introduce `tokenizer.BraceTracker`, refactor the 11 simple readers.
PR 2: introduce `languages/braceish.Reader` (the keyword-driven config
struct), port Zig + Kotlin + Scala + Swift + Solidity + PHP to it. Leave
Go, Rust, JS, Perl bespoke (their quirks justify the LOC).

Done when:
- `cloc languages/` drops by ~600 LOC.
- Parity test green on every touched language.

### Phase G — walk + I/O polish (~1 PR, ~1 day) — **PARTIAL** (`a85d66e`)

F-22 (gitignore frame trimming) landed. F-23 (md5 double-read) is
deferred: cache-the-bytes is the right fix but it adds memory pressure
for large repos; (dev,ino) dedup would change user-visible semantics (the
existing parity test asserts content-based dedup matching upstream
lizard). Left for a follow-up where we decide whether to diverge from
lizard or accept the read cost.

Findings: **F-22, F-23**.

Walk hardening + single-read for dedup-then-tokenise. Low priority — only
land after the bigger phases settle.

### Phase H — CLI & BDD polish (~1 PR, ~1 day) — **PARTIAL** (`2e2bf35`)

F-34 (parameterised Gherkin step), F-35 (no-default Lang in World.Reset),
and F-37 (CLI exit via sentinel error, not os.Exit) landed. F-32
(Formatter registry) and F-36 (collapse --xml/--csv/... into --format)
are deferred — they were flagged "optional but nice" in the original
write-up and the existing switch is small enough that adding a registry
would expand the public API without a corresponding readability win.

Findings: **F-32, F-34, F-35, F-36, F-37**.

Formatter registry + one Gherkin step for all languages. CLI exit path via
return value.

---

## 4. Out of scope

- The `lizard-upstream/` tree. Read-only by policy.
- Performance work beyond what the cleanup naturally yields. `BENCHMARKS.md`
  numbers are already competitive (parity §8); no need for micro-tuning
  until a profile says so.
- The `parity` test corpus itself. The infra is fine; if a language
  diverges (`divergences.md`), fix the language, not the harness.

---

## 5. Verification matrix

Every phase must pass:

| Gate | Command |
|---|---|
| Build | `go build ./...` |
| Vet | `go vet ./...` |
| Lint | `golangci-lint run` |
| Unit | `go test ./...` |
| BDD | `go test ./features/...` |
| Parity (optional) | `go test -tags parity ./test/parity/...` |
| ik quality gate | CI workflow `ik.yml` (score ≥ 50, see `.ik.yaml`) |

For Phase D and E (breaking) add: a single-line CHANGELOG entry, a
`Deprecated:` godoc on every renamed symbol, and at least one example in
`examples/` that uses the new API.

---

## 6. Anti-goals

In keeping with the project's CLAUDE.md guidance:

- **No new abstractions for hypothetical needs.** `BraceTracker` is
  justified by 11 call sites. A `Reader` factory is justified by F-15 (a
  real bug). Don't add a generic plugin system.
- **No feature-flagged dual paths.** When Phase E lands, the old free
  functions become wrappers, not behind a flag.
- **No "while we're here" rewrites of the tokenizer.** It works, parity
  passes, leave it.
