package migrations

import "embed"

// Files contains the versioned SQL migrations distributed with the binary.
//
//go:embed *.sql
var Files embed.FS
