package asr

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStopTranscribeWorkerWaitsForExit(t *testing.T) {
	segments := make(chan segmentChunk)
	workerDone := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		stopTranscribeWorker(segments, workerDone)
		close(stopped)
	}()

	select {
	case _, ok := <-segments:
		if ok {
			t.Fatal("segments channel remained open")
		}
	case <-time.After(time.Second):
		t.Fatal("segments channel was not closed")
	}

	select {
	case <-stopped:
		t.Fatal("worker cleanup returned before worker exit")
	default:
	}

	close(workerDone)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("worker cleanup did not finish")
	}
}

func TestWaitForRetryStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := waitForRetry(ctx, time.Minute)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForRetry error=%v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("waitForRetry took %s after cancellation", elapsed)
	}
}
