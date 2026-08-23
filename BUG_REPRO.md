# Bug Reproduction

Baseline: `origin/green_base_bug_001`

The mock request path removes cancellation from the request context before the service applies a fixed response delay. Canceling the client context therefore leaves the delay and downstream hit processing running.

Run:

```text
go test -race ./internal/service -count=20 -timeout=5s -run '^TestBug01_CanceledMockRequestStopsDelay$'
```

Expected result: the test reports that a canceled request remained in the delay.
