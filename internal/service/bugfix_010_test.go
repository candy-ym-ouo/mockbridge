package service

import (
	"testing"
)

func TestBug10_ShutdownAndCloseAreIdempotent(t *testing.T) {
	_, records, closeDB := testRepositories(t)
	defer closeDB()
	svc := NewRecordService(records, 8, nil)
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("record service shutdown panicked: %v", recovered)
		}
	}()
	svc.Shutdown()
	svc.Close()
}
