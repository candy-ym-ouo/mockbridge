# Bug Reproduction

Baseline: `origin/green_base_bug_002`

The scenario snapshot is published as a slice and the matcher sorts that slice in place. A request can therefore reorder shared snapshot state while another request is reading it.

Run:

```text
go test -race ./internal/matcher -count=20 -timeout=5s -run '^TestBug02_MatchDoesNotReorderPublishedSnapshot$'
```

Expected result: the test reports that the published snapshot was reordered.
