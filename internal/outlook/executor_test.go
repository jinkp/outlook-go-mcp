//go:build windows

package outlook

import (
	"context"
	"errors"
	"sync/atomic"
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

func TestCOMExecutorRetriesOnCOMFailure(t *testing.T) {
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	var callCount atomic.Int32
	err := executor.Submit(context.Background(), func() error {
		n := callCount.Add(1)
		if n < 3 {
			return ErrCOMFailure
		}
		return nil // succeed on 3rd attempt
	})

	if err != nil {
		t.Fatalf("Submit() error = %v, want nil (should succeed on retry)", err)
	}

	if got := callCount.Load(); got != 3 {
		t.Fatalf("job called %d times, want 3", got)
	}

	// Session should have been closed+reconnected between retries.
	if session.closeCalls < 2 {
		t.Fatalf("closeCalls = %d, want >= 2 (reconnect between retries)", session.closeCalls)
	}
}

func TestCOMExecutorDoesNotRetryNonCOMErrors(t *testing.T) {
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	var callCount atomic.Int32
	wantErr := ErrInvalidParams
	err := executor.Submit(context.Background(), func() error {
		callCount.Add(1)
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Submit() error = %v, want %v", err, wantErr)
	}

	if got := callCount.Load(); got != 1 {
		t.Fatalf("job called %d times, want 1 (no retry for non-COM errors)", got)
	}
}

func TestCOMExecutorReturnsErrorAfterMaxRetries(t *testing.T) {
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	var callCount atomic.Int32
	err := executor.Submit(context.Background(), func() error {
		callCount.Add(1)
		return ErrCOMFailure
	})

	if !errors.Is(err, ErrCOMFailure) {
		t.Fatalf("Submit() error = %v, want %v", err, ErrCOMFailure)
	}

	if got := callCount.Load(); got != int32(maxCOMRetries) {
		t.Fatalf("job called %d times, want %d (maxCOMRetries)", got, maxCOMRetries)
	}
}

func TestCOMExecutorRetriesOnConnectFailureThenSucceeds(t *testing.T) {
	session := &fakeOutlookSession{
		connectErr: ErrCOMFailure,
	}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	// After 2 failed connects, allow it to succeed.
	go func() {
		time.Sleep(1500 * time.Millisecond)
		session.connectErr = nil
	}()

	err := executor.Submit(context.Background(), func() error {
		return nil
	})

	if err != nil {
		t.Fatalf("Submit() error = %v, want nil (should succeed after connect retry)", err)
	}

	if session.connectCalls < 2 {
		t.Fatalf("connectCalls = %d, want >= 2", session.connectCalls)
	}
}

func TestCOMExecutorRecoversPanicAsCOMFailureAndRetries(t *testing.T) {
	session := &fakeOutlookSession{}
	executor := NewCOMExecutor(session)

	if err := executor.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer executor.Stop()

	var callCount atomic.Int32
	err := executor.Submit(context.Background(), func() error {
		n := callCount.Add(1)
		if n == 1 {
			panic("access violation")
		}
		return nil // succeed on 2nd attempt
	})

	if err != nil {
		t.Fatalf("Submit() error = %v, want nil (panic should be retried)", err)
	}

	if got := callCount.Load(); got != 2 {
		t.Fatalf("job called %d times, want 2", got)
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
