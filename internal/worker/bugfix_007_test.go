package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type bug07Job struct{ runs atomic.Int32 }

func (j *bug07Job) Name() string { return "bug07" }
func (j *bug07Job) Run(ctx context.Context) error {
	j.runs.Add(1)
	<-ctx.Done()
	return nil
}

func TestBug07_SchedulerStartsOneLoopPerJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	job := &bug07Job{}
	scheduler := NewScheduler(nil)
	scheduler.Add(job, time.Millisecond)
	scheduler.Start(ctx)
	deadline := time.Now().Add(time.Second)
	for job.runs.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(5 * time.Millisecond)
	if got := job.runs.Load(); got != 1 {
		cancel()
		scheduler.Wait()
		t.Fatalf("job started %d concurrent loops, want 1", got)
	}
	cancel()
	scheduler.Wait()
}
