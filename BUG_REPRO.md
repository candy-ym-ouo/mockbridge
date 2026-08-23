# Bug Reproduction

Baseline: `origin/green_base_bug_004`

Rendering a plain response without configured headers returns a nil header map. The handler later adds response metadata to that map, producing a runtime panic.

Run:

```text
go test -race ./internal/responder -count=20 -timeout=5s -run '^TestBug04_RenderAlwaysReturnsWritableHeaders$'
```

Expected result: the test reports that rendering returned nil headers.
