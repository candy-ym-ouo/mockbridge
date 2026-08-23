# Bug Reproduction

Baseline: `origin/green_base_bug_010`

The explicit shutdown path closes the record queue and the deferred close path closes it again. The second close panics instead of following an idempotent service lifecycle.

Run:

```text
go test -race ./internal/service -count=20 -timeout=5s -run '^TestBug10_ShutdownAndCloseAreIdempotent$'
```

Expected result: the test reports a close-of-closed-channel panic.
