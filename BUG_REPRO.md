# Bug Reproduction

Baseline: `origin/green_base_bug_006`

Concurrent request record submissions share one mutable staging field in `RecordService`. A later request can replace that field before the earlier request sends its record to the queue.

Run:

```text
go test -race ./internal/service -count=20 -timeout=5s -run '^TestBug06_SubmitRequestKeepsConcurrentRecordsDistinct$'
```

Expected result: the test reports fewer distinct request IDs than submitted requests or the race detector reports the shared write.
