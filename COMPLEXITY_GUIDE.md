# Complexity Guide

## Hardline

- `TestFileComplexityGuard` is the hard gate for every Go file under `internal/`.
- `TestModuleComplexityGuard` is the hard gate for every ownership module under `internal/`.
- Source files must stay at `<=80` complexity and `<=400` code lines.
- Modules must stay at `<=1200` source complexity and `<=6000` source code lines.
- Test files must stay at `<=800` code lines.
- Module test totals are reported for visibility but do not fail the guard.
- Do not add exemptions and do not raise the limits to make a change pass.

## When the Guard Fails

1. Read the failing file or module and label each block with repo vocabulary from `VOCABULARY.md`.
2. Split the failing file by one responsibility before changing behavior.
3. If the module still fails, move a stable responsibility into a vocabulary-named ownership module under `internal/`.
4. Re-run the guard after each split so the next move is based on the current hotspot.

## Preferred Split Order

- Split pure transformations first: parsers, codecs, renderers, chunkers, formatters, summary builders.
- Split data-pipeline stages next: `scan` vs `parse` vs `projection` vs `subagent`.
- Split archive work next: `sync` vs `import analysis` vs archive config or filesystem helpers.
- Split canonical-store work next: `rebuild` vs `catalog` vs `transcript` vs `search corpus` vs binary codec.
- Split TUI work next: `browser` list state vs transcript pane orchestration, `viewer` search vs render, `transcript` segments vs preview rendering.

## When to Extract a Module

- If a file still breaches the cap after a responsibility split, the problem is usually mixed orchestration.
- Extract a dedicated module under `internal/` when one consumer needs a stable seam to talk to another flow.
- Keep interfaces in the consumer module, not the provider module.
- Name the new module with vocabulary terms only. If the term does not exist, add it to `VOCABULARY.md` first.
- Do not create generic catch-all modules like `util`, `helper`, `service`, `core`, or `manager`.

## Test Split Rules

- Split oversized test files by behavior, not by assertion style.
- Keep helpers close to the behavior that uses them unless they are shared across the whole scanner or store flow.
- Move tests with the code instead of exporting internals only for test access.
- Keep scenario coverage at the outer layer even when unit tests are redistributed.

## What Not to Do

- Do not keep adding branches to the failing file while the guard is red.
- Do not combine unrelated fixes into the same extraction.
- Do not hide complexity behind untyped maps or `any`.
- Do not keep a flat layout by force when the usage boundary is already clear.

## Verification

```bash
gofmt -w <changed-files>
go test ./internal/app -run TestFileComplexityGuard -count=1
go test ./internal/app -run TestModuleComplexityGuard -count=1
go test ./...
golangci-lint run ./...
go test -race ./...
```

- If the metrics changed, refresh `COMPLEXITY_BASELINE.md` with:

```bash
go test ./internal/app -run TestComplexityBaselineDocument -count=1 -update
go test ./internal/app -run TestComplexityBaselineDocument -count=1
```
