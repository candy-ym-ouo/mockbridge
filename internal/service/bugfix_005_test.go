package service

import (
	"context"
	"errors"
	"testing"

	"mockbridge/internal/repository"
)

func TestBug05_RecordNotFoundSurvivesServiceWrapping(t *testing.T) {
	_, records, closeDB := testRepositories(t)
	defer closeDB()
	_, err := NewRecordService(records, 8, nil).Get(context.Background(), 999999)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("errors.Is(%v, ErrNotFound)=false", err)
	}
}
