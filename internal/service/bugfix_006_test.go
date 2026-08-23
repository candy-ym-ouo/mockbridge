package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mockbridge/internal/model"
)

func TestBug06_SubmitRequestKeepsConcurrentRecordsDistinct(t *testing.T) {
	_, records, closeDB := testRepositories(t)
	svc := NewRecordService(records, 512, nil)
	defer func() {
		svc.Close()
		closeDB()
	}()
	const total = 64
	done := make(chan struct{}, total)
	for i := 0; i < total; i++ {
		go func(i int) {
			svc.SubmitRequest(context.Background(), model.CallRecord{RequestID: fmt.Sprintf("bug06-%d", i), Method: "GET", Path: "/bug06", CreatedAt: time.Now()})
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < total; i++ {
		<-done
	}
	svc.Close()
	page, err := svc.Query(context.Background(), model.RecordQuery{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, record := range page.List {
		seen[record.RequestID] = true
	}
	if len(seen) != total {
		t.Fatalf("got %d distinct request IDs, want %d", len(seen), total)
	}
}
