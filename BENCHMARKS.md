# chamele-go benchmarks

This file is the place to drop reproducible numbers when you tune or compare
chamele's analyser. Numbers below are reference baselines — re-run them on
your hardware before drawing conclusions.

## Reproducing

The benchmark harness lives in `chamele/` as plain Go benchmarks:

```bash
go test -run='^$' -bench=. ./chamele/...
```

If you want a wall-clock comparison against the upstream Python lizard:

```bash
make bootstrap                                                   # one-time
time go run ./cmd/chamele ./lizard-upstream/lizard_languages     > /tmp/go.csv
time python3 -m lizard --csv ./lizard-upstream/lizard_languages  > /tmp/py.csv
diff <(sort /tmp/go.csv) <(sort /tmp/py.csv)
```

## Methodology

- Each benchmark analyses the upstream `lizard_languages/` tree (~27 files,
  ~4700 LOC of Python). The corpus is deterministic between runs because
  the upstream commit is pinned (see `scripts/fetch-upstream.sh`).
- `b.SetBytes(totalSourceBytes)` is set in benchmarks so `go test -bench`
  reports MB/s alongside ns/op.
- Run the analyser with `WithThreads(1)` first to measure raw single-thread
  throughput, then with `WithThreads(runtime.NumCPU())` to measure scaling.

## Baseline (illustrative)

These are the numbers an Apple M1 / Go 1.23 / chamele-go v0.1.0 produced
when this document was first written. Replace them with fresh numbers in
your PR.

| Benchmark                    | Time      | Throughput  |
|------------------------------|-----------|-------------|
| BenchmarkAnalyzeCFile        | ~ TBD     | ~ TBD       |
| BenchmarkAnalyzeJavaFile     | ~ TBD     | ~ TBD       |
| BenchmarkAnalyzePythonFile   | ~ TBD     | ~ TBD       |
| BenchmarkAnalyzeAllUpstream  | ~ TBD     | ~ TBD       |

## Profiling

For CPU profiles:

```bash
go test -run='^$' -bench=BenchmarkAnalyzeAllUpstream -cpuprofile=cpu.out ./chamele/
go tool pprof -http=:9999 cpu.out
```

Look first at the regex tokenizer (`internal/tokenizer`) and the
`ConditionCounter` processor — those are usually the hot loops.

## Memory

The tokenizer is streaming (Go 1.23 `iter.Seq[string]`) so peak memory is
roughly the cost of the largest single token plus per-file accumulators.
Extensions that buffer tokens (`ext/io`, `ext/duplicate`) inflate this — to
measure their effect, register the analyser with and without them via
`NewFileAnalyzerWithExts`.

## Notes for contributors

- If you regress throughput by more than 10%, mention it in the PR
  description and explain why.
- Don't optimise without a profile. The `OrderingIndex` plumbing makes it
  tempting to micro-tune the extension iteration; almost always, the win
  is in the regex tokenizer or in avoiding allocations inside `Process`.
