package worker

import (
	"context"
	"testing"
	"time"
)

type bug03Job struct{}

func (bug03Job) Name() string { return "bug03" }
func (bug03Job) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func TestBug03_WaitDoesNotReturnBeforeWorkerRegistration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	scheduler := NewScheduler(nil)
	scheduler.Add(bug03Job{}, time.Hour)
	scheduler.Start(ctx)
	waitDone := make(chan struct{})
	go func() {
		scheduler.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
		t.Fatal("Wait returned before the started worker was registered")
	case <-time.After(10 * time.Millisecond):
	}
	cancel()
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after cancellation")
	}
}
