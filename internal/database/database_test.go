package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
)

func TestOpenAndMigrateIdempotent(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err = Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&n); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestEveryConnectionEnforcesForeignKeys(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "pragma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err = Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	conn1, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close()
	conn2, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
	for i, conn := range []*sql.Conn{conn1, conn2} {
		var foreignKeys, timeout int
		if err = conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
			t.Fatalf("connection %d foreign_keys=%d err=%v", i, foreignKeys, err)
		}
		if err = conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&timeout); err != nil || timeout != 5000 {
			t.Fatalf("connection %d busy_timeout=%d err=%v", i, timeout, err)
		}
	}
	res, err := conn1.ExecContext(ctx, "INSERT INTO contracts(key,name,created_at,updated_at) VALUES('cascade','Cascade','now','now')")
	if err != nil {
		t.Fatal(err)
	}
	contractID, _ := res.LastInsertId()
	if _, err = conn1.ExecContext(ctx, "INSERT INTO scenarios(contract_id,name,created_at,updated_at) VALUES(?,?,?,?)", contractID, "default", "now", "now"); err != nil {
		t.Fatal(err)
	}
	if _, err = conn2.ExecContext(ctx, "DELETE FROM contracts WHERE id=?", contractID); err != nil {
		t.Fatal(err)
	}
	var scenarios int
	if err = conn1.QueryRowContext(ctx, "SELECT COUNT(*) FROM scenarios WHERE contract_id=?", contractID).Scan(&scenarios); err != nil || scenarios != 0 {
		t.Fatalf("cascade scenarios=%d err=%v", scenarios, err)
	}
}

func TestMemoryDatabaseUsesOneSharedSchema(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var n int
			if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='contracts'").Scan(&n); err != nil || n != 1 {
				t.Errorf("schema count=%d err=%v", n, err)
			}
		}()
	}
	wg.Wait()
}
