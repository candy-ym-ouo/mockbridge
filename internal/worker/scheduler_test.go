package worker

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"

	"mockbridge/internal/database"
	"mockbridge/internal/model"
	"mockbridge/internal/repository"
	"mockbridge/internal/service"
	"testing"
	"time"
)

type testJob struct {
	runs chan struct{}
	err  error
}

func (j testJob) Name() string              { return "test" }
func (j testJob) Run(context.Context) error { j.runs <- struct{}{}; return j.err }
func TestSchedulerAndCron(t *testing.T) {
	cases := []struct {
		expr string
		ok   bool
	}{{"* * * * *", true}, {"*/5 * * * *", true}, {"1 * * * *", false}, {"bad", false}, {"0-10 * * * *", true}}
	now := time.Date(2026, 8, 23, 10, 5, 0, 0, time.UTC)
	for _, c := range cases {
		if got := CronMatch(c.expr, now); got != c.ok {
			t.Errorf("%s=%v", c.expr, got)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	runs := make(chan struct{}, 2)
	s := NewScheduler(slog.Default())
	s.Add(testJob{runs: runs, err: errors.New("boom")}, 5*time.Millisecond)
	s.Start(ctx)
	select {
	case <-runs:
	case <-time.After(time.Second):
		t.Fatal("job not run")
	}
	time.Sleep(time.Millisecond)
	if len(s.Status()) != 1 {
		t.Fatal("missing status")
	}
	cancel()
	s.Wait()
}

func TestJobs(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "jobs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err = database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	mockRepo := repository.NewMockRepository(db)
	recordRepo := repository.NewRecordRepository(db)
	scenarios, err := service.NewScenarioService(ctx, mockRepo)
	if err != nil {
		t.Fatal(err)
	}
	contracts := service.NewContractService(mockRepo, scenarios)
	c, err := contracts.Create(ctx, model.ContractInput{Key: "jobs/test", Name: "jobs", Enabled: true, Priority: 100})
	if err != nil {
		t.Fatal(err)
	}
	second, err := scenarios.Create(ctx, c.Key, model.ScenarioInput{Name: "cron", MatchRules: model.MatchRules{Method: "GET", Path: "/api/mock/jobs"}, Response: model.ResponseDef{Status: 200}, Fault: model.FaultDef{Status: 500}, Switch: model.SwitchDef{SwitchToScenario: "default", Cron: "* * * * *"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = scenarios.Activate(ctx, c.Key, second.ID, "manual"); err != nil {
		t.Fatal(err)
	}
	records := service.NewRecordService(recordRepo, 8, slog.Default())
	defer records.Close()
	records.Submit(model.CallRecord{RequestID: "old", Method: "GET", Path: "/x", CreatedAt: time.Now().Add(-48 * time.Hour)})
	records.Submit(model.CallRecord{RequestID: "new", Method: "GET", Path: "/x", ContractKey: c.Key, Matched: true, ResponseStatus: 200, TotalMS: 10, CreatedAt: time.Now()})
	time.Sleep(250 * time.Millisecond)
	if err = (CleanerJob{Records: records, Retention: 24 * time.Hour}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if err = (StatsJob{Records: records}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	job := NewSwitcherJob(scenarios)
	if err = job.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if scenarios.Snapshots()[0].Scenario.Name != "default" {
		t.Fatal("cron switch did not activate default")
	}
}
