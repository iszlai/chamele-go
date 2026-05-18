# chamele-go

A pure-Go, 1-to-1 feature-complete port of [lizard](https://github.com/terryyin/lizard),
the extensible cyclomatic complexity analyzer for 27+ programming languages.

> Status: **porting complete, cleanup in flight**.
> - [`PLAN.md`](./PLAN.md) — original porting plan (27 languages, 19
>   extensions, 7 output formats — all landed).
> - [`CLEANUP.md`](./CLEANUP.md) — phased refactor plan to drop duplication,
>   remove dead code, and finish the idiomatic-Go pass. Pick any phase.

## What lizard does

- Counts NLOC (lines of code without comments), cyclomatic complexity (CCN),
  token count, parameter count, function length and nesting depth.
- Works without needing imports/headers resolved — it estimates structural
  complexity from tokens, not a real AST.
- Supports C, C++, C#, Erlang, Fortran, GDScript, Go, Java, JavaScript (ES6 +
  JSX), Kotlin, Lua, Objective-C, Perl, PHP, PL/SQL, Python, R, Ruby, Rust,
  Scala, Solidity, Structured Text (ST), Swift, TTCN-3, TypeScript (+ TSX),
  Vue, Zig.
- Outputs: tabular, XML (cppncss), CSV, HTML, Checkstyle XML, clang/MSVC
  warning formats.

## Why a Go port

- Single static binary; no Python runtime, no pip, no venv on the host.
- Embeddable as a Go library — drop-in replacement for the multiple external
  complexity tools we currently shell out to (`radon`, `eslint`,
  `pmd`, `lizard`, etc.).
- Easier to ship inside a Docker base image or a CI container.

## Pinned upstream

This port targets lizard upstream commit
[`4ad5454`](https://github.com/terryyin/lizard/commit/4ad545491e8a141a72bb52b8d719b1842c5a7d1b)
(just past v1.22.1). The upstream tree is checked out under `lizard-upstream/`
as a read-only reference; do **not** modify it.

## License

MIT. The upstream lizard is MIT-licensed; we keep the original copyright
notice in `NOTICE`. See [`PLAN.md` §2](./PLAN.md#2-license) for the audit.
