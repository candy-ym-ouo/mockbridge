# Bug Reproduction

Baseline: `origin/green_base_bug_009`

`CleanBefore` obtains a dedicated database connection but does not close it on the successful path. Repeated cleanup calls retain connections in the pool.

Run:

```text
go test -race ./internal/repository -count=20 -timeout=5s -run '^TestBug09_CleanBeforeReturnsDedicatedConnection$'
```

Expected result: the test observes connections still in use after cleanup returns.
