# Contributing to chamele-go

## Quick start

1. Run `make bootstrap` once to fetch `lizard-upstream/` and sync testdata.
2. Read `PLAN.md` §§ 1–6 before writing any code.
3. Run `make test` to verify everything compiles and tests pass.

## Upstream reference

The pinned upstream commit is `4ad5454` (lizard v1.22.1+).
The upstream tree lives in `lizard-upstream/` (gitignored; fetched by `make bootstrap`).
When in doubt about behaviour, run:

```sh
python3 -m lizard --csv yourfile.<ext>
```

and match the output.

## Branch naming

`phase-<n>-<thing>` — e.g. `phase-1-tokenizer`, `phase-3-java`.
Keep PRs small: one reader or one extension per PR is ideal.

## Test expectations

- Every ported reader must translate 100% of its upstream test file.
- Every skip must be documented in `docs/divergences.md`.
- BDD scenarios in `features/` cover the public API and CLI behaviour.

## Re-syncing upstream

See `PLAN.md` §17 row 26. To bump the pin:

1. Update `PIN` in `scripts/fetch-upstream.sh`.
2. Run `make bootstrap`.
3. Diff `lizard-upstream/CHANGELOG.md` for new readers/extensions.
4. Port new tests/readers and update PLAN.md.
