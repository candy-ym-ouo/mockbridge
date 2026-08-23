package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"mockbridge/internal/database"
)

func TestBug09_CleanBeforeReturnsDedicatedConnection(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "bug09.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := NewRecordRepository(db)
	for i := 0; i < 4; i++ {
		if _, err = repo.CleanBefore(context.Background(), time.Now(), 10); err != nil {
			t.Fatal(err)
		}
	}
	if inUse := db.Stats().InUse; inUse != 0 {
		t.Fatalf("cleaner left %d database connections in use", inUse)
	}
}
