# Bug Reproduction

Baseline: `origin/green_base_bug_003`

Scheduler worker registration happens inside the newly created goroutine. `Wait` can run before `Add`, return too early, and leave the worker lifecycle outside the wait boundary.

Run:

```text
go test -race ./internal/worker -count=20 -timeout=2s -run '^TestBug03_WaitDoesNotReturnBeforeWorkerRegistration$'
```

Expected result: the test reaches the scheduler wait boundary before the worker is registered, or times out while the lifecycle is inconsistent.
