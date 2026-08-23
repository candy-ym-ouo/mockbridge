# Bug Reproduction

Baseline: `origin/green_base_bug_007`

Scheduler starts two loops for each registered job. The same job can consequently execute concurrently, duplicating work and sharing state that was designed for one loop.

Run:

```text
go test -race ./internal/worker -count=20 -timeout=5s -run '^TestBug07_SchedulerStartsOneLoopPerJob$'
```

Expected result: the test observes more than one execution loop for the single registered job.
