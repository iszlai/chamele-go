# chamele-go — Porting Plan

A 1-to-1 feature-complete Go port of [lizard](https://github.com/terryyin/lizard).
Written so a junior engineer or coding agent can pick up any phase without
prior context. Every phase has: **scope**, **definition of done**, **files to
touch**, and **explicit references back to the Python source**.

The upstream Python tree lives in `./lizard-upstream/` (pinned at commit
`4ad5454`, just past v1.22.1). Use it as the canonical reference for behaviour
— when in doubt, run the Python lizard against a sample input and match its
output byte-for-byte where possible.

---

## Table of contents

1. [Goals & non-goals](#1-goals--non-goals)
2. [License](#2-license)
3. [Upstream inventory](#3-upstream-inventory)
4. [Target architecture](#4-target-architecture)
5. [Repository layout](#5-repository-layout)
6. [Testing strategy — three layers](#6-testing-strategy--three-layers)
7. [Phase 0 — bootstrap](#phase-0--bootstrap)
8. [Phase 1 — tokenizer + state machine + core types](#phase-1--tokenizer--state-machine--core-types)
9. [Phase 2 — analysis engine](#phase-2--analysis-engine)
10. [Phase 3 — first language readers (C/C++/Java)](#phase-3--first-language-readers-cccjava)
11. [Phase 4 — Python, Go, JavaScript readers](#phase-4--python-go-javascript-readers)
12. [Phase 5 — remaining 20 language readers](#phase-5--remaining-20-language-readers)
13. [Phase 6 — extensions](#phase-6--extensions)
14. [Phase 7 — CLI + output formats](#phase-7--cli--output-formats)
15. [Phase 8 — differential testing against Python lizard](#phase-8--differential-testing-against-python-lizard)
16. [Phase 9 — BDD/Gherkin port-level acceptance suite](#phase-9--bddgherkin-port-level-acceptance-suite)
17. [Phase 10 — docs, examples, release v0.1](#phase-10--docs-examples-release-v01)
18. [Cross-cutting conventions](#cross-cutting-conventions)
19. [Resolved decisions](#17-resolved-decisions)
20. [Appendix A — full file map of upstream](#appendix-a--full-file-map-of-upstream)
21. [Appendix B — per-language reader checklist](#appendix-b--per-language-reader-checklist)
22. [Appendix C — per-extension checklist](#appendix-c--per-extension-checklist)

---

## 1. Goals & non-goals

### Goals (must-have for v1.0)

- **Behavioural parity** with lizard v1.22.1 on the public CLI surface:
  - same set of supported file extensions and languages,
  - same CCN / NLOC / token / parameter / length / nesting-depth numbers on
    the canonical lizard test corpus (within rounding for averages),
  - same exit code semantics (`-i` warning gate),
  - same `#lizard forgive`, `#lizard forgive global`, `#lizard forgives(...)`
    directive semantics,
  - same `whitelizard.txt` whitelist semantics,
  - same `.gitignore` filtering behaviour.
- **Embeddable Go library**: `import "github.com/<org>/chamele-go/lizard"`
  exposes `Analyze`, `AnalyzeFile`, `AnalyzeFiles` with strongly-typed
  results (`FileInformation`, `FunctionInfo`).
- **Single static binary** for the CLI (`lizard` or `chamele`, TBD — see
  §17 Q1).
- **All 27 language readers** ported (see Appendix B).
- **All 22 extensions** ported (see Appendix C).
- **All 6 output formats** ported (tabular, XML, CSV, HTML, Checkstyle,
  clang-warning, MSVS-warning).
- **Concurrency** via a worker pool (replaces Python's
  `multiprocessing.Pool`), default = `runtime.NumCPU()`.
- **Public API contract documented** with godoc + examples.

### Non-goals (v1.0)

- Implementing every lizard CLI flag exactly the same — minor renames are OK
  if godoc is clear (see §17 Q3).
- Pixel-perfect HTML output. We preserve the data and rough layout, not the
  exact whitespace.
- The Cursor-rules / `.cursor/`, `appengine_config.py`, `index.py`, web
  hosting files, and the website templates. They are not part of the
  analyzer.
- Re-implementing the embeddable-Python `sys.path` hacks from
  `_script_dirs()` in `lizard.py`.
- A REST API or web server (lizard's `app.yaml` / GAE bits). Skip.

---

## 2. License

### Upstream license

`lizard-upstream/LICENSE.txt` is a standard MIT permission grant
(verbatim text in §17 below). The relevant clauses:

> Permission is hereby granted, free of charge, to any person obtaining a
> copy of this software ... to deal in the Software without restriction,
> including without limitation the rights to use, copy, modify, merge,
> publish, distribute, sublicense, and/or sell copies of the Software ...
> subject to the following conditions:
>
> The above copyright notice and this permission notice shall be included in
> all copies or substantial portions of the Software.

In addition, `lizard.py` itself carries an Apache-2.0 header at the top of
the file (lines 1–14), which is unusual — likely a leftover from an early
version. The repo-level LICENSE.txt is the canonical one, and MIT is a
superset of what Apache requires for **us as redistributors** (Apache adds
patent grant + state-of-changes requirement). To be safe we:

### What chamele-go must do

1. **Keep `lizard-upstream/LICENSE.txt` in the repo as-is** so the upstream
   copyright stays attached.
2. **Add a `NOTICE` file at the repo root** stating chamele-go is a Go port
   of lizard, naming "Terry Yin and other contributors" as the upstream
   author and including the MIT permission notice.
3. **chamele-go's own license: MIT** (matches upstream exactly; see §17
   row 2). A single `LICENSE` file at the repo root.
4. **In every Go source file ported from a specific Python file**, add a
   header comment noting the source file. Example:
   ```go
   // Package lizard analyzes source-code complexity.
   //
   // This file is derived from lizard's lizard.py (commit 4ad5454),
   // copyright Terry Yin and contributors, MIT-licensed.
   // See NOTICE for the full upstream notice.
   ```
5. **Do not rename or remove the upstream copyright in `LICENSE.txt`** —
   that is the load-bearing piece for legal compliance.

**Conclusion: port and open-source distribution are allowed.** No
copyleft concerns. We are not required to share modifications, but we will
because the whole point is an open-source library.

---

## 3. Upstream inventory

Numbers below are line counts of upstream `.py` files (excluding tests).
They are an upper bound on how much Python logic we need to translate.

| Area | Lines | Files | Notes |
|------|-------|-------|-------|
| Engine (`lizard.py`)              | 1163 | 1   | CLI, FileAnalyzer, NestingStack, FileInfoBuilder, processors, OutputScheme, whitelist, gitignore, hashing |
| Tokenizer base (`code_reader.py`) | 238  | 1   | regex tokenizer + `CodeStateMachine` + `CodeReader` |
| Language readers                  | 4669 | 27  | one file per language + 5 helpers (`clike`, `golike`, `rubylike`, `script_language`, `js_style_regex_expression`) |
| Extensions                        | 1857 | 22  | metrics + 4 output formats + utilities |
| Tests                             | ~15k | 50+ | pytest fixtures, includes our parity-target |
| **Total Python under port**       | ~7900 | ~50 | |

Test corpus to use as the parity oracle:
- `lizard-upstream/test/testCyclomaticComplexity.py` — table tests, easy
  port.
- `lizard-upstream/test/test_languages/*.py` — 12k LOC of per-language unit
  tests; **these are the spec**. Translate or replay them.
- `lizard-upstream/test/test_languages/testdata/` — input fixtures (Perl &
  Python); copy verbatim into `testdata/` in chamele-go.
- `lizard-upstream/test/test_extensions/*.py` — extension unit tests.
- `lizard-upstream/test/testTokenizer.py` — tokenizer table tests.

Full file map: **Appendix A**.

---

## 4. Target architecture

### Mental model

```
   ┌────────────┐    ┌──────────────────┐    ┌─────────────────────────┐    ┌──────────┐
   │  source    │ →  │  tokenize        │ →  │  pipeline of            │ →  │ FileInfo │
   │  bytes     │    │  (regex scanner) │    │  preprocessors +        │    │ (NLOC,   │
   │            │    │                  │    │  language state machine │    │  CCN, …) │
   └────────────┘    └──────────────────┘    └─────────────────────────┘    └──────────┘
                                                       ↑
                                                       │  extensions (each is a
                                                       │  pipeline stage or a
                                                       │  cross-file post-process)
                                                       │
```

The Python tokenizer yields `string` tokens. Each "processor"
(comment_counter, line_counter, token_counter, condition_counter, and any
extensions) is a generator that **transforms** the stream — typically just
side-effects on `context` and yielding the same tokens, but they can drop
tokens (comments) or inject synthetic ones (e.g. `cpre` macro handling).

The state machine (`CodeStateMachine`) walks tokens and mutates
`FileInfoBuilder.current_function` to populate `FunctionInfo`s.

### Translating to idiomatic Go

| Python idiom | Go translation |
|---|---|
| `def gen(...): yield token` token generator | `iter.Seq[string]` (Go 1.23+ range-over-func), pulled with `for tok := range gen(...)`. We target **Go 1.23+** (1.26 is the current host); the iterator type pulls its weight here. |
| `self._state = self._foo` method-pointer state | `type stateFn func(tok string) stateFn` returning the next state, or interface `State{ Next(tok string) State }`. **Recommend the func-pointer style** — exactly mirrors the Python pattern with no boxing. |
| `parallel_states = (A(ctx), B(ctx))` | `[]parallelState` slice on the reader; each is `func(tok string)` (or a `*Machine` value). |
| `@CodeStateMachine.read_inside_brackets_then("{}", "_state_global")` decorator | A helper function `readInsideBracketsThen(open, close rune, then stateFn, body func(tok string))` returning a `stateFn`. Replicate semantics — opening/closing bracket counting + delayed body call. |
| `@CodeStateMachine.read_until_then(';')` | Helper `readUntilThen(stop string, body func(tok string, collected []string))` returning a `stateFn`. |
| `FileInfoBuilder.__getattr__` delegating to `_nesting_stack` | Embed `*NestingStack` in `FileInfoBuilder` so Go's method promotion gives the same delegation. |
| `patch()` / `patch_append_method()` monkey-patching (`lizardnd.py`) | Idiomatic Go: extensions register hooks instead of patching shared state. Per-function extension data lives in a `map[*FunctionInfo]ExtensionData` keyed on the function pointer, OR `FunctionInfo` has an `Ext map[string]any` field for opaque storage. **Pick the second**, it's flatter. |
| `pathspec` for `.gitignore` | `github.com/sabhiram/go-gitignore` (the closest match to gitwildmatch semantics). |
| `multiprocessing.Pool.imap_unordered` | `errgroup.Group` + a fixed-size goroutine pool, fan-in via a buffered channel. |
| `argparse` | `github.com/spf13/cobra` + `pflag`. Trade-off: stdlib `flag` is lighter but doesn't handle subcommands or grouped help well; lizard has many flags, cobra is the better fit. |

### Concurrency model

```
              ┌─────────────────┐
   files ──→  │ jobs chan       │ ──→ N workers ──→  results chan ──→ printer
              └─────────────────┘
```

- Workers each construct a fresh `FileAnalyzer` per file (analyzers are
  stateful) and call `AnalyzeSourceCode(path, bytes)`.
- The cross-file post-processing step (`fan-in`/`fan-out`,
  `lizardduplicate`) runs **after** all file results have been collected,
  on the main goroutine, so it sees deterministic order.
- For ordered output (CLI matches lizard's order), workers return
  `(idx, FileInformation)` and the printer reorders by index.

### Memory

- A 1 MB source file produces O(tokens) intermediate state. Stream tokens
  through `iter.Seq` so we never hold the whole token list at once except
  for extensions that genuinely need it (`lizardio` keeps the token list
  for fan-in/out).
- For duplicate detection (`lizardduplicate`), tokens are hashed and
  buffered per-function — keep memory bounded with the same
  `sample_size` / `collapse_repeat_tokens` knobs as upstream.

---

## 5. Repository layout

```
chamele-go/
├── README.md
├── PLAN.md                          ← this file
├── NOTICE                           ← upstream attribution (Phase 0)
├── LICENSE                          ← MIT
├── go.mod                           ← module github.com/iszlai/chamele-go (go 1.23)
├── go.sum
├── Makefile
├── .golangci.yml
├── .goreleaser.yaml                 ← (Phase 10)
├── .gitignore                       ← includes /lizard-upstream
│
├── lizard-upstream/                 ← (gitignored) read-only reference, fetched via scripts/
│
├── scripts/
│   ├── fetch-upstream.sh            ← clones terryyin/lizard at the pinned SHA
│   └── parity-corpus.sh             ← refreshes test/parity/corpus from public repos
│
├── .github/
│   └── workflows/
│       ├── ci.yml                   ← Go 1.23 + latest, ubuntu/macos/windows
│       └── parity.yml               ← differential test against Python lizard
│
├── cmd/
│   └── chamele/                     ← CLI binary
│       └── main.go
│
├── chamele/                         ← public library, the import root
│   ├── doc.go
│   ├── analyze.go                   ← Analyze (slice + functional options), AnalyzeFile, AnalyzeFiles
│   ├── options.go                   ← Options struct + WithLanguages, WithThreads, WithExclude, …
│   ├── fileinfo.go                  ← FileInformation
│   ├── funcinfo.go                  ← FunctionInfo
│   ├── nesting.go                   ← Namespace, NestingStack, BareNesting
│   ├── builder.go                   ← FileInfoBuilder
│   ├── processors.go                ← preprocessing, comment_counter, line_counter, token_counter, condition_counter
│   ├── analyzer.go                  ← FileAnalyzer
│   ├── warnings.go                  ← warning_filter, whitelist_filter
│   ├── walk.go                      ← get_all_source_files, NESTED gitignore, md5 dedup
│   ├── output_scheme.go             ← OutputScheme + caption registry
│   ├── extension.go                 ← Extension interface + RegisterExtension (public)
│   └── version.go                   ← const Version = "x.y.z"
│
├── internal/
│   ├── tokenizer/                   ← regex tokenizer + CodeStateMachine
│   │   ├── tokenizer.go
│   │   ├── statemachine.go
│   │   ├── reader.go                ← internal CodeReader engine
│   │   └── helpers.go               ← read_inside_brackets_then, read_until_then
│   └── stringx/
│       ├── unicode.go               ← BOM detection
│       └── lineend.go               ← CRLF→LF normalisation (file-read time)
│
├── languages/                       ← public; stable Register API from v0.1
│   ├── languages.go                 ← Reader interface, Register(reader), GetReaderForFilename(path)
│   ├── all/                         ← side-effect import that registers every reader
│   │   └── all.go
│   ├── clike/
│   ├── java/
│   ├── golang/                      ← package name "golang" (Go keyword conflict)
│   ├── python/
│   ├── typescript/
│   ├── … (full list in Appendix B)
│   └── tsx/
│
├── ext/                             ← extensions, opt-in by import
│   ├── ext.go                       ← compile-time assertion that each pkg's New() satisfies chamele.Extension
│   ├── all/
│   │   └── all.go                   ← side-effect: registers every built-in extension
│   ├── nd/                          ← nesting depth
│   ├── ns/                          ← nested control structures
│   ├── modified/
│   ├── mccabe/
│   ├── io/                          ← fan-in / fan-out
│   ├── cpre/
│   ├── ignoreassert/
│   ├── duplicate/
│   ├── dupparams/
│   ├── boolcount/
│   ├── exitcount/
│   ├── gotocount/
│   ├── statementcount/
│   ├── dependencycount/
│   ├── complextags/
│   ├── wordcount/
│   ├── dumpcomments/
│   ├── outside/
│   └── nonstrict/
│
├── output/                          ← output formatters (all buffer-then-emit)
│   ├── tabular.go
│   ├── csv.go
│   ├── xml.go                       ← cppncss-compatible
│   ├── html.go
│   ├── checkstyle.go
│   ├── clang_warning.go
│   └── msvs_warning.go
│
├── features/                        ← BDD layer (Phase 9) — godog wired into `go test`
│   ├── analyze.feature
│   ├── walk_filter.feature          ← gitignore, exclude, dedup, language filter
│   ├── forgiveness.feature          ← #lizard forgive directives
│   ├── whitelist.feature
│   ├── output_tabular.feature
│   ├── output_csv.feature
│   ├── output_xml.feature
│   ├── output_html.feature
│   ├── output_checkstyle.feature
│   ├── cli_exit_codes.feature
│   ├── languages/                   ← one .feature per language as acceptance gate
│   │   ├── go.feature
│   │   ├── python.feature
│   │   ├── java.feature
│   │   └── …
│   ├── extensions/
│   │   ├── nesting_depth.feature
│   │   ├── fan_in_out.feature
│   │   ├── duplicate.feature
│   │   └── …
│   └── steps/                       ← shared step definitions (one Go package)
│       ├── steps_test.go            ← TestMain wires godog into `go test`
│       ├── analyze_steps.go         ← Given/When/Then for Analyze API
│       ├── cli_steps.go             ← Given/When/Then for `chamele` binary
│       ├── source_steps.go          ← "Given a Python file containing: …"
│       ├── output_steps.go          ← "Then the CSV should equal: …"
│       └── world.go                 ← shared scenario state
│
├── testdata/                        ← copied verbatim from upstream/test_languages/testdata
│   └── (perl_*.pl, python_*.py, utf.c, …)
│
└── test/
    ├── parity/                      ← differential tests against Python lizard (tag-gated)
    │   ├── parity_test.go
    │   └── corpus/                  ← curated multi-language sources
    └── golden/                      ← golden snapshots for output formats
```

### Package boundary rules

- `chamele/` is the **only** package external callers should import for
  typical use. It re-exports the types they need (`FileInformation`,
  `FunctionInfo`, `Options`, `Extension`, `Result`).
- `languages/<lang>/` are imported for **side effects** that register the
  reader, OR explicitly when a caller wants a custom subset:
  ```go
  import _ "github.com/iszlai/chamele-go/languages/all" // registers all 27
  // or:
  import _ "github.com/iszlai/chamele-go/languages/golang"
  import _ "github.com/iszlai/chamele-go/languages/java"
  ```
- `internal/tokenizer` is **not** exported — language authors who want to
  build a reader use `languages.Register(reader)` against the stable
  `languages.Reader` interface.
- `ext/<name>` packages each export a `New() chamele.Extension`
  constructor so callers can opt-in to specific metrics.
- The `languages.Reader` and `chamele.Extension` interfaces are **public
  v0.1 commitments** (§17 row 17). Breaking changes to them follow SemVer.

---

## 6. Testing strategy — three layers

chamele-go ships three independent test layers. Each catches a different
class of bug. **All three must be green** for a release.

```
   ┌───────────────────────────────────────────────────────────────┐
   │ Layer 1 — Translated unit tests       (per Go package)       │
   │   What:  upstream pytest cases re-expressed as Go table tests │
   │   Why:   precise spec for each language reader + extension    │
   │   When:  written alongside the code in every phase            │
   │   How:   `go test ./languages/<lang>` etc.                    │
   └───────────────────────────────────────────────────────────────┘
   ┌───────────────────────────────────────────────────────────────┐
   │ Layer 2 — Port-level BDD acceptance   (features/ at root)    │
   │   What:  Gherkin scenarios over the public Go API and CLI     │
   │   Why:   readable spec + protects against regressions at      │
   │          the *port* boundary (library API, CLI, output)       │
   │   When:  Phase 9 (after migration completes)                  │
   │   How:   godog wired into `go test ./features/...`            │
   └───────────────────────────────────────────────────────────────┘
   ┌───────────────────────────────────────────────────────────────┐
   │ Layer 3 — Differential testing        (Python lizard oracle) │
   │   What:  run both implementations on a corpus, diff CSV       │
   │   Why:   catches semantic drift the unit tests didn't cover   │
   │   When:  Phase 8 (continuous after that)                      │
   │   How:   `go test -tags parity ./test/parity/...` in CI       │
   └───────────────────────────────────────────────────────────────┘
```

### Why three layers and not one?

- **Unit tests** are precise but fragile to refactor. They prove the
  *implementation* matches Python lizard's behaviour case-by-case.
- **BDD/Gherkin** is durable across refactors (it tests *behaviour at the
  port*, not internal symbols). It also serves as living documentation —
  a reviewer reading `features/forgiveness.feature` learns exactly how
  `#lizard forgive` works without reading code.
- **Differential** catches the unknown-unknowns: cases the upstream test
  suite never covered, but real source files exercise.

### The user's testing preference

The user has stated:
> "after migration my expectation is to have port level test suite using
> cucumber/gherkin style bdd"

This is the **primary acceptance layer** post-migration. Layer 1 (unit
tests) is what we use during the port; Layer 2 (BDD) is what we rely on
going forward. Layer 3 (differential) is the safety net.

### Per-PR test gate

| Change type | Layer 1 | Layer 2 | Layer 3 |
|---|---|---|---|
| New language reader | new tests in `languages/<lang>/*_test.go` | new `features/languages/<lang>.feature` | must not regress |
| New extension | new tests in `ext/<name>/*_test.go` | new `features/extensions/<name>.feature` | must not regress |
| Tokenizer change | tests in `internal/tokenizer/` | not directly | must not regress |
| CLI flag change | tests in `cmd/chamele/` | new scenario in `features/cli_*.feature` | not affected |
| Output format change | golden file under `test/golden/` | new scenario in `features/output_*.feature` | matters if differential includes that format |

---

## Phase 0 — bootstrap

### Scope

Get the repo into a state where every subsequent phase can run
`go test ./...` and have it pass on an empty codebase.

### Tasks

1. Create `go.mod`:
   ```
   module github.com/iszlai/chamele-go
   go 1.23
   ```
2. Add direct dependencies (per §17 row 16):
   ```
   require (
       github.com/spf13/cobra v1.x
       github.com/spf13/pflag v1.x
       github.com/cucumber/godog v0.14.x
       github.com/sabhiram/go-gitignore v1.x
       golang.org/x/sync v0.x          // errgroup
   )
   ```
3. Create `LICENSE` (MIT — full text of the upstream LICENSE.txt; replace
   the copyright line with `Copyright 2026 Lehel Iszlai and chamele-go
   contributors`).
4. Create `NOTICE`:
   ```
   chamele-go
   Copyright 2026 Lehel Iszlai and contributors

   This product is a derivative work of lizard
   (https://github.com/terryyin/lizard), Copyright Terry Yin and
   contributors, MIT-licensed. The upstream MIT license text is
   included verbatim in lizard-upstream/LICENSE.txt when the upstream
   tree is fetched via scripts/fetch-upstream.sh.
   ```
5. Add `.gitignore` containing `/lizard-upstream/` (the upstream tree is
   NOT vendored in git; it's fetched on demand — §17 row 20).
6. Add `scripts/fetch-upstream.sh`:
   ```bash
   #!/usr/bin/env bash
   set -euo pipefail
   PIN=4ad545491e8a141a72bb52b8d719b1842c5a7d1b
   if [[ -d lizard-upstream/.git ]]; then
       git -C lizard-upstream fetch --depth 1 origin "$PIN"
       git -C lizard-upstream checkout "$PIN"
   else
       git clone --depth 1 https://github.com/terryyin/lizard.git lizard-upstream
       git -C lizard-upstream fetch --depth 1 origin "$PIN"
       git -C lizard-upstream checkout "$PIN"
   fi
   # Mirror testdata into our tree so per-language tests don't need the upstream tree at runtime.
   rsync -a --delete lizard-upstream/test/test_languages/testdata/ testdata/
   ```
   First-time contributor or CI must run this once.
7. Add `Makefile` with: `bootstrap` (runs fetch-upstream.sh), `build`,
   `test`, `lint`, `bdd` (alias for `go test ./features/...`),
   `parity` (`go test -tags parity ./test/parity/...`), `clean`.
8. Add `.github/workflows/ci.yml` running on Go 1.23 (floor) + 1.26
   (latest) on ubuntu/macos/windows. Steps: `go vet ./...`,
   `golangci-lint run`, `go test ./...` (includes BDD). The first
   contributor to bump the floor must update §17 row 5 in the same PR.
9. Add `.github/workflows/parity.yml`: runs `make bootstrap`, installs
   Python 3.11 + `pip install lizard==1.22.1 pathspec`, runs `go test
   -tags parity ./test/parity/...`, uploads CSV diffs on failure.
10. Add `.golangci.yml` with at least: errcheck, govet, ineffassign,
    staticcheck, gofmt, goimports, gosec.
11. Add the package skeletons listed in §5 with a `// Package …` doc
    comment but no real code yet. Each phase will fill them in.
12. Add `CONTRIBUTING.md` pointing at PLAN.md, the upstream commit pin,
    and a "run `make bootstrap` first" line.
13. Add `features/steps/steps_test.go` with the godog `TestMain`
    skeleton so `go test ./features/...` works even with no scenarios.
14. Add `.cursor/rules/` or similar (optional) capturing the codebase
    conventions in §16 for AI-assisted contributors.

### Definition of done

- `make bootstrap && make build && make test` succeeds (no real tests
  yet, but BDD shim doesn't error).
- CI is green on a placeholder PR.
- `LICENSE`, `NOTICE`, `CONTRIBUTING.md`, `Makefile`, `.gitignore`,
  workflow files, `scripts/fetch-upstream.sh` all present.
- `lizard-upstream/` exists locally (after bootstrap) but is gitignored.

---

## Phase 1 — tokenizer + state machine + core types

### Scope

Port the **language-agnostic** plumbing: regex tokenizer, state machine
helpers, and the data types (`FunctionInfo`, `FileInformation`, `Namespace`,
`NestingStack`, `FileInfoBuilder`).

### Files to touch

| New Go file | Ported from |
|---|---|
| `internal/tokenizer/tokenizer.go` | `lizard_languages/code_reader.py:128–207` (`generate_tokens`) |
| `internal/tokenizer/statemachine.go` | `lizard_languages/code_reader.py:11–86` (`CodeStateMachine`) |
| `internal/tokenizer/reader.go` | `lizard_languages/code_reader.py:89–238` (`CodeReader`) |
| `internal/tokenizer/helpers.go` | `lizard_languages/code_reader.py:64–86` (`read_inside_brackets_then`, `read_until_then`) |
| `chamele/funcinfo.go` | `lizard.py:302–372` (`FunctionInfo`) |
| `chamele/fileinfo.go` | `lizard.py:375–397` (`FileInformation`) |
| `chamele/nesting.go` | `lizard.py:280–438` (`Nesting`, `Namespace`, `BARE_NESTING`, `NestingStack`) |
| `chamele/builder.go` | `lizard.py:441–523` (`FileInfoBuilder`) |

### Critical details

#### Tokenizer regex

The Python pattern in `code_reader.py:142–170` is one big alternation. Notes
for the Go port:

- **Go's `regexp` is RE2 → no backreferences, no lookarounds.** Most of the
  lizard pattern is safe. The problematic part is the C++ template
  heuristic on line 165:
  ```
  \<(?=(?:[^<>?]*\?)+[^<>]*\>)(?:[\w\s,.?]|(?:extends))+\>
  ```
  It uses `(?=...)` lookahead. **Plan:** drop the lookahead, accept the
  unconditional pattern `\<(?:[\w\s,.?]|(?:extends))+\>`, then in a tiny
  post-filter Go function reject matches that don't contain a `?` (cheap —
  one byte scan). This preserves semantics.
- `re.M | re.S` map to Go's `(?ms)` inline flags. Use `regexp.MustCompile`
  with the flag prefix; no need for `regexp.Compile` at runtime since the
  pattern is constant.
- Performance: pre-compile the tokenizer regex once per language at
  `init()`, store on the reader struct.
- The `(?aiLmsux)` flag-extraction trick in `code_reader.py:188–201` is
  used by some language readers (e.g. `r.py`, `perl.py`) to pass case-
  insensitivity into the additional pattern. Mirror with a Go helper
  `extractAndApplyFlags(addition string) (cleaned string, flags string)`
  that returns the cleaned regex source + the `(?…)` prefix to prepend.

#### State machine semantics to preserve

- `next(state, token=None)` returns the result of calling the new state on
  the token if a token is given. Some languages rely on this re-entrancy
  (`go.py`, `clike.py`).
- `to_exit` flag — when a sub-state machine sets it, the outer caller
  pops it. Use a returned `bool` in Go (`type stateFn func(tok string)
  (next stateFn, exit bool)`).
- `sub_state(state, callback, token)` runs a nested machine until it
  `statemachine_return()`s, then calls the callback. Implement as an
  explicit stack of `stateFn` slices on the machine.
- `last_token` is read by several readers (e.g. `clike` distinguishes
  initializer braces from class-body braces). Mirror as a field on the
  machine.

#### `FunctionInfo` field set (Phase 1 baseline — extensions add more)

| Field | Type | Notes |
|---|---|---|
| `Name`, `LongName` | `string` | qualified name vs full signature |
| `Filename` | `string` | |
| `StartLine`, `EndLine` | `int` | |
| `CyclomaticComplexity` | `int` | seed value `1` |
| `NLOC` | `int` | seed `1` |
| `TokenCount` | `int` | seed `1` (first token) |
| `FullParameters` | `[]string` | parsed param list, with default values + types still attached |
| `TopNestingLevel` | `int` | |
| `FanIn`, `FanOut`, `GeneralFanOut` | `int` | populated by `ext/io` |
| `MaxNestingDepth` | `int` | populated by `ext/nd` |
| `ForgivenMetrics` | `map[string]struct{}` | `#lizard forgives(...)` |
| `Ext` | `map[string]any` | opaque per-extension data |

Provide computed getters:
- `Length() int { return e.EndLine - e.StartLine + 1 }`
- `ParameterCount() int` — runs the same regex as
  `lizard.py:339–348` to strip default values and type annotations,
  returning bare names. Exclude empty entries (trailing commas).
- `UnqualifiedName() string` — last `::`-split segment.
- `Location() string` — `" name@start-end@filename"`.

### Tests (Phase 1)

- `internal/tokenizer/tokenizer_test.go` — port every case in
  `lizard-upstream/test/testTokenizer.py`. Including edge cases like
  multi-line strings, raw strings, macros with line continuation.
- `chamele/funcinfo_test.go` — table tests for `ParameterCount` on
  representative parameter strings (`""`, `"int a, char *b"`,
  `"a: int = 5, b: str"`, `"a, b,"` trailing-comma case).

### Definition of done

- All tokenizer tests pass.
- `FunctionInfo`, `FileInformation`, `NestingStack`, `FileInfoBuilder`
  compile and have unit tests.
- A skeleton `FileAnalyzer` exists but does nothing yet (next phase).

---

## Phase 2 — analysis engine

### Scope

Wire up the processor pipeline and produce useful `FileInformation` for a
**single hardcoded language** (use a stub C reader that just splits on
whitespace) so we can exercise the engine end-to-end.

### Files to touch

| New Go file | Ported from |
|---|---|
| `chamele/processors.go` | `lizard.py:526–584` (5 processors) |
| `chamele/analyzer.go` | `lizard.py:586–620` (`FileAnalyzer`) |
| `chamele/analyze.go` | `lizard.py:80–98` (`analyze`, `analyze_files`) + new `Analyze*` API |
| `chamele/options.go` | new — functional options (`With*` constructors) and `Options` struct |
| `chamele/walk.go` | `lizard.py:929–1009` (`get_all_source_files`, md5 dedup, **plus** nested `.gitignore` handling — §17 row 13) |
| `chamele/warnings.go` | `lizard.py:628–678` (warning + whitelist filters) |
| `chamele/output_scheme.go` | `lizard.py:681–786` (`OutputScheme`, captions) |
| `internal/stringx/lineend.go` | new — CRLF→LF normalisation at file-read time (§17 row 15) |

### Critical details

#### Processor pipeline

Python ordering, `lizard.py:1096–1102`:

```
[preprocessing, comment_counter, line_counter, token_counter, condition_counter]
```

Mirror this exact order. Extensions can be inserted with an `OrderingIndex`
(see `lizardcpre.py:11` — `ordering_index = 0` puts it before everything).

#### `comment_counter` forgiveness logic

`lizard.py:532–551` — when a comment contains:
- `#lizard forgive global` → set `context.forgive_global = true`
- `#lizard forgives(metrics)` → split CSV inside parens, add to
  `current_function.forgiven_metrics`
- `#lizard forgive` (any other suffix) → set `context.forgive = true`
- contains `"GENERATED CODE"` → halt processing (return early, no more
  tokens yielded for this file)

This logic must be in `processors.go`, not delegated to each language —
it's language-agnostic.

#### `line_counter`

Walks tokens, tracks newlines, calls `context.AddNLOC(count + newline)`.
Key edge cases (from `lizard.py:554–567`):
- A token containing embedded newlines (multi-line string, `/* ... */`
  comment) counts each as a line.
- `\n` token resets the per-line state.

#### `walk.go` semantics

- `filepath.WalkDir` over each input path.
- Resolve a reader via `languages.GetReaderForFilename(path)`; skip files
  with no match.
- `lans` filter: if `WithLanguages(...)` was given, also drop files whose
  reader's `LanguageNames` doesn't intersect.
- Exclude pattern: each pattern is fnmatch-style (`*`, `?`,
  `[...]`). Use `path.Match` in a loop.
- **Nested `.gitignore` handling** (§17 row 13 — we diverge from upstream
  here, fixing a known bug): maintain a stack of `gitignore.IgnoreParser`
  instances as we descend. On entering a directory, push a new parser for
  any `.gitignore` found there; on leaving, pop. A file is excluded iff
  any parser in the stack matches it (innermost wins for negation
  patterns, matching git's actual behaviour). Implementation: use
  `github.com/sabhiram/go-gitignore` per-directory, combined with a
  walk-state struct that tracks the active stack.
- This **diverges from upstream** (which only reads the first `.gitignore`
  it finds). Document the divergence in `docs/divergences.md`.
- MD5 dedup: hash each file's bytes once, skip if seen.

### Tests (Phase 2)

- End-to-end test: analyse a fake `*.c` file with the stub reader, check
  NLOC, token count, and that the line counter handles `\n` correctly.
- Test `walk.go`'s `.gitignore` handling, exclude patterns, dedup by md5,
  language filter.
- Test `OutputScheme` produces the same captions as Python (lock with a
  golden file).

### Definition of done

- The engine can run preprocessors + a stub reader and produce a
  `FileInformation` with non-zero NLOC.
- `Analyze("/tmp/path")` returns an `[]FileInformation` over a directory.
- Walk behaviour matches lizard's on a hand-crafted corpus.

---

## Phase 3 — first language readers (C/C++/Java)

### Scope

Port the **most complex and most reused** readers first, because every
other reader extends them.

`clike.py` is the parent of: java, csharp, javascript, typescript, objc,
swift, scala, php, kotlin. Get this right and 9 readers fall into place.

### Files to touch

| New Go file | Ported from | LOC (PY) |
|---|---|---|
| `languages/clike/clike.go` | `lizard_languages/clike.py` | 398 |
| `languages/java/java.go` | `lizard_languages/java.py` | 252 |
| `languages/languages.go` | `lizard_languages/__init__.py` | 67 |

### Critical details

#### `CLikeReader` decomposition

The Python class has **three parallel states**:
1. `CLikeStates` — function detection (name, parameter list, body)
2. `CLikeNestingStackStates` — `{ } class namespace template [[attr]]`
3. `CppRValueRefStates` — `&&` is r-value ref, not logical-and; `typedef`
   handling

Mirror in Go as three machines stored in the reader struct, run in
parallel for every token.

#### Preprocess hook

`clike.py:44–63` walks tokens, joins `~Foo` into a single token (for C++
destructors), parses `#if/#elif/#include` directives. The
`process_token`/`preprocess` hooks need to be reflected in the Go
`Reader` interface:

```go
type Reader interface {
    Tokenize(src []byte) iter.Seq[string]
    Preprocess(tokens iter.Seq[string]) iter.Seq[string] // optional
    ProcessToken(tok string) (handled bool)              // optional
    ParallelStates() []StateMachine
    Conditions() ConditionSet
    Extensions() []string                                 // file extensions, e.g. {"c","cpp",…}
    LanguageNames() []string                              // {"cpp","c"}
}
```

#### Java specifics worth flagging

- `JavaStates._consume_java_expression_tokens` (java.py:32–47) — handles
  `Foo.class`, `Type::method` so they don't trigger function detection.
- Records (`record Foo(...) {}`) — recently added (Java 14+). The compact
  constructor `Foo {}` block shouldn't create a separate function.
- Nested classes inside method bodies via `JavaFunctionBodyStates`.
- Annotations `@Override` skipping.

### Tests (Phase 3)

- Translate `lizard-upstream/test/testCAndCPP.py` (797 LOC) and
  `lizard-upstream/test/testJava.py` (685 LOC) to Go table tests.
- Add `internal/tokenizer/clike_token_test.go` for raw strings,
  digit-separators (`1'000'000`), line continuations.

### Definition of done

- `Analyze` over a C/C++ file produces the same CCN/NLOC/token/parameter
  counts as the Python tests assert.
- Java test suite passes.
- `lang := languages.Get("cpp")` works.

---

## Phase 4 — Python, Go, JavaScript readers

### Scope

These are independent of clike's parent and can be done in parallel.

| New Go file | Ported from | LOC | Depends on |
|---|---|---|---|
| `languages/python/python.go` | `python.py` | 152 | `script_language.py` (52 LOC) |
| `languages/golang/golang.go` | `go.py` + `golike.py` | 46 + 85 | none |
| `languages/javascript/javascript.go` | `javascript.py` + `js_style_regex_expression.py` | TBD | clike |

### Critical details

#### Python indentation tracking

`python.py:11–24` — the `PythonIndents` helper opens/closes nesting
levels based on leading whitespace. Translate as a `pythonIndents`
struct on the reader; `preprocess` calls `SetNesting` each line.

#### Triple-quoted-string-as-comment heuristic

`python.py:52–85` — a top-level `"""..."""` that isn't a docstring and
isn't part of an assignment expression is treated as a comment (NLOC
deducted). Carry the `_lastMeaningfulToken` field on the reader.

#### Go `golike` state machine

`golike.py:8–84` — the `func` keyword pushes a new function; tells apart
methods (`func (s *S) Name(...)`), generics (`Name[T any]`), and
interface methods. Watch out for backtick strings (`go.py:25–41`
overrides `__call__` to bypass condition counting inside backticks).

### Tests (Phase 4)

- `testPython.py` (517 LOC), `testPythonForgive.py` (63 LOC),
  `testGo.py` (168 LOC), `testJavaScript.py` (839 LOC), `testES6.py`
  (513 LOC).

### Definition of done

- All four readers pass their respective ported test suites.
- They co-exist in the registry (selectable via `Analyze(...,
  WithLanguages("python","go"))`).

---

## Phase 5 — remaining 20 language readers

### Scope

Port the rest. Group by dependency to parallelise.

### Order

1. **clike children** (depend on Phase 3): csharp, objc, swift, scala,
   php, kotlin, typescript, tsx.
2. **Standalone**: lua, rust, fortran, kotlin, solidity, erlang, zig,
   r, plsql, st, ttcn, vue, gdscript.
3. **rubylike children**: ruby, perl (both use `rubylike.py`).

For each reader, follow Appendix B's checklist (file, parent class, LOC,
upstream test file).

### Definition of done

- Every reader's upstream test file is translated and passes.
- `languages/all/all.go` (a side-effect import that registers every
  reader) is a single `import` line that brings them all in.

---

## Phase 6 — extensions

### Scope

Port the 22 extensions. Each is independent and can be parallelised.

### Approach

Define `lizard.Extension` interface:

```go
type Extension interface {
    Name() string
    OrderingIndex() int               // 0 = before built-in processors, default = after
    Process(tokens iter.Seq[string], reader languages.Reader) iter.Seq[string]
    CrossFileProcess(files []FileInformation) []FileInformation // optional
    FunctionInfoColumns() []ColumnSpec   // additional columns for output
    PrintResult(w io.Writer) error        // optional, for word-cloud etc.
}
```

Per-function extension data lives in `FunctionInfo.Ext[<ext-name>]`.
Type-assert in the column getter:

```go
func (e *NDExtension) Value(f *FunctionInfo) any {
    if d, ok := f.Ext["nd"].(*NDData); ok {
        return d.MaxNestingDepth
    }
    return 0
}
```

### Per-extension notes

See **Appendix C** for upstream file, LOC, FUNCTION_INFO columns, and
known caveats per extension.

The biggest ones:
- `ext/duplicate` (lizardduplicate.py, 314 LOC) — non-trivial; uses a
  sliding window hash over normalised tokens. Port the algorithm
  literally; do not try to rewrite cleverly.
- `ext/io` (lizardio.py, 92 LOC) — needs the full token list per
  function for cross-file fan-in/out. Make sure the extension is the
  only one storing tokens (memory concern).
- `ext/nd` (lizardnd.py, 215 LOC) — heavy mutation of context state.
  Port the state machine literally before refactoring.

### Definition of done

- Every extension's upstream test file translated and green.
- The default CLI exposes them via `-E<name>` flags, same as upstream.

---

## Phase 7 — CLI + output formats

### Scope

User-facing surface: `cmd/lizard/main.go`, output formatters, exit-code
semantics.

### Files to touch

| New Go file | Ported from |
|---|---|
| `cmd/chamele/main.go` | `lizard.py:1108–1162` (`main`) |
| `cmd/chamele/cmd_root.go` | `lizard.py:114–276` (`arg_parser`, `parse_args`) using cobra |
| `output/tabular.go` | `lizard.py:789–873` (`print_and_save_modules`, `print_warnings`, `print_total`) |
| `output/xml.go` | `lizard_ext/xmloutput.py` (168 LOC) |
| `output/csv.go` | `lizard_ext/csvoutput.py` (56 LOC) |
| `output/html.go` | `lizard_ext/htmloutput.py` (229 LOC) — data-equivalent only (§17 row 14) |
| `output/checkstyle.go` | `lizard_ext/checkstyleoutput.py` (33 LOC) |
| `output/clang_warning.go` | `lizard.py:902–907` |
| `output/msvs_warning.go` | `lizard.py:910–915` |

All output formatters **buffer-then-emit** (§17 row 11): they receive the
full `[]FileInformation` and write a complete document. No streaming.

### CLI flag mapping

Modernised flags only — **no legacy aliases** (§17 row 3). The full list
(lizard.py:127–273) — every flag must be present:

| Python flag (upstream) | chamele flag | Behaviour |
|---|---|---|
| `-l, --languages` | `-l, --languages` (repeatable) | language filter |
| `-V, --verbose` | `-V, --verbose` | long function names in output |
| `-C, --CCN` | `-C, --ccn` | CCN warning threshold (default 15) |
| `-f, --input_file` | `-f, --input-file` | read paths from file |
| `-o, --output_file` | `-o, --output-file` | write output here; format inferred from extension |
| `-L, --length` | `-L, --length` | length warning threshold (default 1000) |
| `-a, --arguments` | `-a, --arguments` | param count threshold (default 100) |
| `-w, --warnings_only` | `-w, --warnings-only` | clang-format warnings only |
| `--warning-msvs` | `--warning-msvs` | MSVS-format warnings only |
| `-i, --ignore_warnings` | `-i, --ignore-warnings` | exit-code gate |
| `-x, --exclude` (repeatable) | `-x, --exclude` | fnmatch exclude |
| `-t, --working_threads` | `-t, --threads` | worker count (default `runtime.NumCPU()` — §17 row 4) |
| `-X, --xml` | `-X, --xml` | XML output |
| `--csv` | `--csv` | CSV output |
| `-H, --html` | `-H, --html` | HTML output |
| `-m, --modified` | `-m, --modified` | enables `modified` extension |
| `--checkstyle` | `--checkstyle` | Checkstyle XML |
| `-E, --extension` | `-E, --extension` (repeatable) | enable extension by name |
| `-s, --sort` | `-s, --sort` (repeatable) | sort warnings by field |
| `-T, --Threshold` | `-T, --threshold` (repeatable) | `field=value` threshold setter (parsed exactly like upstream's `DictAction`) |
| `-W, --whitelist` | `-W, --whitelist` | whitelist file (default `./whitelizard.txt`) |
| `-N, --ND` (via `ext/nd`) | `-N, --nd` | ND threshold (default 7) |
| `--NS` (via `ext/ns`) | `--ns` | NS threshold (default 3) |
| (new) | `--version` | print `Version` from `chamele/version.go` |

CLI is a **single root command** (§17 row 10): `chamele [flags] <paths...>`.
Subcommands are not added in v0.1.

### Exit code semantics

```go
if opts.IgnoreWarnings >= 0 && warningCount > opts.IgnoreWarnings {
    os.Exit(1)
}
```

`-i -1` → never error. Mirror upstream verbatim.

### Tests (Phase 7)

- Golden tests on every output format with a tiny fixed input
  (`testdata/golden_in.c`, expected `golden_out.xml`, …).
- Run the CLI in a subprocess against a sample dir, check exit code.

### Definition of done

- `chamele-go --help` matches lizard's help layout in spirit.
- Every output format round-trips on a curated fixture.

---

## Phase 8 — differential testing against Python lizard

### Scope

The translated unit tests verify **what we wrote down**. The differential
test verifies **what we missed**.

### Setup

1. Vendor a known-good Python lizard install in CI via a `requirements.txt`
   pinned to v1.22.1 plus `pathspec`.
2. Curate `test/parity/corpus/` with at least 3 small files per supported
   language — pull from real OSS projects (with attribution comments).
3. For each file, run both engines with `--csv`, parse to a canonical
   form, diff.

### CSV is the comparison format

Both engines emit a stable per-function CSV (verbose mode). Compare on:

- `nloc, ccn, token_count, parameter_count, length, name, start_line, end_line`

We do NOT compare on float averages (rounding differences). We do NOT
compare on long_name (string normalisation differs across token regex
edge cases — track separately, don't gate CI on it for v1.0).

### Acceptance threshold

For v1.0:
- **Identical** on per-function `nloc, ccn, token_count, parameter_count`.
- **Identical** on function names for top-level functions; nested /
  anonymous functions allowed to diverge by name (track in CHANGELOG).
- Any difference is a bug-with-a-ticket, not a blocker for v0.1 — but a
  blocker for v1.0.

### Tests

- `test/parity/parity_test.go` — `go test -tags parity ./test/parity/...`.
- `.github/workflows/parity.yml` — installs Python lizard, runs the
  diff test, uploads CSV diffs as artifacts on failure.

### Definition of done

- Parity passes on the curated corpus.
- A documented divergence list lives in `docs/divergences.md` for any
  known differences (e.g. how Python lizard handles a malformed file vs.
  how we handle it).

---

## Phase 9 — BDD/Gherkin port-level acceptance suite

### Scope

Build the durable acceptance layer the user expects post-migration. This
is **Layer 2** in §6 — port-level scenarios in Gherkin, driven by
[cucumber/godog](https://github.com/cucumber/godog), wired into
`go test ./features/...`.

This phase **does not** add or change behaviour. It captures what we built
in Phases 1–8 as readable acceptance criteria, locking the public surface
against regression.

### Why this is its own phase

By Phase 9, every reader and extension exists with translated unit tests.
The BDD layer:

- exercises the **public ports** (library API, CLI binary, output
  formatters) — not internal symbols, so it survives refactors;
- is **readable spec** — a non-coder can audit `features/forgiveness.feature`
  and understand exactly how `#lizard forgive` works;
- catches **integration bugs** the unit tests missed because each unit
  test mocks half the pipeline;
- gives **agentic contributors** a runnable definition of "feature done".

### Files to touch

| New file | Purpose |
|---|---|
| `features/steps/steps_test.go` | godog TestSuite wiring; registers every step package; lives in `_test.go` so binary is unaffected |
| `features/steps/world.go` | shared scenario state: source code under test, captured FileInformation, captured CLI stdout/stderr/exit-code |
| `features/steps/source_steps.go` | `Given a <lang> file containing:` (docstring), `Given a directory <path> with:` (data table of file → content) |
| `features/steps/analyze_steps.go` | `When I analyze it`, `When I analyze it with options <table>` |
| `features/steps/cli_steps.go` | `When I run "chamele <args>"` — spawns the built binary in a tempdir |
| `features/steps/output_steps.go` | `Then the CSV output should equal:`, `Then function "<name>" should have CCN <n>`, `Then the exit code should be <n>` |
| `features/analyze.feature` | core analysis behaviour: NLOC, CCN, token, parameter count |
| `features/walk_filter.feature` | gitignore (nested), exclude patterns, md5 dedup, language filter |
| `features/forgiveness.feature` | `#lizard forgive`, `forgive global`, `forgives(metric_list)` |
| `features/whitelist.feature` | whitelizard.txt rules and matching |
| `features/output_tabular.feature` | default tabular output structure |
| `features/output_csv.feature` | CSV columns, escaping, verbose mode |
| `features/output_xml.feature` | cppncss XML structure, averages, sums |
| `features/output_html.feature` | data-equivalent HTML structure |
| `features/output_checkstyle.feature` | Checkstyle XML element shape |
| `features/cli_exit_codes.feature` | `--ignore-warnings` gate semantics |
| `features/languages/<lang>.feature` | one per supported language — function detection, CCN per construct, language-specific edge cases |
| `features/extensions/<ext>.feature` | one per extension — ND, NS, fan-in/out, modified CCN, duplicate, etc. |

### godog wiring

```go
// features/steps/steps_test.go
package steps

import (
    "os"
    "testing"

    "github.com/cucumber/godog"
)

func TestFeatures(t *testing.T) {
    suite := godog.TestSuite{
        ScenarioInitializer: InitializeScenario,
        Options: &godog.Options{
            Format:   "pretty",
            Paths:    []string{"../"},      // features/*.feature, features/**/*.feature
            TestingT: t,
            Tags:     os.Getenv("GODOG_TAGS"), // empty = run all
        },
    }
    if suite.Run() != 0 {
        t.Fatal("BDD suite failed")
    }
}

func InitializeScenario(sc *godog.ScenarioContext) {
    w := &World{}
    sc.Before(func(_ context.Context, _ *godog.Scenario) (context.Context, error) {
        w.Reset()
        return nil, nil
    })
    registerSourceSteps(sc, w)
    registerAnalyzeSteps(sc, w)
    registerCLISteps(sc, w)
    registerOutputSteps(sc, w)
}
```

### Style guide for .feature files

- **One Feature per file** — a feature = one user-facing capability.
- **Background** — use for repeated setup (`Given chamele is configured
  with default options`).
- **Scenario Outline + Examples** — use for parametrised cases like
  "every language detects a function with N parameters".
- **Tag** scenarios as `@slow`, `@cli` (spawns subprocess), `@io`
  (touches disk), so CI can scope runs.
- Step phrasing: stick to a small, reusable vocabulary. Examples:
  - `Given a Go file containing:` (docstring follows)
  - `When I analyze it`
  - `Then the function "X" should have CCN <n>`
  - `Then the exit code should be <n>`
  - `Then the output should equal:` (docstring follows)
  Don't write language-specific verbs (`When I lizard it`) — keep generic.

### Example feature file

```gherkin
Feature: Cyclomatic complexity in Go source
  As a developer using chamele as a library
  I want CCN computed from token-level structure
  So that I can flag functions that exceed a threshold

  Background:
    Given chamele is configured with default options

  Scenario: A function with no branches has CCN 1
    Given a Go file containing:
      """
      package x
      func hello() { fmt.Println("hi") }
      """
    When I analyze it
    Then the function "hello" should have CCN 1
    And the function "hello" should have 0 parameters

  Scenario Outline: Common control-flow constructs each add 1 to CCN
    Given a Go file containing:
      """
      package x
      func f(a int) { <body> }
      """
    When I analyze it
    Then the function "f" should have CCN <ccn>

    Examples:
      | body                            | ccn |
      | if a > 0 { return }             |  2  |
      | for i := 0; i < a; i++ { x() }  |  2  |
      | switch a { case 1: }            |  2  |
      | if a > 0 && a < 10 { x() }      |  3  |
```

### How features map to translated unit tests

The translated unit tests in `languages/<lang>/*_test.go` are precise and
low-level (one assertion per case). The BDD scenarios are **broader, fewer,
and behaviour-focused**. For each language we aim for:

- **~10–30 scenarios** per language (cover the major constructs +
  edge cases that have ever caused bugs).
- **NOT one scenario per upstream test case** — that's Layer 1's job.

If a scenario fails: triage by reading the scenario name. The unit tests
will usually pinpoint which translation case regressed.

### Tests / CI

- `make bdd` runs `go test ./features/...` (alias for the same).
- CI runs BDD as part of the default `go test ./...` invocation in
  `.github/workflows/ci.yml`.
- On failure, godog's pretty formatter highlights the failing step. Step
  definitions should produce diff-friendly error messages
  (`expected CCN 2, got 3`).

### Definition of done

- Every public surface (library API, CLI, each output format) has at
  least one Gherkin feature file.
- Every supported language has a `features/languages/<lang>.feature`.
- Every extension has a `features/extensions/<ext>.feature`.
- `go test ./features/...` passes on all CI platforms.
- The README has a "Read the features" section pointing newcomers to
  `features/` as the canonical user docs.

---

## Phase 10 — docs, examples, release v0.1

### Scope

- Godoc: every exported symbol has a doc comment, complete with examples
  for `chamele.Analyze`, `chamele.AnalyzeFile`, `chamele.WithLanguages`,
  etc.
- `examples/` directory with three runnable Go programs:
  1. Analyse a single file, print CCN per function.
  2. Walk a repo and produce CSV.
  3. Embed in a CI tool to fail-on-threshold.
- `cmd/chamele/README.md` — CLI usage cheat-sheet.
- A short `BENCHMARKS.md` reporting wall-clock vs Python lizard on a
  100k-LOC corpus (no fixed target — §17 row 18 — just transparency).
- `docs/divergences.md` — every deliberate behavioural difference from
  Python lizard (nested gitignore, default `--threads`, CRLF
  normalisation, modernised flag names, HTML layout).
- `docs/extending.md` — how to write a custom language reader or
  extension using the public `languages.Register` and
  `chamele.RegisterExtension` APIs (§17 row 17).
- Tag `v0.1.0`. `.goreleaser.yaml` builds binaries for linux/amd64,
  linux/arm64, darwin/amd64, darwin/arm64, windows/amd64.

### Definition of done

- `go install github.com/iszlai/chamele-go/cmd/chamele@v0.1.0` works.
- pkg.go.dev shows clean godoc.
- Release notes name every supported language and extension.
- README links to features/ as the canonical behaviour spec.

---

## 16. Cross-cutting conventions

These apply to every phase. A junior engineer or coding agent picking up
work should re-read this section before opening a PR.

### Style

- `gofmt` + `goimports` mandatory.
- `golangci-lint run` clean.
- Test names: `TestUnit_<Type>_<Behaviour>` for table tests,
  `TestE2E_<Scenario>` for integration.
- No `interface{}` unless wrapping a true ad-hoc value (use generics if
  the type is uniform).

### Mapping conventions

- **Python `self.foo = bar` mutation** in `CodeStateMachine` →
  exported `Foo` on the Go struct **only** if external code reads it;
  otherwise unexported.
- **Python `_foo` (single underscore)** → unexported in Go.
- **Python `_state = self._other_state`** → return the next `stateFn`
  from the current state. The state machine engine reassigns it.
- **Method decorators (`@CodeStateMachine.read_inside_brackets_then`)**
  → helper functions that wrap the body, e.g.:
  ```go
  m.next(readInsideBracketsThen('{', '}', m._stateGlobal, func(tok string) {
      // body
  }))
  ```

### Documentation style for ported files

Top of every ported Go file:

```go
// Derived from lizard's <upstream-path> (commit 4ad5454).
// Copyright Terry Yin and contributors, MIT-licensed.
```

### Test parity

Each ported test file's name mirrors the upstream one, lowercased:
- `testCAndCPP.py` → `clike_test.go`
- `testJava.py` → `java_test.go`
- `testGo.py` → `golang_test.go`

Inside, group tests with `t.Run("subtest", …)` matching the pytest
parametrize labels so a failure traces back to a specific upstream case.

### Adding a new language

1. Pick an extension list (`["foo"]`).
2. Write the `Reader` (extends `clike` or stands alone).
3. Add a side-effect `init()` that calls
   `languages.Register(NewReader)`.
4. Translate the upstream test file → `foo_test.go` (Layer 1).
5. Add `features/languages/foo.feature` covering the language's main
   constructs (Layer 2 — see Phase 9 for the style guide).
6. Add to `languages/all/all.go`.
7. Add to the `--languages` CLI help enumeration.
8. Tick the row in Appendix B.

### Adding a new extension

1. Implement `chamele.Extension` in `ext/<name>/`.
2. Export `New() chamele.Extension`.
3. Translate the upstream extension test (Layer 1).
4. Add `features/extensions/<name>.feature` covering the metric the
   extension adds (Layer 2).
5. Wire into `ext/all/all.go`.
6. Add a CLI flag if the extension has its own threshold (see `ext/nd`
   for the pattern).

---

## 17. Resolved decisions

These were settled before Phase 0 starts. If you disagree with one of
them mid-port, open an issue — don't quietly deviate.

| # | Decision | Notes |
|---|---|---|
| 1 | **Module path: `github.com/iszlai/chamele-go`. Binary name: `chamele`.** | Avoids PATH conflict with Python lizard on the same machine. |
| 2 | **License: MIT only.** | Matches upstream lizard. Single `LICENSE` file at the root. Every ported Go file still carries the upstream attribution header (see §16). |
| 3 | **CLI flags: modernised only, no legacy aliases.** | Go/POSIX convention (`--threads`, `--input-file`, `--ccn`). Scripts that currently call Python lizard need a one-time flag update before they can swap to `chamele`. The full flag table in Phase 7 is the source of truth. |
| 4 | **Default thread count: `runtime.NumCPU()`.** | Diverges from upstream's `1`. CLI flag `--threads N` overrides. Document the divergence in `docs/divergences.md`. |
| 5 | **Go version floor: 1.23.** | Unlocks `iter.Seq` range-over-func for the token pipeline. `go.mod` declares `go 1.23`. CI tests on 1.23 and latest (1.26). |
| 6 | **Coverage gate per reader: 100% of upstream tests, minus explicitly-tracked skips.** | Every skip is one line in `docs/divergences.md` with a reason (Python-specific quirk, deliberate semantic change, known bug etc.). PR reviewer rejects unjustified `t.Skip`. |
| 7 | **Hosting: `iszlai/chamele-go`, public from day 1.** | MIT license is on every commit. `NOTICE` and `LICENSE` live at the repo root from the first push. |
| 8 | **Public library Go package: `chamele`.** | `import "github.com/iszlai/chamele-go/chamele"`; call sites read `chamele.Analyze(...)`. Internally consistent with module + binary name. |
| 9 | **CLI library: `spf13/cobra` + `spf13/pflag`.** | Mature, handles ~25 flags with grouped help. Worth the extra direct dep. |
| 10 | **Output ordering: always deterministic.** | Workers index their results; printer sorts before emission. Trades O(n) result-slice memory for stable diffs. |
| 11 | **Per-file error policy: skip + log to stderr (match upstream).** | Library callers get an empty `FileInformation` for failed files; check `if fi.IsEmpty()`. CI doesn't fail a whole run on one bad file. |
| 12 | **BDD framework: cucumber/godog.** | Official Go Cucumber. `.feature` files in `features/` at repo root. |
| 13 | **`.gitignore`: walk nested files (fix upstream bug).** | Deliberate divergence from Python lizard's single-`.gitignore` behaviour. Document in `docs/divergences.md`. Matches git's actual semantics. |
| 14 | **BDD features directory: `features/` at repo root.** | Standard Cucumber convention. Shared step package in `features/steps/`. |
| 15 | **BDD invocation: plain `go test ./features/...`.** | godog wires into `testing.T` via TestSuite shim. Contributors run BDD by default. |
| 16 | **Step definitions: one shared `features/steps/` package.** | Steps are reused across features. Easy to find a step from a .feature file. |
| 17 | **Public API shape: slice-returning + functional options.** | `chamele.Analyze(paths, chamele.WithLanguages("go"), chamele.WithThreads(8)) ([]FileInformation, error)`. Simple, familiar. Streaming variant deferred to v0.2 if demand arises. |
| 18 | **CLI command structure: single command.** | `chamele [flags] <paths...>`. No subcommands in v0.1; future `chamele check`, `chamele report` are additive and non-breaking. |
| 19 | **Output emission: buffer all, emit at end.** | Required for XML/HTML wrapping documents and the cppncss summary section. Memory cost negligible (records are tiny). |
| 20 | **Version stamp: hardcoded `const Version` in `chamele/version.go`.** | Matches upstream's `lizard_ext/version.py`. Bumped explicitly per release. |
| 21 | **HTML output: data-equivalent only.** | Same metrics, same table structure; CSS/layout/look-and-feel can diverge. Documented in `docs/divergences.md`. |
| 22 | **Line endings: normalise CRLF→LF at file read.** | One place to handle it (`internal/stringx/lineend.go`). State machines see only `\n`. |
| 23 | **Direct deps: cobra, pflag, godog, go-gitignore, errgroup.** | Pragmatic minimum, all well-maintained. Listed explicitly in `go.mod`. |
| 24 | **Plugin API: public from v0.1.** | `languages.Register(reader)` and `chamele.RegisterExtension(ext)` are part of v0.1 surface. Breaking changes to the `Reader` / `Extension` interfaces follow SemVer. |
| 25 | **Performance: no fixed target.** | `BENCHMARKS.md` reports vs Python lizard on a fixed corpus; releases don't gate on the number. Avoids premature optimisation. |
| 26 | **Upstream re-sync: manual, every 3–6 months.** | Bump the pin in `scripts/fetch-upstream.sh`, diff `CHANGELOG.md`, port new tests/readers. Cadence noted in `CONTRIBUTING.md`. |
| 27 | **`lizard-upstream/`: gitignored, fetched on demand.** | `scripts/fetch-upstream.sh` clones at the pinned SHA before parity tests. Keeps repo lean. |

---

## Appendix A — full file map of upstream

### Engine + tokenizer

| Upstream file | LOC | Maps to |
|---|---|---|
| `lizard.py` | 1163 | `lizard/*.go` (split per concern) |
| `lizard_languages/__init__.py` | 67 | `languages/languages.go` |
| `lizard_languages/code_reader.py` | 238 | `internal/tokenizer/*.go` |

### Language readers (27)

| File | LOC | Parent | Test file | LOC (test) |
|---|---|---|---|---|
| `clike.py` | 398 | — | `testCAndCPP.py` | 797 |
| `csharp.py` | 38 | clike | `testCsharp.py` | 219 |
| `erlang.py` | 124 | — | `testErlang.py` | 158 |
| `fortran.py` | 260 | — | `testFortran.py` | 525 |
| `gdscript.py` | 90 | — | `testGDScript.py` | 43 |
| `go.py` | 46 | golike | `testGo.py` | 168 |
| `golike.py` | 85 | — (helper) | — | — |
| `java.py` | 252 | clike | `testJava.py` | 685 |
| `javascript.py` | 76 | clike | `testJavaScript.py`, `testES6.py` | 839 + 513 |
| `js_style_regex_expression.py` | TBD | — (helper) | — | — |
| `kotlin.py` | 73 | clike | `testKotlin.py` | 226 |
| `lua.py` | 49 | — | `testLua.py` | 177 |
| `objc.py` | 73 | clike | `testObjC.py` | 115 |
| `perl.py` | 333 | rubylike | `testPerl.py`, `test_perl_func_names.py` | 919 + 20 |
| `php.py` | 272 | clike | `testPHP.py` | 563 |
| `plsql.py` | 422 | — | `testPLSQL.py` | 751 |
| `python.py` | 152 | script_language | `testPython.py`, `testPythonForgive.py` | 517 + 63 |
| `r.py` | 293 | — | `testR.py` | 588 |
| `ruby.py` | 65 | rubylike | `testRuby.py` | 435 |
| `rubylike.py` | 109 | — (helper) | — | — |
| `rust.py` | 34 | — | `testRust.py` | 142 |
| `scala.py` | 49 | clike | `testScala.py` | 130 |
| `script_language.py` | 31 | — (helper) | — | — |
| `solidity.py` | 27 | clike | `testSolidity.py` | 47 |
| `st.py` | 143 | — | `testSt.py` | 404 |
| `swift.py` | 76 | clike | `testSwift.py` | 277 |
| `tnsdl.py` | 97 | — | — | — |
| `tsx.py` | 167 | typescript | `testTSX.py`, `testJSX.py` | 739 + 434 |
| `ttcn.py` | 64 | — | `testTTCN.py` | 386 |
| `typescript.py` | 639 | clike | `testTypeScript.py` | 1228 |
| `vue.py` | 34 | — | `testVue.py` | 75 |
| `zig.py` | 31 | — | `testZig.py` | 242 |

### Extensions (22)

See **Appendix C** below.

### Other engine tests (must port)

| File | LOC | Notes |
|---|---|---|
| `testCyclomaticComplexity.py` | ~60 | CCN parametrize table |
| `testTokenizer.py` | ~120 | tokenizer table |
| `testAnalyzer.py` | TBD | engine integration |
| `testHelpers.py` | 29 | helpers used by every language test |
| `testApplication.py` | TBD | CLI behaviour |
| `testFilesFilter.py` | TBD | walk / exclude / dedup |
| `testCommentOptions.py` | TBD | `#lizard forgive` semantics |
| `testNestingDepth.py` | TBD | ND extension integration |
| `testNestedStructures.py` | TBD | NS extension integration |
| `testCyclomaticComplexity.py` | TBD | |
| `testExtension.py` | TBD | extension framework |
| `testOutput.py`, `testOutputCSV.py`, `testOutputHTML.py`, `testOutputFile.py` | TBD | output formats |

---

## Appendix B — per-language reader checklist

For each language, the porter must produce:

- [ ] `languages/<lang>/<lang>.go` — `Reader` struct + parallel state machines
- [ ] `languages/<lang>/<lang>_test.go` — translation of the upstream pytest
- [ ] An `init()` that calls `languages.Register(NewReader)`
- [ ] Entry in `languages/all/all.go` (side-effect import)
- [ ] Listed in the CLI `--help` `--languages` enumeration
- [ ] README.md table updated

Mark a reader "shipped" only when its upstream test file is 100%
translated and green.

### Tracking matrix

| Language | Status | Tests passing | Notes |
|---|---|---|---|
| C/C++ | ☐ | 0 / 797 | |
| C# | ☐ | 0 / 219 | |
| Erlang | ☐ | 0 / 158 | |
| Fortran | ☐ | 0 / 525 | |
| GDScript | ☐ | 0 / 43 | |
| Go | ☐ | 0 / 168 | |
| Java | ☐ | 0 / 685 | |
| JavaScript | ☐ | 0 / 839 | |
| ES6 | ☐ | 0 / 513 | |
| JSX | ☐ | 0 / 434 | |
| Kotlin | ☐ | 0 / 226 | |
| Lua | ☐ | 0 / 177 | |
| Objective-C | ☐ | 0 / 115 | |
| Perl | ☐ | 0 / 939 | |
| PHP | ☐ | 0 / 563 | |
| PL/SQL | ☐ | 0 / 751 | |
| Python | ☐ | 0 / 580 | |
| R | ☐ | 0 / 588 | |
| Ruby | ☐ | 0 / 435 | |
| Rust | ☐ | 0 / 142 | |
| Scala | ☐ | 0 / 130 | |
| Solidity | ☐ | 0 / 47 | |
| ST | ☐ | 0 / 404 | |
| Swift | ☐ | 0 / 277 | |
| TSX | ☐ | 0 / 739 | |
| TTCN | ☐ | 0 / 386 | |
| TypeScript | ☐ | 0 / 1228 | |
| Vue | ☐ | 0 / 75 | |
| Zig | ☐ | 0 / 242 | |

---

## Appendix C — per-extension checklist

| Extension | Upstream file | LOC | Adds to FunctionInfo | Notes |
|---|---|---|---|---|
| Nesting depth | `lizardnd.py` | 215 | `max_nesting_depth` | Heavy context mutation. Port literally first. |
| Nested structures | `lizardns.py` | 136 | `max_nested_structures` | Uses bracket-counting decorator. |
| Modified CCN | `lizardmodified.py` | 25 | replaces `cyclomatic_complexity` semantics for switch | Triggered via `-m` flag too. |
| McCabe (alt CCN) | `lizardmccabe.py` | 41 | `mccabe` | Includes `?` weight; close to default CCN but counts logical operators differently. |
| Fan-in / fan-out | `lizardio.py` | 92 | `fan_in`, `fan_out`, `general_fan_out` | Cross-file post-process; keeps tokens per fn. |
| C preprocessor | `lizardcpre.py` | 37 | — | Skips code inside `#else` ... `#endif` branches; runs first (`ordering_index = 0`). |
| Ignore assert | `lizardignoreassert.py` | 21 | — | Drops tokens inside `assert(...)`. |
| Duplicate detection | `lizardduplicate.py` | 314 | `duplicates` (cross-file) | Sliding hash window. |
| Duplicate param list | `lizardduplicated_param_list.py` | 53 | `duplicate_param_list` | Reports identical param signatures. |
| Bool count | `lizardboolcount.py` | 34 | `bool_count` | |
| Exit count | `lizardexitcount.py` | 22 | `exit_count` | `return`/`exit` per function. |
| Goto count | `lizardgotocount.py` | 16 | `goto_count` | |
| Statement count | `lizardstatementcount.py` | 32 | `statement_count` | Counts semicolons. |
| Dependency count | `lizarddependencycount.py` | 57 | `dependency_count` | Imports per file. |
| Complex tags | `lizardcomplextags.py` | 25 | `complex_tags` | |
| Word count | `lizardwordcount.py` | 231 | — (custom `print_result`) | Produces tag-cloud HTML; reuses `keywords.py`. |
| Dump comments | `lizarddumpcomments.py` | 26 | — | Debug aid. |
| Outside (global) | `lizardoutside.py` | 12 | — | Treats global as one function. |
| Nonstrict | `lizardnonstrict.py` | 14 | — | Marker; sets `silent_all_others`. |
| (utility) `keywords.py` | — | 80 | — | Stop-word list for word count. |
| (utility) `default_ordered_dict.py` | — | 30 | — | Use `map` + slice of keys in Go. |
| (output) `xmloutput.py` | — | 168 | — | Phase 7. |
| (output) `htmloutput.py` | — | 229 | — | Phase 7. |
| (output) `csvoutput.py` | — | 56 | — | Phase 7. |
| (output) `checkstyleoutput.py` | — | 33 | — | Phase 7. |
| (utility) `auto_open.py` | — | 36 | — | BOM detection helper. |

---

---

## Status quo — 2026-05-17

### What is done

- **Phase 0** — bootstrap: go.mod, Makefile, CI/parity workflows, .gitignore, LICENSE, NOTICE, CONTRIBUTING.md, all package skeletons, godog BDD shim.
- **Phase 1** — tokenizer + state machine + core types: `internal/tokenizer` (GenerateTokens, Machine, helpers), `chamele` (FunctionInfo, FileInformation, NestingStack, FileInfoBuilder), 29 tests green.
- **Phase 2** — analysis engine: 5 processors (Preprocessing, CommentCounter, LineCounter, TokenCounter, ConditionCounter), FileAnalyzer, Analyze/AnalyzeFile/AnalyzeFiles, walk with nested gitignore + md5 dedup, WarningFilter, WhitelistFilter, OutputScheme.
- **Phase 3** — C/C++ and Java readers: CLikeReader (3 parallel state machines), JavaReader (depth-tracking, annotation skip, class body nesting). 25 C++ + 10 Java tests green.
- **Phase 4** — Python, Go, JavaScript: PythonReader (indentation-based), GoReader (brace-depth + funcDepths, IsInsideFunction logic), JSReader (PushNewFunction/EndOfFunction). All pass.
- **Phase 5** — all 27 language readers registered in `languages/all/all.go`. Readers with tests: C/C++, C#, Erlang, Fortran, GDScript, Go, Java, JS, Kotlin, Lua, ObjC, Perl, PHP, PLSQL, Python, R, Ruby, Rust, Scala, Solidity, ST, Swift, TSX, TTCN-3, TypeScript, Vue, Zig. **21 test packages green.**
- **Phase 6** — extension framework: updated Extension interface (Process takes ctx), CrossFileExtension, Printer. Implemented: modified, ignoreassert, exitcount, gotocount, statementcount, boolcount, outside, nonstrict, dumpcomments. Stubs (pass-through): cpre, mccabe, ns, nd, duplicate, dupparams, dependencycount, complextags, wordcount, io. All in `ext/all/all.go`.
- **Phase 7** — CLI + output formats: `cmd/chamele/main.go` with all 25 flags (cobra), tabular/CSV/XML/HTML/Checkstyle/clang/MSVS formatters. `chamele.xsl` vendored locally. Binary builds and produces correct output end-to-end.
- **Phase 8** — parity test framework: `test/parity/parity_test.go` (build tag `parity`), 19 Perl/Python corpus files in `testdata/`. CCN differences are hard failures; NLOC/token are soft (logged). Perl ternary CCN is marked a known soft divergence. **Passes with `go test -tags parity ./test/parity/...`**
- **Phase 9** — BDD suite: step definitions (source, analyze, output), feature files: `analyze.feature`, `forgiveness.feature`, `features/languages/go.feature`. 14 scenarios / 64 steps — all green. godog upgraded to v0.15.1.
- **Phase 10** — `docs/divergences.md` (8 entries), `examples/` (3 programs), version bumped to `v0.1.0`, `.goreleaser.yaml`.

### What is NOT done (open items)

1. **Language test files missing** (9 languages have no `_test.go`): erlang, fortran, gdscript, lua, plsql, r, ruby, st, vue.
2. **Extension stubs** (5 pass-through with no real logic): `ext/nd`, `ext/ns`, `ext/mccabe`, `ext/duplicate`, `ext/io`.
3. **BDD feature files missing**: walk_filter, whitelist, output_tabular/csv/xml/html/checkstyle, cli_exit_codes, and one per language (26 missing) and one per extension (19 missing, `features/extensions/` is empty).
4. **Docs**: `BENCHMARKS.md` and `docs/extending.md` not written.
5. **CI golangci-lint** — currently failing because golangci-lint v2 `.golangci.yml` schema issues. The valid v2 schema uses `linters.settings` (not `linters-settings`), formatters as a top-level key, and `issues` with changed sub-keys. Next step: look up the exact v2.12 reference schema at https://raw.githubusercontent.com/golangci/golangci-lint/v2.12.2/.golangci.reference.yml and rewrite `.golangci.yml` accordingly, then fix the 20 errcheck/staticcheck/ineffassign/gofmt issues flagged in the last run.
6. **Lint errors to fix** (from last CI run):
   - `cmd/chamele/main.go:142` — unchecked `fmt.Sscanf` return
   - `cmd/chamele/main.go:154` — unchecked `fh.Close()` in defer
   - `features/steps/source_steps.go:102,105` — unchecked `f.Close()`
   - `output/checkstyle.go:12,13,31,33` — unchecked `fmt.Fprintln/Fprintf`
   - `output/clang_warning.go:14` — unchecked `fmt.Fprintf`
   - `output/csv.go:24` — unchecked `fmt.Fprintf`
   - `languages/golang/golang.go:60`, `languages/lua/lua.go:57`, `languages/php/php.go:46` — gofmt issues
   - `output/tabular.go:105` — ineffassign `fnCount`
   - `output/tabular.go:140` — staticcheck: use `append(all, files[i].Functions...)`
   - `chamele/builder.go:141` — staticcheck: remove embedded field from selector
   - `chamele/output_scheme.go:76` — staticcheck: use `fmt.Fprintf` instead of `WriteString(fmt.Sprintf(...))`
   - `languages/fortran/fortran.go:117`, `languages/perl/perl.go:138`, `languages/ruby/ruby.go:101` — staticcheck: tagged switch
   - `languages/plsql/plsql.go:61` — unused field `inFunc`

---

## Quick start for a new contributor

You just got handed this plan. To start:

1. Read §§ 1–6 (one sitting, ~25 minutes). §6 (Testing strategy) and §17
   (Resolved decisions) are the load-bearing references.
2. Run `make bootstrap` once to fetch `lizard-upstream/` locally.
3. Pick the lowest-numbered phase that isn't done. The Appendix B/C
   matrices show which sub-tasks are open.
4. Read the upstream Python file for your sub-task **once, fully**,
   before writing any Go.
5. Open a branch `phase-<n>-<thing>`. Keep PRs small (one reader or one
   extension per PR is ideal).
6. Write tests in this order:
   - Translate the upstream Python test file → Go table tests (Layer 1).
     Make them compile, mark them `t.Skip("not yet implemented")`.
   - For a new reader/extension, add a `features/.../<x>.feature`
     covering the main user-facing behaviour (Layer 2).
   - Implement until both layers pass.
7. Cite the upstream file + line range in your PR description.
8. Cross off the checkbox in Appendix B or C as part of the PR.

When stuck:
- Run the Python lizard on the exact input that confuses you:
  `python3 -m lizard --csv yourfile.<ext>` — it's the ground truth.
- Read `lizard-upstream/CHANGELOG.md` if behaviour seems weird; it's
  often historical.
- The state-machine pattern is the hard part. If the Python code uses a
  decorator you haven't seen, search Phase 1 for the helper that
  replaces it.
- If a Gherkin step doesn't exist yet, add it to
  `features/steps/<area>_steps.go` and reuse from then on. Don't
  duplicate step phrasings.

Good luck.
