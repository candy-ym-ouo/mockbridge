# Bug Reproduction

Baseline: `origin/green_base_bug_005`

The repository converts a missing-record sentinel to text, and the service wraps that text with a non-unwrappable format. The HTTP layer can no longer recognize the domain error with `errors.Is`.

Run:

```text
go test -race ./internal/service -count=20 -timeout=5s -run '^TestBug05_RecordNotFoundSurvivesServiceWrapping$'
```

Expected result: `errors.Is` does not recognize `repository.ErrNotFound`.
