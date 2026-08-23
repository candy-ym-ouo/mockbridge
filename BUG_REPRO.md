# Bug Reproduction

Baseline: `origin/green_base_bug_008`

The switcher ignores activation errors and continues as if the scheduled transition succeeded. Scheduler status therefore loses the failure and the caller cannot respond to it.

Run:

```text
go test -race ./internal/worker -count=20 -timeout=5s -run '^TestBug08_SwitcherPropagatesActivationFailure$'
```

Expected result: the test reports success from the switcher even though activation uses an invalid target.
