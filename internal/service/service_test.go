package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"mockbridge/internal/database"
	"mockbridge/internal/model"
	"mockbridge/internal/repository"
)

func testRepositories(t *testing.T) (*repository.MockRepository, *repository.RecordRepository, func()) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "service.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(context.Background(), db); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return repository.NewMockRepository(db), repository.NewRecordRepository(db), func() { db.Close() }
}

func TestScenarioDefaultsArePersisted(t *testing.T) {
	mocks, _, closeDB := testRepositories(t)
	defer closeDB()
	ctx := context.Background()
	if _, err := mocks.CreateContract(ctx, model.ContractInput{Key: "defaults", Name: "Defaults", Enabled: true, Priority: 100}); err != nil {
		t.Fatal(err)
	}
	scenarios, err := NewScenarioService(ctx, mocks)
	if err != nil {
		t.Fatal(err)
	}
	got, err := scenarios.Create(ctx, "defaults", model.ScenarioInput{Name: " normalized ", MatchRules: model.MatchRules{Method: " get ", Path: "/api/mock/defaults"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "normalized" || got.MatchRules.Method != "GET" || got.Response.Status != 200 {
		t.Fatalf("defaults not persisted: %+v", got)
	}
	if got.Fault.Status != 500 || !json.Valid([]byte(got.Fault.Body)) {
		t.Fatalf("invalid fault defaults: %+v", got.Fault)
	}
	if got.Response.Headers == nil || got.Fault.OnCalls == nil {
		t.Fatalf("response and fault defaults must be empty arrays: response=%#v calls=%#v", got.Response.Headers, got.Fault.OnCalls)
	}
}

func TestRecordSubmitAndCloseAreConcurrentSafe(t *testing.T) {
	_, records, closeDB := testRepositories(t)
	defer closeDB()
	svc := NewRecordService(records, 256, slog.New(slog.NewTextHandler(io.Discard, nil)))
	start := make(chan struct{})
	var submitters sync.WaitGroup
	for worker := 0; worker < 24; worker++ {
		submitters.Add(1)
		go func(worker int) {
			defer submitters.Done()
			<-start
			for n := 0; n < 80; n++ {
				svc.Submit(model.CallRecord{RequestID: "race", Method: "GET", Path: "/api/mock/race", ResponseStatus: 200, CreatedAt: time.Now()})
			}
		}(worker)
	}
	close(start)
	var closers sync.WaitGroup
	for i := 0; i < 3; i++ {
		closers.Add(1)
		go func() { defer closers.Done(); svc.Close() }()
	}
	submitters.Wait()
	closers.Wait()
	svc.Submit(model.CallRecord{RequestID: "after-close"})
}

func TestRawQueryAndDelayLimit(t *testing.T) {
	raw := "b=two+words&a=1&a=x%26y"
	record := baseRecord(MockRequest{Method: "GET", Path: "/x", RawQuery: raw}, time.Now())
	if record.QueryString != raw {
		t.Fatalf("query=%q want %q", record.QueryString, raw)
	}
	if got := chooseDelay(model.DelayDef{FixedMS: 20_000}); got != 10_000 {
		t.Fatalf("fixed delay=%d", got)
	}
	if got := chooseDelay(model.DelayDef{MinMS: 15_000, MaxMS: 20_000}); got != 10_000 {
		t.Fatalf("range delay=%d", got)
	}
}

func TestFaultBodyTemplateRenderingAndFallback(t *testing.T) {
	for _, tc := range []struct {
		name      string
		faultBody string
		wantBody  string
	}{
		{name: "rendered", faultBody: `{"id":"{{path.id}}","count":{{state.counter}}}`, wantBody: `{"id":"42","count":1}`},
		{name: "invalid template falls back to configured body", faultBody: `{{oops`, wantBody: `{{oops`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mocks, _, closeDB := testRepositories(t)
			defer closeDB()
			ctx := context.Background()
			contract, err := mocks.CreateContract(ctx, model.ContractInput{Key: "fault", Name: "Fault", Enabled: true, Priority: 100})
			if err != nil {
				t.Fatal(err)
			}
			scenarios, err := NewScenarioService(ctx, mocks)
			if err != nil {
				t.Fatal(err)
			}
			_, err = scenarios.Update(ctx, contract.Key, contract.DefaultScenarioID, model.ScenarioInput{
				Name:       "default",
				MatchRules: model.MatchRules{Method: "GET", Path: "/api/mock/fault/{id}"},
				Response:   model.ResponseDef{Status: 200, Body: `{}`},
				Fault:      model.FaultDef{Enabled: true, Status: 503, Rate: 1, Body: tc.faultBody},
			})
			if err != nil {
				t.Fatal(err)
			}
			got := NewMockService(scenarios).Process(ctx, MockRequest{Method: "GET", Path: "/api/mock/fault/42"})
			if got.Status != 503 || string(got.Body) != tc.wantBody || !got.Record.InjectedFault {
				t.Fatalf("status=%d body=%q record=%+v", got.Status, got.Body, got.Record)
			}
		})
	}
}
