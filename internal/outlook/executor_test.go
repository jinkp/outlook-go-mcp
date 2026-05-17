//go:build windows

package outlook

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCOMExecutorStartCallsSessionConnect(t *testing.T) {
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	if session.connectCalls != 1 {
		t.Fatalf("connectCalls = %d, want 1", session.connectCalls)
	}
}

func TestCOMExecutorSubmitRunsJobAndReturnsResult(t *testing.T) {
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	wantErr := errors.New("job failed")
	err := executor.Submit(context.Background(), func() error {
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Submit() error = %v, want %v", err, wantErr)
	}
}

func TestCOMExecutorSubmitReturnsContextErrorWhenCancelledBeforeJobRuns(t *testing.T) {
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	blockerStarted := make(chan struct{})
	releaseBlocker := make(chan struct{})

	go func() {
		_ = executor.Submit(context.Background(), func() error {
			close(blockerStarted)
			<-releaseBlocker
			return nil
		})
	}()

	select {
	case <-blockerStarted:
	case <-time.After(time.Second):
		t.Fatal("blocking job did not start")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := executor.Submit(ctx, func() error {
		return nil
	})
	close(releaseBlocker)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Submit() error = %v, want %v", err, context.Canceled)
	}
}

func TestCOMExecutorStopCausesSubsequentSubmitToFail(t *testing.T) {
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	executor.Stop()

	err := executor.Submit(context.Background(), func() error {
		return nil
	})

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("Submit() error = %v, want %v", err, ErrNotConnected)
	}
}

type fakeOutlookSession struct {
	connectCalls int
	closeCalls   int
	connected    bool
	connectErr   error
	closeErr     error
}

func (s *fakeOutlookSession) Connect() error {
	s.connectCalls++
	if s.connectErr != nil {
		return s.connectErr
	}
	s.connected = true
	return nil
}

func (s *fakeOutlookSession) Close() error {
	s.closeCalls++
	s.connected = false
	return s.closeErr
}

func (s *fakeOutlookSession) IsConnected() bool {
	return s.connected
}
