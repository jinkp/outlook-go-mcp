//go:build windows

package outlook

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCOMExecutorStartDoesNotConnectEagerly(t *testing.T) {
	// Lazy connect: Start() must not call session.Connect().
	// The connection happens on the first Submit() call.
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	if session.connectCalls != 0 {
		t.Fatalf("connectCalls = %d after Start(), want 0 (lazy connect)", session.connectCalls)
	}
}

func TestCOMExecutorSubmitConnectsLazilyOnFirstCall(t *testing.T) {
	// First Submit() should trigger Connect() exactly once.
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	_ = executor.Submit(context.Background(), func() error { return nil })

	if session.connectCalls != 1 {
		t.Fatalf("connectCalls = %d after first Submit(), want 1", session.connectCalls)
	}

	// Second Submit() must NOT reconnect.
	_ = executor.Submit(context.Background(), func() error { return nil })
	if session.connectCalls != 1 {
		t.Fatalf("connectCalls = %d after second Submit(), want still 1", session.connectCalls)
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
