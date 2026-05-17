# Deliberate divergences from Python lizard

This file documents every place where chamele intentionally (or knowingly)
differs from Python lizard v1.22.1 behaviour. Each entry has a reason and a
tracking note.

## 1. Nested `.gitignore` files honoured at every directory level

**Python lizard** reads only the first `.gitignore` it finds when walking a
directory tree.

**chamele** uses `go-gitignore` per directory and stacks parsers as it
descends, matching git's actual multi-level `.gitignore` semantics.

*Rationale:* This is a bug fix, not a regression. Nested ignores are the
correct behaviour.

## 2. Default thread count — `runtime.NumCPU()` instead of 1

Python lizard defaults to a single worker (no parallelism). chamele defaults
to `runtime.NumCPU()`. Override with `--threads 1` to match the single-worker
behaviour.

*Rationale:* Modern hardware; single-threaded default was a Python
multiprocessing limitation.

## 3. CLI flag names modernised

Long flags use Go/POSIX kebab-case (`--input-file`, `--warnings-only`,
`--ignore-warnings`, `--ccn`) instead of the mixed underscores of lizard
(`--input_file`, `--warnings_only`). The short forms (`-C`, `-i`, `-w`, etc.)
are unchanged. Scripts that invoke the Python binary need a one-time flag rename
to use chamele.

## 4. CRLF → LF normalisation at file read time

All input files are normalised from `\r\n` to `\n` before tokenisation.
Python lizard does the same via Python's universal newline support. Behaviour
is equivalent.

## 5. HTML output layout

The HTML formatter produces a valid but minimal HTML table. Python lizard's
HTML uses custom templates with CSS. The **data** (function metrics) is
identical; the visual layout differs. See §17 row 21 of PLAN.md.

## 6. Perl ternary CCN

Python lizard counts `?` and `:` as separate conditions for the ternary
operator, AND also appears to count comparison operators (`>`, `<`, etc.) in
some contexts, leading to higher CCN values for Perl ternary-heavy functions.
chamele counts only `?` and `:` as stated in the upstream code.

*Status:* Under investigation for v1.0. Parity test marks this as a soft
divergence.

## 7. Perl NLOC and token count

Python lizard's Perl preprocessor handles `#` comment accumulation slightly
differently, producing different intermediate token counts. The token_count
field for Perl files is typically 40–60% higher in Python lizard.
CCN, parameter_count, and function detection are unaffected.

*Status:* Known tokenizer divergence; tracked for v1.0 fix.

## 8. Performance

chamele is single-threaded slower than Python lizard on tiny corpora (< 20
files) due to Go binary startup. On ≥ 30 files chamele is 2–3× faster
single-threaded, and 6–10× faster with `runtime.NumCPU()` workers.
